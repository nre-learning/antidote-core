package kubernetes

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	ot "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"
	logrus "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/nre-learning/antidote-core/config"
	"github.com/nre-learning/antidote-core/db"
	models "github.com/nre-learning/antidote-core/db/models"
	"github.com/nre-learning/antidote-core/scheduler"
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
	NC     *nats.Conn
	Config config.AntidoteConfig
	Db     db.DataManager

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

	HealthChecker LessonHealthChecker
}

func NewKubernetesBackend(nc *nats.Conn, acfg config.AntidoteConfig, adb db.DataManager) (*KubernetesBackend, error) {

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
	clientCrd, err := crdclient.NewForConfig(kubeConfig) // Client for creating instances of the network CRD
	if err != nil {
		logrus.Fatalf("Unable to create new kubernetes crd client - %v", err)
	}

	// Start scheduler
	k := KubernetesBackend{
		KubeConfig:    kubeConfig,
		Client:        cs,
		ClientExt:     csExt,
		ClientCrd:     clientCrd,
		NC:            nc,
		Config:        acfg,
		Db:            adb,
		HealthChecker: &scheduler.LessonHealthCheck{},
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

func (k *KubernetesBackend) handleRequestCREATE(sc ot.SpanContext, newRequest services.LessonScheduleRequest) {
	span := ot.StartSpan("scheduler_lsr_create", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(k.Config.InstanceID, newRequest.LiveLessonID)
	span.LogEvent(fmt.Sprintf("Generated namespace name %s", nsName))

	ll, err := k.Db.GetLiveLesson(span.Context(), newRequest.LiveLessonID)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return
	}

	err = s.createK8sStuff(span.Context(), newRequest)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		_ = k.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		return
	}

	err = s.waitUntilReachable(span.Context(), ll)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		_ = k.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		return
	}

	err = k.Db.UpdateLiveLessonStatus(span.Context(), ll.ID, models.Status_CONFIGURATION)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		_ = k.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		return
	}

	err = k.ConfigureStuff(span.Context(), nsName, ll, newRequest)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		_ = k.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		return
	}

	lesson, err := k.Db.GetLesson(span.Context(), newRequest.LessonSlug)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return
	}

	span.LogEvent(fmt.Sprintf("Inserting ready delay of %d seconds", lesson.ReadyDelay))
	if lesson.ReadyDelay > 0 {
		time.Sleep(time.Duration(lesson.ReadyDelay) * time.Second)
	}

	_ = k.Db.UpdateLiveLessonStatus(span.Context(), ll.ID, models.Status_READY)

	// Inject span context and send LSR into NATS
	tracer := ot.GlobalTracer()
	var t services.TraceMsg
	if err := tracer.Inject(span.Context(), ot.Binary, &t); err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
	}
	reqBytes, _ := json.Marshal(newRequest)
	t.Write(reqBytes)
	s.NC.Publish(services.LsrCompleted, t.Bytes())
}

func (k *KubernetesBackend) handleRequestMODIFY(sc ot.SpanContext, newRequest services.LessonScheduleRequest) {
	span := ot.StartSpan("scheduler_lsr_modify", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(k.Config.InstanceID, newRequest.LiveLessonID)

	ll, err := k.Db.GetLiveLesson(span.Context(), newRequest.LiveLessonID)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return
	}

	err = k.ConfigureStuff(span.Context(), nsName, ll, newRequest)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		_ = k.Db.UpdateLiveLessonError(span.Context(), ll.ID, true)
		return
	}

	err = k.Db.UpdateLiveLessonStatus(span.Context(), ll.ID, models.Status_READY)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
	}

	err = s.boopNamespace(span.Context(), nsName)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
	}
}

func (k *KubernetesBackend) handleRequestBOOP(sc ot.SpanContext, newRequest services.LessonScheduleRequest) {
	span := ot.StartSpan("scheduler_lsr_boop", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(k.Config.InstanceID, newRequest.LiveLessonID)

	err := s.boopNamespace(span.Context(), nsName)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
	}
}

// handleRequestDELETE handles a livelesson deletion request by first sending a delete request
// for the corresponding namespace, and then cleaning up local state.
func (k *KubernetesBackend) handleRequestDELETE(sc ot.SpanContext, newRequest services.LessonScheduleRequest) {
	span := ot.StartSpan("scheduler_lsr_delete", ot.ChildOf(sc))
	defer span.Finish()

	nsName := generateNamespaceName(k.Config.InstanceID, newRequest.LiveLessonID)
	err := s.deleteNamespace(span.Context(), nsName)
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
	}

	// Make sure to not return earlier than this, so we make sure the state is cleaned up
	// no matter what
	_ = k.Db.DeleteLiveLesson(span.Context(), newRequest.LiveLessonID)
}

// createK8sStuff is a high-level workflow for creating all of the things necessary for a new instance
// of a livelesson. Pods, services, networks, networkpolicies, ingresses, etc to support a new running
// lesson are all created as part of this workflow.
func (k *KubernetesBackend) createK8sStuff(sc ot.SpanContext, req services.LessonScheduleRequest) error {
	span := ot.StartSpan("scheduler_k8s_create_stuff", ot.ChildOf(sc))
	defer span.Finish()

	ns, err := s.createNamespace(span.Context(), req)
	if err != nil {
		log.Error(err)
	}

	// Set network policy ONLY after configuration has had a chance to take place. Once this is in place,
	// only config pods spawned by Jobs will have internet access, so if this takes place earlier, lessons
	// won't initially come up at all.
	if !k.Config.AllowEgress {
		s.createNetworkPolicy(span.Context(), ns.Name)
	}

	// Sync TLS certificate into the lesson namespace (and optionally, docker pull credentials)
	_ = s.syncSecret(span.Context(), k.Config.SecretsNamespace, ns.ObjectMeta.Name, k.Config.TLSCertName)
	if k.Config.PullCredName != "" {
		_ = s.syncSecret(span.Context(), k.Config.SecretsNamespace, ns.ObjectMeta.Name, k.Config.PullCredName)
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
	if usesJupyterLabGuide(lesson) {

		jupyterEp := &models.LiveEndpoint{
			Name:  "jupyterlabguide",
			Image: fmt.Sprintf("antidotelabs/jupyter:%s", s.BuildInfo["imageVersion"]),
			Ports: []int32{8888},
		}

		// Add to the endpoints map as well as the ordered list, so the loop below picks it up at the end.
		ll.LiveEndpoints[jupyterEp.Name] = jupyterEp
		epOrdered = append(epOrdered, jupyterEp.Name)

		nsName := generateNamespaceName(k.Config.InstanceID, req.LiveLessonID)

		_, err := s.createIngress(
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
		_, err := s.createNetwork(span.Context(), c, fmt.Sprintf("%s-%s-net", connection.A, connection.B), connection.Subnet, req)
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
		newPod, err := s.createPod(span.Context(),
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
			svc, err := s.createService(
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
				_, err := s.createIngress(
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
				rdy, err := s.getPodStatus(pod)
				if err != nil {
					s.recordPodLogs(span.Context(), ll.ID, pod.Name, initContainerName)
					s.recordPodLogs(span.Context(), ll.ID, pod.Name, "")
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
			s.recordPodLogs(span.Context(), ll.ID, pod.Name, initContainerName)

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

func (s *AntidoteScheduler) configureStuff(sc ot.SpanContext, nsName string, ll models.LiveLesson, newRequest services.LessonScheduleRequest) error {
	span := ot.StartSpan("scheduler_configure_stuff", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("llID", ll.ID)
	span.LogFields(log.Object("llEndpoints", ll.LiveEndpoints))

	s.killAllJobs(span.Context(), nsName, "config")

	wg := new(sync.WaitGroup)
	wg.Add(len(ll.LiveEndpoints))
	allGood := true
	for i := range ll.LiveEndpoints {

		// Ignore any endpoints that don't have a configuration option
		if ll.LiveEndpoints[i].ConfigurationType == "" {
			wg.Done()
			continue
		}

		job, err := s.configureEndpoint(span.Context(), ll.LiveEndpoints[i], newRequest)
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
				completed, statusCount, err := s.getJobStatus(span, job, newRequest)
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
// This allows Syringe to pull lesson data from either Git, or from a local filesystem - the latter of which being very useful for lesson
// development.
func (s *AntidoteScheduler) getVolumesConfiguration(sc ot.SpanContext, lessonSlug string) ([]corev1.Volume, []corev1.VolumeMount, []corev1.Container, error) {
	span := ot.StartSpan("scheduler_get_volumes", ot.ChildOf(sc))
	defer span.Finish()

	lesson, err := s.Db.GetLesson(span.Context(), lessonSlug)
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
				strings.TrimPrefix(lesson.LessonDir, fmt.Sprintf("%s/", path.Clean(s.Config.CurriculumDir))),
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
				Path: s.Config.CurriculumDir,
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

// PurgeOldSessions cleans up old LiveSessions according to the configured LiveSessionTTL
func (s *AntidoteScheduler) PurgeOldSessions(sc ot.SpanContext) error {
	span := ot.StartSpan("scheduler_purgeoldsessions", ot.ChildOf(sc))
	defer span.Finish()

	lsList, err := s.Db.ListLiveSessions(span.Context())
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}

	lsTTL := time.Duration(s.Config.LiveSessionTTL) * time.Minute

	for _, ls := range lsList {
		createdTime := time.Since(ls.CreatedTime)

		// No need to continue if this session hasn't even exceeded the TTL
		if createdTime <= lsTTL {
			continue
		}

		llforls, err := s.getLiveLessonsForSession(span.Context(), ls.ID)
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

		err = s.Db.DeleteLiveSession(span.Context(), ls.ID)
		if err != nil {
			span.LogFields(log.Error(err))
			ext.Error.Set(span, true)
			return err
		}
	}

	return nil
}

// PurgeOldLessons identifies any kubernetes namespaces that are operating with our antidoteId,
// and among those, deletes the ones that have a lastAccessed timestamp that exceeds our configured
// TTL. This function is meant to be run in a loop within a goroutine, at a configured interval. Returns
// a slice of livelesson IDs to be deleted by the caller (not handled by this function)
func (k *KubernetesBackend) PurgeOldLessons(sc ot.SpanContext) ([]string, error) {
	span := ot.StartSpan("scheduler_purgeoldlessons", ot.ChildOf(sc))
	defer span.Finish()

	nameSpaces, err := k.Client.CoreV1().Namespaces().List(metav1.ListOptions{
		// VERY Important to use this label selector, otherwise you'll delete way more than you intended
		LabelSelector: fmt.Sprintf("antidoteManaged=yes,antidoteId=%s", k.Config.InstanceID),
	})
	if err != nil {
		return nil, err
	}

	// No need to GC if no matching namespaces exist
	if len(nameSpaces.Items) == 0 {
		span.LogFields(log.Int("gc_namespaces", 0))
		return []string{}, nil
	}

	liveLessonsToDelete := []string{}
	oldNameSpaces := []string{}
	for n := range nameSpaces.Items {

		i, err := strconv.ParseInt(nameSpaces.Items[n].ObjectMeta.Labels["lastAccessed"], 10, 64)
		if err != nil {
			return []string{}, err
		}
		lastAccessed := time.Unix(i, 0)

		if time.Since(lastAccessed) < time.Duration(k.Config.LiveLessonTTL)*time.Minute {
			continue
		}

		lsID := nameSpaces.Items[n].ObjectMeta.Labels["liveSession"]

		// An error from this function shouldn't impact the cleanup of this livelesson. The only reason we're checking
		// it here is so we can safely look at the "Persistent" field. Otherwise, an error here is moot.
		ls, err := k.Db.GetLiveSession(span.Context(), lsID)
		if err == nil {
			if ls.Persistent {
				span.LogEvent("Skipping GC, session marked persistent")
				continue
			}
		}

		liveLessonsToDelete = append(liveLessonsToDelete, nameSpaces.Items[n].ObjectMeta.Labels["liveLesson"])
		oldNameSpaces = append(oldNameSpaces, nameSpaces.Items[n].ObjectMeta.Name)
	}

	// No need to GC if no old namespaces exist
	if len(oldNameSpaces) == 0 {
		span.LogFields(log.Int("gc_namespaces", 0))
		return []string{}, nil
	}

	var wg sync.WaitGroup
	wg.Add(len(oldNameSpaces))
	for n := range oldNameSpaces {
		go func(ns string) {
			defer wg.Done()
			s.deleteNamespace(span.Context(), ns)
		}(oldNameSpaces[n])
	}
	wg.Wait()
	span.LogFields(log.Int("gc_namespaces", len(oldNameSpaces)))

	return liveLessonsToDelete, nil

}
