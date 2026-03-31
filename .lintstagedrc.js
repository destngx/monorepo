const legacyPrefixes = [
  'apps/wealth-management-legacy/',
  'apps/cloudinary-photos-app/',
  'apps/portfolio-landpage/',
  'libs/wealth-management-legacy/',
  'libs/cloudinary-photos-app/',
  'libs/portfolio-landpage/',
];

const isLegacyFile = (file) => legacyPrefixes.some((prefix) => file.startsWith(prefix));
const quote = (file) => `"${file}"`;

export default {
  '*.{js,jsx,ts,tsx,mjs,svelte}': (files) => {
    const activeFiles = files.filter((file) => !isLegacyFile(file));
    if (activeFiles.length === 0) return [];
    const args = activeFiles.map(quote).join(' ');
    return [`eslint --fix ${args}`, `prettier --write ${args}`];
  },
  '*.{json,css,scss,md,html,graphql}': (files) => {
    const activeFiles = files.filter((file) => !isLegacyFile(file));
    if (activeFiles.length === 0) return [];
    const args = activeFiles.map(quote).join(' ');
    return [`prettier --write ${args}`];
  },
  '*.go': (files) => {
    const activeFiles = files.filter((file) => !isLegacyFile(file));
    if (activeFiles.length === 0) return [];
    const args = activeFiles.map(quote).join(' ');
    return [`gofmt -w ${args}`];
  },
};
