import { helper } from '@ember/component/helper';

export function isDefined([value] /*, hash*/) {
  let actualValue = value;
  return typeof actualValue !== 'undefined';
}

export default helper(isDefined);
