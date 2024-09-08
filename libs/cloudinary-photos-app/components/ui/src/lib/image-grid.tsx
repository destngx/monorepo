'use client';

import { ReactNode } from 'react';
import { SearchResult } from '@mono/cloudinary-photos-app/types';

const MAX_COLUMNS = 4;

export function ImageGrid({
  images,
  getImage,
}: {
  images: SearchResult[];
  getImage: (imagedata: SearchResult) => ReactNode;
}) {
  function getColumns(colIndex: number) {
    return images.filter((resource, idx) => idx % MAX_COLUMNS === colIndex);
  }

  return (
    <div className="columns-1 md:columns-3 xl:columns-4 gap-4">
      {[images].map((column, idx) => (
        <div key={idx} className="flex flex-col gap-4">
          {column.map(getImage)}
        </div>
      ))}
    </div>
  );
}
