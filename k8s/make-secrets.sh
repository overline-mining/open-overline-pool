#!/bin/bash

. ./config

kubectl create secret generic api-config-file --from-file=../pool_configs/config.api.json
kubectl create secret generic proxy-config-file --from-file=../pool_configs/config.proxy.json
kubectl create secret generic unlocker-config-file --from-file=../pool_configs/config.unlocker.json
kubectl create secret generic payouts-config-file --from-file=../pool_configs/config.payouts.json

kubectl create secret generic pool-miner-key --from-literal=${POOL_MINER_KEY}
kubectl create secret generic pool-miner-scookie --from-literal=${POOL_NODE_SCOOKIE}
kubectl create secret generic pool-miner-private-key --from-literal=$(cat ${POOL_MINER_PVT_KEY})
kubectl create secret generic pool-fee-key --from-literal=${POOL_FEE_KEY}
