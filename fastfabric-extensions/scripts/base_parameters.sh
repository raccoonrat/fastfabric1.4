#!/usr/bin/env bash

source custom_parameters.sh

export FABRIC_ROOT=$GOPATH/src/github.com/hyperledger/fabric
export FABRIC_CFG_PATH=${FABRIC_ROOT}/fastfabric-extensions/scripts #change this if you want to copy the script folder somewhere else before modifying it

export CORE_PEER_ID=${FAST_PEER_ADDRESS}.${PEER_DOMAIN}
export CORE_PEER_ADDRESS=${FAST_PEER_ADDRESS}:7051
export CORE_PEER_LISTENADDRESS=0.0.0.0:7051
export CORE_PEER_GOSSIP_EXTERNALENDPOINT=${FAST_PEER_ADDRESS}:7051
export CORE_PEER_CHAINCODEADDRESS=${ENDORSER_ADDRESS}:7052
export CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:7052
export CORE_PEER_LOCALMSPID=Org1MSP

export ORDERER_GENERAL_LISTENADDRESS=0.0.0.0
export ORDERER_GENERAL_GENESISMETHOD=file
export ORDERER_GENERAL_LEDGERTYPE=ram
export ORDERER_GENERAL_GENESISFILE=${FABRIC_CFG_PATH}/channel-artifacts/genesis.block
export ORDERER_GENERAL_LOCALMSPID=OrdererMSP