package main

import (
	"os"

	"github.com/saykoooo/calc_go/internal/agent"
	"github.com/saykoooo/calc_go/internal/application"
)

func main() {
	argLength := len(os.Args[1:])

	if argLength == 0 {
		app := application.New()
		app.RunServer()
	} else if os.Args[1] == "--agent" {
		agent.RunAgent()
	} else if os.Args[1] == "--all" {
		go agent.RunAgent()
		app := application.New()
		app.RunServer()
	}
}
