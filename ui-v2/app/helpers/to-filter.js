import { helper } from '@ember/component/helper';
import ucfirst from 'consul-ui/utils/ucfirst';
const convert = function(str, map) {
  const replacement = map.find(function(arr) {
    const key = arr[0];
    return str.startsWith(key);
  });
  if (str.indexOf('"') === -1) {
    const replaced = str.replace(replacement[0], '');
    return replacement[1].replace('%s', replaced).replace('%S', ucfirst(replaced));
  }
  return str;
};
export function toFilter([values], attrs) {
  if (!values) {
    return null;
  }
  if (!Array.isArray(values)) {
    values = values.split('\n');
  }
  return values
    .map(function(item) {
      return convert(item, attrs.map);
    }, '')
    .join(' and ');
}

export default helper(toFilter);
