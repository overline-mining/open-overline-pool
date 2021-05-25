#!/bin/bash
set -eo pipefail

#docker build -t local/zano -f Dockerfile.zano .
docker build -t local/open-zano-pool-api -f Dockerfile.api .
docker build -t local/open-zano-pool-frontend -f Dockerfile.frontend .

