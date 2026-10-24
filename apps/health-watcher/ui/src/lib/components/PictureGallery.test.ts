import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import type { Picture } from '$lib/api/types';
import { session } from '$lib/state/session.svelte';
import PictureGallery from './PictureGallery.svelte';

const picture = (over: Partial<Picture>): Picture => ({
  id: 'pic-1',
  pageId: 'page-1',
  position: 0,
  kind: 'xray',
  caption: '',
  box: { x0: 0.1, y0: 0.1, x1: 0.9, y1: 0.6 },
  width: 800,
  height: 600,
  createdAt: '2026-01-01T00:00:00Z',
  ...over,
});

const pageOf = (id: string | undefined) => (id === 'page-1' ? { number: 1, name: 'Page 1 · X-quang phổi' } : undefined);

describe('PictureGallery', () => {
  it('shows each picture with its kind, caption and page, opening at full size', () => {
    session.tenantId = 'tenant-1';
    render(PictureGallery, {
      pictures: [picture({}), picture({ id: 'pic-2', kind: 'endoscopy', caption: 'Tai: Màng nhĩ bình thường' })],
      pageOf,
      pageHref: (id: string) => `/ui/documents/doc-1?tab=pages&page=${id}`,
    });
    const film = screen.getByAltText('X-ray');
    expect(film.getAttribute('src')).toBe('/api/tenants/tenant-1/pictures/pic-1/image');
    expect(film.closest('a')?.getAttribute('target')).toBe('_blank');
    expect(screen.getByAltText('Endoscopy: Tai: Màng nhĩ bình thường')).toBeTruthy();
    expect(screen.getByText('Tai: Màng nhĩ bình thường')).toBeTruthy();
    const links = screen.getAllByRole('link', { name: 'Page 1 · X-quang phổi' });
    expect(links[0].getAttribute('href')).toBe('/ui/documents/doc-1?tab=pages&page=page-1');
  });
});
