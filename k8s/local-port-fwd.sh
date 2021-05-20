#!/bin/bash

trap 'kill %1; kill %2' SIGINT
kubectl port-forward open-overline-pool-frontend 8080 7020 &
kubectl port-forward open-overline-pool-api 21111 &
kubectl port-forward open-overline-pool-proxy 12111 11112
