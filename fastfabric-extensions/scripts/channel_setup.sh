#!/bin/bash
source base_parameters.sh

export CORE_PEER_MSPCONFIGPATH=./crypto-config/peerOrganizations/${PEER_DOMAIN}/users/Admin@${PEER_DOMAIN}/msp
export CORE_PEER_ADDRESS=${FAST_PEER_ADDRESS}:7051

peer channel create -o ${ORDERER_ADDRESS}:7050 -c fastfabric -f ./channel-artifacts/channel.tx
peer channel join -b fastfabric.block

for i in ${ENDORSER_ADDRESS[@]}
do
    export CORE_PEER_ADDRESS=$(get_correct_peer_address ${i}):7051
    peer channel join -b fastfabric.block
done

if [[ ${FAST_PEER_ADDRESS} != "localhost" ]]
then
    p_addr=${FAST_PEER_ADDRESS}.${PEER_DOMAIN}
else
    p_addr=${FAST_PEER_ADDRESS}
fi

export CORE_PEER_ADDRESS=${p_addr}:7051
peer channel update -o $(get_correct_orderer_address):7050 -c fastfabric -f ./channel-artifacts/anchor_peer.tx
