package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	pb "app/event"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

const broker = "localhost:9092"

func main() {
	topic := "events-topic"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	wg.Add(1)
	go Produce(ctx, topic, "producer1", &wg)

	wg.Add(1)
	go Produce(ctx, topic, "producer2", &wg)

	getTopicsAndUnreadMessages(broker)

	wg.Add(1)
	go consume(ctx, broker, topic, "consumer-group-1", &wg)

	wg.Add(1)
	go consume(ctx, broker, topic, "consumer-group-2", &wg)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop,
		syscall.SIGHUP,  // reconfigure
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGTERM, // Kubernetes best practices: https://cloud.google.com/blog/products/containers-kubernetes/kubernetes-best-practices-terminating-with-grace
		syscall.SIGQUIT)

	select {
	case sig := <-stop:
		slog.Info(`terminated by signal`, `signal`, sig)
		cancel()

	case <-ctx.Done():
		slog.Info(`terminated by context`)
	}

	wg.Wait()
	slog.Info(`App closed`)
}

func getTopicsAndUnreadMessages(broker string) (err error) {
	// Create an AdminClient to list topics
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": broker})
	if err != nil {
		slog.Error(`msg`, `Err`, err)
		return
	}
	defer adminClient.Close()

	// List metadata for all topics
	metadata, err := adminClient.GetMetadata(nil, false, 5000)
	if err != nil {
		slog.Error(`msg`, `Err`, err)
		return
	}

	// Create a Consumer to fetch offsets
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
		"group.id":          "offset-checker",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		slog.Error(`offset-checker`, `Err`, err)
		return
	}
	defer consumer.Close()

	fmt.Println("Available topics and unread message counts:")
	for topic, topicMetadata := range metadata.Topics {
		var totalUnreadMessages int64 = 0

		for _, partition := range topicMetadata.Partitions {
			// Get earliest and latest offsets for the partition
			low, high, err := consumer.QueryWatermarkOffsets(topic, partition.ID, 5000)
			if err != nil {
				slog.Error(`QueryWatermarkOffsets`, `Err`, err)
				continue
			}

			// Calculate unread messages
			unreadMessages := high - low
			totalUnreadMessages += unreadMessages
		}

		fmt.Printf("Topic: %s, Unread Messages: %d\n", topic, totalUnreadMessages)
	}

	return
}

func consume(ctx context.Context, broker, topic, groupId string, wg *sync.WaitGroup) {
	defer wg.Done()

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
		"group.id":          groupId,
		"auto.offset.reset": "earliest",
		// "session.timeout.ms": 1000,
	})
	if err != nil {
		slog.Error(`NewConsumer`, `Err`, err)
		return
	}
	defer c.Close()

	err = c.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		slog.Error(`SubscribeTopics`, `Err`, err)
		return
	}

	// Consume messages until context is cancelled
	for {
		select {
		case <-ctx.Done():
			return

		default:
			msg, err := c.ReadMessage(1 * time.Second)
			if err == nil {
				fmt.Printf("%s ==> %s\n", msg.TopicPartition, string(msg.Value))

				// Unmarshal the protobuf message
				event := &pb.Event{}
				err := proto.Unmarshal(msg.Value, event)
				if err != nil {
					slog.Error(`Unmarshal`, `Err`, err)
					continue
				}

				fmt.Println("Received", event.GetName(), event.GetId())
			} else if !err.(kafka.Error).IsTimeout() {
				// The client will automatically try to recover from all errors.
				// Timeout is not considered an error because it is raised by
				// ReadMessage in absence of messages.
				slog.Error(`ReadMessage`, `Err`, err)
			}

		}
	}
}

func Produce(ctx context.Context, topic, id string, wg *sync.WaitGroup) {
	defer wg.Done()

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"client.id":         id,
		"acks":              "all",
	})
	if err != nil {
		slog.Error(`NewProducer`, `Err`, err)
		return
	}

	defer p.Close()

	// Start delivery report goroutine
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("* Producer %s: Delivery failed: %v\n", id, ev.TopicPartition.Error)
				} else {
					fmt.Printf("* Producer %s: Delivered message to topic %s [%d] at offset %v\n",
						id, *ev.TopicPartition.Topic, ev.TopicPartition.Partition, ev.TopicPartition.Offset)
				}
			}
		}
	}()

	// Produce messages until context is cancelled
	counter := 0
	for {

		select {
		case <-ctx.Done():
			return

		default:
			counter++

			// Create an event
			event := &pb.Event{}
			event.SetName(fmt.Sprintf("Event-%s-%d", id, counter))
			event.SetId(uuid.New().String())

			// Marshal event to protobuf
			eventBytes, err := proto.Marshal(event)
			if err != nil {
				slog.Error(`Marshal`, `Err`, err)
				continue
			}

			// Produce message
			err = p.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
				Value:          eventBytes,
				Key:            []byte(event.GetId()),
			}, nil)

			if err != nil {
				slog.Error(`Producer`, `Err`, err)
			}

			// Sleep to avoid flooding
			time.Sleep(2 * time.Second)
		}
	}
}
