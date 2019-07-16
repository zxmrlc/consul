import Route from '@ember/routing/route';
import { inject as service } from '@ember/service';
import { hash } from 'rsvp';
import { get } from '@ember/object';
import { routes } from 'consul-ui/router';

export default Route.extend({
  settings: service('settings'),
  queryParams: routes.dc.services.show._options.query,
  model: function(params) {
    const settings = get(this, 'settings');
    const dc = this.modelFor('dc').dc.Name;
    return hash({
      slug: params.name,
      urls: settings.findBySlug('urls'),
      dc: dc,
      item: undefined,
    });
  },
  setupController: function(controller, model) {
    controller.setProperties(model);
  },
});
