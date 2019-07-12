#!/bin/bash
source base_parameters.sh

export FABRIC_LOGGING_SPEC=WARN
export CORE_PEER_MSPCONFIGPATH=${FABRIC_CFG_PATH}/crypto-config/peerOrganizations/${PEER_DOMAIN}/peers/${FAST_PEER_ADDRESS}.${PEER_DOMAIN}/msp

rm /var/hyperledger/production/* -r # clean up data from previous runs
(cd ${FABRIC_ROOT}/peer/ && go install)

# peer node start can be run without the storageAddr and endorserAddr parameters. In that case those modules will not be decoupled to different nodes
s=""
if [[ ! -z ${STORAGE_ADDRESS} ]]
then
    echo "Starting with decoupled storage server ${STORAGE_ADDRESS}"
    s="--storageAddr $(get_correct_peer_address ${STORAGE_ADDRESS}):10000"
fi

e=""
for i in ${ENDORSER_ADDRESS[@]}
do
    echo "Starting with decoupled endorser server $i"
    e="$e --endorserAddr $(get_correct_peer_address ${i}):10000"
done
peer node start ${s} ${e}

