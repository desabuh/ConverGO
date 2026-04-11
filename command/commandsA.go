package command

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/desabuh/convergo/cvrdt"
	"github.com/desabuh/convergo/file"
	"github.com/desabuh/convergo/p2p"
)

const SITE_ID = "1"
const DOMAIN = "TEST_DOMAIN"
const DOMAIN_PATH = "./test_domain_concurrent/"

func guard(ctx context.Context, cond bool) bool {
	return cond || ctx.Err() != nil
}

var CommandRegistry CommandParser = NewCommandParser().
	Register(
		"edit",
		WithTimeoutCommand(CommandFunc(func(ctx context.Context, args []string, done chan<- ResultMessage) {
			const NUM_PARAMS = 3
			const INSERT_NUM_PARAMS = 4

			if len(args) < NUM_PARAMS {
				done <- FromErrStr("Command should have at least %d arguments", NUM_PARAMS)
				return
			}

			opStr := args[0]

			var opType cvrdt.OpType

			if opStr == "insert" {
				opType = cvrdt.Insertion
			} else if opStr == "delete" {
				opType = cvrdt.Deletion
			} else {
				done <- FromErrStr("First argument should be operation mode 'insert' or 'delete' not %s", opStr)
				return
			}

			filepath := args[1]

			pos, err := strconv.Atoi(args[2])

			if err != nil {
				done <- FromErrStr("Position should be an integer")
				return
			}

			content := ""
			if opType == cvrdt.Insertion {
				if len(args) < INSERT_NUM_PARAMS {
					done <- FromErrStr("Command edit should also provide 'content' parameter'")
					return
				}
				content = args[3]
			}

			currentInfo, err := Registry.GetFileCtxInfo(filepath)

			isCtxExist := err == nil

			if !isCtxExist {
				err := Registry.CreateFileCtx(filepath, time.Duration(3*time.Second))

				if err != nil {
					done <- FromError(err)
					return
				}

				currentInfo, err = Registry.GetFileCtxInfo(filepath)

				if err != nil {
					done <- FromError(err)
					return
				}
			}

			newState := cvrdt.GetNewStateFromOp(
				cvrdt.CreateNewLocalOp(opType, pos, content, SITE_ID),
			)

			mergedState := currentInfo.ReadOnlyState.Merge(newState)

			currentInfo.ReadOnlyState = mergedState
			err = Registry.UpdateFileCtx(currentInfo)

			if err != nil {
				done <- FromError(err)
				return
			}

			if opType == cvrdt.Deletion {
				currentInfo, _ = Registry.GetFileCtxInfo(filepath)
				content = currentInfo.ReadOnlyState[1].Char()
			}

			done <- FromSuccess("%s operation of %s in position %d was a success", nil, opStr, content, pos)

		}), 2*time.Second),
	).
	Register(
		"read",
		WithTimeoutCommand(CommandFunc(func(ctx context.Context, args []string, done chan<- ResultMessage) {

			if len(args) < 1 {
				done <- FromErrStr("File path argument should be supplied")
			}

			filepath := args[0]

			content, err := Registry.GetFileContent(filepath)

			if err != nil {
				done <- FromError(err)
				return
			}

			done <- FromSuccess("File %s was read with success", content, filepath)

		}), 1*time.Second),
	).
	Register(
		"history",
		WithTimeoutCommand(CommandFunc(func(ctx context.Context, args []string, done chan<- ResultMessage) {

			if len(args) < 1 {
				done <- FromErrStr("File path argument should be supplied")
				return
			}

			filepath := args[0]

			ctxInfo, err := Registry.GetFileCtxInfo(filepath)

			if err != nil {
				done <- FromError(err)
				return
			}

			var opContent string = ""
			for _, crdtOp := range ctxInfo.ReadOnlyState {
				opContent = opContent + crdtOp.OpId().String() + "\n"
			}

			if opContent == "" {
				done <- FromErrStr("No operation have been applied to file: %s", filepath)
				return
			}

			done <- FromSuccess(
				"%d operation have been found on %s file",
				opContent,
				len(ctxInfo.ReadOnlyState),
				filepath,
			)

		}), 1*time.Second),
	).
	Register(
		"init",
		WithTimeoutCommand(CommandFunc(func(ctx context.Context, args []string, done chan<- ResultMessage) {

			if Network.Address != "" {
				done <- FromErrStr("listening peer is already initialized on address %s", Network.Address)
				return
			}

			if len(args) < 1 {
				done <- FromErrStr("ip address should be specified")
				return
			}

			addr, err := net.ResolveTCPAddr("tcp", args[0])

			if err != nil {
				done <- FromErrStr("Could not create peer: %w", err)
				return
			}

			go func() {
				err := Network.ListenFor(addr)
				done <- FromError(err)

			}()

			if err != nil {
				done <- FromErrStr("Could not listen of address %s: %w", addr.String(), err)
				return
			}

			done <- FromSuccess("Peer initialization, waiting on port %s", nil, addr.String())
		},
		), 2*time.Second),
	).
	Register(
		"pair",
		WithTimeoutCommand(CommandFunc(func(ctx context.Context, args []string, done chan<- ResultMessage) {

			if len(args) < 1 {
				done <- FromErrStr("ip address should be specified")
				return
			}

			targetAddr, err := net.ResolveTCPAddr("tcp", args[0])

			if err != nil {
				done <- FromErrStr("Could not create peer: %w", err)
				return
			}

			err = Network.Add(ctx, targetAddr)

			if err != nil {
				done <- FromErrStr("Cannot pair with %s: %w", targetAddr.String(), err)
				return
			}

			peerName := <-Network.GetReceiveCh()

			hostName := ctx.Value("hostname").(string)

			err = Network.Send(ctx, hostName, targetAddr)

			if err != nil {
				done <- FromErrStr("Pair error with %s | %s: %w", targetAddr, peerName, err)
				return
			}

			//done <- FromSuccess("Peer with address %s is successfully paired", nil, targetAddr.String())

		},
		), 2*time.Second),
	)

var Registry *file.FileRegistry[cvrdt.CvRDTState] = file.CreateNewWootFileRegistry(SITE_ID, DOMAIN, DOMAIN_PATH)
var Network *p2p.TCPLayer[string] = p2p.NewTCPTransport[string]()
