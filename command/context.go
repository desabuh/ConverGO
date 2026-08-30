package command

import "context"

type CommandContext[S any] struct {
	context.Context

	App  S
	Args []string

	reply chan<- ResultMessage
}

func (c *CommandContext[S]) Reply(result ResultMessage) {
	c.reply <- result
}

func (c *CommandContext[S]) Success(message string) {
	c.reply <- ResultMessage{
		Status:  Success,
		Message: message,
	}
}

func (c *CommandContext[S]) Error(err error) {
	c.reply <- ResultMessage{
		Status:  Error,
		Message: err.Error(),
	}
}
