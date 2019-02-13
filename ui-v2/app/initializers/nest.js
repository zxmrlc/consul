import ServicesRoute from 'consul-ui/routes/dc/services/index';
import ServicesController from 'consul-ui/controllers/dc/services/index';
import NodesRoute from 'consul-ui/routes/dc/nodes/index';
import NodesController from 'consul-ui/controllers/dc/nodes/index';

import CSSElementQueries from 'npm:css-element-queries';

export function initialize(application) {
  CSSElementQueries.ElementQueries.init();
  const obj = {
    'dc/services': {
      route: ServicesRoute,
      controller: ServicesController,
    },
    'dc/nodes': {
      route: NodesRoute,
      controller: NodesController,
    },
  };
  Object.keys(obj).forEach(function(key) {
    ['route', 'controller'].forEach(function(type) {
      if (obj[key][type]) {
        const cls = obj[key][type];
        if (type === 'route') {
          cls.reopen({
            templateName: `${key}/index`,
          });
        }
        application.register(`${type}:${key}`, cls.extend());
        application.unregister(`${type}:${key}/index`);
        // application.reset(`${type}:${key}/index`);
        // application.register(`${type}:${key}/index`, (type === 'route' ? Route : Controller).extend({}));
      }
    });
  });
}

export default {
  initialize,
};
