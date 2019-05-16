export FABRIC_ROOT=$GOPATH/src/github.com/hyperledger/fabric
export FABRIC_CFG_PATH=$FABRIC_ROOT/fastfabric-extensions/scripts
if [ -d ./crypto-config ]; then rm -r ./crypto-config; fi
if [ -d ./channel-artifacts ]; then rm -r ./channel-artifacts; fi
mkdir channel-artifacts
./bin/cryptogen generate --config=crypto-config.yaml
./bin/configtxgen -outputBlock ./channel-artifacts/genesis.block -profile OneOrgOrdererGenesis -channelID fastfabric-system-channel
./bin/configtxgen -outputCreateChannelTx ./channel-artifacts/channel.tx -profile OneOrgChannel -channelID fastfabric
./bin/configtxgen -outputAnchorPeersUpdate ./channel-artifacts/anchor_peer.tx -profile OneOrgChannel -asOrg Org1MSP -channelID fastfabric
