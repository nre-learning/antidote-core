// TODO - all of the reachable stuff below should be mandated somehow. The question is, should this be moved into the backend but mandated through interfaces,
// or centralized, and mandate that all of the plugins to use it via documented convention?

package reachability

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	ot "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"

	db "github.com/nre-learning/antidote-core/db"
	models "github.com/nre-learning/antidote-core/db/models"
	"golang.org/x/crypto/ssh"
)

func isEpReachable(ep *models.LiveEndpoint) (bool, error) {

	// rTest is used to give structure to the reachability tests we want to run.
	// This function will first construct a slice of rTests based on information available in the
	// LiveLesson, and then will subsequently run tests based on each rTest.
	type rTest struct {
		name   string
		method string
		host   string
		port   int32
	}

	rTests := []rTest{}
	var mapMutex = &sync.Mutex{}
	reachableMap := map[string]bool{}
	for _, rt := range rTests {
		reachableMap[rt.name] = false
	}

	// If no presentations, add a single rTest using the first available Port.
	// (we aren't currently testing for all Ports)
	if len(ep.Presentations) == 0 {

		if len(ep.Ports) == 0 {
			// Should never happen due to earlier code, but good to be safe
			return false, errors.New("Endpoint has no Ports")
		}

		rTests = append(rTests, rTest{
			name:   ep.Name,
			method: "tcp",
			host:   ep.Host,
			port:   ep.Ports[0],
		})
	}
	for p := range ep.Presentations {
		rTests = append(rTests, rTest{
			name:   fmt.Sprintf("%s-%s", ep.Name, ep.Presentations[p].Name),
			method: string(ep.Presentations[p].Type),
			host:   ep.Host,
			port:   ep.Presentations[p].Port,
		})
	}

	// Last, iterate over the rTests and spawn goroutines for each test
	wg := new(sync.WaitGroup)
	wg.Add(len(rTests))
	for _, rt := range rTests {
		ctx := context.Background()

		// Timeout for an individual test
		ctx, _ = context.WithTimeout(ctx, 10*time.Second)
		go func(ctx context.Context, rt rTest) {
			defer wg.Done()

			testResult := false

			// Not currently doing an HTTP health check, but one could easily be added.
			// rt.method is already being set to "http" for corresponding presentations
			if rt.method == "ssh" {
				testResult = sshTest(rt.host, int(rt.port))
			} else {
				testResult = tcpTest(rt.host, int(rt.port))
			}

			mapMutex.Lock()
			defer mapMutex.Unlock()
			reachableMap[rt.name] = testResult

		}(ctx, rt)
	}

	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()

	select {
	case <-c:
		for _, r := range reachableMap {
			if !r {
				return false, nil
			}
		}
		return true, nil
	case <-time.After(time.Second * 5):
		return false, nil
	}

}

// WaitUntilReachable waits until an entire livelesson is reachable
func WaitUntilReachable(sc ot.SpanContext, dm db.DataManager, ll models.LiveLesson) error {
	span := ot.StartSpan("scheduler_wait_until_reachable", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("liveLessonID", ll.ID)
	span.LogFields(log.Object("liveEndpoints", ll.LiveEndpoints))

	// reachableTimeLimit controls how long we wait for each goroutine to finish
	// as well as in general how long we wait for all of them to finish. If this is exceeded,
	// the livelesson is marked as failed.
	reachableTimeLimit := time.Second * 600

	finishedEps := map[string]bool{}
	wg := new(sync.WaitGroup)
	wg.Add(len(ll.LiveEndpoints))
	for n := range ll.LiveEndpoints {
		ep := ll.LiveEndpoints[n]
		ctx := context.Background()
		ctx, _ = context.WithTimeout(ctx, reachableTimeLimit)

		go func(sc ot.SpanContext, ctx context.Context, ep *models.LiveEndpoint) {
			span := ot.StartSpan("scheduler_ep_reachable_test", ot.ChildOf(sc))
			defer span.Finish()
			span.SetTag("epName", ep.Name)
			span.SetTag("epSSHCreds", fmt.Sprintf("%s:%s", ep.SSHUser, ep.SSHPassword))

			defer wg.Done()
			for {
				epr, err := isEpReachable(ep)
				if err != nil {
					span.LogFields(log.Error(err))
					ext.Error.Set(span, true)
					return
				}
				if epr {
					finishedEps[ep.Name] = true
					_ = dm.UpdateLiveLessonTests(span.Context(), ll.ID, int32(len(finishedEps)), int32(len(ll.LiveEndpoints)))
					span.LogEvent("Endpoint has become reachable")
					return
				}

				select {
				case <-time.After(1 * time.Second):
					continue
				case <-ctx.Done():
					return
				}
			}
		}(span.Context(), ctx, ep)
	}

	// Wait for each endpoint's goroutine to finish, either through completion,
	// or through context cancelation due to timer expiring.
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		//
	case <-time.After(reachableTimeLimit):
		//
	}

	if len(finishedEps) != len(ll.LiveEndpoints) {
		err := errors.New("Timeout waiting for LiveEndpoints to become reachable")
		span.LogFields(
			log.Error(err),
			log.Object("failedEps", finishedEps),
		)
		ext.Error.Set(span, true)
		return err
	}

	return nil
}

// sshTest is an important health check to run especially for interactive endpoints,
// so that we know the endpoint is not only online but ready to receive SSH connections
// from the user via the Web UI
func sshTest(host string, port int) bool {
	strPort := strconv.Itoa(int(port))

	// Using made-up creds, since we only care that SSH is viable for this simple health test.
	sshConfig := &ssh.ClientConfig{
		User:            "foobar",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password("foobar"),
		},
		// TODO(mierdin): This still doesn't seem to work properly for "hung" ssh servers. Having to rely
		// on the outer select/case timeout at the moment.
		Timeout: time.Second * 2,
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", host, strPort), sshConfig)
	if err != nil {
		// For a simple health check, we only care that SSH is responding, not that auth is solid.
		// Thus the made-up creds. If we get this message, then all is good.
		if strings.Contains(err.Error(), "unable to authenticate") {
			return true
		}
		return false
	}
	defer conn.Close()

	return true
}

func tcpTest(host string, port int) bool {
	strPort := strconv.Itoa(int(port))
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, strPort), 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
