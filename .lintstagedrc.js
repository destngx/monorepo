module.exports = {
  '*.{js,json,css,scss,md,ts,html,graphql}': ['nx format:write --files', 'nx affected:lint --fix --files'],
};
