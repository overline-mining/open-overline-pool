import EmberObject from '@ember/object';
import { computed } from '@ember/object';

var Block = EmberObject.extend({
  variance: computed('difficulty', 'shares', function() {
    let percent = this.get('shares') / this.get('difficulty');
    if (!percent) {
      return 0;
    }
    return percent;
  }),

  isLucky: computed('variance', function() {
    return this.get('variance') <= 1.0;
  }),

  isOk: computed('orphan', 'uncle', function() {
    console.log('orphan from object: ' + this.get('orphan')); // eslint-disable-line no-console
    return !this.get('orphan');
  }),

  formatReward: computed('reward', function() {
    if (!this.get('orphan')) {
      let value = parseInt(this.get('reward')) * 0.000000000000000001;
      console.log("reward from backend : " + this.get('reward')); // eslint-disable-line no-console
      console.log(`parsed value: ${value}`); // eslint-disable-line no-console
      return value.toFixed(6);
    } else {
      return 0;
    }
  })
});

export default Block;
