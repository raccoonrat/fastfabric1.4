#!/bin/bash
export FABRIC_CFG_PATH=$HOME/fabric_channel_setup
export CORE_PEER_LOCALMSPID=Org1MSP
export ENDORSER_ADDRESS = "" # address of endorser
export DOMAIN="" # peer domain as defined in crypto-config.yaml
export ORDERER_ADDRESS="" # change to address of the orderer
export CORE_PEER_ADDRESS=$ENDORSER_ADDRESS:7051
export CORE_PEER_MSPCONFIGPATH=./crypto-config/peerOrganizations/$DOMAIN/users/Admin@$DOMAIN/msp
peer chaincode install -l golang -n benchmark -v 1.0 -o $ORDERER_ADDRESS:7050 -p "github.com/hyperledger/fabric/fastfabric-extensions/chaincode"

a="'{\"Args\":[\"init\",\"0\", \"1\", \"0\"]}'"
echo peer chaincode instantiate -o $ORDERER_ADDRESS:7050 -C fastfabric -n benchmark -v 1.0 -c $a | bash

sleep 5

./chaincode_account_setup.sh $1 $2 $3
