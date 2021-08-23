## Open Source Overline Mining Pool

![Miner's stats page](https://user-images.githubusercontent.com/1068089/106371835-e4035080-632e-11eb-9fdc-73c3bb420b5b.png)

### Features

* Support for HTTP and Stratum mining
* Detailed block stats with luck percentage and full reward
* Parity nodes rpc failover built in
* Modern beautiful Ember.js frontend
* Separate stats for workers: can highlight timed-out workers so miners can perform maintenance of rigs
* JSON-API for stats
* kubenetes based deployment for all elements of mining pool for maximum reliability

#### Proxies (not yet available)

* [Overline-Proxy](https://github.com/sammy007/ether-proxy) HTTP proxy with web interface
* [Stratum Proxy](https://github.com/Atrides/eth-proxy) for Overline

### Bringing a pool up in a linux environment

Dependencies:

  * go >= 1.13
  * bcnode (does it even have versions at this point, it's more a stream of consciousness)
  * redis-server >= 2.8.0
  * nodejs ~ 10 LTS
  * nginx
  * kubernetes >= 1.20 (use minikube for local / non-production builds)

**I highly recommend to use Ubuntu 20.04 LTS.**

This whole installation is containerized and comes with a full kubernetes based setup.
Below we will walk through instructions for running the pool on minikube in a testing environment.


1. Clone this repository `git clone https://github.com/overline-mining/open-overline-pool.git`

2. Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

3. Install [minikube](https://minikube.sigs.k8s.io/docs/start/#binary-download)

4. Update bcnode image:

    ```bash
    cd open-overline-pool/docker
    ./build-images.sh
    ```

5.1 Setup secrets:

    ```bash
    cd open-overline-pool/k8s
    # NOTE -->  edit the file "config" to contain whatever addresses, http auth passwords, and keys you wish to use
    ./make-secrets.sh
    # NOTE --> you will need to re-run this after you changed something, or part of the pool wont work
    ```

5.2 Set the bcnode to bootstrap mode:
    
    ```bash
    nano open-overline-pool/bcnode/deployment.yml
    # NOTE -->  edit the file way at the bottom to this:
    
             name: vol-bcnode-db
        command: ['sh','-c']
        args:
        - rm -r /data/db;     # !!!!!!!To start the bootstrap mode, it will fail if this isnt included
          if [ ! -d "/data/db" ]; then
            echo "nameserver 8.8.8.8" >> /etc/resolv.conf;
            echo "nameserver 8.8.4.4" >> /etc/resolv.conf;
            apt-get update && apt-get install -y wget unzip;
            until [ -f .uploaded ]; do sleep 1; ls -lh _easysync_db.tar.gz; done;
            tar -xvzf _easysync_db.tar.gz -C /data --strip-components=1;   #!!!! set --strip-components to 1        
            rm /data/db/IDENTITY;
            rm /data/.chainstate.db;
            rm _easysync_db.tar.gz;
            rm .uploaded;
          fi;
          echo "done!";
    
    #Exit and save the file once edited
    
    ```

6. Initialize bcnode (you will need a chainstate snapshot saved as a `.tar.gz` file):

    ```bash
    cd open-overline-pool/k8s
    kubectl apply -f bcnode/
    ./upload-db.sh $(kubectl get pods | grep bcnode | awk '{print $1}') /path/to/chainstate.tar.gz
    # wait for a bit for the node's initialization container to unpack the chainstate
    # follow progress with
    kubectl logs $(kubectl get pods | grep bcnode | awk '{print $1}') -c get-bcnode-db-container -f --tail 10
    # once that is done follow the bcnode logs and wait for it to sync
    kubectl logs $(kubectl get pods | grep bcnode | awk '{print $1}') -c bcnode -f --tail 10
    ```

7. While the bcnode is syncing, setup redis.

    ```bash
    kubectl apply -f redis/
    ```

8. Once the node has fully synced, change the node deployment.yml back from Bootstrap mode, so it wont clear itself on every start

    ```bash
    
    nano /bcnode/deployment.yml
    # NOTE -->  edit the file way at the bottom to this:
    
             name: vol-bcnode-db
        command: ['sh','-c']
        args:
        - rm -r /data/db/IDENTITY;     # !!!!!!!To prevent the bootstrap, so the node wont wipe itself on startup
          if [ ! -d "/data/db" ]; then
            echo "nameserver 8.8.8.8" >> /etc/resolv.conf;
            echo "nameserver 8.8.4.4" >> /etc/resolv.conf;
            apt-get update && apt-get install -y wget unzip;
            until [ -f .uploaded ]; do sleep 1; ls -lh _easysync_db.tar.gz; done;
            tar -xvzf _easysync_db.tar.gz -C /data --strip-components=1;       
            rm /data/db/IDENTITY;
            rm /data/.chainstate.db;
            rm _easysync_db.tar.gz;
            rm .uploaded;
          fi;
          echo "done!";

    # Exit and save the file after you changed it.

    # Then we need to kill the node to apply the settings.
    kubectl delete -f bcnode/deployment.yml
    # Now lets apply the settings
    kubectl apply -f bcnode/deployment.yml

    ```

9. Once the bcnode is back up and synced again we bring open-overline-pool online as follows:

    ```bash
    kubectl apply -f open-overline-pool/
    ./local-port-forward.sh
    ```

10. You should now be able to point a browser to `localhost` and see the splash page. You can also test that the pool is accepting jobs by pointing a overline-compatible stratum miner at it.

#### Customization

You can customize the layout using built-in web server with live reload:

    ember server --port 8082 --environment development

**Don't use built-in web server in production**.

Check out <code>www/app/templates</code> directory and edit these templates
in order to customise the frontend.

### Configuration

Configuration is actually simple, just read it twice and think twice before changing defaults.

**Don't copy config directly from this manual. Use the example config from the package,
otherwise you will get errors on start because of JSON comments.**

```javascript
{
  // Set to the number of CPU cores of your server
  "threads": 2,
  // Prefix for keys in redis store
  "coin": "eth",
  // Give unique name to each instance
  "name": "main",

  "proxy": {
    "enabled": true,

    // Bind HTTP mining endpoint to this IP:PORT
    "listen": "0.0.0.0:8888",

    // Allow only this header and body size of HTTP request from miners
    "limitHeadersSize": 1024,
    "limitBodySize": 256,

    /* Set to true if you are behind CloudFlare (not recommended) or behind http-reverse
      proxy to enable IP detection from X-Forwarded-For header.
      Advanced users only. It's tricky to make it right and secure.
    */
    "behindReverseProxy": false,

    // Stratum mining endpoint
    "stratum": {
      "enabled": true,
      // Bind stratum mining socket to this IP:PORT
      "listen": "0.0.0.0:8008",
      "timeout": "120s",
      "maxConn": 8192,
      "tls": false,
      "certFile": "/path/to/cert.pem",
      "keyFile": "/path/to/key.pem"
    },

    // Try to get new job from node in this interval
    "blockRefreshInterval": "120ms",
    "stateUpdateInterval": "3s",
    // Require this share difficulty from miners
    "difficulty": 2000000000,

    /* Reply error to miner instead of job if redis is unavailable.
      Should save electricity to miners if pool is sick and they didn't set up failovers.
    */
    "healthCheck": true,
    // Mark pool sick after this number of redis failures.
    "maxFails": 100,
    // TTL for workers stats, usually should be equal to large hashrate window from API section
    "hashrateExpiration": "3h",

    "policy": {
      "workers": 8,
      "resetInterval": "60m",
      "refreshInterval": "1m",

      "banning": {
        "enabled": false,
        /* Name of ipset for banning.
        Check http://ipset.netfilter.org/ documentation.
        */
        "ipset": "blacklist",
        // Remove ban after this amount of time
        "timeout": 1800,
        // Percent of invalid shares from all shares to ban miner
        "invalidPercent": 30,
        // Check after after miner submitted this number of shares
        "checkThreshold": 30,
        // Bad miner after this number of malformed requests
        "malformedLimit": 5
      },
      // Connection rate limit
      "limits": {
        "enabled": false,
        // Number of initial connections
        "limit": 30,
        "grace": "5m",
        // Increase allowed number of connections on each valid share
        "limitJump": 10
      }
    }
  },

  // Provides JSON data for frontend which is static website
  "api": {
    "enabled": true,
    "listen": "0.0.0.0:8080",
    // Collect miners stats (hashrate, ...) in this interval
    "statsCollectInterval": "5s",
    // Purge stale stats interval
    "purgeInterval": "10m",
    // Fast hashrate estimation window for each miner from it's shares
    "hashrateWindow": "30m",
    // Long and precise hashrate from shares, 3h is cool, keep it
    "hashrateLargeWindow": "3h",
    // Collect stats for shares/diff ratio for this number of blocks
    "luckWindow": [64, 128, 256],
    // Max number of payments to display in frontend
    "payments": 50,
    // Max numbers of blocks to display in frontend
    "blocks": 50,

    /* If you are running API node on a different server where this module
      is reading data from redis writeable slave, you must run an api instance with this option enabled in order to purge hashrate stats from main redis node.
      Only redis writeable slave will work properly if you are distributing using redis slaves.
      Very advanced. Usually all modules should share same redis instance.
    */
    "purgeOnly": false
  },

  // Check health of each node in this interval
  "upstreamCheckInterval": "5s",

  /* List of parity nodes to poll for new jobs. Pool will try to get work from
    first alive one and check in background for failed to back up.
    Current block template of the pool is always cached in RAM indeed.
  */
  "upstream": [
    {
      "name": "main",
      "url": "http://127.0.0.1:8545",
      "timeout": "10s"
    },
    {
      "name": "backup",
      "url": "http://127.0.0.2:8545",
      "timeout": "10s"
    }
  ],

  // This is standard redis connection options
  "redis": {
    // Where your redis instance is listening for commands
    "leadEndpoint": "redis-leader:6379",
    "followEndpoint": "redis-follower:6379",
    "poolSize": 10,
    "database": 0,
    "password": ""
  },

  // This module periodically remits ether to miners
  "unlocker": {
    "enabled": false,
    // Pool fee percentage
    "poolFee": 1.0,
    // Pool fees beneficiary address (leave it blank to disable fee withdrawals)
    "poolFeeAddress": "",
    // Donate 10% from pool fees to developers
    "donate": true,
    // Unlock only if this number of blocks mined back
    "depth": 120,
    // Simply don't touch this option
    "immatureDepth": 20,
    // Keep mined transaction fees as pool fees
    "keepTxFees": false,
    // Run unlocker in this interval
    "interval": "10m",
    // Parity node rpc endpoint for unlocking blocks
    "daemon": "http://127.0.0.1:8545",
    // Rise error if can't reach parity
    "timeout": "10s"
  },

  // Pay out miners using this module
  "payouts": {
    "enabled": false,
    // Require minimum number of peers on node
    "requirePeers": 25,
    // Run payouts in this interval
    "interval": "12h",
    // Parity node rpc endpoint for payouts processing
    "daemon": "http://127.0.0.1:8545",
    // Rise error if can't reach parity
    "timeout": "10s",
    // Address with pool balance
    "address": "0x0",
    // Let parity to determine gas and gasPrice
    "autoGas": true,
    // Gas amount and price for payout tx (advanced users only)
    "gas": "21000",
    "gasPrice": "50000000000",
    // Send payment only if miner's balance is >= 0.5 Ether
    "threshold": 500000000,
    // Perform BGSAVE on Redis after successful payouts session
    "bgsave": false
  }
}
```

If you are distributing your pool deployment to several servers or processes,
create several configs and disable unneeded modules on each server. (Advanced users)

I recommend this deployment strategy:

* Mining instance - 1x (it depends, you can run one node for EU, one for US, one for Asia)
* Unlocker and payouts instance - 1x each (strict!)
* API instance - 1x

### Notes

* Unlocking and payouts are sequential, 1st tx go, 2nd waiting for 1st to confirm and so on. You can disable that in code. Carefully read `docs/PAYOUTS.md`.
* Also, keep in mind that **unlocking and payouts will halt in case of backend or node RPC errors**. In that case check everything and restart.
* You must restart module if you see errors with the word *suspended*.
* Don't run payouts and unlocker modules as part of mining node. Create separate configs for both, launch independently and make sure you have a single instance of each module running.
* If `poolFeeAddress` is not specified all pool profit will remain on coinbase address. If it specified, make sure to periodically send some dust back required for payments.

### Credits

Originally made by sammy007, modifications for overline and kubernetes by lgray. Licensed under GPLv3.

#### Contributors

[Alex Leverington](https://github.com/subtly)

### Donations are highly appreciated!

ETH/OL: `0xf34fa87db39d15471bebe997860dcd49fc259318`

BTC: `13xBjyBFeqiW1eipFGiS1YQvw9HMuAx3bp`

NEO: `AJodL5DbcASbYPNyVBvYNiZnUpyRtqNpJU`

WAV: `3P6Vaod2dVdk9542QhVAroXimR1m6ThXLjh`

LSK: `4823425666801418479L`

Original author ETH: `0xb85150eb365e7df0941f0cf08235f987ba91506a`
