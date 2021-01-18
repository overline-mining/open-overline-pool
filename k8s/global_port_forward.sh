#!/bin/bash

trap 'kill %1;' SIGINT
kubectl port-forward --address 0.0.0.0 open-overline-pool-frontend 8080 7020 &
kubectl port-forward --address 0.0.0.0 open-overline-pool-api 6283 3141 3142
