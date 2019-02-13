import Component from '@ember/component';
import { inject as service } from '@ember/service';
import { get, set } from '@ember/object';
import SlotsMixin from 'ember-block-slots';
export default Component.extend(SlotsMixin, {
  dom: service('dom'),
  tagName: '',
  willRender: function() {
    this._super(...arguments);
    set(this, 'enabled', this._isRegistered('two'));
  },
  actions: {
    change: function(e) {
      const sections = [...document.querySelectorAll('section')];
      const value = e.target.value * 0.1;

      sections[0].style.width = `${value}%`;
      sections[1].style.width = `${100 - value}%`;
      get(this, 'dom')
        .components('.dom-recycling,.list-collection')
        .forEach(function(item) {
          item.resize({
            type: 'resize',
            detail: {
              width: 0,
            },
          });
        });
    },
  },
});
