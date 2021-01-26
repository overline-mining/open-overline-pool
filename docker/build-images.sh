#!/bin/bash
set -eo pipefail

POOL_TARBALL_PATH=../..
POOL_TARBALL_NAME=open-overline-pool.tar.gz
cp ${POOL_TARBALL_PATH}/${POOL_TARBALL_NAME} ./${POOL_TARBALL_NAME}

#docker build --build-arg POOL_TARBALL_NAME=${POOL_TARBALL_NAME} -t local/open-overline-pool-api -f Dockerfile.api .
#docker build --build-arg POOL_TARBALL_NAME=${POOL_TARBALL_NAME} -t local/mining-api-reformatter -f Dockerfile.reformatter .
docker build --build-arg POOL_TARBALL_NAME=${POOL_TARBALL_NAME} -t local/open-overline-pool-frontend -f Dockerfile.frontend .
