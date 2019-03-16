/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package example

import (
	"bufio"
	"context"
	"fmt"
	"github.com/fabric_extension"
	"github.com/fabric_extension/block_cache"
	"github.com/fabric_extension/crypto"
	exp "github.com/fabric_extension/experiment"
	"github.com/fabric_extension/grpcmocks"
	"github.com/fabric_extension/stopwatch"
	benchutil "github.com/fabric_extension/util"
	val "github.com/fabric_extension/validator"
	"github.com/golang/protobuf/proto"
	configtxtest "github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	"github.com/hyperledger/fabric/core/ledger/testutil"
	"github.com/hyperledger/fabric/protos/common"
	"golang.org/x/sync/semaphore"
	"log"
	"os/user"
	"sort"
	"sync"
)

var app *App
var committer *Committer
var consenter *Consenter
var validator *txvalidator.TxValidator

var usr *user.User

type InitParams struct{
	Id       string
	GenBlock *common.Block
	Ledger   ledger.PeerLedger
}

var initParams InitParams

func Start(params exp.ExperimentParams) {
	initialize(params)

	for _, tp := range params.ThreadParams {
		fabric_extension.PipelineWidth = tp.BlockPipeline
		fabric_extension.ThreadsPerBlock = tp.TxPipeline
		var err error
		validator, err = val.Init(initParams.Id, initParams.GenBlock, initParams.Ledger)
		handleError(err,true)

		fmt.Println("==============================================")
		fmt.Printf(
			"Current block pipeline width: %d\n" +
			"Current transaction pipeline width: %d\n", tp.BlockPipeline, tp.TxPipeline)

		for _, bp := range params.BlockParams {
			fmt.Println("-------------------")
			fmt.Printf("Current block size: %d\n", bp.BlockSize)
			ExecuteExperiment(bp)
		}
	}
	ledgermgmt.Close()
}

func initialize(params exp.ExperimentParams) {
	var err error
	usr, err = user.Current()
	if err != nil {
		log.Fatal(err)
	}

	grpcmocks.StartClients(params.GrpcAddress)

	//call a helper method to load the core.yaml
	testutil.SetupCoreYAMLConfig()

	ledgermgmt.InitializeTestEnv()


	gb, _ := configtxtest.MakeGenesisBlock(params.ChannelId)
	blocks.Cache.Put(gb)
	peerLedger, err := ledgermgmt.CreateLedger(gb)
	handleError(err, true)

	app = ConstructAppInstance(peerLedger)
	initParams = InitParams{params.ChannelId,gb,peerLedger}


	crypto.SetChainID(params.ChannelId)

	handleError(err, true)

	consenter = ConstructConsenter()
	committer = ConstructCommitter(peerLedger)
}

func ExecuteExperiment(params exp.BlockParams) {
	commitLogFile := benchutil.OpenFile(fmt.Sprintf("%s/commit_%d_%d.log", usr.HomeDir,
		params.BlockSize, params.Repetitions))
	defer commitLogFile.Close()

	stopwatch.SetOutput("Commit", commitLogFile)

	initApp(params)

	assertSetup(params)

	bs := prepareExperimentBlocks(params)

	stopwatch.MeasureWithComment("Commit",
		fmt.Sprintf("threads/block:%d, threads/pipeline:%d",
			fabric_extension.ThreadsPerBlock, fabric_extension.PipelineWidth),
		func() { CommitAllBlocks(bs) })
	stopwatch.Flush()

	assertResult(params)
}

func initApp(params exp.BlockParams) {
	logger.Debug("Entering initApp()")

	//transfers include 2 accounts, so need twice the number as a setup
	totalAccountCount := 2 * params.BlockSize * params.Repetitions
	output := [][]byte{}
	var envs []*common.Envelope
	for accountNo := 0; accountNo < totalAccountCount; {

		accounts := make(map[string]int, params.BlockSize)
		for len(accounts) < params.BlockSize {
			accounts[fmt.Sprint("account", accountNo)] = 1
			accountNo++
		}

		env, err := app.Init(accounts)
		handleError(err, true)
		envs = append(envs, env)

		b,err :=proto.Marshal(env)

		output= append(output,b)
	}

	rawBlock := consenter.ConstructBlock(envs...)
	blocks.Cache.Put(rawBlock)

	err := validator.ValidateByNo(rawBlock.Header.Number)
	handleError(err, true)

	err = committer.Commit(rawBlock.Header.Number)
	handleError(err, true)

	logger.Debug("Exiting initApp()")
	dumpTxs(output, "init_dump.log")
}

func assertSetup(params exp.BlockParams) {
	balances := QueryBalances(params)
	if len(balances) != 2*params.BlockSize*params.Repetitions {
		log.Fatal("Not all accounts setup. Expected",2*params.BlockSize*params.Repetitions, ", actual", len(balances) )
	}
	for i, balance := range balances {
		if balance != 1 {
			log.Fatal("Account ", i, " is not setup correctly")
		}
	}
	fmt.Println("All accounts setup correctly")
}

func assertResult(params exp.BlockParams) {
	balances := QueryBalances(params)

	for i, balance := range balances {
		if balance != 2*(i%2) {
			log.Fatal("Account ", i, " has balance ", balance, ", should be ", 2*(i%2))
		}
	}
	fmt.Println("All transactions successfully committed")
}

func QueryBalances(params exp.BlockParams) []int {
	totalAccountCount := 2 * params.BlockSize * params.Repetitions
	accounts := make([]string, 0, totalAccountCount)
	for i := 0; i < totalAccountCount; i++ {
		accounts = append(accounts, fmt.Sprintf("account%d", i))
	}
	balances, err := app.QueryBalances(accounts)
	handleError(err, true)
	return balances
}

func prepareExperimentBlocks(params exp.BlockParams) []*common.Block {
	bs := make([]*common.Block, params.Repetitions)
	w := sync.WaitGroup{}
	for i := 0; i < params.Repetitions; i++ {
		w.Add(1)
		go func(iteration int) {
			defer w.Done()
			offset := 2 * params.BlockSize * iteration
			bs[iteration] = createExperimentBlock(params.BlockSize, offset)
		}(i)
	}
	w.Wait()
	sort.Slice(bs, func(i,j int)bool{return bs[i].Header.Number < bs[j].Header.Number})
	return bs
}
func createExperimentBlock(blockSize int, offset int) *common.Block {
	txs := make([]*common.Envelope, 0, blockSize)
	output := make([][]byte,blockSize)
	for i := 0; i+1 < 2*blockSize; i += 2 {
		tx, err := app.TransferFunds(
			fmt.Sprint("account", offset+i),
			fmt.Sprint("account", offset+i+1),
			1)
		handleError(err, true)
		txs = append(txs, tx)
		b,err :=proto.Marshal(tx)

		output[i/2]=b
	}

	dumpTxs(output, "tx_dump.log")

	// act as ordering service (consenter) to create a Raw Block from the Transaction
	return consenter.ConstructBlock(txs...)
}
func dumpTxs(txs [][]byte, filename string) {
	dumpFile := benchutil.OpenFile(fmt.Sprintf("%s/%s", usr.HomeDir, filename))
	defer dumpFile.Close()
	writer := bufio.NewWriter(dumpFile)
	defer writer.Flush()
	envs := &grpcmocks.Envelopes{Envs:txs}
	b,err :=proto.Marshal(envs)
	handleError(err, true)
	writer.Write(b)

}

func CommitAllBlocks(bs []*common.Block) {
	fmt.Println("Last block no:", bs[len(bs)-1].Header.Number)

	finalized :=make(chan bool, len(blocks.Cache))
	for i:=0; i<len(blocks.Cache);i++{
		finalized <- true
	}

	cachedBlocks := cache(bs, finalized)
	validatedBlocks := validate(cachedBlocks)

	expectedBlockNo := bs[0].Header.Number
	lastBlockNo := expectedBlockNo + uint64(len(bs))
	backlog := make(map[uint64]uint64, len(bs))
	fmt.Println("Start of Commit")
	for i := expectedBlockNo; i < lastBlockNo; {
		var blockNo uint64
		var ok bool
		if blockNo, ok = backlog[i]; !ok {
			blockNo, ok = <-validatedBlocks
			if !ok {
				break
			}
			if blockNo != i {
				backlog[blockNo] = blockNo
				continue
			}
		}

		var err error
		// act as committing peer to commit the Raw Block
		err = committer.Commit(blockNo)
		finalized <-true
		handleError(err, true)
		i++
	}
}

func cache(bs []*common.Block, finalized chan bool) chan uint64 {
	cachedBlocks := make(chan uint64, len(blocks.Cache)-1)
	go func() {
		for _, b := range bs {
			<-finalized
			blocks.Cache.Put(b)
			cachedBlocks <- b.Header.Number
		}
		close(cachedBlocks)
		fmt.Println("All blocks cached")
	}()
	fmt.Println("Cache initialized")
	return cachedBlocks
}

func validate(cache <-chan uint64) <-chan uint64 {
	preValidated :=make(chan uint64,len(cache))
	go func() {

		var weight int64
		if fabric_extension.PipelineWidth/2<1{
			weight = 1
		}else{
			weight = int64(fabric_extension.PipelineWidth/2)
		}
		weighted := semaphore.NewWeighted(weight)
		msc := crypto.GetService()
		wg := &sync.WaitGroup{}
		for blockNo := range cache {
			weighted.Acquire(context.Background(), 1)
			wg.Add(1)
			go preValidate(msc, blockNo, preValidated, weighted, wg)
		}
		wg.Wait()
		close(preValidated)
		fmt.Println("Prevalidation completed")
	}()
	fmt.Println("Prevalidation started")

	output := make(chan uint64,len(cache))
	go func() {
		var weight int64
		if fabric_extension.PipelineWidth/2<1{
			weight = 1
		}else{
			weight = int64(fabric_extension.PipelineWidth/2)
		}
		weighted := semaphore.NewWeighted(weight)
		wg := &sync.WaitGroup{}
		for blockNo := range preValidated {
			weighted.Acquire(context.Background(), 1)
			wg.Add(1)
			go validateBlock(blockNo, weighted, output, wg)
		}
		wg.Wait()
		close(output)
		fmt.Println("Validation completed")
	}()
	fmt.Println("Validation started")
	return output
}

func preValidate(msc *crypto.MspMessageCryptoService, blockNo uint64, preValidated chan uint64, w *semaphore.Weighted, wg *sync.WaitGroup) {
	defer wg.Done()
	defer w.Release(1)
	if err := msc.VerifyBlockByNo(blockNo); err == nil {
		preValidated <- blockNo
	}else{
		fmt.Println("Prevalidation for block", blockNo, "failed:", err)
	}
}

func validateBlock( blockNo uint64, weighted *semaphore.Weighted, output chan uint64, wg *sync.WaitGroup) {
		defer wg.Done()
		defer weighted.Release(1)
		err := validator.ValidateByNo(blockNo)
		handleError(err, true)
		output <-blockNo
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