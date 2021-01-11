#!/bin/bash

curl -d '{"jsonrpc":"2.0","id":0,"method":"getSyncStatus","params":[]}' \
     -H 'Content-Type: application/json' \
     -u ':correct-horse-battery-staple' http://overline:3000/rpc --verbose

echo
