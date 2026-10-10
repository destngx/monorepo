import { describe, expect, it } from 'vitest';
import { soundcloudEmbedUrl } from './soundcloud';

const playlist = 'https://api.soundcloud.com/playlists/1000012141';

describe('soundcloudEmbedUrl', () => {
  it('points the widget player at the playlist', () => {
    const url = new URL(soundcloudEmbedUrl(playlist, { color: '#e2306c' }));

    expect(url.origin + url.pathname).toBe('https://w.soundcloud.com/player/');
    expect(url.searchParams.get('url')).toBe(playlist);
  });

  it('tints the controls with the accent colour', () => {
    const url = new URL(soundcloudEmbedUrl(playlist, { color: '#e2306c' }));

    expect(url.searchParams.get('color')).toBe('#e2306c');
  });

  it('starts playing on open and keeps the visual player without comments or reposts', () => {
    const url = new URL(soundcloudEmbedUrl(playlist, { color: '#e2306c' }));

    expect(Object.fromEntries(url.searchParams)).toMatchObject({
      auto_play: 'true',
      visual: 'true',
      show_comments: 'false',
      show_reposts: 'false',
    });
  });
});
