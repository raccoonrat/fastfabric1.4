/*
Copyright IBM Corp. 2017 All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package main is the entrypoint for the orderer binary
// and calls only into the server.Main() function.  No other
// function should be included in this package.
package main

import (
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	"github.com/hyperledger/fabric/orderer/common/server"
	"os"
	"strconv"
)

const (
    MaxMessageCount = 100
    AbsoluteMaxBytes  = 512 // KB
    PreferredMaxBytes = 512 // KB
)

var envvars = map[string]string{
    "ORDERER_GENERAL_LISTENADDRESS":                                "0.0.0.0",
    "ORDERER_GENERAL_GENESISPROFILE":                              genesisconfig.SampleDevModeSoloProfile,
    "ORDERER_GENERAL_LEDGERTYPE":                                  "ram",
    "ORDERER_GENERAL_LOGLEVEL":                                    "ERROR",
    "ORDERER_KAFKA_VERBOSE":                                       "false",
    genesisconfig.Prefix + "_ORDERER_BATCHSIZE_MAXMESSAGECOUNT":   strconv.Itoa(MaxMessageCount),
    genesisconfig.Prefix + "_ORDERER_BATCHSIZE_ABSOLUTEMAXBYTES":  strconv.Itoa(AbsoluteMaxBytes) + " KB",
    genesisconfig.Prefix + "_ORDERER_BATCHSIZE_PREFERREDMAXBYTES": strconv.Itoa(PreferredMaxBytes) + " KB",
    genesisconfig.Prefix + "_ORDERER_KAFKA_BROKERS":               "[localhost:9092, localhost:9092, localhost:9092]",
}

func main() {
    server.SetDevFabricConfigPath()
    for key, value := range envvars {
        os.Setenv(key, value)
    }
    os.Setenv("ORDERER_GENERAL_GENESISPROFILE", genesisconfig.SampleDevModeKafkaProfile)

    server.Main()
}

