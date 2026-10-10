export const site = {
  // Production origin, used for canonical URLs, Open Graph, RSS, the sitemap and robots.txt.
  url: 'https://destngx.vercel.app',
  name: 'destnguyxn',
  title: 'Dinh Nguyen - DevOps & Software Engineer',
  description:
    'Quang Dinh Nguyen Pham, a DevOps and software engineer in Ho Chi Minh City. Cloud infrastructure, tooling, side projects and notes.',
  locale: 'en',
} as const;

export const nav = [
  { label: 'Work', href: '/projects' },
  { label: 'Writing', href: '/blog' },
  { label: 'About', href: '/#story' },
  { label: 'Contact', href: '/#contact' },
] as const;
