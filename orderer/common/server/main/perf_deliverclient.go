// Copyright IBM Corp. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"github.com/fabric_extension"
	"github.com/fabric_extension/stopwatch"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/fabric_extension/block_cache"
	"github.com/fabric_extension/commit"
	benchutil "github.com/fabric_extension/util"
	"github.com/fabric_extension/experiment"
	"github.com/fabric_extension/grpcmocks"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/ledger/kvledger/example"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	"github.com/hyperledger/fabric/core/ledger/testutil"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
	"math"
	"os"
	"github.com/hyperledger/fabric/protos/orderer"
	val "github.com/fabric_extension/validator"
	"os/signal"
	"path/filepath"
	"sync"

	// "sync"
	"os/user"
	"time"
	crypto2 "github.com/fabric_extension/crypto"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/localmsp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)


var (
   oldest  = &ab.SeekPosition{Type: &ab.SeekPosition_Oldest{Oldest: &ab.SeekOldest{}}}
    newest  = &ab.SeekPosition{Type: &ab.SeekPosition_Newest{Newest: &ab.SeekNewest{}}}
    maxStop = &ab.SeekPosition{Type: &ab.SeekPosition_Specified{Specified: &ab.SeekSpecified{Number: math.MaxUint64}}}
    //oldest = &ab.SeekPosition{Type: &ab.SeekPosition_Specified{Specified: &ab.SeekSpecified{Number: uint64(100)}}}
)

type deliverClient struct {
    client    ab.AtomicBroadcast_DeliverClient
    channelID string
    signer    crypto.LocalSigner
    quiet     bool
}

func newDeliverClient(client ab.AtomicBroadcast_DeliverClient, channelID string, signer crypto.LocalSigner, quiet bool) *deliverClient {
    return &deliverClient{client: client, channelID: channelID, signer: signer, quiet: quiet}
}

func (r *deliverClient) seekHelper(start *ab.SeekPosition, stop *ab.SeekPosition) *cb.Envelope {
    env, err := utils.CreateSignedEnvelope(cb.HeaderType_DELIVER_SEEK_INFO, r.channelID, r.signer, &ab.SeekInfo{
        Start:    start,
        Stop:     stop,
        Behavior: ab.SeekInfo_BLOCK_UNTIL_READY,
    }, 0, 0)
    if err != nil {
        panic(err)
    }
    return env
}

func (r *deliverClient) seekOldest() error {
    return r.client.Send(r.seekHelper(oldest, maxStop))
}

func (r *deliverClient) seekNewest() error {
    return r.client.Send(r.seekHelper(newest, maxStop))
}

func (r *deliverClient) seekSingle(blockNumber uint64) error {
    specific := &ab.SeekPosition{Type: &ab.SeekPosition_Specified{Specified: &ab.SeekSpecified{Number: blockNumber}}}
    return r.client.Send(r.seekHelper(specific, specific))
}

func runDeliverClient( channelID string,
    serverAddr string,
    seek int,
    quiet bool,
    goroutines uint64,
    r uint64 ,
    msgSize uint64,
    messages uint64,
    numBlocks uint64,
    HomeDir string,
    signer crypto.LocalSigner) {

	fmt.Println("Running....", r)

	if seek < -2 {
		fmt.Println("Wrong seek value.")
		flag.PrintDefaults()
	}

	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}
	client, err := ab.NewAtomicBroadcastClient(conn).Deliver(context.TODO())
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}

	s := newDeliverClient(client, channelID, signer, quiet)
	switch seek {
	case -2:
		err = s.seekOldest()
	case -1:
		err = s.seekNewest()
	default:
		err = s.seekSingle(uint64(seek))
	}
	if err != nil {
		fmt.Println("Received error:", err)
	}
	msgs := make(chan *orderer.DeliverResponse, fabric_extension.PipelineWidth)
	mcs := crypto2.GetService()
	go s.receive(msgs)

	var weight int64
	if fabric_extension.PipelineWidth/2<1{
		weight = 1
	}else{
		weight = int64(fabric_extension.PipelineWidth/2)
	}
	weighted := semaphore.NewWeighted(weight)

	const reps = 10
	const blocksize = 100

	sequence := commit.GetSequence()
	verified := commit.GetVerified()
	validated := commit.GetValidated()
	go commitBlocks(sequence, validated)
	go validate(verified, validated)
	firstBlock:=true

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func(){
		for range c {
			assertResult(blocksize, reps)
			stopwatch.Flush()
			os.Exit(0)
		}
	}()

	for msg := range msgs {
		switch t := msg.Type.(type) {
		case *orderer.DeliverResponse_Status:
			fmt.Println("Got status ", t)
			continue
		case *orderer.DeliverResponse_Block:
			seqNum := t.Block.Header.Number
			//fmt.Println("Block ", seqNum)
			if seqNum ==0 {
				if !firstBlock {
					fmt.Println("this should happen only once")
				}
				txsFilter := util.NewTxValidationFlagsSetValue(len(t.Block.Data.Data), peer.TxValidationCode_VALID)
				t.Block.Metadata.Metadata[cb.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter
				initialize(t.Block)

				//initApp(blocksize, reps)
				//assertSetup(blocksize, reps)

				firstBlock = false
			}else {
				blocks.Cache.Put(t.Block)
				sequence <-seqNum

				weighted.Acquire(context.Background(), 1)
				go func(w *semaphore.Weighted) {
					defer w.Release(1)

					if err := mcs.VerifyBlockByNo(seqNum); err != nil {
						logger.Errorf("[%s] Error verifying block with sequnce number %d, due to %s", channelID, seqNum, err)
						return
					}
					verified <- seqNum
				}(weighted)
			}
		default:
			logger.Warningf("[%s] Received unknown: ", channelID, t)
			return
		}
	}

}
func commitBlocks(sequence chan uint64, validated chan uint64) {
	backlog := make(map[uint64]uint64, 1024)
	var blockNo uint64
	var ok bool
	for nextToCommit := range sequence {
		if blockNo, ok = backlog[nextToCommit & 1023]; !ok || blockNo != nextToCommit {
			for bn := range validated{
				if bn == nextToCommit {
					blockNo = bn
					break
				}
				backlog[bn & 1023] = bn
			}
		}

		if err := committer.Commit(blockNo); err != nil {
			logger.Panicf("Cannot commit block to the ledger due to %+v", errors.WithStack(err))
		}

		stopwatch.Now("end2end")
	}
}
var logger *logging.Logger // package-level logger

func main() {

	logger = flogging.MustGetLogger("deliverclient")
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


    var serverAddr string
    var seek int
    var quiet bool
    var goroutines uint64
    var numRuns uint64 
    var msgSize uint64
    var messages uint64
    var numBlocks uint64


    flag.StringVar(&serverAddr, "server", fmt.Sprintf("%s:%d", conf.General.ListenAddress, conf.General.ListenPort), "The RPC server to connect to.")
	flag.StringVar(&storageAddr, "storage", fmt.Sprintf("%s:%d", conf.General.ListenAddress, conf.General.ListenPort), "The storage RPC server to connect to.")
    flag.StringVar(&channelID, "channelID", localconfig.Defaults.General.SystemChannel, "The channel ID to deliver from.")
    flag.BoolVar(&quiet, "quiet", true, "Only print the block number, will not attempt to print its block contents.")
    flag.IntVar(&seek, "seek", -2, "Specify the range of requested blocks."+
        "Acceptable values:"+
        "-2 (or -1) to start from oldest (or newest) and keep at it indefinitely."+
        "N >= 0 to fetch block N only.")
    flag.Uint64Var(&goroutines, "goroutines", 2, "The number of concurrent go routines to broadcast the messages on")
    flag.Uint64Var(&msgSize, "size", 1024, "Message size")
    flag.Uint64Var(&messages, "messages", 10000, "#transactions")
    flag.Uint64Var(&numRuns, "run", 1, "Run ")
    flag.Parse()

    experiment.Current = experiment.ExperimentParams{ChannelId:channelID, GrpcAddress:storageAddr}

    numBlocks = 999
    // user
    usr, err := user.Current()
    if err != nil {
        fmt.Println("Error: ", err)
    }
    fmt.Println("seeking.. ", seek)

    now := time.Now()
	stopwatch.SetOutput("end2end", benchutil.OpenFile(fmt.Sprintf("%s/end2end.log", usr.HomeDir)))
	stopwatch.SetOutput("deliver", benchutil.OpenFile(fmt.Sprintf("%s/deliver.log", usr.HomeDir)))


    for r := uint64(0); r < numRuns; r++ {
        
        runDeliverClient(channelID,
                serverAddr ,
                seek ,
                quiet ,
                goroutines ,
                r ,
                msgSize ,
                messages,
                numBlocks,
                usr.HomeDir,
                signer)
            
    }

    fmt.Println(numRuns, "time taken:", time.Since(now))
    fmt.Println("Done")

}

var storageAddr string
var channelID string

func initialize(block *common.Block) {
	grpcmocks.StartClients(storageAddr)
	blocks.Cache.Put(block)

	//call a helper method to load the core.yaml
	testutil.SetupCoreYAMLConfig()

	ledgermgmt.InitializeTestEnv()

	peerLedger, err := ledgermgmt.CreateLedger(block)
	handleError(err, true)

	app = example.ConstructAppInstance(peerLedger)

	crypto2.SetChainID(channelID)

	handleError(err, true)

	consenter = example.ConstructConsenter()
	committer = example.ConstructCommitter(peerLedger)
	validator, err = val.Init(channelID,block,peerLedger)
}

var app *example.App
var committer *example.Committer
var consenter *example.Consenter
var validator *txvalidator.TxValidator

func (s *deliverClient) receive(messages chan *orderer.DeliverResponse) {
	defer close(messages)
	for {
		stopwatch.Measure("deliver", func() {
			msg, err := s.client.Recv()
			if err != nil {
				stopwatch.Flush()
				panic(fmt.Sprintf("[%s] Receive error: %s\n", s.channelID, err.Error()))
				return
			}
			messages <- msg
			})
	}
}

func handleError(err error, quit bool) {
	if err != nil {
		if quit {
			panic(fmt.Errorf("Error: %s\n", err))
		} else {
			fmt.Printf("Error: %s\n", err)
		}
	}
}

func validate(verified <-chan uint64, validated chan uint64){
	fmt.Println("Validation started")
	var weight int64
	if fabric_extension.PipelineWidth/2<1{
		weight = 1
	}else{
		weight = int64(fabric_extension.PipelineWidth/2)
	}
	weighted := semaphore.NewWeighted(weight)
	wg := &sync.WaitGroup{}
	for blockNo := range verified {
		weighted.Acquire(context.Background(), 1)
		wg.Add(1)
		go validateBlock(blockNo, weighted, validated, wg)
	}
	wg.Wait()
	close(validated)
	fmt.Println("Validation completed")
}

func validateBlock( blockNo uint64, weighted *semaphore.Weighted, output chan uint64, wg *sync.WaitGroup) {
	defer wg.Done()
	defer weighted.Release(1)
	err := validator.ValidateByNo(blockNo)
	handleError(err, true)
	output <-blockNo
}
func assertResult(blocksize, reps int) {
	balances := QueryBalances(blocksize, reps)
	iscorrect:=true
	for i, balance := range balances {
		if balance != 2*(i%2) {
			fmt.Println("Account ", i, " has balance ", balance, ", should be ", 2*(i%2))
			iscorrect= false
			break
		}
	}
	if iscorrect {
		fmt.Println("All transactions successfully committed")
	}
}

func QueryBalances(blocksize, reps int) []int {
	totalAccountCount := 2 * blocksize * reps
	accounts := make([]string, 0, totalAccountCount)
	for i := 0; i < totalAccountCount; i++ {
		accounts = append(accounts, fmt.Sprintf("account%d", i))
	}
	balances, err := app.QueryBalances(accounts)
	handleError(err, true)
	return balances
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


func dirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.IsDir()
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
