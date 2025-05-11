package agent

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"context"

	"github.com/saykoooo/calc_go/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Task struct {
	ID            string  `json:"id"`
	Arg1          float64 `json:"arg1"`
	Arg2          float64 `json:"arg2"`
	Operation     string  `json:"operation"`
	OperationTime int32   `json:"operation_time"`
}

var (
	shutdownCh = make(chan struct{})
	serverPort string
	wg         sync.WaitGroup
)

var client proto.OrchestratorClient

func initGRPCClient() {
	conn, err := grpc.Dial("localhost:"+serverPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	client = proto.NewOrchestratorClient(conn)
}

func RunAgent() {
	serverPort = os.Getenv("GRPC_PORT")
	if serverPort == "" {
		serverPort = "5000"
	}

	initGRPCClient()
	computingPower, _ := strconv.Atoi(os.Getenv("COMPUTING_POWER"))
	if computingPower <= 0 {
		computingPower = 1
	}

	log.Printf("Agent: Starting %d worker threads", computingPower)

	wg.Add(computingPower)
	for i := 0; i < computingPower; i++ {
		go worker()
	}

	<-shutdownCh
	log.Println("Agent: Shutting down agent...")
	wg.Wait()
}

func worker() {
	defer wg.Done()
	for {
		select {
		case <-shutdownCh:
			return
		default:
			task, err := getTask()
			if err != nil {
				log.Printf("Agent: Failed to get task: %v. Retrying in 1 second...", err)
				time.Sleep(1 * time.Second)
				continue
			}

			log.Printf("Agent: Received task: ID=%s, Operation=%s, Arg1=%.2f, Arg2=%.2f",
				task.ID, task.Operation, task.Arg1, task.Arg2)

			result, err := compute(task.Arg1, task.Arg2, task.Operation)
			if err != nil {
				log.Printf("Agent: Error during computation: %v", err)
				continue
			}

			log.Printf("Agent: Computation result for task %s: %.2f", task.ID, result)

			time.Sleep(time.Duration(task.OperationTime) * time.Millisecond)

			if err := sendResult(task.ID, result); err != nil {
				log.Printf("Agent: Failed to send result for task %s: %v", task.ID, err)
			}
		}
	}
}

func getTask() (*Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetTask(ctx, &proto.GetTaskRequest{})
	if err != nil {
		return nil, err
	}

	return &Task{
		ID:            resp.Id,
		Arg1:          resp.Arg1,
		Arg2:          resp.Arg2,
		Operation:     resp.Operation,
		OperationTime: int32(resp.OperationTime),
	}, nil
}

func sendResult(id string, result float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.SubmitResult(ctx, &proto.ResultRequest{
		Id:     id,
		Result: result,
	})
	return err
}

func compute(a, b float64, op string) (float64, error) {
	switch op {
	case "+":
		return a + b, nil
	case "-":
		return a - b, nil
	case "*":
		return a * b, nil
	case "/":
		if b == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return a / b, nil
	default:
		return 0, fmt.Errorf("unknown operation: %s", op)
	}
}
