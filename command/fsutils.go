package command

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/desabuh/convergo/utils"
)

// type MutexWriter struct {
// 	wr io.Writer
// 	mu sync.Mutex
// }

// Write(p []byte) (n int, err error)

func initControlFlowCommand() {
	CommandRegistry.Register(
		"wait",
		WithTimeoutCommand(CommandFunc(func(ctx context.Context, args []string, done chan<- ResultMessage) {

			if len(args) < 1 {
				done <- FromErrStr("A time to wait should be supplied in ms")
			}

			timeStr := args[0]

			timeToWait, err := strconv.Atoi(timeStr)

			if err != nil {
				done <- FromErrStr("Time supplied should be an integer: %w", err)
			}

			if timeToWait < 0 {
				done <- FromErrStr("Time supplied should be greater than 0")
			}

			time.Sleep(time.Duration(timeToWait) * time.Millisecond)

			done <- FromSuccess("Successfully waited %d ms", nil, timeToWait)

		}), time.Duration(math.MaxInt64)),
	)
}

func CheckCommand(ctx context.Context, file *os.File, out io.Writer, parser CommandParser) {
	initControlFlowCommand() //init useful control flow commands

	scanner := bufio.NewScanner(file)

	go func() {
		<-ctx.Done()

		if errors.Is(ctx.Err(), context.Canceled) {
			fmt.Print("ctx done channel termination, proceed to close the file\n")
			file.Close()
			return
		}
	}()

	for scanner.Scan() {
		line := scanner.Text()

		cmd, err := parser.Parse(line)

		if err != nil {
			fmt.Println(err)
			continue
		}

		cmdDone := make(chan ResultMessage, 1)

		cmd.Execute(ctx, nil, cmdDone)

		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				//fmt.Print("ctx done channel termination, proceed to close the file\n")
				file.Close()
				return
			}
			fmt.Println(ctx.Err())
		case rep := <-cmdDone:
			utils.Logger.Log(out, func(w io.Writer) {
				fmt.Fprintln(out, rep.Message)
				if rep.Payload != nil {
					fmt.Fprintln(out, rep.Payload)
				}
			},
			)
		}

		//channel should be closed to avoid command goroutine leaks
		close(cmdDone)
	}
}
