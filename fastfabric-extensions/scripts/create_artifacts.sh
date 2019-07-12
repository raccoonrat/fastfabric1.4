#!/usr/bin/env bash
bash base_parameters.sh

if [ ! -f crypto-config.yaml.bak ]; then
    cp crypto-config.yaml crypto-config.yaml.bak
fi

(cat crypto-config.yaml.bak | sed "s/ORDERER_DOMAIN/$ORDERER_DOMAIN/g" | sed "s/ORDERER_ADDRESS/$ORDERER_ADDRESS/g"| sed "s/PEER_DOMAIN/$PEER_DOMAIN/g"| sed "s/FAST_PEER_ADDRESS/$FAST_PEER_ADDRESS/g") > crypto-config.yaml

if [ ! -f configtx.yaml.bak ]; then
    cp configtx.yaml configtx.yaml.bak
fi
(cat configtx.yaml.bak | sed "s/ORDERER_DOMAIN/$ORDERER_DOMAIN/g" | sed "s/ORDERER_ADDRESS/$ORDERER_ADDRESS/g"| sed "s/PEER_DOMAIN/$PEER_DOMAIN/g"| sed "s/FAST_PEER_ADDRESS/$FAST_PEER_ADDRESS/g") > configtx.yaml

if [ -d ./crypto-config ]; then rm -r ./crypto-config; fi
if [ -d ./channel-artifacts ]; then rm -r ./channel-artifacts; fi
mkdir channel-artifacts
./bin/cryptogen generate --config=crypto-config.yaml
./bin/configtxgen -configPath ./ -outputBlock ./channel-artifacts/genesis.block -profile OneOrgOrdererGenesis -channelID fastfabric-system-channel
./bin/configtxgen -configPath ./ -outputCreateChannelTx ./channel-artifacts/channel.tx -profile OneOrgChannel -channelID fastfabric
./bin/configtxgen -configPath ./ -outputAnchorPeersUpdate ./channel-artifacts/anchor_peer.tx -profile OneOrgChannel -asOrg Org1MSP -channelID fastfabric
