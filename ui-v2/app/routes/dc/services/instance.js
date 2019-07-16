import Route from '@ember/routing/route';

export default Route.extend({
  model: function(params) {
    return {
      slug: [params.id, params.node, params.name].join('/'),
      dc: this.modelFor('dc').dc.Name,
      item: undefined,
      proxy: undefined,
    };
  },
  setupController: function(controller, model) {
    controller.setProperties(model);
  },
});
