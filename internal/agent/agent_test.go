package agent

import (
	"context"
	"testing"

	"github.com/saykoooo/calc_go/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

func TestCompute(t *testing.T) {
	tests := []struct {
		name     string
		a        float64
		b        float64
		op       string
		expected float64
		err      bool
	}{
		{"Addition", 5, 3, "+", 8, false},
		{"Subtraction", 5, 3, "-", 2, false},
		{"Multiplication", 5, 3, "*", 15, false},
		{"Division", 6, 3, "/", 2, false},
		{"Division by zero", 6, 0, "/", 0, true},
		{"Unknown operation", 5, 3, "%", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compute(tt.a, tt.b, tt.op)
			if (err != nil) != tt.err {
				t.Errorf("compute(%v, %v, %q) error = %v, expected error = %v", tt.a, tt.b, tt.op, err, tt.err)
				return
			}
			if result != tt.expected {
				t.Errorf("compute(%v, %v, %q) = %v, expected %v", tt.a, tt.b, tt.op, result, tt.expected)
			}
		})
	}
}

type MockOrchestratorClient struct {
	mock.Mock
}

func (m *MockOrchestratorClient) GetTask(ctx context.Context, in *proto.GetTaskRequest, opts ...grpc.CallOption) (*proto.TaskResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*proto.TaskResponse), args.Error(1)
}

func (m *MockOrchestratorClient) SubmitResult(ctx context.Context, in *proto.ResultRequest, opts ...grpc.CallOption) (*proto.SubmitResultResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*proto.SubmitResultResponse), args.Error(1)
}

func TestGetTask(t *testing.T) {
	mockClient := new(MockOrchestratorClient)
	client = mockClient

	expectedTask := &proto.TaskResponse{
		Id:            "task1",
		Arg1:          2,
		Arg2:          3,
		Operation:     "+",
		OperationTime: 100,
	}

	mockClient.On("GetTask", mock.Anything, &proto.GetTaskRequest{}).Return(expectedTask, nil)

	task, err := getTask()
	assert.NoError(t, err)
	assert.Equal(t, expectedTask.Id, task.ID)
	assert.Equal(t, expectedTask.Arg1, task.Arg1)
	assert.Equal(t, expectedTask.Arg2, task.Arg2)
	assert.Equal(t, expectedTask.Operation, task.Operation)
	assert.Equal(t, expectedTask.OperationTime, task.OperationTime)

	mockClient.AssertExpectations(t)
}

func TestSendResult(t *testing.T) {
	mockClient := new(MockOrchestratorClient)
	client = mockClient

	mockClient.On("SubmitResult", mock.Anything, &proto.ResultRequest{
		Id:     "task1",
		Result: 5,
	}).Return(&proto.SubmitResultResponse{}, nil)

	err := sendResult("task1", 5)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}
