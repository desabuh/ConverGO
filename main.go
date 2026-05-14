package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/desabuh/convergo/command"
	"github.com/desabuh/convergo/cvrdt"
	"github.com/desabuh/convergo/file"
	"github.com/desabuh/convergo/p2p"
)

type RequestType string

type Message struct {
	typ     RequestType
	content any
}

func ToCvrdtSelector(data p2p.Payload) (cvrdt.CvRDTState, error) {
	var res []cvrdt.WootOperation
	err := data.Decode(&res)

	if err != nil {
		return nil, err
	}

	result := make(cvrdt.CvRDTState, len(res))

	for i, op := range res {
		result[i] = op
	}

	return result, err
}

func main() {

	//---------------------------------------------------

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// defer func() {
	// 	os.Remove("C:/Users/maste/Documents/converGO/ConverGO/test_domain_concurrent/file.txt")
	// 	os.Remove("C:/Users/maste/Documents/converGO/ConverGO/test_domain_concurrent/file2.txt")
	// }()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("Received signal: %v\n", sig)
		cancel()
	}()

	//var cmd string
	var localHostInfo p2p.PeerHostInfo

	// if os.Args[1] == "first" {
	// 	cmd = "command"

	// 	localHostInfo, _ = p2p.CreateNewHostInfo("1", os.Args[1], "localhost:8085")

	// } else {
	// 	cmd = "command2"

	// 	localHostInfo, _ = p2p.CreateNewHostInfo("2", os.Args[1], "localhost:8086")
	// }

	fmt.Println("DSADASDsads")

	id := os.Getenv("PEER_ID")
	name := os.Getenv("PEER_NAME")
	port := os.Getenv("LOCAL_PORT")

	dns := os.Getenv("DNS")

	if id == "" || name == "" || port == "" || dns == "" {
		panic(fmt.Errorf("ID, NAME, DNS and LOCAL_PORT env variable should be supplied"))
	}

	localHostInfo, _ = p2p.CreateNewHostInfo(id, name, dns+":"+port)
	//cmd = "command2"

	fmt.Printf("localHostInfo.Address: %v\n", localHostInfo.Address)

	ctx = context.WithValue(ctx, "hostname", localHostInfo)

	//f, err := file.OpenFile("C:/Users/maste/Documents/converGO/ConverGO/test_domain_concurrent/"+cmd+".txt", os.O_RDWR)
	//f, err := file.OpenFile("./test_domain_concurrent/"+cmd+".txt", os.O_RDWR)
	//fmt.Printf("os.Getenv(\"COMMAND_FILE\"): %v\n", os.Getenv("COMMAND_FILE"))
	f, err := file.OpenFile(os.Getenv("COMMAND_FILE"), os.O_RDWR)
	defer f.Close()

	if err != nil {
		panic(err)
	}

	fmt.Printf("f: %v\n", f)

	command.CheckCommand(ctx, f, os.Stdout, command.CommandRegistry)

	fmt.Println("DSADASD")

	//<-sigChan
	//time.Sleep(1 * time.Second)

	fmt.Print("shutting down")

	//time.Sleep(3 * time.Second)

	//---------------------------------------------------

	//command.CheckCommand(ctx, os.Stdin, command.CommandRegistry)

	// if err := command.CheckCommand(ctx, os.Stdin, cmdR); err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	// }

	// cmdDone := make(chan command.ResultMessage, 1)

	// fmt.Print("Start\n")

	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// cmd, err := command.CommandRegistry.Parse("edit insert file.txt 0 Hi")
	// //read file.txt

	// if err != nil {
	// 	panic(err)
	// }

	// cmd.Execute(ctx, nil, cmdDone)

	// select {
	// case <-ctx.Done():
	// 	fmt.Println(ctx.Err())
	// case rep := <-cmdDone:
	// 	fmt.Println(rep.Message)

	// }

	// time.Sleep(4 * time.Second)

	// cmd, err = command.CommandRegistry.Parse("edit delete file.txt 0")

	// if err != nil {
	// 	panic(err)
	// }

	// cmd.Execute(ctx, nil, cmdDone)

	// select {
	// case <-ctx.Done():
	// 	fmt.Println(ctx.Err())
	// case rep := <-cmdDone:
	// 	fmt.Println(rep.Message)

	// }

	// time.Sleep(4 * time.Second)

	// cmd, err = command.CommandRegistry.Parse("read file.txt")

	// if err != nil {
	// 	panic(err)
	// }

	// cmd.Execute(ctx, nil, cmdDone)

	// select {
	// case <-ctx.Done():
	// 	fmt.Print("FAIL")
	// case rep := <-cmdDone:
	// 	fmt.Println(rep.Message)
	// 	fmt.Print("File content: \n")
	// 	fmt.Println(rep.Payload)

	// }

	// fmt.Print("END")

}
