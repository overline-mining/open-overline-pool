#!/bin/sh
set -e

# TODO: make it break if the patching yields no matches?

echo "*** Applying patch to re-enable RPC miner..."
sed -i 's/BC_RUST_MINER/BC_RPC_MINER/g' ./lib/mining/officer.js
sed -i 's/if[[:space:]]\+(false[[:space:]]\+||[[:space:]]\+BC_RPC_MINER)[[:space:]]\+{/if (BC_RPC_MINER) {/g' ./lib/mining/officer.js

#echo "*** Applying some more monkey patching..."
#perl -MFile::Slurp -0pe 'BEGIN {$r = read_file("/tmp/monkey-patch"); chomp($r)}s/if \(response\.getResult\(\) === MinerResponseResult\.CANCELED\) \{.*?\}/$r/s' -i ./lib/mining/officer.js

echo "*** Applying patch to override the RPC miner's address..."
sed -i "s/const[[:space:]]\+GRPC_MINER_URL[[:space:]]\+=.*/const GRPC_MINER_URL = 'localhost:50052'/g" ./lib/rpc/client.js

echo "*** Starting bcnode..."

echo "ARGS ->" $@

exec ./bin/cli "$@"

