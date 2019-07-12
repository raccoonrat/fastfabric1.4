#!/bin/bash
source base_parameters.sh

export CORE_PEER_MSPCONFIGPATH=./crypto-config/peerOrganizations/${PEER_DOMAIN}/users/Admin@${PEER_DOMAIN}/msp
for i in ${ENDORSER_ADDRESS[@]}
do
    export CORE_PEER_ADDRESS=${i}:7051
    peer chaincode install -l golang -n benchmark -v 1.0 -o ${ORDERER_ADDRESS}:7050 -p "github.com/hyperledger/fabric/fastfabric-extensions/chaincode"
    a="'{\"Args\":[\"init\",\"0\", \"1\", \"0\"]}'"
    echo peer chaincode instantiate -o ${ORDERER_ADDRESS}:7050 -C fastfabric -n benchmark -v 1.0 -c ${a} | bash
done

sleep 5

bash chaincode_account_setup.sh $1 $2 $3
