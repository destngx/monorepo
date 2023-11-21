'use client';
import { Avatar, AvatarFallback, AvatarImage, Button } from '@nx-pnpm-monorepo/cloudinary-photos-app/components/ui';
import { MenuHamburger } from '@nx-pnpm-monorepo/cloudinary-photos-app/components/icons';
import { ReactNode, useState } from 'react';
import Image from 'next/image';
import { SideMenu } from '@nx-pnpm-monorepo/cloudinary-photos-app/components/ui/server';

export function MainLayout({ children }: { children: ReactNode }) {
  const [isOpenedMenu, setIsOpenedMenu] = useState(true);
  return (
    <>
      <div className="border-b">
        <div className="flex h-16 items-center px-4 container mx-auto">
          <Button className={'block'} onClick={() => setIsOpenedMenu(!isOpenedMenu)}>
            <MenuHamburger />
          </Button>
          <Image src="/album.png" width="50" height="50" alt="icon of this photo album app" />
          DestNgX Photos
          <div className="ml-auto flex items-center space-x-4">
            <Avatar>
              <AvatarImage src="" alt="@destnguyxn" />
              <AvatarFallback>CN</AvatarFallback>
            </Avatar>
          </div>
        </div>
      </div>
      <div className="flex">
        {!isOpenedMenu && <SideMenu />}
        <div className="w-full px-4 pt-8">{children}</div>
      </div>
    </>
  );
}
