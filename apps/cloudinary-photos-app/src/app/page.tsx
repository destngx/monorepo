'use client';

import { CldImage, CldUploadButton } from 'next-cloudinary';
import { useState } from 'react';

export type UploadResult = {
  info: { public_id: string };
  event: 'success';
};
export default function Home() {
  const [imageId, setImageId] = useState('');

  return (
    <main className="flex min-h-screen flex-col items-center justify-between p-24">
      <CldUploadButton
        onUpload={(results: UploadResult) => setImageId(results.info?.public_id)}
        uploadPreset="nmdr2f41"
      />

      {imageId && <CldImage width="500" height="300" src={imageId} sizes="100vw" alt="Description of my image" />}
    </main>
  );
}
