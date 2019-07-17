import Helper from '@ember/component/helper';
import { inject as service } from '@ember/service';
import { get } from '@ember/object';

export default Helper.extend({
  container: service('search'),
  compute([name, items, search]) {
    return get(this, 'container')
      .searchable(name)
      .add(items)
      .search(search);
  },
});
