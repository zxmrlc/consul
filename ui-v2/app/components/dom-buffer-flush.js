import { inject as service } from '@ember/service';
import { get } from '@ember/object';
import Component from '@ember/component';
const append = function(e) {
  if (get(this, 'name') === e.detail.name) {
    this.element.appendChild(e.detail.content);
    [...get(this, 'dom').components('.app-view', this.element)].forEach(function(item) {
      item.didInsertElement();
    });
  }
};
export default Component.extend({
  buffer: service('dom-buffer'),
  dom: service('dom'),
  name: 'two-pane',
  init: function() {
    this._super(...arguments);
    this._append = append.bind(this);
  },
  didInsertElement: function() {
    get(this, 'buffer').on('add', this._append);
  },
  didDestroyElement: function() {
    get(this, 'buffer').off('add', this._append);
  },
});
