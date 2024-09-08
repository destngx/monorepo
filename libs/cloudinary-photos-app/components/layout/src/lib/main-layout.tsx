'use client';
import { Avatar, AvatarFallback, AvatarImage, Button } from '@mono/cloudinary-photos-app/components/ui';
import { MenuHamburger } from '@mono/cloudinary-photos-app/components/icons';
import { ReactNode, useState } from 'react';
import Image from 'next/image';
import { SideMenu } from '@mono/cloudinary-photos-app/components/ui/server';
import Link from 'next/link';

export function MainLayout({ children }: { children: ReactNode }) {
  const [isOpenedMenu, setIsOpenedMenu] = useState(true);
  return (
    <>
      <div className="border-b">
        <div className="flex h-16 items-center px-4 container mx-auto">
          <Button className={'block'} onClick={() => setIsOpenedMenu(!isOpenedMenu)}>
            <MenuHamburger />
          </Button>
          <Link className={'flex my-auto align-middle items-center'} href={'/'}>
            <Image src="/album.png" width="50" height="50" alt="icon of this photo album app" />
            <span className="ml-2 text-2xl font-bold">DestNgX Photos</span>
          </Link>
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
