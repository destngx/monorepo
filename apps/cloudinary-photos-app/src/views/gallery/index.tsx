import { GalleryGrid } from '@/widgets/gallery-grid';
import cloudinary from 'cloudinary';
import { SearchForm } from '@/widgets/search-form';
import { UploadButton } from '@/widgets/upload-button';
import { SearchResult } from '@mono/cloudinary-photos-app/types';

export default async function GalleryPage({
  searchParams: { search = '' },
}: any) {
  const results = (await cloudinary.v2.search
    .expression(`resource_type:image${search != '' ? ` AND tags=${search}` : ''}`)
    .sort_by('created_at', 'desc')
    .with_field('tags')
    .max_results(30)
    .execute()) as { resources: SearchResult[] };

  return (
    <section>
      <div className="flex flex-col gap-8">
        <div className="flex justify-between">
          <h1 className="text-4xl font-bold">Gallery</h1>
          <UploadButton />
        </div>

        <SearchForm initialSearch={search} />

        <GalleryGrid images={results.resources} />
      </div>
    </section>
  );
}

// Named export for routing delegation
export { GalleryPage };
