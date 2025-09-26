package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/Ahmed-Armaan/grpc-logStore.git/logStore/logStore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func healthCheck(check pb.PingClient) {
	res, err := check.Ping(context.Background(), &pb.PingReq{Ping: "ping"})
	if err != nil {
		log.Fatal("Counld not reach server")
	}

	fmt.Println(res.Pong)
}

func publishLog(client pb.LogserviceClient, message string, level pb.Loglevel) {
	entry := &pb.LogEntry{
		Message:   message,
		Timestamp: timestamppb.Now(),
		Details:   "Test details",
		Level:     level,
	}

	res, err := client.PublishLog(context.Background(), &pb.LogEntryReq{Entry: entry})
	if err != nil {
		log.Fatalf("PublishLog failed: %v", err)
	}
	fmt.Println("Log inserted:", res.GetInserted())
}

func getLog(client pb.LogserviceClient, latest bool, all bool, head int) {
	stream, err := client.Getlog(context.Background(), &pb.Logretrivalreq{
		Latest: latest,
		All:    all,
		Head:   int32(head),
	})
	if err != nil {
		log.Fatalf("GetLog failed: %v", err)
	}

	for {
		logEntry, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("GetLog receive failed: %v", err)
		}
		fmt.Printf("Received log: %s [%d]\n", logEntry.GetMessage(), logEntry.GetLevel())
	}
}

func clearLog(client pb.LogserviceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := client.Clearlog(ctx, &emptypb.Empty{})
	if err != nil {
		log.Fatalf("ClearLog failed: %v", err)
	}

	if resp != nil {
		log.Printf("ClearLog success: %v", resp.Cleared)
	} else {
		log.Println("ClearLog returned nil response")
	}
}

func listen(client pb.LogserviceClient, level pb.Loglevel) {
	stream, err := client.Listen(context.Background(), &pb.ListenReq{
		Loglevel: int32(level),
	})
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}

	for {
		logEntry, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("GetLog receive failed: %v", err)
		}
		fmt.Printf("Received log: %s [%d]\n", logEntry.GetMessage(), logEntry.GetLevel())
	}
}

func main() {
	conn, err := grpc.NewClient("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	//check := pb.NewPingClient(conn)
	client := pb.NewLogserviceClient(conn)

	//healthCheck(check)
	publishLog(client, "message 1", 2)
	publishLog(client, "message 2", 0)
	getLog(client, true, false, -1)
	clearLog(client)
	listen(client, 2)
}
