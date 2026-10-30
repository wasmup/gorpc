package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"runtime"
	"time"

	"google.golang.org/grpc"

	pb "app/event"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: slog.LevelInfo})))
	slog.Info(`Go`, `Version`, runtime.Version(), `OS`, runtime.GOOS, `ARCH`, runtime.GOARCH, `now`, time.Now(), `Local`, time.Local)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		slog.Error(`tcp listen`, `Err`, err)
		return
	}

	grpcServer := grpc.NewServer()
	pb.RegisterEventServiceServer(grpcServer, &eventServer{})

	err = grpcServer.Serve(lis)
	if err != nil {
		slog.Error(`grpc`, `Err`, err)
		return
	}
}

type eventServer struct {
	pb.UnimplementedEventServiceServer
}

func (s *eventServer) SendEvent(ctx context.Context, req *pb.Event) (*pb.SendEventResponse, error) {
	fmt.Println(req.GetName(), req.GetId())

	r := &pb.SendEventResponse{}
	r.SetStatus("OK")
	return r, nil
}
