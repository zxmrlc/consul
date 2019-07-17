import { helper } from '@ember/component/helper';

export function errorMessage([error] /*, hash*/) {
  const temp = error.message.split('\n');
  temp.splice(0, 2);
  return temp.join('\n');
}

export default helper(errorMessage);
