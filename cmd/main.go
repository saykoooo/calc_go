package main

import (
	"os"

	"github.com/saykoooo/calc_go/internal/agent"
	"github.com/saykoooo/calc_go/internal/application"
)

func main() {
	argLength := len(os.Args[1:])

	if os.Args[1] == "--agent" {
		agent.RunAgent()
	}
	if os.Args[1] == "--all" {
		go agent.RunAgent()
	}
	if argLength == 0 || os.Args[1] == "--all" {
		app := application.New()
		go app.RunGRPCServer()
		app.RunServer()
	}
}
