#!/bin/bash

trap 'kill %1;kill %2' SIGINT
kubectl port-forward --address 0.0.0.0 open-overline-pool-frontend 80 7020 &
kubectl port-forward --address 0.0.0.0 open-overline-pool-api 21111 &
kubectl port-forward --address 0.0.0.0 open-overline-pool-proxy 12111 11112
