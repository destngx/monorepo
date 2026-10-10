export type Social = { label: string; href: string; handle: string };

export type Role = {
  title: string;
  /** Left empty on purpose until the owner confirms which employers can be named publicly. */
  company?: string;
  period: string;
  summary: string;
  highlights: string[];
  stack: string[];
};

export type Milestone = { year: string; text: string };
export type Hobby = { name: string; note: string; sfx: string };
export type SkillGroup = { title: string; skills: string[] };

export const profile = {
  name: 'Quang Dinh Nguyen Pham',
  shortName: 'Dinh',
  handle: 'destnguyxn',
  role: 'DevOps & Software Engineer',
  location: 'Ho Chi Minh City, Vietnam',
  email: 'nguyenphamquangdinh99@gmail.com',
  greeting: "Yo! I'm Dinh.",
  tagline: 'I keep clusters calm, pipelines fast, and I build odd little tools on the side.',
  intro:
    'I run cloud infrastructure and CI/CD for a living: AWS, Kubernetes, GitOps, and the monitoring that tells me ' +
    'when something is about to catch fire. Outside of work I tinker in a monorepo full of side projects, from a ' +
    'Go AI gateway to a macOS posture coach.',
} as const;

export const socials: Social[] = [
  { label: 'GitHub', href: 'https://github.com/destngx', handle: '@destngx' },
  { label: 'LinkedIn', href: 'https://www.linkedin.com/in/erudi-lumos/', handle: 'erudi-lumos' },
  { label: 'Email', href: `mailto:${profile.email}`, handle: profile.email },
];

export const roles: Role[] = [
  {
    title: 'DevOps Engineer',
    period: '2024 - now',
    summary: 'Own CI/CD and cloud infrastructure for a portfolio of product teams.',
    highlights: [
      'Run GitLab CI/CD pipelines for 10+ projects.',
      'Cut deployment costs by about 50% while keeping 99.9% uptime.',
      'Operate several Kubernetes clusters with GitOps.',
      'Built observability on Grafana, Prometheus and Loki with Slack alerting.',
    ],
    stack: ['AWS', 'Kubernetes', 'GitLab CI', 'Terraform', 'GitOps', 'Grafana'],
  },
  {
    title: 'Team Leader',
    period: '2023',
    summary: 'Led five projects from kickoff to production across ten teams.',
    highlights: [
      'Scoped requirements, split work and managed delivery risk.',
      'Shipped a serverless document platform with OpenSearch.',
      'Built feedback analysis with text tokenization to triage issues.',
    ],
    stack: ['AWS Lambda', 'OpenSearch', 'NestJS'],
  },
  {
    title: 'Back-End Developer',
    period: '2022 - 2023',
    summary: 'Built internal systems with clean architecture and microservices.',
    highlights: [
      'Designed an attendance system used by 400 employees.',
      'Built a harmful-content detector with Django and YOLOv7.',
      'Halved build and deploy time with Docker optimisation and zero-downtime deploys.',
    ],
    stack: ['NestJS', 'Django', 'PostgreSQL', 'Docker'],
  },
];

export const milestones: Milestone[] = [
  { year: '1999', text: 'Born in Lam Dong, in the Vietnamese highlands.' },
  { year: '2022', text: 'B.Sc. in Software Engineering, University of Science, VNU-HCM.' },
  { year: '2022', text: 'First job as a back-end developer.' },
  { year: '2024', text: 'Moved into DevOps full time.' },
  { year: 'now', text: 'Building side projects in public in a single monorepo.' },
];

export const skillGroups: SkillGroup[] = [
  { title: 'Cloud & infra', skills: ['AWS', 'Terraform', 'Kubernetes', 'Docker', 'GitOps', 'NGINX'] },
  { title: 'Delivery', skills: ['GitLab CI', 'GitHub Actions', 'Nx', 'Zero-downtime deploys'] },
  { title: 'Observability', skills: ['Prometheus', 'Grafana', 'Loki', 'Alerting'] },
  { title: 'Languages', skills: ['TypeScript', 'Go', 'Python', 'Swift', 'Shell'] },
  { title: 'Back end', skills: ['NestJS', 'FastAPI', 'Fiber', 'PostgreSQL', 'Redis'] },
  { title: 'Front end', skills: ['Svelte', 'React', 'Tailwind CSS'] },
];

export const hobbies: Hobby[] = [
  { name: 'Bass guitar', note: 'Creating grooves and rhythms.', sfx: 'ブン' },
  { name: 'Cycling', note: '200 km and 300 km pendants achieved.', sfx: 'シャー' },
  { name: 'Running', note: 'Pushing limits one step at a time.', sfx: 'タッ' },
  { name: 'Manga', note: 'Diving into visual storytelling. Also where this site got its look.', sfx: 'ドン' },
  { name: 'Legend of Zelda', note: "Exploring Hyrule's adventures.", sfx: 'ヤッ' },
  { name: 'Gamification', note: 'Turning challenges into games.', sfx: 'ピコ' },
];

export const photographyUrl = 'https://destnguyxn-photos.vercel.app/';
