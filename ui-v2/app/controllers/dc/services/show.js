import Controller from '@ember/controller';
import { get, set } from '@ember/object';
import { inject as service } from '@ember/service';
import { queryParams } from 'consul-ui/router';
export default Controller.extend({
  dom: service('dom'),
  queryParams: queryParams('dc.services.show'),
  init: function() {
    this.searchParams = {
      serviceInstance: 's',
    };
    this._super(...arguments);
  },
  setProperties: function() {
    this._super(...arguments);
    // This method is called immediately after `Route::setupController`, and done here rather than there
    // as this is a variable used purely for view level things, if the view was different we might not
    // need this variable
    set(this, 'selectedTab', 'instances');
  },
  actions: {
    change: function(e) {
      set(this, 'selectedTab', e.target.value);
      // Ensure tabular-collections sizing is recalculated
      // now it is visible in the DOM
      get(this, 'dom')
        .components('.tab-section input[type="radio"]:checked + div table')
        .forEach(function(item) {
          if (typeof item.didAppear === 'function') {
            item.didAppear();
          }
        });
    },
  },
});
