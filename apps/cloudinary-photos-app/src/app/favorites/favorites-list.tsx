'use client';

import { useEffect, useState } from 'react';
import { CloudinaryImage, ImageGrid } from '@nx-pnpm-monorepo/cloudinary-photos-app/components/ui';
import { SearchResult } from '@nx-pnpm-monorepo/cloudinary-photos-app/types';

export default function FavoritesList({ initialResources }: { initialResources: SearchResult[] }) {
  const [resources, setResources] = useState(initialResources);

  useEffect(() => {
    setResources(initialResources);
  }, [initialResources]);

  return (
    <ImageGrid
      images={resources}
      getImage={(imageData: SearchResult) => {
        return (
          <CloudinaryImage
            key={imageData.public_id}
            imagedata={imageData}
            width="400"
            height="300"
            alt="an image of something"
            onUnheart={(unheartedResource) => {
              setResources((currentResources) =>
                currentResources.filter((resource) => resource.public_id !== unheartedResource.public_id),
              );
            }}
          />
        );
      }}
    />
  );
}
