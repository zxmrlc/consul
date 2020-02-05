import Route from '@ember/routing/route';
import { inject as service } from '@ember/service';
import { hash } from 'rsvp';

export default Route.extend({
  repo: service('settings'),
  dcRepo: service('repository/dc'),
  nspacesRepo: service('repository/nspace/disabled'),
  model: function(params) {
    return hash({
      dcs: this.dcRepo.findAll(),
      nspaces: this.nspacesRepo.findAll(),
      nspace: this.nspacesRepo.getActive(),
    }).then(model => {
      return hash({
        ...model,
        ...{
          dc: this.dcRepo.getActive(null, model.dcs),
        },
      });
    });
  },
  setupController: function(controller, model) {
    controller.setProperties(model);
  },
});
