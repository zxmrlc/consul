import Component from '@ember/component';
import { inject as service } from '@ember/service';
import { get, set } from '@ember/object';

import WithListeners from 'consul-ui/mixins/with-listeners';

const replace = function(
  obj,
  prop,
  value,
  destroy = (prev = null, value) => (typeof prev === 'function' ? prev() : null)
) {
  const prev = get(obj, prop);
  if (prev !== value) {
    destroy(prev, value);
  }
  return set(obj, prop, value);
};

export default Component.extend(WithListeners, {
  tagName: 'span',

  data: service('blocking'),

  onchange: function() {},
  onerror: function() {},

  didInsertElement: function() {
    this._super(...arguments);
    const options = {
      rootMargin: '0px',
      threshold: 1.0,
    };

    const observer = new IntersectionObserver((entries, observer) => {
      entries.map(item => {
        set(this, 'isIntersecting', item.isIntersecting);
        if (!item.isIntersecting) {
          this.actions.close.bind(this)();
        } else {
          this.actions.open.bind(this)();
        }
      });
    }, options);
    observer.observe(this.element);
    this.listen(() => {
      this.actions.close.bind(this)();
      observer.disconnect();
    });
  },
  didReceiveAttrs: function() {
    this._super(...arguments);
    if (this.element && get(this, 'isIntersecting')) {
      this.actions.open.bind(this)();
    }
  },
  actions: {
    open: function() {
      // keep this argumentless
      const src = get(this, 'src');
      const filter = get(this, 'filter');

      const source = replace(
        this,
        'source',
        get(this, 'data').open(`${src}${filter ? `?filter=${filter}` : ``}`, this),
        (prev, source) => {
          // Makes sure any previous source (if different) is ALWAYS closed
          get(this, 'data').close(prev, this);
        }
      );
      const remove = this.listen(source, {
        message: e => this.onchange(e),
        error: e => this.onerror(e),
      });
      replace(this, '_remove', remove);
      const currentEvent = source.getCurrentEvent();
      if (currentEvent) {
        this.onchange(currentEvent);
      }
    },
    close: function() {
      // keep this argumentless
      get(this, 'data').close(get(this, 'source'), this);
      replace(this, '_remove', null);
      set(this, 'source', undefined);
    },
  },
});
