package command

import (
	"context"
	"fmt"
	"time"
)

// functional interface to execute commands
// Execute() method should accept a context and a send-only channel to report eventual error
// the way this interface is written is not intended to report result (except for errors)
type Command interface {
	Execute(ctx context.Context, done chan<- error)
}

// function decorator to wrap functional interface Command
// it should be used wether using closure in the command usage is important
// done is an error channel (nil means no error is got at the end)
type CommandFunc func(ctx context.Context, done chan<- error)

func (f CommandFunc) Execute(ctx context.Context, done chan<- error) {
	f(ctx, done)
}

func WithTimeoutCommand(base CommandFunc, timeout time.Duration) CommandFunc {
	return func(ctx context.Context, done chan<- error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		taskChannel := make(chan error, 1)

		go func() {
			base(ctx, taskChannel)
			taskChannel <- nil
		}()

		select {
		case err := <-taskChannel:
			done <- err
		case <-ctx.Done():
			done <- ctx.Err()
		}
	}

}

type CommandParser struct {
	commands map[string]Command
}

func NewCommandParser() CommandParser {
	return CommandParser{
		commands: make(map[string]Command),
	}
}

func (p *CommandParser) Parse(input string) (Command, error) {
	if command, exists := p.commands[input]; exists {
		return command, nil
	}
	return nil, fmt.Errorf("command not found: %s", input)
}

func (p *CommandParser) Register(input string, action Command) {
	p.commands[input] = action
}
