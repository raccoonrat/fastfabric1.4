#!/bin/bash
export FABRIC_ROOT=$GOPATH/src/github.com/hyperledger/fabric
export FABRIC_CFG_PATH=$FABRIC_ROOT/fastfabric-extensions/scripts
export FABRIC_LOGGING_SPEC=WARN
export ORDERER_GENERAL_LISTENADDRESS=0.0.0.0
export ORDERER_GENERAL_GENESISMETHOD=file
export ORDERER_GENERAL_LEDGERTYPE=ram
export ORDERER_GENERAL_GENESISFILE=$FABRIC_CFG_PATH/channel-artifacts/genesis.block
export ORDERER_GENERAL_LOCALMSPID=OrdererMSP # as defined in crypto-config.yaml
export DOMAIN="" # orderer domain as defined in crypto-config.yaml
export ORDERER_ADDRESS="" # change to address of the orderer
export ORDERER_GENERAL_LOCALMSPDIR=$FABRIC_CFG_PATH/crypto-config/ordererOrganizations/$DOMAIN/orderers/$ORDERER_ADDRESS/msp
(cd $FABRIC_ROOT/orderer/ && go install)
orderer start
