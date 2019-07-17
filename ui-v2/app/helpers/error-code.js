import { helper } from '@ember/component/helper';

export function errorCode([error] /*, hash*/) {
  if (typeof error !== 'undefined') {
    return error.errors[0].status;
  }
}

export default helper(errorCode);
