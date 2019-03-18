package server

import (
	"fmt"
	"github.com/fabric_extension/grpcmocks"
	"os"
	"path/filepath"
)

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
	devConfigDir, err :=  GetDevConfigDir()
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

type ArrayFlags []string

func (i *ArrayFlags) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *ArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var Endorsers []grpcmocks.StorageClient
