package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "app/event"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	grpcErrs = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "app_grpc_client_send_errs",
	}, []string{"msg"})

	responseTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "app_grpc_client_send",
		Buckets: []float64{0.0001, 0.0002, 0.0005, 0.001, 0.002, 0.005, 0.010, 0.020, 0.050, 0.1, 0.2, 0.5, 1, 2, 5}, // 16 buckets
	})
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: slog.LevelInfo})))
	slog.Info(`Go`, `Version`, runtime.Version(), `OS`, runtime.GOOS, `ARCH`, runtime.GOARCH, `now`, time.Now(), `Local`, time.Local)

	creds := insecure.NewCredentials()
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(creds))
	if err != nil {
		slog.Error(`msg`, `Err`, err)
		return
	}
	defer conn.Close()

	client := pb.NewEventServiceClient(conn)

	event := &pb.Event{}
	event.SetName("name")
	event.SetId("id")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	resp, err := client.SendEvent(ctx, event)
	d := time.Since(start)
	responseTime.Observe(d.Seconds())
	if err != nil {
		code := status.Code(err)
		grpcErrs.WithLabelValues(code.String()).Inc()

		slog.Error(`SendEvent`, `code`, code, `duration`, d, `Err`, err)
		return
	}

	fmt.Println("Response from server:", resp)
}
