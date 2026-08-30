package command

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"github.com/desabuh/convergo/log"
)

const COMMAND_ID = "COMMAND"

type CommandLoop[S io.Closer] struct {
	parser *CommandParser[S]
	logger log.GlobalLogger
	reader io.Reader
	app    S
}

func NewCommandLoop[S io.Closer](
	app S,
	parser *CommandParser[S],
	reader io.Reader,
	loggerFactory log.GlobalLoggerFactory,
) *CommandLoop[S] {
	return &CommandLoop[S]{
		app:    app,
		parser: parser,
		reader: reader,
		logger: loggerFactory.Create(COMMAND_ID),
	}
}

func (l *CommandLoop[S]) Run(ctx context.Context) error {

	defer func() {

		err := l.app.Close()

		l.logger.Log("App termination...")
		if err != nil {
			l.logger.Log("App termination error: %v", err)
		}
	}()

	scanner := bufio.NewScanner(l.reader)

	for scanner.Scan() {

		line := scanner.Text()

		command, err := l.parser.Parse(line)

		if err != nil {
			l.logger.Log("Command error: %v", err)

			continue
		}

		reply := make(chan ResultMessage, 1)

		cmdCtx := CommandContext[S]{
			Context: ctx,
			App:     l.app,
			Args:    command.Args,
			reply:   reply,
		}

		command.Command.Execute(cmdCtx)

		select {

		case <-ctx.Done():
			return ctx.Err()

		case result := <-reply:

			prefix := "Command Success: "

			if result.Status == Error {
				prefix = "Command Failure: "
			}

			content := prefix + result.Message

			if result.Payload != nil {
				content = content + fmt.Sprintf("\n%v", result.Payload)
			}

			l.logger.Log(content)
		}
	}
	return scanner.Err()
}
