#!/bin/bash
export FABRIC_ROOT=$GOPATH/src/github.com/hyperledger/fabric
export FABRIC_CFG_PATH=$FABRIC_ROOT/fastfabric-extensions/scripts
export CORE_PEER_LOCALMSPID=Org1MSP
export ENDORSER_ADDRESS = "" # address of endorser
export DOMAIN="" # peer domain as defined in crypto-config.yaml
export ORDERER_ADDRESS="" # change to address of the orderer
export CORE_PEER_ADDRESS=$ENDORSER_ADDRESS:7051
export CORE_PEER_MSPCONFIGPATH=./crypto-config/peerOrganizations/$DOMAIN/users/Admin@$DOMAIN/msp

index=$1
remainder=$2
count=0

while [ $remainder -gt 0 ]
do
    if [ $remainder -gt 100000 ]
    then
        count=100000
    else
        count=$remainder
    fi
    remainder=$((remainder - count))

    a="'{\"Args\":[\"init\",\"$index\", \"$count\", \"$3\"]}'"
    echo Instantiating accounts $index to $((index + count -1 ))
    echo "peer chaincode invoke -o $ORDERER_ADDRESS:7050 -C fastfabric -n benchmark -c $a" | bash
    index=$((index + count))
done
echo All done!
