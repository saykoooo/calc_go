package agent

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/saykoooo/calc_go/internal/db"
	"github.com/saykoooo/calc_go/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

type testServer struct {
	proto.UnimplementedOrchestratorServer
}

func (s *testServer) GetTask(ctx context.Context, req *proto.GetTaskRequest) (*proto.TaskResponse, error) {
	return &proto.TaskResponse{
		Id:        "test-task",
		Arg1:      2,
		Arg2:      3,
		Operation: "+",
	}, nil
}

func (s *testServer) SubmitResult(ctx context.Context, req *proto.ResultRequest) (*proto.SubmitResultResponse, error) {
	if req.Id != "test-task" || req.Result != 5 {
		return nil, fmt.Errorf("unexpected result")
	}
	return &proto.SubmitResultResponse{}, nil
}

func initTestGRPCServer() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	proto.RegisterOrchestratorServer(s, &testServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestAgent_Integration(t *testing.T) {
	db.Init(":memory:")
	defer db.Stop()

	initTestGRPCServer()

	serverPort = "50051"
	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial buffer net: %v", err)
	}
	client = proto.NewOrchestratorClient(conn)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		task, err := getTask()
		if err != nil {
			t.Errorf("Failed to get task: %v", err)
		}
		if task.Operation != "+" || task.Arg1 != 2 || task.Arg2 != 3 {
			t.Errorf("Unexpected task: %+v", task)
		}

		result, err := compute(task.Arg1, task.Arg2, task.Operation)
		if err != nil {
			t.Errorf("Failed to compute: %v", err)
		}
		if result != 5 {
			t.Errorf("Expected 5, got %.2f", result)
		}

		err = sendResult(task.ID, result)
		if err != nil {
			t.Errorf("Failed to send result: %v", err)
		}
	}()

	wg.Wait()
}
