'use server';
import cloudinary from 'cloudinary';
import { Folder, SearchResult } from '@nx-pnpm-monorepo/cloudinary-photos-app/types';
import { ExifParserFactory } from 'ts-exif-parser';

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

export async function getFolders() {
  const { folders } = (await cloudinary.v2.api.root_folders()) as {
    folders: Folder[];
  };
  // eslint-disable-next-line @typescript-eslint/no-unsafe-return
  return folders;
}

export async function getBufferFromUrl(url: string) {
  const response = await fetch(url);
  const buffer = await response.arrayBuffer();
  const parser = ExifParserFactory.create(buffer);
  const Data = ExifParserFactory.create(buffer).parse();
  // console.log(url, Data);
}
