'use client';

import ThemeToggleButton from './theme-toggle-button';
import { SCToggleButton } from './soundcloud-button';
import React, { ComponentProps } from 'react';

const Navbar = (props: ComponentProps<'div'>) => {
  // const { path } = props;

  return (
    <div className="fixed bottom-6 h-fit w-12 backdrop-blur bg-white/20 rounded p-1 m-1 ">
      <div className={'container py-4 h-full flex gap-2 flex-col justify-between items-center align-middle'}>
        <SCToggleButton />
        <ThemeToggleButton />
      </div>
    </div>
  );
};

export { Navbar };
