// Copyright IBM Corp. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"github.com/fabric_extension"
	"github.com/fabric_extension/block_cache"
	"github.com/fabric_extension/commit"
	"github.com/fabric_extension/experiment"
	"github.com/fabric_extension/grpcmocks"
	"github.com/fabric_extension/stopwatch"
	benchutil "github.com/fabric_extension/util"
	val "github.com/fabric_extension/validator"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/ledger/kvledger/example"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	"github.com/hyperledger/fabric/core/ledger/testutil"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/orderer/common/server"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
	"log"
	"math"
	"os"
	"os/signal"
	"sync"

	crypto2 "github.com/fabric_extension/crypto"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/localmsp"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/utils"
	"os/user"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	oldest  = &ab.SeekPosition{Type: &ab.SeekPosition_Oldest{Oldest: &ab.SeekOldest{}}}
	newest  = &ab.SeekPosition{Type: &ab.SeekPosition_Newest{Newest: &ab.SeekNewest{}}}
	maxStop = &ab.SeekPosition{Type: &ab.SeekPosition_Specified{Specified: &ab.SeekSpecified{Number: math.MaxUint64}}}
	logger *logging.Logger // package-level logger

	storageAddr string
	endorseAddr server.ArrayFlags
	channelID string
	storage grpcmocks.StorageClient
)

func main() {
	logger = flogging.MustGetLogger("deliverclient")
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

	var serverAddr string
	var seek int
	var quiet bool

	flag.StringVar(&serverAddr, "server", fmt.Sprintf("%s:%d", conf.General.ListenAddress, conf.General.ListenPort), "The RPC server to connect to.")
	flag.StringVar(&storageAddr, "storage", fmt.Sprintf("%s:%d", conf.General.ListenAddress, conf.General.ListenPort), "The storage RPC server to connect to.")
	flag.Var(&endorseAddr, "endorse", "The endorse RPC server to connect to.")
	flag.StringVar(&channelID, "channelID", localconfig.Defaults.General.SystemChannel, "The channel ID to deliver from.")
	flag.BoolVar(&quiet, "quiet", true, "Only print the block number, will not attempt to print its block contents.")
	flag.IntVar(&seek, "seek", -2, "Specify the range of requested blocks."+
		"Acceptable values:"+
		"-2 (or -1) to start from oldest (or newest) and keep at it indefinitely."+
		"N >= 0 to fetch block N only.")
	flag.Parse()

	experiment.Current = experiment.ExperimentParams{ChannelId: channelID, GrpcAddress: storageAddr}

	// user
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Error: ", err)
	}
	fmt.Println("seeking.. ", seek)

	now := time.Now()
	stopwatch.SetOutput("end2end", benchutil.OpenFile(fmt.Sprintf("%s/end2end.log", usr.HomeDir)))
	stopwatch.SetOutput("deliver", benchutil.OpenFile(fmt.Sprintf("%s/deliver.log", usr.HomeDir)))

	runDeliverClient(channelID, serverAddr, seek, quiet, signer)


	fmt.Println("time taken:", time.Since(now))
	fmt.Println("Done")

}

func initialize(block *common.Block) {
	fmt.Println("Storage server address:", storageAddr)
	storage = grpcmocks.StartClient(storageAddr)
	fmt.Println("Endorser server address:", endorseAddr)
	for _, addr := range endorseAddr {
		end := grpcmocks.StartClient(addr)
		server.Endorsers = append(server.Endorsers, end)
		_, err := end.CreateLedger(context.Background(), block)
		if err != nil {
			log.Fatal(err)
		}
	}
	_, err := storage.CreateLedger(context.Background(), block)
	if err != nil {
		log.Fatal(err)
	}

	blocks.Cache.Put(block)

	//call a helper method to load the core.yaml
	testutil.SetupCoreYAMLConfig()

	ledgermgmt.InitializeTestEnv()

	peerLedger, err := ledgermgmt.CreateLedger(block)
	handleError(err, true)

	crypto2.SetChainID(channelID)

	handleError(err, true)

	committer = example.ConstructCommitter(peerLedger)
	validator, err = val.Init(channelID, block, peerLedger)

	go sendOutput()
}

var output chan uint64
func sendOutput() {
	output = make(chan uint64, 100000)
	for no := range output {
		b, err := blocks.Cache.Get(no)
		if err != nil {
			fmt.Println("blockNo:", no)
			panic(err)
		}
		storage.Store(context.Background(), b.Rawblock)
		for _, en := range server.Endorsers {
			en.Store(context.Background(), b.Rawblock)
		}
	}
}

var committer *example.Committer
var validator *txvalidator.TxValidator

func handleError(err error, quit bool) {
	if err != nil {
		if quit {
			panic(fmt.Errorf("Error: %s\n", err))
		} else {
			fmt.Printf("Error: %s\n", err)
		}
	}
}

func runDeliverClient(channelID string,
	serverAddr string,
	seek int,
	quiet bool,
	signer crypto.LocalSigner) {

	fmt.Println("Running....")

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
	if fabric_extension.PipelineWidth/2 < 1 {
		weight = 1
	} else {
		weight = int64(fabric_extension.PipelineWidth / 2)
	}
	weighted := semaphore.NewWeighted(weight)

	const reps = 10
	const blocksize = 100

	sequence := commit.GetSequence()
	verified := commit.GetVerified()
	validated := commit.GetValidated()
	go commitBlocks(sequence, validated)
	go validate(verified, validated)
	firstBlock := true

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
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
			if seqNum == 0 {
				if !firstBlock {
					fmt.Println("this should happen only once")
				}
				txsFilter := util.NewTxValidationFlagsSetValue(len(t.Block.Data.Data), peer.TxValidationCode_VALID)
				t.Block.Metadata.Metadata[cb.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter
				initialize(t.Block)

				firstBlock = false
			} else {
				blocks.Cache.Put(t.Block)
				sequence <- seqNum

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
		if blockNo, ok = backlog[nextToCommit&1023]; !ok || blockNo != nextToCommit {
			for bn := range validated {
				if bn == nextToCommit {
					blockNo = bn
					break
				}
				backlog[bn&1023] = bn
			}
		}

		if err := committer.Commit(blockNo); err != nil {
			logger.Panicf("Cannot commit block to the ledger due to %+v", errors.WithStack(err))
		}
		output <- blockNo
		stopwatch.Now("end2end")
	}
}

func validate(verified <-chan uint64, validated chan uint64) {
	fmt.Println("Validation started")
	var weight int64
	if fabric_extension.PipelineWidth/2 < 1 {
		weight = 1
	} else {
		weight = int64(fabric_extension.PipelineWidth / 2)
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

func validateBlock(blockNo uint64, weighted *semaphore.Weighted, output chan uint64, wg *sync.WaitGroup) {
	defer wg.Done()
	defer weighted.Release(1)
	err := validator.ValidateByNo(blockNo)
	handleError(err, true)
	output <- blockNo
}

func assertResult(blocksize, reps int) {
	balances := queryBalances(blocksize, reps)
	iscorrect := true
	for i, balance := range balances {
		if int(balance) != 2*(i%2) {
			fmt.Println("Account ", i, " has balance ", balance, ", should be ", 2*(i%2))
			iscorrect = false
			break
		}
	}
	if iscorrect {
		fmt.Println("All transactions successfully committed")
	}
}

func queryBalances(blocksize, reps int) []int32 {
	totalAccountCount := 2 * blocksize * reps
	accounts := make([]string, 0, totalAccountCount)
	for i := 0; i < totalAccountCount; i++ {
		accounts = append(accounts, fmt.Sprintf("account%d", i))
	}
	balances, err := storage.QueryBalances(context.Background(), &grpcmocks.Accounts{Accs: accounts})
	handleError(err, true)
	return balances.Bal
}

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
