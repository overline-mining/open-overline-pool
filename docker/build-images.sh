#!/bin/bash
set -eo pipefail

docker build -t local/bcnode -f Dockerfile.zano .
docker build -t local/open-zano-pool-api -f Dockerfile.api .
docker build -t local/open-overline-pool-frontend -f Dockerfile.frontend .

