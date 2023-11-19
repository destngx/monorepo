import { useState } from 'react';
import Link from 'next/link';
import { Pencil } from 'lucide-react';
import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@nx-pnpm-monorepo/cloudinary-photos-app/components/ui';
import { Menu } from '@nx-pnpm-monorepo/cloudinary-photos-app/components/icons';
import { AddToAlbumDialog } from './add-to-album-dialog';
import { SearchResult } from '@nx-pnpm-monorepo/cloudinary-photos-app/types';

export function ImageMenu({ image }: { image: SearchResult }) {
  const [isOpened, setIsOpened] = useState(false);

  return (
    <div className={'absolute top-2 right-2 border-0'}>
      <DropdownMenu open={isOpened} onOpenChange={setIsOpened}>
        <DropdownMenuTrigger asChild>
          <Button variant="secondary" className="w-8 h-8 p-0">
            <Menu />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-40">
          <DropdownMenuItem asChild>
            <AddToAlbumDialog image={image} onClose={() => setIsOpened} />
          </DropdownMenuItem>
          <DropdownMenuItem asChild>
            <Button className="cursor-pointer flex justify-start pl-4" asChild variant="ghost">
              <Link href={`/detail?publicId=${encodeURIComponent(image.public_id)}`}>
                <Pencil className="mr-2 w-4 h-4" />
                Detail
              </Link>
            </Button>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
