import { z } from 'zod';

export const projectStatuses = ['active', 'shipped', 'experiment', 'archived'] as const;

export const projectMetaSchema = z.object({
  title: z.string().min(1),
  summary: z.string().min(1).max(240),
  year: z.number().int().min(2000),
  status: z.enum(projectStatuses),
  stack: z.array(z.string()).min(1),
  repo: z.url().optional(),
  live: z.url().optional(),
  featured: z.boolean().default(false),
  order: z.number().int().default(99),
  sfx: z.string().max(6).optional(),
});

export const postMetaSchema = z.object({
  title: z.string().min(1),
  description: z.string().min(1).max(240),
  date: z.coerce.date(),
  updated: z.coerce.date().optional(),
  tags: z.array(z.string()).default([]),
  draft: z.boolean().default(false),
});

export type ProjectMeta = z.infer<typeof projectMetaSchema>;
export type PostMeta = z.infer<typeof postMetaSchema>;
export type Project = ProjectMeta & { slug: string };
export type Post = PostMeta & { slug: string };
