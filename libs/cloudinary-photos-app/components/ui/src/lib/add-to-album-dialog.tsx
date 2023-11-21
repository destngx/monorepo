import { FolderPlus } from 'lucide-react';
import { forwardRef, useState } from 'react';
import { addImageToAlbum } from '../server/actions';
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  Input,
  Label,
} from '@nx-pnpm-monorepo/cloudinary-photos-app/components/ui';

import { SearchResult } from '@nx-pnpm-monorepo/cloudinary-photos-app/types';

export const AddToAlbumDialog = forwardRef(({ image, onClose }: { image: SearchResult; onClose: () => void }, ref) => {
  const [albumName, setAlbumName] = useState('');
  const [isOpened, setIsOpened] = useState(false);

  return (
    <Dialog
      open={isOpened}
      onOpenChange={(newOpenState) => {
        setIsOpened(newOpenState);
        if (!newOpenState) {
          onClose();
        }
      }}
    >
      <DialogTrigger asChild>
        <Button variant="ghost">
          <FolderPlus className="mr-2 h-4 w-4" />
          <span>Add to Album</span>
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Add Image to Album</DialogTitle>
          <DialogDescription>Type an album you want to move this image into</DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="name" className="text-right">
              Album
            </Label>
            <Input
              onChange={(e) => setAlbumName(e.currentTarget.value)}
              id="album-name"
              value={albumName}
              className="col-span-3"
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            onClick={() => {
              onClose();
              setIsOpened(false);
              void addImageToAlbum(image, albumName);
            }}
            type="submit"
          >
            Add
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
});
