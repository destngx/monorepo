'use client';

import { Button, Input, Label } from '@mono/cloudinary-photos-app/components/ui';
import { useEffect, useState } from 'react';
import { CldImage } from 'next-cloudinary';
import { getBufferFromUrl, getImageDetail } from '@mono/cloudinary-photos-app/components/ui/server';
import { allExpanded, darkStyles, JsonView } from 'react-json-view-lite';
import 'react-json-view-lite/dist/index.css';

export default function EditPage({
  searchParams: { publicId },
}: any) {
  const [transformation, setTransformation] = useState<
    undefined | 'generative-fill' | 'blur' | 'grayscale' | 'pixelate' | 'bg-remove'
  >();

  const [pendingPrompt, setPendingPrompt] = useState('');
  const [prompt, setPrompt] = useState('');
  const [imageDetail, setImageDetail] = useState<unknown>();

  useEffect(() => {
    void (async () => {
      setImageDetail(await getImageDetail(publicId));
    })();
  }, []);
  return (
    <section>
      <div className="flex flex-col gap-8">
        <div className="flex justify-between">
          <h1 className="text-4xl font-bold">Edit {publicId}</h1>
        </div>

        <div className="flex gap-4">
          <div className="flex flex-col gap-4">
            <Button
              onClick={() => {
                setTransformation('generative-fill');
                setPrompt(pendingPrompt);
              }}
            >
              Apply Generative Fill
            </Button>
            <Label>Prompt</Label>
            <Input value={pendingPrompt} onChange={(e) => setPendingPrompt(e.currentTarget.value)} />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-12">
          <CldImage
            onLoad={(e) => void getBufferFromUrl(e.currentTarget.src)}
            src={publicId}
            width="400"
            height="300"
            alt="some image"
          />

          {transformation === 'generative-fill' && (
            <CldImage
              src={publicId}
              width="1400"
              height="900"
              alt="some image"
              crop="pad"
              fillBackground={{
                prompt,
              }}
            />
          )}
          <JsonView data={imageDetail ?? {}} shouldExpandNode={allExpanded} style={darkStyles} />
        </div>
      </div>
    </section>
  );
}
