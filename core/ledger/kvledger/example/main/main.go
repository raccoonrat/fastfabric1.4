package main

import (
	"flag"
	"fmt"
	"github.com/fabric_extension/experiment"
	"github.com/fabric_extension/util"
	"github.com/hyperledger/fabric/core/ledger/kvledger/example"
	"log"
	"os"
	"os/user"
	"runtime/pprof"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write mem profile to file")
var threadprofile = flag.String("threadprofile", "", "write thread profile to file")
var grprofile = flag.String("grprofile", "", "write goroutine profile to file")
var server = flag.String("server", "localhost", "server address")
var port = flag.String("port", "10000", "port")

func main() {
	flag.Parse()

	params := experiment.ExperimentParams{
		BlockParams: []experiment.BlockParams{{100, 1000}},
		GrpcAddress: fmt.Sprint(*server, ":", *port),
		ChannelId:   "test"}

	if file, ok := isCpuProfiling(); ok {
		fmt.Println("Start profiling")
		defer fmt.Println("Stop profiling")
		defer file.Close()
		pprof.StartCPUProfile(file)
		defer pprof.StopCPUProfile()
	}

	//	params.ThreadParams = make([]example.ThreadParams, 0, 49*49)
	//	for i := 1; i < 50; i++ {
	//		for j := 1; j < 50; j++ {
	//			params.ThreadParams = append(params.ThreadParams,
	//				example.ThreadParams{
	//					BlockPipeline: i,
	//					TxPipeline:    j})
	//		}
	//	}
	//
	params.ThreadParams = []experiment.ThreadParams{{31, 25}}
	experiment.Current = params
	example.Start(params)

	if file, ok := isMemProfiling(); ok {
		defer fmt.Println("Writing heap profile")
		defer file.Close()
		pprof.WriteHeapProfile(file)
	}

	if file, ok := isThreadProfiling(); ok {
		defer fmt.Println("Writing thread profile")
		defer file.Close()
		pprof.Lookup("threadcreate").WriteTo(file, 1)
	}
	if file, ok := isGRProfiling(); ok {
		defer fmt.Println("Writing goroutine profile")
		defer file.Close()
		pprof.Lookup("goroutine").WriteTo(file, 1)
	}
}
func isThreadProfiling() (*os.File, bool) {
	if *threadprofile == "" {
		return nil, false
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	f := util.OpenFile(fmt.Sprintf("%s/%s", usr.HomeDir, *threadprofile))
	if err != nil {
		log.Fatal(err)
	}
	return f, true
}

func isCpuProfiling() (*os.File, bool) {
	if *cpuprofile == "" {
		return nil, false
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	f := util.OpenFile(fmt.Sprintf("%s/%s", usr.HomeDir, *cpuprofile))
	if err != nil {
		log.Fatal(err)
	}
	return f, true
}

func isMemProfiling() (*os.File, bool) {
	if *memprofile == "" {
		return nil, false
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	f := util.OpenFile(fmt.Sprintf("%s/%s", usr.HomeDir, *memprofile))
	if err != nil {
		log.Fatal(err)
	}
	return f, true
}

func isGRProfiling() (*os.File, bool) {
	if *grprofile == "" {
		return nil, false
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	f := util.OpenFile(fmt.Sprintf("%s/%s", usr.HomeDir, *grprofile))
	if err != nil {
		log.Fatal(err)
	}
	return f, true
}
