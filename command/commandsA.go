package command

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/desabuh/convergo/cvrdt"
	"github.com/desabuh/convergo/p2p"
	"github.com/desabuh/convergo/utils"
)

const SITE_ID = "1"
const DOMAIN_PATH = "./test_domain_concurrent/"

var DEFAULT_OUT_STREAM = os.Stdout

func guard(ctx context.Context, cond bool) bool {
	return cond || ctx.Err() != nil
}

var client *p2p.CvrdtNetClient

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

			currentInfo, err := client.Registry.GetFileCtxInfo(filepath)

			isCtxExist := err == nil

			var creationLog ResultMessage

			if !isCtxExist {
				err := client.Registry.CreateFileCtx(filepath, time.Duration(3*time.Second))

				creationLog = FromSuccess("A new file context %s was created", nil, filepath)

				if err != nil {
					done <- ConcatResultMessages(creationLog, FromError(err))
					return
				}

				currentInfo, err = client.Registry.GetFileCtxInfo(filepath)

				if err != nil {
					done <- ConcatResultMessages(creationLog, FromError(err))
					return
				}
			}

			newState := cvrdt.GetNewStateFromOp(
				cvrdt.CreateNewLocalOp(opType, pos, content, SITE_ID),
			)

			mergedState := currentInfo.ReadOnlyState.Merge(newState)

			currentInfo.ReadOnlyState = mergedState
			_, err = client.Registry.UpdateFileCtx(currentInfo)

			if err != nil {
				done <- ConcatResultMessages(creationLog, FromError(err))
				return
			}

			if opType == cvrdt.Deletion {
				currentInfo, _ = client.Registry.GetFileCtxInfo(filepath)
				content = currentInfo.ReadOnlyState[1].Char()
			}

			done <- ConcatResultMessages(creationLog, FromSuccess("%s operation of %s in position %d was a success", nil, opStr, content, pos))

		}), 2*time.Second),
	).
	Register(
		"push",
		WithTimeoutCommand(CommandFunc(func(ctx context.Context, args []string, done chan<- ResultMessage) {

			err := client.BroadcastRegistryOverTransport(ctx)

			if err != nil {
				done <- FromErrStr("Push broadcasting failure %v", err)
			}

			done <- FromSuccess("Push broadcasting successfull!", nil)

		}), 2*time.Second),
	).
	Register(
		"read",
		WithTimeoutCommand(CommandFunc(func(ctx context.Context, args []string, done chan<- ResultMessage) {

			if client == nil {
				done <- FromErrStr("CvrdtClient should be initialized to visualize file contexts content")
			}

			if len(args) < 1 {
				done <- FromErrStr("File context <filepath> argument should be supplied")
			}

			filepath := args[0]

			content, err := client.Registry.GetFileContent(filepath)

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

			if client == nil {
				done <- FromErrStr("CvrdtClient should be initialized to visualize file contexts history")
			}

			if len(args) < 1 {
				done <- FromErrStr("Should specify file context <pathfile>")
				return
			}

			filepath := args[0]

			ctxInfo, err := client.Registry.GetFileCtxInfo(filepath)

			//ctxInfo, err := Registry.GetFileCtxInfo(filepath)

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

			if client != nil {
				done <- FromErrStr("CvrdtClient already initialized on on address %s", client.Id.Address)
			}

			if len(args) < 1 {
				done <- FromErrStr("Init command should supply <ip:port> command")
			}

			hostName := ctx.Value("hostname").(p2p.PeerHostInfo)

			shakeCodec := p2p.NewCodec(p2p.JsonEncoder[p2p.PeerHostInfo]{}, p2p.JsonDecoder[p2p.PeerHostInfo]{})

			transportCodec := p2p.NewCodec(p2p.JsonEnvelopeEncoder[p2p.PeerMetadata]{}, p2p.JsonEnvelopeDecoder[p2p.PeerMetadata]{})

			client = p2p.CreateNewTCPWootBasedClient(hostName, DOMAIN_PATH, shakeCodec, transportCodec)

			go client.WaitForMessages(context.Background())

			go func() {
				err := client.Init()
				utils.Logger.NlLog(DEFAULT_OUT_STREAM, "peer waiting interface closed: %v", err)
				return
			}()

			done <- FromSuccess("Peer initialization, waiting on address %s", nil, hostName.Address.String())

		},
		), 2*time.Second),
	).
	Register(
		"pair",
		WithTimeoutCommand(CommandFunc(func(ctx context.Context, args []string, done chan<- ResultMessage) {

			if client == nil {
				done <- FromErrStr("CvrdtClient should be initialized to pair with others clients")
			}

			if len(args) < 1 {
				done <- FromErrStr("Pair ip address of the target client should be specified")
				return
			}

			err := client.Pair(ctx, args[0])

			if err != nil {
				done <- FromErrStr("Could not pair with target client: %w", err)
				return
			}

			done <- FromSuccess("Target client on address %s is successfully paired", nil, args[0])

		},
		), 10*time.Second),
	)
