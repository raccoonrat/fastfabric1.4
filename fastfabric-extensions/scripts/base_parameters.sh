#!/usr/bin/env bash
source custom_parameters.sh

get_correct_address () {
    if [[ $1 != "localhost" ]]
    then
        addr=$1.$2
    else
        addr=$1
    fi

    return ${addr}
}

get_correct_peer_address(){
    return $(get_correct_address $1 ${PEER_DOMAIN})
}

get_correct_orderer_address(){
    return $(get_correct_address $ORDERER_ADDRESS ${ORDERER_DOMAIN})
}


export FABRIC_ROOT=$GOPATH/src/github.com/hyperledger/fabric
export FABRIC_CFG_PATH=${FABRIC_ROOT}/fastfabric-extensions/scripts #change this if you want to copy the script folder somewhere else before modifying it

p_addr=$(get_correct_peer_address $FAST_PEER_ADDRESS)
export CORE_PEER_ID=${p_addr}
export CORE_PEER_ADDRESS=${p_addr}:7051
export CORE_PEER_GOSSIP_EXTERNALENDPOINT=${p_addr}:7051
export CORE_PEER_CHAINCODEADDRESS=${p_addr}:7052


export CORE_PEER_LISTENADDRESS=0.0.0.0:7051
export CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:7052
export CORE_PEER_LOCALMSPID=Org1MSP

export ORDERER_GENERAL_LISTENADDRESS=0.0.0.0
export ORDERER_GENERAL_GENESISMETHOD=file
export ORDERER_GENERAL_LEDGERTYPE=ram
export ORDERER_GENERAL_GENESISFILE=${FABRIC_CFG_PATH}/channel-artifacts/genesis.block
export ORDERER_GENERAL_LOCALMSPID=OrdererMSP