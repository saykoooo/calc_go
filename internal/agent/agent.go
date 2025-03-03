package agent

import (
	"fmt"
	"log"
	"time"
)

type Task struct {
	ID            string  `json:"id"`
	Arg1          float64 `json:"arg1"`
	Arg2          float64 `json:"arg2"`
	Operation     string  `json:"operation"`
	OperationTime int     `json:"operation_time"`
}

func main() {
	computingPower := 1
	log.Printf("Starting %d worker threads", computingPower)
	for i := 0; i < computingPower; i++ {
		go worker()
	}

	select {}
}

func worker() {
	for {
		task, err := getTask()
		if err != nil {
			log.Printf("Failed to get task: %v. Retrying in 1 second...", err)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("Received task: ID=%s, Operation=%s, Arg1=%.2f, Arg2=%.2f", task.ID, task.Operation, task.Arg1, task.Arg2)

		result, err := compute(task.Arg1, task.Arg2, task.Operation)
		if err != nil {
			log.Printf("Error during computation: %v", err)
			continue
		}

		log.Printf("Computation result for task %s: %.2f", task.ID, result)

		time.Sleep(time.Duration(task.OperationTime) * time.Millisecond)

		if err := sendResult(task.ID, result); err != nil {
			log.Printf("Failed to send result for task %s: %v", task.ID, err)
			continue
		}

		log.Printf("Result sent successfully for task %s", task.ID)
	}
}

func getTask() (*Task, error) {
	var response struct {
		Task Task `json:"task"`
	}

	return &response.Task, nil
}

func sendResult(id string, result float64) error {
	return nil
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
