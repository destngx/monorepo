import cloudinary from 'cloudinary';
import AlbumGrid from './album-grid';
import { SearchResult } from '@mono/cloudinary-photos-app/types';
import { ForceRefresh } from '@mono/cloudinary-photos-app/components/ui';

export default async function GalleryPage({
  params: { albumName },
}: any) {
  const results = (await cloudinary.v2.search
    .expression(`resource_type:image AND folder=${albumName}`)
    .sort_by('created_at', 'desc')
    .with_field('tags')
    .max_results(30)
    .execute()) as { resources: SearchResult[] };

  return (
    <section>
      <ForceRefresh />
      <div className="flex flex-col gap-8">
        <div className="flex justify-between">
          <h1 className="text-4xl font-bold">Album {albumName}</h1>
        </div>

        <AlbumGrid images={results.resources} />
      </div>
    </section>
  );
}
