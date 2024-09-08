'use client';

import { CldImage, CldUploadButton, CloudinaryUploadWidgetResults } from 'next-cloudinary';
import { useState } from 'react';
import { getBufferFromUrl } from '@mono/cloudinary-photos-app/components/ui/server';

export type UploadResult = {
  info: { public_id: string };
  event: 'success';
};
export default function Home() {
  const [imageId, setImageId] = useState('');

  return (
    <main className="flex min-h-screen flex-col items-center justify-between p-24">
      <div>
        <CldUploadButton
          className="bg-pink-400 p-4 rounded hover:bg-purple-400 hover:cursor-pointer transition"
          onUploadAdded={(results: CloudinaryUploadWidgetResults) => {
            const data = results.info;
            if (typeof data === 'string') {
              return;
            }
            void getBufferFromUrl(data.secure_url);
            setImageId(data.public_id);
          }}
          uploadPreset="nmdr2f41"
        />
      </div>
      {imageId && <CldImage width="500" height="300" src={imageId} sizes="100vw" alt="Description of my image" />}
    </main>
  );
}
