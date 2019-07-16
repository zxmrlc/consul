import Component from '@ember/component';
import { get, set, computed } from '@ember/object';
import { inject as service } from '@ember/service';
import { next } from '@ember/runloop';

const diff = function(prev, items) {
  return items
    .filter(item => !prev.includes(item))
    .concat(prev.filter(item => items.includes(item)))
    .filter(item => item !== '');
};
export default Component.extend({
  dom: service('dom'),
  classNames: ['phrase-editor'],
  item: '',
  values: null,
  value: '',
  data: computed('values.[]', function() {
    const data = get(this, 'values').join('\n');
    return data.length === 0 ? null : data;
  }),
  didReceiveAttrs: function() {
    this._super(...arguments);
    const value = get(this, 'value') || '';
    const prev = get(this, 'values') || [];
    set(this, 'values', []);
    const items = value
      .toString()
      .split(' and ')
      .join('\n')
      .split('\n');
    if (diff(prev, items).length > 0) {
      set(this, 'values', items);
      next(() => {
        this.search({ target: this });
      });
    }
  },
  didInsertElement: function() {
    this._super(...arguments);
    // TODO: use {{ref}}
    this.input = get(this, 'dom').element('input', this.element);
    this.didReceiveAttrs();
  },
  onchange: function(e) {},
  search: function(e) {
    // TODO: Temporarily continue supporting `searchable`
    let searchable = get(this, 'searchable');
    if (searchable) {
      if (!Array.isArray(searchable)) {
        searchable = [searchable];
      }
      searchable.forEach(item => {
        item.search(get(this, 'values'));
      });
    }
    if (get(this, 'data.length') === 0) {
      set(this, 'value', null);
    }
    this.onchange(e);
  },
  oninput: function(e) {},
  onkeydown: function(e) {},
  actions: {
    keydown: function(e) {
      switch (e.keyCode) {
        case 8: // backspace
          if (e.target.value == '' && get(this, 'values').length > 0) {
            this.actions.remove.bind(this)(get(this, 'values').length - 1);
          }
          break;
        case 27: // escape
          set(this, 'values', []);
          this.search({ target: this });
          break;
      }
      this.onkeydown({ target: this });
    },
    input: function(e) {
      set(this, 'item', e.target.value);
      this.oninput({ target: this });
    },
    remove: function(index, e) {
      get(this, 'values').removeAt(index, 1);
      this.search({ target: this });
      this.input.focus();
    },
    add: function(e) {
      const item = get(this, 'item').trim();
      if (item !== '') {
        set(this, 'item', '');
        const currentItems = get(this, 'values') || [];
        const items = new Set(currentItems).add(item);
        if (items.size > currentItems.length) {
          set(this, 'values', [...items].filter(item => item !== ''));
          this.search({ target: this });
        }
      }
    },
  },
});
