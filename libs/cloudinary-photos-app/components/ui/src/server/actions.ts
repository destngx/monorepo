'use server';

import cloudinary from 'cloudinary';
import { SearchResult } from '@nx-pnpm-monorepo/cloudinary-photos-app/types';

export async function addImageToAlbum(image: SearchResult, album: string) {
  await cloudinary.v2.api.create_folder(album);

  let parts = image.public_id.split('/');
  if (parts.length > 1) {
    parts = parts.slice(1);
  }
  const publicId = parts.join('/');

  await cloudinary.v2.uploader.rename(image.public_id, `${album}/${publicId}`);
}

export async function setAsFavoriteAction(publicId: string, isFavorite: boolean) {
  if (isFavorite) {
    await cloudinary.v2.uploader.add_tag('favorite', [publicId]);
  } else {
    await cloudinary.v2.uploader.remove_tag('favorite', [publicId]);
  }
}
