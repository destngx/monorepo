import * as React from 'react';

import { Button } from '@mono/cloudinary-photos-app/components/ui';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@mono/cloudinary-photos-app/components/ui';
import Link from 'next/link';

import { Folder } from '@mono/cloudinary-photos-app/types';

export type AlbumCardProps = { folder: Folder };

export function AlbumCard({ folder }: AlbumCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{folder.name}</CardTitle>
        <CardDescription>All your {folder.name} images</CardDescription>
      </CardHeader>
      <CardContent></CardContent>
      <CardFooter className="flex justify-between">
        <Button asChild>
          <Link href={`/albums/${folder.name}`}>View Album</Link>
        </Button>
      </CardFooter>
    </Card>
  );
}
