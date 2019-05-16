#!/bin/bash
export FABRIC_CFG_PATH=$FABRIC_ROOT/fastfabric-extensions/scripts
export CORE_PEER_LOCALMSPID=Org1MSP
export DOMAIN="" # peer domain as defined in crypto-config.yaml
export FAST_PEER_ADDRESS="" # change to address of fast peer
export ENDORSER_ADDRESS = "" # address of endorser
export ORDERER_ADDRESS="" # change to address of the orderer
export CORE_PEER_MSPCONFIGPATH=./crypto-config/peerOrganizations/$DOMAIN/users/Admin@$DOMAIN/msp
export CORE_PEER_ADDRESS=$FAST_PEER_ADDRESS:7051
peer channel create -o $ORDERER_ADDRESS:7050 -c fastfabric -f ./channel-artifacts/channel.tx
peer channel join -b fastfabric.block
export CORE_PEER_ADDRESS=$ENDORSER_ADDRESS:7051
peer channel join -b fastfabric.block
export CORE_PEER_ADDRESS=$FAST_PEER_ADDRESS:7051
peer channel update -o $ORDERER_ADDRESS:7050 -c fastfabric -f ./channel-artifacts/anchor_peer.tx