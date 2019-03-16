/*
Copyright IBM Corp. 2017 All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package main is the entrypoint for the orderer binary
// and calls only into the server.Main() function.  No other
// function should be included in this package.
package main

import (
    "fmt"
    genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
    "github.com/hyperledger/fabric/orderer/common/server"
    "os"
	"path/filepath"
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
    genesisconfig.Prefix + "_ORDERER_KAFKA_BROKERS":               "[gho9:9092, gho10:9092, gho11:9092]",
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



func main() {
    SetDevFabricConfigPath()
    for key, value := range envvars {
        os.Setenv(key, value)
    }
    os.Setenv("ORDERER_GENERAL_GENESISPROFILE", genesisconfig.SampleDevModeKafkaProfile)
    
    server.Main()
}

