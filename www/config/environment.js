'use strict';

module.exports = function(environment) {
  let ENV = {
    modulePrefix: 'open-overline-pool',
    environment: environment,
    rootURL: '/',
    locationType: 'hash',
    EmberENV: {
      FEATURES: {
        // Here you can enable experimental features on an ember canary build
        // e.g. 'with-controller': true
      }
    },

    APP: {
      // API host and port
      ApiUrl: 'http://157.245.116.220:6283/', // 185.209.114.83

      // HTTP mining endpoint
      HttpHost: 'http://157.245.116.220',
      HttpPort: 3142,

      // Stratum mining endpoint
      StratumHost: '157.245.116.220',
      StratumPort: 3141,

      // Fee and payout details
      PoolFee: '1%',
      PayoutThreshold: '10 Overline',

      // For network hashrate (change for your favourite fork)
      BlockTime: 1.4764e8
    }
  };

  if (environment === 'development') {
    /* Override ApiUrl just for development, while you are customizing
      frontend markup and css theme on your workstation.
    */
    //ENV.APP.ApiUrl = 'http://localhost:8081/'
     ENV.APP.LOG_RESOLVER = true;
     ENV.APP.LOG_ACTIVE_GENERATION = true;
     ENV.APP.LOG_TRANSITIONS = true;
     ENV.APP.LOG_TRANSITIONS_INTERNAL = true;
     ENV.APP.LOG_VIEW_LOOKUPS = true;
  }

  if (environment === 'test') {
    // Testem prefers this...
    ENV.locationType = 'none';

    // keep test console output quieter
    ENV.APP.LOG_ACTIVE_GENERATION = false;
    ENV.APP.LOG_VIEW_LOOKUPS = false;

    ENV.APP.rootElement = '#ember-testing';
  }

  if (environment === 'production') {

  }

  return ENV;
};
