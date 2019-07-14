
# FastFabric


**Note:** This is a fork of the Hyperledger Fabric repository (https://github.com/hyperledger/fabric) and contains the source code for the FastFabric publication (https://arxiv.org/abs/1901.00910).

The work done for the publication is found in the `fastfabric` branch. The focus was set on testing core performance of the block commitment process, so the code in that branch is not intended to be run as a fully setup system.

All described optimizations from the publication (except transaction header separation) were reimplemented for the Fabric 1.4 release version in branch `fastfabric-1.4`. With that new implementation, a full Fabric network can be set up and it also includes helper scripts. **Caveat:** DEBUG level logging has been completely disabled, because it slowed down the system dramatically even when a higher logging level was chosen. In the following, we describe how to create a test setup using the `fastfabric-1.4` branch.


## Prerequisites

- The Hyperledger Fabric prerequisites are installed
- `$GOPATH` and `$GOPATH/bin` are added to `$PATH`
- The instructions assume that the repository is cloned to `$GOPATH/src/github.com/hyperledger/fabric`
- Add the `cryptogen`and `configtxgen` binaries to a new `$GOPATH/src/github.com/hyperledger/fabric/fastfabric-extensions/scripts/bin` folder
- Ensure that the folder `/var/hyperledger/production/` exists and the user executing the scripts has sufficient permissions to read and write to that folder.

## Instructions

All following steps use scripts from the  `fabric/fastfabric-extensions/scripts` folder. These example scripts create a network consisting of a single orderer and a single peer. Optionally, the endorsement and ledger storage can be run on separate servers as described in our paper.
- Open `custom_parameters.sh` and fill in the addresses and the domains of the orderer and the peer. In order to run the peer with decoupled endorsement and/or storage fill in `ENDORSER_ADDRESS` and/or `STORAGE_ADDRESS` as well. `ENDORSER_ADDRESS` expects an array of addresses, so it is possible to run more than one decoupled endorser for the same peer.
- Execute `create_artifact.sh` to create the prerequisite files to setup the network, channel and anchor peer.
- Execute `run_orderer.sh` on the respective server that should form the ordering service.
- (Optional) Execute `run_storage.sh` on the server that should persist the blockchain and world state.
- (Optional) Execute `run_endorser.sh` on the server that should act as a decoupled endorser.
- Execute `run_fastpeer.sh` on the server that should act as the known peer of the network.
- Execute `channel_setup.sh` on any server in the network to set up a channel named _fastfabric_
- Execute `chaincode_setup.sh` on any server in the network. Command should have the form `bash chaincode_setup.sh [lower limit of account index range] [upper limit of account index range] [value per account]`. **Example**: `bash chaincode_setup.sh 0 10 100`. This will install a chaincode named _benchmark_ with the designated number of accounts, which will accept _query_ or _transfer_ invocations. **Example**: To transfer 10 coins from _account 13_ to _account 27_ via cli command do `peer chaincode invoke -C fastfabric -n benchmark -c '{"Args":["transfer","account13","account27", "10"]}'`.

This should set up an orderer, a persistent storage peer (optional), an endorser peer (optional), a fast validation peer, a channel and an example chaincode. **Important:** If the optional decoupled storage and endorers are used, the set up order must be storage -> endorser -> fast peer because of their communication dependencies.
