package main

import (
	// "context"
	"context"
	"fmt"
	"submersible/tty"
	"time"
	// "submersible/tty"
)

// Currently testing formatting, etc. for bash
func main() {

	cmd, err := tty.NewTerminal(context.Background(), "bash", "--norc", "-i")
	if err != nil {
		panic("failed to start pty: " + err.Error())
	}
	cmd.Start()
	errs := cmd.Errors()
	go func() {
		for {
			cmd.RunCommand([]byte("cd ..\n"))
			cmd.RunCommand([]byte("pwd\n"))
			time.Sleep(time.Second)
		}
	}()
	serr := cmd.Out()
	for {
		select {
		case b := <-serr:
			fmt.Println("stdout", string(b))
		case err := <-errs:
			fmt.Println("errs", err.Error())
		}
	}
}
