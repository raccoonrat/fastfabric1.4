// Copyright IBM Corp. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"github.com/fabric_extension/grpcmocks"
	"github.com/fabric_extension/stopwatch"
	"github.com/hyperledger/fabric/orderer/common/server"
	"os/user"

	"github.com/gogo/protobuf/proto"
	"os"
	"sync"
	"time"


	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/localmsp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	benchutil "github.com/fabric_extension/util"
	"github.com/hyperledger/fabric/protos/common"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"gopkg.in/cheggaaa/pb.v1"
)

func main() {
	server.SetDevFabricConfigPath()
	conf, err := localconfig.Load()
	if err != nil {
		fmt.Println("failed to load config:", err)
		os.Exit(1)
	}

	// Load local MSP
	err = mspmgmt.LoadLocalMsp(conf.General.LocalMSPDir, conf.General.BCCSP, conf.General.LocalMSPID)
	if err != nil { // Handle errors reading the config file
		fmt.Println("Failed to initialize local MSP:", err)
		os.Exit(0)
	}

	signer := localmsp.NewSigner()
	//init variables
	var channelID string
	var serverAddr string
	var endorseAddr server.ArrayFlags
	var messages int
	var goroutines int
	var msgSize uint64
	var numRuns uint64
	var bar *pb.ProgressBar

	flag.StringVar(&serverAddr, "server", fmt.Sprintf("%s:%d", conf.General.ListenAddress, conf.General.ListenPort), "The RPC server to connect to.")
	flag.Var(&endorseAddr, "endorse", "The endorse RPC server to connect to.")
	flag.StringVar(&channelID, "channelID", localconfig.Defaults.General.SystemChannel, "The channel ID to broadcast to.")
	flag.IntVar(&messages, "messages", 100000, "The number of messages to broadcast.")
	flag.IntVar(&goroutines, "goroutines", 50, "The number of concurrent go routines to broadcast the messages on")
	flag.Uint64Var(&msgSize, "size", 1024, "The size in bytes of the data section for the payload")
	flag.Uint64Var(&numRuns, "run", 5, "Num of times to run")

	flag.Parse()

	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	defer func() {
		_ = conn.Close()
	}()
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}
	for _, addr := range endorseAddr {
		server.Endorsers = append(server.Endorsers, grpcmocks.StartClient(addr))
	}

	usr, err := user.Current()
	if err != nil {
		fmt.Println("Error: ", err)
	}

	stopwatch.SetOutput("toEndorser", benchutil.OpenFile(fmt.Sprintf("%s/toEndorser.log", usr.HomeDir)))
	stopwatch.SetOutput("fromEndorser", benchutil.OpenFile(fmt.Sprintf("%s/fromEndorser.log", usr.HomeDir)))
	stopwatch.SetOutput("broadcast", benchutil.OpenFile(fmt.Sprintf("%s/broadcast.log", usr.HomeDir)))

	stopwatch.Now("toEndorser")

	// run benchmark
	for r := uint64(0); r < numRuns; r++ {
		fmt.Println("Round", r)
		time.Sleep(5 * time.Second)
		fmt.Println("Initializing")
		transactionSize := 10
		client, err := ab.NewAtomicBroadcastClient(conn).Broadcast(context.TODO())
		for accountNo := 0; accountNo < 2*messages; {

			accounts := make(map[string]int32, transactionSize)
			for len(accounts) < transactionSize {
				accounts[fmt.Sprint("account", accountNo)] = 1
				accountNo++
			}

			env, _ := server.Endorsers[0].Init(context.Background(), &grpcmocks.EndorseInit{InitBalances: accounts})
			client.Send(env)
		}
		fmt.Println("Initializating done")
		time.Sleep(5 * time.Second)

		msgs := make(chan *common.Envelope, 100000)
		go endorse(messages, msgs)

		fmt.Println("Waiting for blocks")
		time.Sleep(10 * time.Second)
		fmt.Println("Stop waiting")

		msgsPerGo := messages / goroutines
		roundMsgs := msgsPerGo * goroutines
		if roundMsgs != messages {
			fmt.Println("Rounding messages to", roundMsgs)
		}

		fmt.Println(msgsPerGo, "messages per goroutine")
		bar = pb.New64(int64(roundMsgs))
		bar.ShowPercent = true
		bar.ShowSpeed = true
		bar = bar.Start()

		var wg sync.WaitGroup
		wg.Add(2 * int(goroutines))
		for i := 0; i < goroutines; i++ {
			client, err = ab.NewAtomicBroadcastClient(conn).Broadcast(context.TODO())
			if err != nil {
				fmt.Println("Error connecting:", err)
				return
			}
			s := newBroadcastClient(client, channelID, signer)

			go func(i int, pb *pb.ProgressBar) {
				defer wg.Done()
				if err != nil {
					fmt.Println("Error connecting:", err)
					return
				}

				go func() {
					defer wg.Done()
					for i := 0; i < msgsPerGo; i++ {

						err = s.getAck()
						if err == nil && bar != nil {
							bar.Increment()
						}
					}
					if err != nil {
						fmt.Printf("\nError: %v\n", err)
					}
				}()

				for i := 0; i < msgsPerGo; i++ {
					if err := s.client.Send(<-msgs); err != nil {
						panic(err)
						//fmt.Println(numRuns,"-", err)
					}
					stopwatch.Now("broadcast")
				}
			}(i, bar)
		}

		wg.Wait()
		bar.FinishPrint("----------------------broadcast message finish-------------------------------")
	}
	stopwatch.Flush()
}

func endorse(totalMessages int, msgs chan *common.Envelope) {
	wg := sync.WaitGroup{}

	stopwatch.Now("toEndorser")
	stopwatch.Now("fromEndorser")
	for i := 0; i < totalMessages; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			stopwatch.Now("toEndorser")
			env, err := server.Endorsers[i%len(server.Endorsers)].Endorse(context.Background(), &grpcmocks.Transfer{From: fmt.Sprint("account", 2*i), To: fmt.Sprint("account", 2*i+1), Amount: 1})
			if err != nil {
				panic(err)
			}
			stopwatch.Now("fromEndorser")
			msgs <- env
		}(i)
	}
	wg.Wait()
	close(msgs)
}

type broadcastClient struct {
	client    ab.AtomicBroadcast_BroadcastClient
	signer    crypto.LocalSigner
	channelID string
}

// newBroadcastClient creates a simple instance of the broadcastClient interface
func newBroadcastClient(client ab.AtomicBroadcast_BroadcastClient, channelID string, signer crypto.LocalSigner) *broadcastClient {
	return &broadcastClient{client: client, channelID: channelID, signer: signer}
}

func (s *broadcastClient) broadcast(transaction []byte) error {
	env := &common.Envelope{}
	err := proto.Unmarshal(transaction, env)
	if err != nil {
		panic(err)
	}
	return s.client.Send(env)
}

func (s *broadcastClient) getAck() error {
	msg, err := s.client.Recv()
	if err != nil {
		return err
	}
	if msg.Status != cb.Status_SUCCESS {
		return fmt.Errorf("Got unexpected status: %v - %s", msg.Status, msg.Info)
	}
	return nil
}
