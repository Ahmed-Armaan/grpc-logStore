package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	pb "github.com/Ahmed-Armaan/grpc-logStore.git/logStore/logStore"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
	pb.UnimplementedLogserviceServer
	pb.UnimplementedPingServer
	listeners []chan *Log
	mu        sync.Mutex
}

var Logs = ListInit(nil)

func (s *server) Ping(ctx context.Context, req *pb.PingReq) (*pb.PongRes, error) {
	if strings.ToLower(req.GetPing()) == "ping" {
		return &pb.PongRes{Pong: "pong"}, nil
	}
	return nil, errors.New("invalid message")
}

func (s *server) PublishLog(ctx context.Context, req *pb.LogEntryReq) (*pb.LogEntryRes, error) {
	entry := req.GetEntry()
	if entry.GetMessage() == "" ||
		entry.GetLevel() < pb.Loglevel_warn || entry.GetLevel() > pb.Loglevel_critical ||
		entry.GetTimestamp() == nil {
		return nil, errors.New("invalid request")
	}

	insertingLog := &Log{
		Message:   entry.GetMessage(),
		Timestamp: *entry.GetTimestamp(),
		Details:   entry.GetDetails(),
		Level:     entry.GetLevel(),
	}

	Logs = Logs.InsertLog(insertingLog)

	s.mu.Lock()
	for _, ch := range s.listeners {
		select {
		case ch <- insertingLog:
		default:
			continue
		}
	}
	s.mu.Unlock()

	return &pb.LogEntryRes{Inserted: true}, nil
}

func (s *server) Getlog(req *pb.Logretrivalreq, stream pb.Logservice_GetlogServer) error {
	temp := Logs
	var count int
	switch {
	case req.GetLatest():
		count = 1
	case req.GetAll() || req.GetHead() <= 0:
		count = 1 << 30 // effectively unlimited
	default:
		count = int(req.GetHead())
	}

	ctx := stream.Context()

	for temp != nil && count > 0 {
		if temp.currLog.Message == "List Begin" {
			break
		}
		prev, currLog := temp.GetLog()

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := stream.Send(&pb.LogEntry{
				Message:   currLog.Message,
				Timestamp: &currLog.Timestamp,
				Details:   currLog.Details,
				Level:     currLog.Level,
			}); err != nil {
				return err
			}
		}

		temp = prev
		count--
	}

	return nil
}

func (s *server) Clearlog(ctx context.Context, _ *emptypb.Empty) (*pb.ClearRes, error) {
	Logs = Logs.DeleteAll()
	return &pb.ClearRes{Cleared: true}, nil
}

func (s *server) Listen(req *pb.ListenReq, stream pb.Logservice_ListenServer) error {
	ch := make(chan *Log, 10)
	s.mu.Lock()
	s.listeners = append(s.listeners, ch)
	s.mu.Unlock()

	level := req.GetLoglevel()
	ctx := stream.Context()

	defer func() {
		for idx, c := range s.listeners {
			if c == ch {
				s.mu.Lock()
				s.listeners = append(s.listeners[:idx], s.listeners[idx+1:]...)
				s.mu.Unlock()
				close(c)
				break
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case log := <-ch:
			if log.Level == pb.Loglevel(level) {
				if err := stream.Send(&pb.LogEntry{
					Message:   log.Message,
					Timestamp: &log.Timestamp,
					Details:   log.Details,
					Level:     log.Level,
				}); err != nil {
					return err
				}
			}
		}
	}
}

func main() {
	PORT := 8080
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", PORT))
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	pb.RegisterLogserviceServer(s, &server{})

	log.Printf("server running on port %d", PORT)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
