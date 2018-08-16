package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	pb "github.com/nre-learning/syringe/api/exp/generated"
	scheduler "github.com/nre-learning/syringe/scheduler"
)

func (s *server) RequestLiveLab(ctx context.Context, lp *pb.LabParams) (*pb.LabUUID, error) {

	// TODO(mierdin): need to perform some basic security checks here. Need to check incoming IP address
	// and do some rate-limiting if possible. Alternatively you could perform this on the Ingress

	if lp.SessionId == "" {
		msg := "Session ID cannot be nil"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	// Identify lab definition - return error if doesn't exist by ID
	if _, ok := s.scheduler.LabDefs[lp.LabId]; !ok {
		log.Errorf("Couldn't find lab ID %d", lp.LabId)
		return &pb.LabUUID{}, errors.New("Failed to find referenced Lab ID")
	}

	// Check to see if it already exists in memory. If it does, don't send provision request.
	// Just look it up and send UUID
	log.Infof("Looking up session %s", lp.SessionId)
	if _, ok := s.sessions[lp.SessionId]; ok {
		if labUuid, ok := s.sessions[lp.SessionId][lp.LabId]; ok {

			log.Info("Found session")
			log.Info(s.sessions[lp.SessionId])
			return &pb.LabUUID{Id: labUuid}, nil
		}
	} else {

		// Doesn't exist, prep it with a map value
		s.sessions[lp.SessionId] = map[int32]string{}
	}

	// Generate UUID, make sure it doesn't conflict with another (unlikely but easy to check)
	var newUuid string
	for {
		newUuid = GenerateUUID()
		if _, ok := s.liveLabs[newUuid]; !ok {
			break
		}
	}

	// What if one session has multiple UUIDs? Might happen as they transition between lessons
	// At the very least you should store a second map inside this map, which contains a map of lab IDs for a given session, and then map those to UUIDs
	//
	// TODO(mierdin): consider not having any tables in memory at all. Just make everything function off of namespace names
	// and literally store all state in kubernetes
	//
	// Ensure sessions table is updated with the new session
	s.sessions[lp.SessionId][lp.LabId] = newUuid

	// 3 - if doesn't already exist, put together schedule request and send to channel
	s.scheduler.Requests <- &scheduler.LabScheduleRequest{
		LabDef:    s.scheduler.LabDefs[lp.LabId],
		Operation: scheduler.OperationType_CREATE,
		Uuid:      newUuid,
		Session:   lp.SessionId,
	}

	return &pb.LabUUID{Id: newUuid}, nil
}

func (s *server) DeleteLiveLab(ctx context.Context, lp *pb.LabParams) (*pb.LiveLab, error) {

	// TODO(mierdin): need to perform some security checks here

	if _, ok := s.scheduler.LabDefs[lp.LabId]; !ok {
		return &pb.LiveLab{}, errors.New("Failed to find referenced Lab ID")
	}

	if _, ok := s.sessions[lp.SessionId]; !ok {
		return &pb.LiveLab{}, errors.New("No existing session found to delete")
	}

	if _, ok := s.sessions[lp.SessionId][lp.LabId]; !ok {
		return &pb.LiveLab{}, errors.New("Session exists but isn't currently using the requested lab ID")
	}

	// Delete the session
	delete(s.sessions, lp.SessionId)

	s.scheduler.Requests <- &scheduler.LabScheduleRequest{
		LabDef:    s.scheduler.LabDefs[lp.LabId],
		Operation: scheduler.OperationType_DELETE,
		Uuid:      s.sessions[lp.SessionId][lp.LabId],
		Session:   lp.SessionId,
	}

	return &pb.LiveLab{}, nil
}

func (s *server) GetLiveLab(ctx context.Context, uuid *pb.LabUUID) (*pb.LiveLab, error) {
	// port1, _ := strconv.Atoi(s.labs[0].LabConnections["csrx1"])

	if uuid.Id == "" {
		msg := "Lab UUID cannot be empty"
		log.Error(msg)
		return nil, errors.New(msg)
	}

	log.Infof("Looking up livelab %s", uuid.Id)
	if _, ok := s.liveLabs[uuid.Id]; !ok {
		return nil, errors.New("livelab not found")
	}

	log.Debug("CURRENT LIVELABS")
	log.Debug(s.liveLabs)

	log.Debugf("About to return %s", s.liveLabs[uuid.Id])

	// Return immediately without health check if we already know it's running
	if s.liveLabs[uuid.Id].Ready {
		return s.liveLabs[uuid.Id], nil
	}

	// For now, I'm doing a health check synchronous with the client calling getLiveLab. This will obviously incur a performance
	// hit the first few calls, but I'm mitigating this by updating the livelab in memory with the result, so that eventually,
	// after subsequent calls, the below conditional will return True and we won't have to check the status again.
	// Obviously this isn't ideal for making sure the lab is STILL running after a while, only that it's initially running.
	s.liveLabs[uuid.Id].Ready = isReady(s.liveLabs[uuid.Id])
	return s.liveLabs[uuid.Id], nil

}

func isReady(ll *pb.LiveLab) bool {
	for d := range ll.Endpoints {
		ep := ll.Endpoints[d]
		if isReachable(ep.Port) {
			log.Debugf("%s health check passed on port %d", ep.Name, ep.Port)
		} else {
			log.Debugf("%s health check failed on port %d", ep.Name, ep.Port)
			return false
		}
	}
	return true
}

func isReachable(port int32) bool {
	intPort := strconv.Itoa(int(port))
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("vip.labs.networkreliability.engineering:%s", intPort), 1*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	return true
}
