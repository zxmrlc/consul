import Component from '@ember/component';
import { inject as service } from '@ember/service';
import { get, set } from '@ember/object';

import Slotted from 'block-slots';
import { toFilter } from 'consul-ui/helpers/to-filter';

// these should remain as strings so that searchType is a string
// so comparison helpers work properly in the template
const SEARCH_TYPE_SIMPLE = '0';
const SEARCH_TYPE_ADVANCED = '1';

export default Component.extend(Slotted, {
  // const protected/template
  SEARCH_TYPE_SIMPLE,
  SEARCH_TYPE_ADVANCED,

  //protected/component settings
  classNames: ['search-bar'],

  // services
  dom: service('dom'),

  // protected/template
  searchType: SEARCH_TYPE_SIMPLE,

  // public attributes
  simplemap: null,
  value: '',

  // protected/avoid in template
  didInsertElement: function() {
    this._super(...arguments);
    // use {{ref}}
    set(this, 'editor', get(this, 'dom').component('.phrase-editor', this.element));
    set(this, 'input', get(this, 'dom').elements('input[type=search]', this.element)[1]);
  },

  // protected/template
  actions: {
    change: function(e) {
      let value, data;
      const buttonTrigger = typeof e === 'undefined';
      if (get(this, 'searchType') === SEARCH_TYPE_SIMPLE) {
        if (buttonTrigger) {
          value = get(this, 'editor.data');
        } else {
          value = get(e, 'target.data'); //editor
        }
        data = toFilter([value], { map: get(this, 'filtermap') });
        set(this, 'input.value', data);
      } else {
        if (buttonTrigger) {
          value = get(this, 'input.value');
        } else {
          value = get(e, 'target.value');
        }
        data = value;
        set(this, 'value', data);
      }
      this.onchange({ target: { data: data, value: value } });
    },
  },
});
