#!/bin/bash
source base_parameters.sh

export FABRIC_LOGGING_SPEC=WARN
export CORE_PEER_MSPCONFIGPATH=${FABRIC_CFG_PATH}/crypto-config/peerOrganizations/${PEER_DOMAIN}/peers/${FAST_PEER_ADDRESS}.${PEER_DOMAIN}/msp

rm /var/hyperledger/production/* -r # clean up data from previous runs
(cd ${FABRIC_ROOT}/peer/ && go install)

peer node start -a ${STORAGE_ADDRESS}:10000 -s
