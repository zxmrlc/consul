export default function(visitable, submitable) {
  return submitable({
    visit: visitable('/notices'),
  });
}
