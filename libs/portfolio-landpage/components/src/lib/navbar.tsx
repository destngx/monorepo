'use client';

import { Logo } from './logo';
import NextLink from 'next/link';
import ThemeToggleButton from './theme-toggle-button';
import { SCToggleButton } from './soundcloud-button';
import React, { ComponentProps } from 'react';
import { IoLogoGithub } from 'react-icons/io5';

const Navbar = (props: ComponentProps<'div'>) => {
  // const { path } = props;

  return (
    <div className={'sticky top-0 h-20 w-full z-1 backdrop-blur p-4 bg-slate-900 text-yellow-100'}>
      <div className={'container flex p-2 flex-wrap align-middle items-center justify-between'}>
        <Logo />
        <div className={'flex align-middle items-center flex-grow'}>
          <NextLink href={'/page'}>Page</NextLink>
          <NextLink href={'/post'}>Post</NextLink>
          <NextLink href={'/source'}>
            <IoLogoGithub />
            Source
          </NextLink>
        </div>
        <ThemeToggleButton />
        <SCToggleButton />
        <NextLink href="/" passHref>
          About
        </NextLink>
        <NextLink href="/works" passHref>
          Works
        </NextLink>
        <NextLink href="/posts" passHref>
          Posts
        </NextLink>
        View Source
      </div>
    </div>
  );
};

export { Navbar };
