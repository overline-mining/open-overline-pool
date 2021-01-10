#!/bin/bash

trap 'kill %1;' SIGINT
kubectl port-forward open-ov-pool-frontend 8080 7020 &
kubectl port-forward open-ov-pool-api 6283 3141 3142
