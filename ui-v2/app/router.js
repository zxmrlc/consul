import EmberRouter from '@ember/routing/router';
import config from './config/environment';
import walk from 'consul-ui/utils/routing/walk';
import wildcard from 'consul-ui/utils/routing/wildcard';

import { get } from '@ember/object';
import { assert } from '@ember/debug';

const createQueryParams = function(routes) {
  return function(id) {
    const queryParams = get(routes, `${id}._options.query`);
    assert(`Route ${id} doesn't exist`, queryParams);
    return queryParams;
  };
};

const Router = EmberRouter.extend({
  location: config.locationType,
  rootURL: config.rootURL,
});
export const routes = {
  // Our parent datacenter resource sets the namespace
  // for the entire application
  dc: {
    _options: { path: ':dc' },
    // Services represent a consul service
    services: {
      _options: {
        path: '/services',
        query: {
          s: {
            as: 'filter',
            replace: true,
          },
          // temporary support of old style status
          status: {
            as: 'status',
          },
        },
      },
      // Show an individual service
      show: {
        _options: {
          path: '/:name',
          query: {
            s: {
              as: 'filter',
              replace: true,
            },
          },
        },
      },
      instance: {
        _options: { path: '/:name/:node/:id' },
      },
    },
    // Nodes represent a consul node
    nodes: {
      _options: {
        path: '/nodes',
        query: {
          s: {
            as: 'filter',
            replace: true,
          },
          // temporary support of old style status
          status: {
            as: 'status',
          },
        },
      },
      // Show an individual node
      show: {
        _options: {
          path: '/:name',
          query: {
            s: {
              as: 'filter',
              replace: true,
            },
          },
        },
      },
    },
    // Intentions represent a consul intention
    intentions: {
      _options: { path: '/intentions' },
      edit: {
        _options: { path: '/:id' },
      },
      create: {
        _options: { path: '/create' },
      },
    },
    // Key/Value
    kv: {
      _options: { path: '/kv' },
      folder: {
        _options: { path: '/*key' },
      },
      edit: {
        _options: { path: '/*key/edit' },
      },
      create: {
        _options: { path: '/*key/create' },
      },
      'root-create': {
        _options: { path: '/create' },
      },
    },
    // ACLs
    acls: {
      _options: { path: '/acls' },
      edit: {
        _options: { path: '/:id' },
      },
      create: {
        _options: { path: '/create' },
      },
      policies: {
        _options: { path: '/policies' },
        edit: {
          _options: { path: '/:id' },
        },
        create: {
          _options: { path: '/create' },
        },
      },
      roles: {
        _options: { path: '/roles' },
        edit: {
          _options: { path: '/:id' },
        },
        create: {
          _options: { path: '/create' },
        },
      },
      tokens: {
        _options: { path: '/tokens' },
        edit: {
          _options: { path: '/:id' },
        },
        create: {
          _options: { path: '/create' },
        },
      },
    },
  },
  // Shows a datacenter picker. If you only have one
  // it just redirects you through.
  index: {
    _options: { path: '/' },
  },
  // The settings page is global.
  settings: {
    _options: { path: '/setting' },
  },
  notfound: {
    _options: { path: '/*path' },
  },
};
export const queryParams = createQueryParams(routes);
export const isWildcard = wildcard(routes);
export default Router.map(walk(routes));
