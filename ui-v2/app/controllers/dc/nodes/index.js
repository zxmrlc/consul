import Controller from '@ember/controller';
import { queryParams } from 'consul-ui/router';

export default Controller.extend({
  queryParams: queryParams('dc.nodes'),
});
