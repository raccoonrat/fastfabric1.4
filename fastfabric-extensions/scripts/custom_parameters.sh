#!/usr/bin/env bash


export PEER_DOMAIN=""
export ORDERER_DOMAIN=""

# fill in the addresses without domain suffix and without ports
export FAST_PEER_ADDRESS=""

# leave endorser address and storage address blank if you want to run on a single server
export ENDORSER_ADDRESS=() # can define multiple addresses in the form ( "addr1" "addr2" ... )
export STORAGE_ADDRESS=""

export ORDERER_ADDRESS=""