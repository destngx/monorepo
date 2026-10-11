const WIDGET_URL = 'https://w.soundcloud.com/player/';

type EmbedOptions = {
  /** Hex colour for the play button and waveform. */
  color: string;
};

/** Builds the SoundCloud widget iframe URL for a track or playlist API URL. */
export const soundcloudEmbedUrl = (resourceUrl: string, { color }: EmbedOptions): string => {
  const url = new URL(WIDGET_URL);
  url.search = new URLSearchParams({
    url: resourceUrl,
    color,
    auto_play: 'true',
    visual: 'true',
    show_user: 'true',
    show_teaser: 'true',
    hide_related: 'false',
    show_comments: 'false',
    show_reposts: 'false',
  }).toString();
  return url.href;
};
