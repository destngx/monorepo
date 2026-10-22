const quote = (file) => `"${file}"`;

export default {
  '*.{js,jsx,ts,tsx,mjs,svelte}': (files) => {
    const args = files.map(quote).join(' ');
    return [`eslint --fix ${args}`, `prettier --write ${args}`];
  },
  '*.{json,css,scss,md,html,graphql}': (files) => {
    const args = files.map(quote).join(' ');
    return [`prettier --write ${args}`];
  },
  '*.go': (files) => {
    const args = files.map(quote).join(' ');
    return [`gofmt -w ${args}`];
  },
};
