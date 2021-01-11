#!/bin/bash

curl -d '{"jsonrpc":"2.0","id":0,"method":"stats","params":[]}' \
     -H 'Content-Type: application/json' \
     -u ':correct-horse-battery-staple' http://overline:3000/rpc

echo
