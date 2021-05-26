#!/bin/bash

. ./config

kubectl create secret generic api-config-file --from-file=../pool_configs/config.api.json
kubectl create secret generic proxy-config-file --from-file=../pool_configs/config.proxy.json
kubectl create secret generic unlocker-config-file --from-file=../pool_configs/config.unlocker.json
kubectl create secret generic payouts-config-file --from-file=../pool_configs/config.payouts.json

kubectl create secret generic pool-wallet-file --from-file=${POOL_MINER_WALLET}

kubectl create secret generic pool-miner-key --from-literal=value=${POOL_MINER_KEY}
kubectl create secret generic pool-miner-pwd --from-literal=value=$(cat ${POOL_MINER_PWD})
kubectl create secret generic pool-fee-key --from-literal=value=${POOL_FEE_KEY}
