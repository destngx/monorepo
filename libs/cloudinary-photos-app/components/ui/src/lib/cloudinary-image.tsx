'use client';

import { CldImage, CldImageProps } from 'next-cloudinary';
import { useState, useTransition } from 'react';
import { ImageMenu } from './image-menu';
import { FullHeart, Heart } from '@mono/cloudinary-photos-app/components/icons';
import { SearchResult } from '@mono/cloudinary-photos-app/types';
import { setAsFavoriteAction } from '../server/actions';

export function CloudinaryImage(
  props: {
    imagedata: SearchResult;
    onUnheart?: (unheartedResource: SearchResult) => void;
  } & Omit<CldImageProps, 'src'>,
) {
  const [, startTransition] = useTransition();

  const { imagedata, onUnheart } = props;

  const [isFavorited, setIsFavorited] = useState(imagedata.tags.includes('favorite'));

  return (
    <div
      className={
        'relative min-h-[200px] cursor-pointer rounded-md transition duration-300 hover:border-blue-400 border-2'
      }
    >
      <CldImage className={'rounded-sm'} {...props} src={imagedata.public_id} />
      {isFavorited ? (
        <FullHeart
          onClick={() => {
            onUnheart?.(imagedata);
            setIsFavorited(false);
            startTransition(() => {
              void setAsFavoriteAction(imagedata.public_id, false);
            });
          }}
          className="absolute top-2 left-2 hover:text-white text-red-500 cursor-pointer"
        />
      ) : (
        <Heart
          onClick={() => {
            setIsFavorited(true);
            startTransition(async () => {
              await setAsFavoriteAction(imagedata.public_id, true);
            });
          }}
          className="absolute top-2 left-2 hover:text-red-500 cursor-pointer"
        />
      )}
      <ImageMenu image={imagedata} />
    </div>
  );
}
