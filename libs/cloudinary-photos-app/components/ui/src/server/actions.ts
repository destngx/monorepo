'use server';
import cloudinary from 'cloudinary';
import { Folder, SearchResult } from '@nx-pnpm-monorepo/cloudinary-photos-app/types';
import * as ExifReader from 'exifreader';
import { MongoClient } from 'mongodb';

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

export async function getBufferFromUrl(url: string): Promise<void> {
  const response = await fetch(url);
  const buffer = await response.arrayBuffer();
  const tags = ExifReader.load(buffer, { includeUnknown: true, expanded: true });
  const cloudinaryImageName = url.split('/').pop();
  const cloudinaryImageId = cloudinaryImageName?.split('.').slice(0, -1).join('.');

  const client = new MongoClient(process.env.MONGODB_URI || '', {});
  try {
    await client.connect();
    const database = client.db('cloud-photos-app'); // Choose a name for your database

    const imageDetailCollection = database.collection('image-detail');
    await imageDetailCollection.insertOne({ cloudinaryImageId: cloudinaryImageId, tags: tags });
  } catch (error) {
    console.error(error);
  } finally {
    await client.close();
  }
}

export async function getImageDetail(publicId: string) {
  const client = new MongoClient(process.env.MONGODB_URI || '', {});
  try {
    await client.connect();
    const database = client.db('cloud-photos-app');

    const imageDetailCollection = database.collection('image-detail');
    return await imageDetailCollection.findOne({ cloudinaryImageId: publicId });
  } catch (e) {
    console.error(e);
  } finally {
    await client.close();
  }
}
