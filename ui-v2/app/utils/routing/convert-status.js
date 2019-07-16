export default function(search = '', status = '') {
  if (status) {
    status = `status:${status}`;
    if (search.indexOf(status) === -1) {
      return search
        .split('\n')
        .concat(status)
        .join('\n')
        .trim();
    }
  }

  return search === '' ? undefined : search;
}
