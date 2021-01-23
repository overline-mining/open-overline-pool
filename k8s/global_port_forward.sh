#!/bin/bash

trap 'kill %1;kill %2' SIGINT
kubectl port-forward --address 0.0.0.0 open-overline-pool-frontend 8080 7020 &
kubectl port-forward --address 0.0.0.0 open-overline-pool-api 6283 &
kubectl port-forward --address 0.0.0.0 open-overline-pool-proxy 3141 3142
