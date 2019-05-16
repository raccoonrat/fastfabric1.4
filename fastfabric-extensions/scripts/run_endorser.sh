#!/bin/bash
export FABRIC_ROOT=$GOPATH/src/github.com/hyperledger/fabric
export FABRIC_CFG_PATH=$FABRIC_ROOT/fastfabric-extensions/scripts
export FABRIC_LOGGING_SPEC=WARN
export FAST_PEER_ADDRESS="" # change to address of fast peer
export CORE_PEER_ID=$FAST_PEER_ADDRESS:7051
export ENDORSER_ADDRESS="" # change to address of the endorser
export CORE_PEER_ADDRESS=$ENDORSER_ADDRESS:7051
export CORE_PEER_LISTENADDRESS=0.0.0.0:7051
export CORE_PEER_GOSSIP_EXTERNALENDPOINT=$ENDORSER_ADDRESS:7051
export CORE_PEER_CHAINCODEADDRESS=$ENDORSER_ADDRESS:7052
export CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:7052
export CORE_PEER_LOCALMSPID=Org1MSP # as defined in crypto-config.yaml
export DOMAIN="" # peer domain as defined in crypto-config.yaml
export CORE_PEER_MSPCONFIGPATH=$FABRIC_CFG_PATH/crypto-config/peerOrganizations/$DOMAIN/peers/$FAST_PEER_ADDRESS/msp
rm /var/hyperledger/production/* -r # clean up data from previous runs
(cd $FABRIC_ROOT/peer/ && go install)
export STORAGE_ADDRESS = "" # address of the storage server
peer node start -a $ENDORSER_ADDRESS:10000 -e --storageAddr $STORAGE_ADDRESS:10000
