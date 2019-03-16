package main

import (
	//	"bytes"
	//	"os"
	"fmt"
	"github.com/hyperledger/fabric/peer/node"
	//	"github.com/hyperledger/fabric/common/viperutil"
	//	"github.com/hyperledger/fabric/core/handlers/library"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	//	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	//	"github.com/stretchr/testify/assert"
	//	"google.golang.org/grpc"
)

func SetViper() {
	viper.Set("peer.address", "0.0.0.0:7051")
	viper.Set("peer.listenAddress", "0.0.0.0:7051")
	viper.Set("peer.chaincodeListenAddress", "0.0.0.0:6052")
	viper.Set("peer.fileSystemPath", "./tmp/hyperledger/test")
	viper.Set("chaincode.executetimeout", "30s")
	viper.Set("chaincode.mode", "dev")
	overrideLogModules := []string{"msp", "gossip", "ledger", "cauthdsl", "policies", "grpc"}
	for _, module := range overrideLogModules {
		viper.Set("logging."+module, "INFO")
	}

}

func main() {
	SetViper()
	msptesttools.LoadMSPSetupForTesting()
        wait := make(chan bool)


	go func() {
		fmt.Println("Starting peer..")
		cmd := node.PerfStart()
		//assert.NoError(t, cmd.Execute(), "expected to successfully start command")
		cmd.Execute()
	}()
	<-wait

}
