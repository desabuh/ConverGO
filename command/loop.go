package command

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/desabuh/convergo/log"
)

const COMMAND_ID = "COMMAND"
const TERMINATION_COMMAND = "shutdown_app"

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

func (l *CommandLoop[S]) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	l.cancelCtxOnInterrupt(cancel)

	err := l.startLoop(ctx)

	l.logger.Log("Shutdown command loop with error: %v", err)

}

func (l *CommandLoop[S]) startLoop(ctx context.Context) error {

	defer func() {

		err := l.app.Close()

		l.logger.Log("App termination...")
		if err != nil {
			l.logger.Log("App termination error: %v", err)
		}
	}()

	scanner := NewContextScanner(l.reader)

	for scanner.Scan(ctx) {

		line := scanner.Text()

		if line == TERMINATION_COMMAND {
			l.logger.Log("App termination command received")
			return nil
		}

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

		go command.Command.Execute(cmdCtx)

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

func (l *CommandLoop[S]) cancelCtxOnInterrupt(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()
}

// A wrapper around bufio.Scanner to make it cancellable though a provided context
// this is a trick to stop a blocking Scan() though a cancellable context without having to explicitly closing the underlying reader
type ContextScanner struct {
	*bufio.Scanner
}

func NewContextScanner(r io.Reader) *ContextScanner {
	return &ContextScanner{
		Scanner: bufio.NewScanner(r),
	}
}

func (s *ContextScanner) Scan(ctx context.Context) bool {
	done := make(chan bool, 1)

	go func() {
		done <- s.Scanner.Scan()
	}()

	select {
	case <-ctx.Done():
		return false

	case ok := <-done:
		return ok
	}
}
