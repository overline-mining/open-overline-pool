#!/bin/bash

PODNAME=$1
DBFILE=$2

kubectl cp ${DBFILE} ${PODNAME}:/_easysync_db.tar.gz -c get-bcnode-db-container
touch .uploaded
kubectl cp .uploaded ${PODNAME}:/ -c get-bcnode-db-container
