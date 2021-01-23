#!/bin/bash

kubectl create secret generic api-config-file --from-file=../pool_configs/config.api.json
kubectl create secret generic proxy-config-file --from-file=../pool_configs/config.proxy.json
kubectl create secret generic unlocker-config-file --from-file=../pool_configs/config.unlocker.json
kubectl create secret generic payouts-config-file --from-file=../pool_configs/config.payouts.json
