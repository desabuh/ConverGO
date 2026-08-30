package command

import (
	"context"
	"errors"
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

func ConcatResultMessages(results ...ResultMessage) ResultMessage {
	status := Success
	var payload any
	var messages []string

	for _, r := range results {
		// Skip uninitialized (zero-value) messages
		if r == (ResultMessage{}) {
			continue
		}

		if r.Status == Error {
			status = Error
		}

		if r.Message != "" {
			messages = append(messages, r.Message)
		}

		// Keep last non-nil payload
		if r.Payload != nil {
			payload = r.Payload
		}
	}

	return ResultMessage{
		Status:  status,
		Message: strings.Join(messages, "\n\t"),
		Payload: payload,
	}
}

func WithTimeoutCommand[S any](
	base func(CommandContext[S]),
	timeout time.Duration,
) func(CommandContext[S]) {

	return func(cmd CommandContext[S]) {

		ctx, cancel := context.WithTimeout(
			cmd.Context,
			timeout,
		)
		defer cancel()

		copy := cmd
		copy.Context = ctx

		done := make(chan struct{})

		go func() {
			base(copy)
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			cmd.Error(ctx.Err())
		}
	}
}

// functional interface to execute commands
// Execute() method should accept a context and a send-only channel to report the result of the operation
// the way this interface is written is not intended to report result (except for errors)
type Command[S any] interface {
	Execute(CommandContext[S])
}

// function decorator to wrap functional interface Command
// it should be used wether using closure in the command usage is important
// done is the result channel
type CommandFunc[S any] func(CommandContext[S])

func (f CommandFunc[S]) Execute(cmd CommandContext[S]) {
	f(cmd)
}

type CommandParser[S any] struct {
	commands map[string]Command[S]
}

func NewCommandParser[S any]() *CommandParser[S] {
	return &CommandParser[S]{
		commands: make(map[string]Command[S]),
	}
}

func (p *CommandParser[S]) Parse(
	input string,
) (*ParsedCommand[S], error) {

	fields := strings.Fields(input)

	if len(fields) == 0 {
		return nil, errors.New("empty command")
	}

	commandName := fields[0]

	command, exists := p.commands[commandName]

	if !exists {
		return nil, fmt.Errorf(
			"command not found: %s",
			commandName,
		)
	}

	return &ParsedCommand[S]{
		Command: command,
		Args:    fields[1:],
	}, nil
}

func (p *CommandParser[S]) Register(
	name string,
	command func(CommandContext[S]),
) *CommandParser[S] {

	p.commands[name] = CommandFunc[S](command)
	return p
}

// simple decorator to store internally the argument, the second parameter in Execute method in not needed, only necessary to conform to interface
type ParsedCommand[S any] struct {
	Command Command[S]
	Args    []string
}

func (c *ParsedCommand[S]) Execute(cmd CommandContext[S]) {
	c.Command.Execute(cmd)
}
