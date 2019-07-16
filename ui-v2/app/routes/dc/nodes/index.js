import Route from '@ember/routing/route';
import { queryParams } from 'consul-ui/router';
import convertStatus from 'consul-ui/utils/routing/convert-status';

export default Route.extend({
  queryParams: queryParams('dc.nodes'),
  model: function(params) {
    return {
      ...{
        slug: '*',
        dc: this.modelFor('dc').dc.Name,
      },
      ...((params.s || params.status) && {
        // we check for the old style `status` variable here
        // and convert it to the new style filter=status:critical
        s: convertStatus(params.s, params.status),
      }),
    };
  },
  setupController: function(controller, model) {
    controller.setProperties(model);
  },
});
