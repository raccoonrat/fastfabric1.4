// Copyright IBM Corp. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"github.com/fabric_extension/grpcmocks"
	"github.com/gogo/protobuf/proto"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"time"

	// "strconv"

	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/localmsp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	//"github.com/hyperledger/fabric/protos/utils"
	"github.com/hyperledger/fabric/protos/common"

	benchutil "github.com/fabric_extension/util"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"gopkg.in/cheggaaa/pb.v1"
)

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
	env :=&common.Envelope{}
	err := proto.Unmarshal(transaction, env)
	//env, err := utils.CreateSignedEnvelope(cb.HeaderType_ENDORSER_TRANSACTION, s.channelID, s.signer, &cb.ConfigValue{Value: transaction}, 0, 0)
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

type measurement struct {
	start, end time.Time
	comment    string
}


func (m *measurement) Duration() time.Duration {
	return m.end.Sub(m.start)
}

func (m *measurement) Start() {
	if m.start.IsZero() {
		m.start = time.Now()
	}
}
func (m *measurement) Stop() {
	if !m.start.IsZero() && m.end.IsZero() {
		m.end = time.Now()
	}
}


func main() {
	SetDevFabricConfigPath()
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
	var messages uint64
	var goroutines uint64
	var msgSize uint64
	var numRuns uint64
	var bar *pb.ProgressBar

	flag.StringVar(&serverAddr, "server", fmt.Sprintf("%s:%d", conf.General.ListenAddress, conf.General.ListenPort), "The RPC server to connect to.")
	flag.StringVar(&channelID, "channelID", localconfig.Defaults.General.SystemChannel, "The channel ID to broadcast to.")
	flag.Uint64Var(&messages, "messages", 100000, "The number of messages to broadcast.")
	flag.Uint64Var(&goroutines, "goroutines", 50, "The number of concurrent go routines to broadcast the messages on")
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
	firstRun := true
	// run benchmark
	for r := uint64(0); r < numRuns; r++ {
		fmt.Println("Round", r)
		init := readMessages("init.dump")
		msgs := readMessages("tx.dump")

		msgsPerGo := messages / goroutines
		roundMsgs := msgsPerGo * goroutines
		if roundMsgs != messages {
			fmt.Println("Rounding messages to", roundMsgs)
		}

		fmt.Println(msgsPerGo ,"messages per goroutine")
		bar = pb.New64(int64(roundMsgs))
		bar.ShowPercent = true
		bar.ShowSpeed = true
		bar = bar.Start()


		if firstRun {
			client, err := ab.NewAtomicBroadcastClient(conn).Broadcast(context.TODO())
			if err != nil {
				fmt.Println("Error connecting:", err)
				return
			}
			s := newBroadcastClient(client, channelID, signer)
			for in := range init {
				if err := s.broadcast(in); err != nil {
					panic(err)
					//fmt.Println(numRuns,"-", err)
				}
			}
			time.Sleep(3 * time.Second)
			firstRun = false
		}
		var wg sync.WaitGroup
		wg.Add(2*int(goroutines))
		for i := uint64(0); i < goroutines; i++ {
			go func(i uint64, pb *pb.ProgressBar) {
				client, err := ab.NewAtomicBroadcastClient(conn).Broadcast(context.TODO())
				if err != nil {
					fmt.Println("Error connecting:", err)
					return
				}
				s := newBroadcastClient(client, channelID, signer)
				go func() {
					defer wg.Done()
					for i := uint64(0); i < msgsPerGo; i++ {

						err = s.getAck()
						if err == nil && bar != nil {
							bar.Increment()
						}
					}
					if err != nil {
						fmt.Printf("\nError: %v\n", err)
					}
				}()

				for i := uint64(0); i < msgsPerGo; i++ {
					if err := s.broadcast(<-msgs); err != nil {
						panic(err)
						//fmt.Println(numRuns,"-", err)
					}
				}
				wg.Done()
			}(i, bar)
		}

		wg.Wait()
		bar.FinishPrint("----------------------broadcast message finish-------------------------------")
	}
}
func readMessages(filename string) chan[]byte {
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Error: ", err)
	}
	dumpFile := benchutil.OpenFile(fmt.Sprintf("%s/%s", usr.HomeDir, filename))
	lines, err := ioutil.ReadAll(dumpFile)

	envs := &grpcmocks.Envelopes{}
	proto.Unmarshal(lines, envs)
	txs := make(chan []byte, 100000)

	fmt.Println("Transactions in ", filename,":", len(envs.Envs))
	for _, env := range envs.Envs{
		txs <- env
	}
	close(txs)
	return txs
}




func SetDevFabricConfigPath() (cleanup func()) {

	oldFabricCfgPath, resetFabricCfgPath := os.LookupEnv("FABRIC_CFG_PATH")
	devConfigDir, err := GetDevConfigDir()
	if err != nil {
		fmt.Println("failed to get dev config dir: %s", err)
	}

	err = os.Setenv("FABRIC_CFG_PATH", devConfigDir)
	if resetFabricCfgPath {
		return func() {
			err := os.Setenv("FABRIC_CFG_PATH", oldFabricCfgPath)
			if err != nil {
				fmt.Println("failed to set env", oldFabricCfgPath)
			}
		}
	}

	return func() {
		err := os.Unsetenv("FABRIC_CFG_PATH")
		if err != nil {
			fmt.Println("failed to unset")
		}
	}
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.IsDir()
}


func GetDevConfigDir() (string, error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return "", fmt.Errorf("GOPATH not set")
	}

	for _, p := range filepath.SplitList(gopath) {
		devPath := filepath.Join(p, "src/github.com/hyperledger/fabric/sampleconfig")
		if !dirExists(devPath) {
			continue
		}

		return devPath, nil
	}

	return "", fmt.Errorf("DevConfigDir not found in %s", gopath)
}
