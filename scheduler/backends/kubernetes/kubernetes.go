package kubernetes

import (
	"errors"
	"fmt"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	ot "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"
	logrus "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/nre-learning/antidote-core/config"
	"github.com/nre-learning/antidote-core/db"
	models "github.com/nre-learning/antidote-core/db/models"
	reachability "github.com/nre-learning/antidote-core/scheduler/reachability"
	"github.com/nre-learning/antidote-core/services"

	// Custom Network CRD Types
	networkcrd "github.com/nre-learning/antidote-core/pkg/apis/k8s.cni.cncf.io/v1"

	// Kubernetes Types
	corev1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rest "k8s.io/client-go/rest"

	// Kubernetes clients
	kubernetesCrd "github.com/nre-learning/antidote-core/pkg/client/clientset/versioned"
	kubernetesExt "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubernetes "k8s.io/client-go/kubernetes"
)

type KubernetesBackend struct {
	Config    config.AntidoteConfig
	Db        db.DataManager
	BuildInfo map[string]string

	KubeConfig *rest.Config

	// Allows us to disable GC for testing. Production code should leave this at
	// false
	// DisableGC bool

	// Client for interacting with normal Kubernetes resources
	Client kubernetes.Interface

	// Client for creating CRD defintions
	ClientExt kubernetesExt.Interface

	// Client for creating instances of our network CRD
	ClientCrd kubernetesCrd.Interface
}

func NewKubernetesBackend(acfg config.AntidoteConfig, adb db.DataManager, bi map[string]string) (*KubernetesBackend, error) {

	var err error
	var kubeConfig *rest.Config
	if !acfg.K8sInCluster {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", acfg.K8sOutOfClusterConfigPath)
		if err != nil {
			logrus.Fatalf("Problem using external k8s configuration %s - %v", acfg.K8sOutOfClusterConfigPath, err)
		}
	} else {
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			logrus.Fatalf("Problem using in-cluster k8s configuration - %v", err)
		}
	}
	cs, err := kubernetes.NewForConfig(kubeConfig) // Client for working with standard kubernetes resources
	if err != nil {
		logrus.Fatalf("Unable to create new kubernetes client - %v", err)
	}
	csExt, err := kubernetesExt.NewForConfig(kubeConfig) // Client for creating new CRD definitions
	if err != nil {
		logrus.Fatalf("Unable to create new kubernetes ext client - %v", err)
	}
	clientCrd, err := kubernetesCrd.NewForConfig(kubeConfig) // Client for creating instances of the network CRD
	if err != nil {
		logrus.Fatalf("Unable to create new kubernetes crd client - %v", err)
	}

	k := KubernetesBackend{
		KubeConfig: kubeConfig,
		Client:     cs,
		ClientExt:  csExt,
		ClientCrd:  clientCrd,
		Config:     acfg,
		Db:         adb,

		BuildInfo: bi,
	}

	// Ensure our network CRD is in place (should fail silently if already exists)
	err = k.createNetworkCrd()
	if err != nil {
		return &KubernetesBackend{}, err
	}

	return &k, nil
}

const initContainerName string = "copy-local-files"

// NetworkCrdClient is an interface for the client for our custom
// network CRD. Allows for injection of mocks at test time.
type NetworkCrdClient interface {
	UpdateNamespace(string)
	Create(obj *networkcrd.NetworkAttachmentDefinition) (*networkcrd.NetworkAttachmentDefinition, error)
	Update(obj *networkcrd.NetworkAttachmentDefinition) (*networkcrd.NetworkAttachmentDefinition, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	Get(name string) (*networkcrd.NetworkAttachmentDefinition, error)
	List(opts meta_v1.ListOptions) (*networkcrd.NetworkList, error)
}

func (k *KubernetesBackend) HandleRequestCREATE(sc ot.SpanContext, newRequest services.LessonScheduleRequest) error {
	span := ot.StartSpan("scheduler_lsr_create", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(k.Config.InstanceID, newRequest.LiveLessonID)
	span.LogEvent(fmt.Sprintf("Generated namespace name %s", nsName))

	ll, err := k.Db.GetLiveLesson(span.Context(), newRequest.LiveLessonID)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}

	err = k.createK8sStuff(span.Context(), newRequest)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		_ = k.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		return err
	}

	err = reachability.WaitUntilReachable(span.Context(), k.Db, ll)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		_ = k.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)

		for _, ep := range ll.LiveEndpoints {
			k.recordPodLogs(span.Context(), ll.ID, ep.Name, "")
		}

		return err
	}

	err = k.Db.UpdateLiveLessonStatus(span.Context(), ll.ID, models.Status_CONFIGURATION)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		_ = k.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		return err
	}

	err = k.configureStuff(span.Context(), nsName, ll, newRequest)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		_ = k.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		return err
	}

	lesson, err := k.Db.GetLesson(span.Context(), newRequest.LessonSlug)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}

	span.LogEvent(fmt.Sprintf("Inserting ready delay of %d seconds", lesson.ReadyDelay))
	if lesson.ReadyDelay > 0 {
		time.Sleep(time.Duration(lesson.ReadyDelay) * time.Second)
	}

	_ = k.Db.UpdateLiveLessonStatus(span.Context(), ll.ID, models.Status_READY)

	return nil
}

func (k *KubernetesBackend) HandleRequestMODIFY(sc ot.SpanContext, newRequest services.LessonScheduleRequest) error {
	span := ot.StartSpan("scheduler_lsr_modify", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(k.Config.InstanceID, newRequest.LiveLessonID)

	ll, err := k.Db.GetLiveLesson(span.Context(), newRequest.LiveLessonID)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}

	err = k.configureStuff(span.Context(), nsName, ll, newRequest)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		_ = k.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		return err
	}

	err = k.Db.UpdateLiveLessonStatus(span.Context(), ll.ID, models.Status_READY)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}

	return nil
}

// HandleRequestDELETE handles a livelesson deletion request by first sending a delete request
// for the corresponding namespace, and then cleaning up local state.
func (k *KubernetesBackend) HandleRequestDELETE(sc ot.SpanContext, newRequest services.LessonScheduleRequest) error {
	span := ot.StartSpan("scheduler_lsr_delete", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(k.Config.InstanceID, newRequest.LiveLessonID)
	err := k.deleteNamespace(span.Context(), nsName)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}

	// Make sure to not return earlier than this, so we make sure the state is cleaned up
	// no matter what
	_ = k.Db.DeleteLiveLesson(span.Context(), newRequest.LiveLessonID)

	return nil
}

// createK8sStuff is a high-level workflow for creating all of the things necessary for a new instance
// of a livelesson. Pods, services, networks, networkpolicies, ingresses, etc to support a new running
// lesson are all created as part of this workflow.
func (k *KubernetesBackend) createK8sStuff(sc ot.SpanContext, req services.LessonScheduleRequest) error {
	span := ot.StartSpan("scheduler_k8s_create_stuff", ot.ChildOf(sc))
	defer span.Finish()

	ns, err := k.createNamespace(span.Context(), req)
	if err != nil {
		log.Error(err)
	}

	// Set network policy ONLY after configuration has had a chance to take place. Once this is in place,
	// only config pods spawned by Jobs will have internet access, so if this takes place earlier, lessons
	// won't initially come up at all.
	if !k.Config.AllowEgress {
		k.createNetworkPolicy(span.Context(), ns.Name)
	}

	// Sync TLS certificate into the lesson namespace (and optionally, docker pull credentials)
	_ = k.syncSecret(span.Context(), k.Config.SecretsNamespace, ns.ObjectMeta.Name, k.Config.TLSCertName)
	if k.Config.PullCredName != "" {
		_ = k.syncSecret(span.Context(), k.Config.SecretsNamespace, ns.ObjectMeta.Name, k.Config.PullCredName)
	}

	lesson, err := k.Db.GetLesson(span.Context(), req.LessonSlug)
	if err != nil {
		return err
	}

	ll, err := k.Db.GetLiveLesson(span.Context(), req.LiveLessonID)
	if err != nil {
		return err
	}

	// The LiveEndpoints field of the LiveLesson model is a map, not a slice. This means that the original order from the lesson definition
	// is lost when the API's initializeLiveEndpoints function populates this from the original slice into the unordered LiveLesson map.
	// Unfortunately this map is pretty well embedded in the API at this point, and would be a lot of work on antidote-core and antidote-web
	// to convert this to an ordered data structure like a slice.
	//
	// In most cases, this order doesn't really matter. This code however, is a huge exception - the order in which endpoint pods are created
	// is extremely important, as this is what determines which pods get which IP addresses from the specified subnet.
	//
	// Because this is the only place I'm currently aware of that this order matters, I've inserted this simple logic to create a slice of strings that
	// represent the original order in which these endpoints appeared, and the subsequent loop can use this to create pods in the original order from
	// the lesson definition. HOWEVER, if we discover that other use cases exist that require consistent ordering, we should tackle the conversion
	// of this field to a slice, and remove this workaround.
	epOrdered := []string{}
	for _, lep := range lesson.Endpoints {
		epOrdered = append(epOrdered, lep.Name)
	}

	// Append endpoint and create ingress for jupyter lab guide if necessary
	if models.UsesJupyterLabGuide(lesson) {

		jupyterEp := &models.LiveEndpoint{
			Name:  "jupyterlabguide",
			Image: fmt.Sprintf("antidotelabs/jupyter:%s", k.BuildInfo["imageVersion"]),
			Ports: []int32{8888},
		}

		// Add to the endpoints map as well as the ordered list, so the loop below picks it up at the end.
		ll.LiveEndpoints[jupyterEp.Name] = jupyterEp
		epOrdered = append(epOrdered, jupyterEp.Name)

		nsName := generateNamespaceName(k.Config.InstanceID, req.LiveLessonID)

		_, err := k.createIngress(
			span.Context(),
			ns.ObjectMeta.Name,
			jupyterEp,
			&models.LivePresentation{
				Name:      "web",
				Port:      8888,
				HepDomain: fmt.Sprintf("%s-jupyterlabguide-web.%s", nsName, k.Config.HEPSDomain),
			},
		)
		if err != nil {
			return fmt.Errorf("Unable to create ingress resource - %v", err)
		}
	}

	// Create networks from connections property
	for c := range lesson.Connections {
		connection := lesson.Connections[c]
		_, err := k.createNetwork(span.Context(), c, fmt.Sprintf("%s-%s-net", connection.A, connection.B), connection.Subnet, req)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	createdPods := make(map[string]*corev1.Pod)

	// Create pods and services
	for _, epName := range epOrdered {
		ep := ll.LiveEndpoints[epName]

		// createPod doesn't try to ensure a certain pod status. That's done later
		newPod, err := k.createPod(span.Context(),
			ep,
			getMemberNetworks(ep.Name, lesson.Connections),
			req,
		)
		if err != nil {
			log.Error(err)
			return err
		}

		createdPods[newPod.ObjectMeta.Name] = newPod

		// Expose via service if needed
		if len(newPod.Spec.Containers[0].Ports) > 0 {
			svc, err := k.createService(
				span.Context(),
				newPod,
				req,
			)
			if err != nil {
				log.Error(err)
				return err
			}

			// Update livelesson liveendpoint with cluster IP
			k.Db.UpdateLiveLessonEndpointIP(span.Context(), ll.ID, ep.Name, svc.Spec.ClusterIP)
			if err != nil {
				span.LogFields(log.Error(err))
				ext.Error.Set(span, true)
				return err
			}

		}

		// Create ingresses for http presentations
		for pr := range ep.Presentations {
			p := ep.Presentations[pr]

			if p.Type == "http" {
				_, err := k.createIngress(
					span.Context(),
					ns.ObjectMeta.Name,
					ep,
					p,
				)
				if err != nil {
					log.Error(err)
					return err
				}
			}
		}
	}

	err = k.Db.UpdateLiveLessonStatus(span.Context(), ll.ID, models.Status_BOOTING)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}

	// Before moving forward with network-based health checks, let's look back at the pods
	// we've deployed, and wait until they're in a "Running" status. This allows us to keep a hold
	// of maximum amounts of context for troubleshooting while we have it
	wg := new(sync.WaitGroup)
	wg.Add(len(createdPods))
	cp := &sync.Mutex{}

	failLesson := false

	for name, pod := range createdPods {
		go func(sc ot.SpanContext, name string, pod *corev1.Pod) {
			span := ot.StartSpan("scheduler_pod_status", ot.ChildOf(sc))
			defer span.Finish()
			span.SetTag("podName", name)
			defer wg.Done()

			for i := 0; i < 150; i++ {
				rdy, err := k.getPodStatus(pod)
				if err != nil {
					k.recordPodLogs(span.Context(), ll.ID, pod.Name, initContainerName)
					k.recordPodLogs(span.Context(), ll.ID, pod.Name, "")
					failLesson = true
					return
				}

				if rdy {
					cp.Lock()
					delete(createdPods, name)
					cp.Unlock()
					return
				}

				time.Sleep(2 * time.Second)
			}

			// We would only get to this point if the pod failed to start in the first place.
			// One potential reason for this is a failure in the init container, so we should attempt
			// to gather those logs.
			k.recordPodLogs(span.Context(), ll.ID, pod.Name, initContainerName)

			err = fmt.Errorf("Timed out waiting for %s to start", name)
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			failLesson = true
			return
		}(span.Context(), name, pod)
	}

	wg.Wait()

	// At this point, the only pods left in createdPods should be ones that failed to ready
	if failLesson || len(createdPods) > 0 {

		failedPodNames := []string{}
		for k := range createdPods {
			failedPodNames = append(failedPodNames, k)
		}

		span.LogFields(log.Object("failedPodNames", failedPodNames))
		ext.Error.Set(span, true)
		return errors.New("Some pods failed to start")
	}

	return nil
}

func (k *KubernetesBackend) configureStuff(sc ot.SpanContext, nsName string, ll models.LiveLesson, newRequest services.LessonScheduleRequest) error {
	span := ot.StartSpan("scheduler_configure_stuff", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("llID", ll.ID)
	span.LogFields(log.Object("llEndpoints", ll.LiveEndpoints))

	k.killAllJobs(span.Context(), nsName, "config")

	wg := new(sync.WaitGroup)
	wg.Add(len(ll.LiveEndpoints))
	allGood := true
	for i := range ll.LiveEndpoints {

		// Ignore any endpoints that don't have a configuration option
		if ll.LiveEndpoints[i].ConfigurationType == "" {
			wg.Done()
			continue
		}

		job, err := k.configureEndpoint(span.Context(), ll.LiveEndpoints[i], newRequest)
		if err != nil {
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			return err
		}

		go func() {
			defer wg.Done()

			oldStatusCount := map[string]int32{
				"active":    0,
				"succeeded": 0,
				"failed":    0,
			}
			for i := 0; i < 600; i++ {
				completed, statusCount, err := k.getJobStatus(span, job, newRequest)
				if err != nil {
					allGood = false
					return
				}

				// The use of this map[string]int32 and comparing old with new using DeepEqual
				// allows us to only log changes in status, rather than the periodic spam
				if !reflect.DeepEqual(oldStatusCount, statusCount) {
					span.LogFields(
						log.String("jobName", job.Name),
						log.Object("statusCount", statusCount),
					)
				}

				if completed {
					return
				}
				oldStatusCount = statusCount
				time.Sleep(1 * time.Second)
			}
			allGood = false
			return
		}()

	}

	wg.Wait()

	if !allGood {
		return errors.New("Problem during configuration")
	}

	return nil
}

// getVolumesConfiguration returns a slice of Volumes, VolumeMounts, and init containers that should be used in all pod and job definitions.
// This allows Antidote to pull lesson data from either Git, or from a local filesystem - the latter of which being very useful for lesson
// development.
func (k *KubernetesBackend) getVolumesConfiguration(sc ot.SpanContext, lessonSlug string) ([]corev1.Volume, []corev1.VolumeMount, []corev1.Container, error) {
	span := ot.StartSpan("scheduler_get_volumes", ot.ChildOf(sc))
	defer span.Finish()

	lesson, err := k.Db.GetLesson(span.Context(), lessonSlug)
	if err != nil {
		return nil, nil, nil, err
	}

	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}
	initContainers := []corev1.Container{}

	// Init container will mount the host directory as read-only, and copy entire contents into an emptyDir volume
	initContainers = append(initContainers, corev1.Container{
		Name:  initContainerName,
		Image: "bash",
		Command: []string{
			"bash",
		},

		// In previous versions of this platform, we used the subPath parameter of the VolumeMount to specify the actual lesson directory (i.e. lessons/my-new-lesson) to make available
		// to the lesson endpoints at the /antidote location. This would result in something like /antidote/<lesson files> rather than /antidote/lessons/my-new-lesson/<lesson files>,
		// which was much more convenient to access within the lessons.
		//
		// However, some runtimes don't appear to handle subPath well, as shown here: https://github.com/kata-containers/runtime/issues/2812.
		//
		// Fortunately, we don't **actually** need the subPath field to accomplish the same goal. This field seems to be mostly useful in environments where you want
		// to use the same volume but provide different mount points to different containers or pods. In our case, all pods within a lesson should look the same, and
		// volumes are created at a pod level using emptyDir. So, instead of using subPath in the mount, we can just copy the correct subdirectory here in the init container,
		// and mount the whole volume in the main container. This volume will already be prepped with the relevant subdirectory.
		Args: []string{
			"-c",
			fmt.Sprintf(
				"ls -lha /antidote-ro && cp -r /antidote-ro/%s/* /antidote && adduser -D antidote && chown -R antidote:antidote /antidote && ls -lha /antidote",
				strings.TrimPrefix(lesson.LessonDir, fmt.Sprintf("%s/", path.Clean(k.Config.CurriculumDir))),
			),
		},

		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "host-volume",
				ReadOnly:  true,
				MountPath: "/antidote-ro",
			},
			{
				Name:      "local-copy",
				ReadOnly:  false,
				MountPath: "/antidote",
			},
		},
	})

	// Add outer host volume, should be mounted read-only
	volumes = append(volumes, corev1.Volume{
		Name: "host-volume",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: k.Config.CurriculumDir,
			},
		},
	})

	// Add inner container volume, should be mounted read-write so we can copy files into it
	volumes = append(volumes, corev1.Volume{
		Name: "local-copy",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	// Finally, mount local copy volume as read-write
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "local-copy",
		ReadOnly:  false,
		MountPath: "/antidote",
	})

	return volumes, volumeMounts, initContainers, nil

}

// PruneOldLiveSessions cleans up old LiveSessions according to the configured LiveSessionTTL
func (k *KubernetesBackend) PruneOldLiveSessions(sc ot.SpanContext) error {
	span := ot.StartSpan("scheduler_pruneoldsessions", ot.ChildOf(sc))
	defer span.Finish()

	lsList, err := k.Db.ListLiveSessions(span.Context())
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}

	lsTTL := time.Duration(k.Config.LiveSessionTTL) * time.Minute

	for _, ls := range lsList {
		createdTime := time.Since(ls.CreatedTime)

		// No need to continue if this session hasn't even exceeded the TTL
		if createdTime <= lsTTL {
			continue
		}

		llforls, err := k.Db.GetLiveLessonsForSession(span.Context(), ls.ID)
		if err != nil {
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			return err
		}

		// We don't want/need to clean up this session if there are active livelessons that are using it.
		if len(llforls) > 0 {
			continue
		}

		// TODO(mierdin): It would be pretty rare, but in the event that a livelesson is spun up between the request above
		// and the livesession deletion below, we would encounter the leak bug we saw in 0.6.0. It might be worth seeing if
		// you can lock things somehow between the two.

		err = k.Db.DeleteLiveSession(span.Context(), ls.ID)
		if err != nil {
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			return err
		}
	}

	return nil
}

// PruneOldLiveLessons queries the datamanager for expired livelessons that don't belong to persistent livesessions.
// Once a list of these IDs is obtained, we delete the corresponding Kubernetes namespace, and then delete the livelesson state.
func (k *KubernetesBackend) PruneOldLiveLessons(sc ot.SpanContext) error {
	span := ot.StartSpan("scheduler_pruneoldlessons", ot.ChildOf(sc))
	defer span.Finish()

	liveLessonsToDelete := []string{}
	oldNameSpaces := []string{}

	lls, err := k.Db.ListLiveLessons(span.Context())
	if err != nil {
		return err
	}

	for _, ll := range lls {
		if time.Since(ll.LastActiveTime) < time.Duration(k.Config.LiveLessonTTL)*time.Minute {
			// TODO - log?
			continue
		}

		// An error from this function shouldn't impact the cleanup of this livelesson. The only reason we're checking
		// it here is so we can safely look at the "Persistent" field. Otherwise, an error here is moot.
		ls, err := k.Db.GetLiveSession(span.Context(), ll.SessionID)
		if err == nil {
			if ls.Persistent {
				span.LogEvent("Skipping GC, session marked persistent")
				continue
			}
		}

		liveLessonsToDelete = append(liveLessonsToDelete, ll.ID)

	}

	for i := range liveLessonsToDelete {
		err := k.Db.DeleteLiveLesson(span.Context(), liveLessonsToDelete[i])
		if err != nil {
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(oldNameSpaces))
	for i := range liveLessonsToDelete {
		go func(llID string) {
			defer wg.Done()

			ns := generateNamespaceName(k.Config.InstanceID, llID)
			k.deleteNamespace(span.Context(), ns)

			err := k.Db.DeleteLiveLesson(span.Context(), llID)
			if err != nil {
				span.LogFields(log.Error(err))
				ext.Error.Set(span, true)
			}

		}(liveLessonsToDelete[i])
	}
	wg.Wait()
	span.LogFields(log.Int("gc_namespaces", len(oldNameSpaces)))

	return nil

}

// PruneOrphans seeks out all antidote-managed namespaces, and deletes them.
// This will effectively reset the cluster to a state with all of the remaining infrastructure
// in place, but no running lessons. Antidote doesn't manage itself, or any other Antidote services.
func (k *KubernetesBackend) PruneOrphans() error {

	span := ot.StartSpan("scheduler_prune_orphaned_ns")
	defer span.Finish()

	nameSpaces, err := k.Client.CoreV1().Namespaces().List(metav1.ListOptions{
		// VERY Important to use this label selector, otherwise you'll nuke way more than you intended
		LabelSelector: fmt.Sprintf("antidoteManaged=yes,antidoteId=%s", k.Config.InstanceID),
	})
	if err != nil {
		return err
	}

	// No need to nuke if no namespaces exist with our ID
	if len(nameSpaces.Items) == 0 {
		span.LogFields(log.Int("pruned_orphans", 0))
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(len(nameSpaces.Items))
	for n := range nameSpaces.Items {

		nsName := nameSpaces.Items[n].ObjectMeta.Name
		go func() {
			defer wg.Done()
			k.deleteNamespace(span.Context(), nsName)
		}()
	}
	wg.Wait()

	span.LogFields(log.Int("pruned_orphans", len(nameSpaces.Items)))
	return nil
}
