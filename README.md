
# FastFabric


**Note:** This is a fork of the Hyperledger Fabric repository (https://github.com/hyperledger/fabric) and contains the source code for the FastFabric publication (https://arxiv.org/abs/1901.00910).

The work done for the publication is found in the `fastfabric` branch. The focus was set on testing core performance of the block commitment process, so the code in that branch is not intended to be run as a fully setup system.

All described optimizations from the publication (except transaction header separation) were reimplemented for the Fabric 1.4 release version in branch `fastfabric-1.4`. With that new implementation, a full Fabric network can be set up and it also includes helper scripts. **Caveat:** DEBUG level logging has been completely disabled, because it slowed down the system dramatically even when a higher logging level was chosen. In the following, we describe how to create a test setup using the `fastfabric-1.4` branch.


## Prerequisites

- The Hyperledger Fabric prerequisites are installed
- `$GOPATH` and `$GOPATH/bin` are added to `$PATH`
- The instructions assume that the repository is cloned to `$GOPATH/src/github.com/hyperledger/fabric`
- Add the `cryptogen`and `configtxgen` binaries to a new `$GOPATH/src/github.com/hyperledger/fabric/fastfabric-extensions/scripts/bin` folder

## Instructions

All following steps use scripts from the  `fabric/fastfabric-extensions/scripts` folder.
- `configtx.yaml` describes a simple network setup consisting of a single orderer and a single peer. Modify the script based on the comments in the file.
- `crypto-config.yaml` describes the setup for the msp folder. Modify the script based on the comments in the file. 
- Run `create_artifact.sh` to create the prerequisite files to setup the network, channel and anchor peer.
- Modify `run_orderer.sh` based on the comments in the file and run it on the respective server that should form the ordering service
- Modify `run_storage.sh` based on the comments in the file and run it on the server that should persist the blockchain and world state
- Modify `run_endorser.sh` based on the comments in the file and run it on the server that should act as a decoupled endorser
- Modify `run_fastpeer.sh` based on the comments in the file and run it on the server that should validate incoming blocks
- Modify `channel_setup.sh` based on the comments in the file and run it on any of the servers that has a peer installed
- Modify `chaincode_setup.sh` based on the comments in the file and run it on any of the servers that has a peer installed. Command should have the form `./chaincode_setup.sh [lower limit of account index range] [upper limit of account index range] [value per account]`. Example: `./chaincode_setup.sh 0 10 100`

This should set up an endorser, persistent storage peer, endorser peer and fast validation peer. **Important:** The peers need to be set up in the order storage -> endorser -> fast peer because of their dependencies.
