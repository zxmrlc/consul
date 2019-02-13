import { inject as service } from '@ember/service';
import { get } from '@ember/object';
import { schedule } from '@ember/runloop';
import Component from '@ember/component';
export default Component.extend({
  buffer: service('dom-buffer'),
  name: 'modal',
  getBufferName: function() {
    // TODO: Right now we are only using this for the modal layer
    // moving forwards you'll be able to name your buffers
    return get(this, 'name');
  },
  didInsertElement: function() {
    this._super(...arguments);
    schedule('afterRender', () => {
      get(this, 'buffer').add(this.getBufferName(), this.element);
    });
  },
  didDestroyElement: function() {
    this._super(...arguments);
    get(this, 'buffer').remove(this.getBufferName());
  },
});
