package command

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Status int

const (
	Success Status = iota
	Error
)

// simple struct to return the result of a command
type ResultMessage struct {
	Status  Status
	Message string
	Payload any
}

func FromError(err error) ResultMessage {
	return ResultMessage{
		Status:  Error,
		Message: err.Error(),
		Payload: nil,
	}
}

func FromErrStr(str string, params ...any) ResultMessage {
	return FromError(fmt.Errorf(str, params...))
}

func FromSuccess(msg string, payload any, params ...any) ResultMessage {
	return ResultMessage{
		Status:  Success,
		Message: fmt.Sprintf(msg, params...),
		Payload: payload,
	}
}

// functional interface to execute commands
// Execute() method should accept a context and a send-only channel to report the result of the operation
// the way this interface is written is not intended to report result (except for errors)
type Command interface {
	Execute(ctx context.Context, args []string, done chan<- ResultMessage)
}

// function decorator to wrap functional interface Command
// it should be used wether using closure in the command usage is important
// done is the result channel
type CommandFunc func(ctx context.Context, args []string, done chan<- ResultMessage)

func (f CommandFunc) Execute(ctx context.Context, args []string, done chan<- ResultMessage) {
	f(ctx, args, done)
}

func WithTimeoutCommand(base CommandFunc, timeout time.Duration) CommandFunc {
	return func(ctx context.Context, args []string, done chan<- ResultMessage) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		resChannel := make(chan ResultMessage, 1)

		go func() {
			base(ctx, args, resChannel)
		}()

		select {
		case res := <-resChannel:
			done <- res
		case <-ctx.Done():
			done <- FromError(ctx.Err())

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
	res := strings.Split(input, " ")

	if len(res) < 2 { //command should have a at least an argument
		return &CommandWithArgs{}, fmt.Errorf("command input not complete: %s", res)
	}

	cmdStr := res[0]
	args := res[1:]

	if command, exists := p.commands[cmdStr]; exists {
		return &CommandWithArgs{command, args}, nil
	}
	return &CommandWithArgs{}, fmt.Errorf("command not found: %s", cmdStr)
}

func (p CommandParser) Register(input string, action Command) CommandParser {
	p.commands[input] = action
	return p
}

// simple decorator to store internally the argument, the second parameter in Execute method in not needed, only necessary to conform to interface
type CommandWithArgs struct {
	cmd  Command
	args []string
}

func (c *CommandWithArgs) Execute(ctx context.Context, args []string, done chan<- ResultMessage) {
	c.cmd.Execute(ctx, c.args, done)
}
