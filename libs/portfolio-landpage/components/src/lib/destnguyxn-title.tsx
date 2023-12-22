import React from 'react';

export const DestnguyxnTitle = ({ isAnimated, className }: { isAnimated: boolean; className?: string }) => {
  return (
    <div
      className={
        (!isAnimated ? 'bg-white/10 backdrop-blur-sm hover:backdrop-blur-xl' : '') +
        ' select-none z-10 m-2 content-around w-fit fixed ' +
        className
      }
    >
      <p
        className={
          (isAnimated ? 'animate-pulse ' : '') +
          ' backdrop-blur bg-amber-800/50 font-extrabold text-gray-900 text-6xl md:text-8xl text dark:text-white'
        }
      >
        DEST
      </p>
      <p className={(isAnimated ? 'animate-pulse ' : '') + 'text-gray-900 text-6xl md:text-8xl dark:text-white'}>
        NGUYXN
      </p>
    </div>
  );
};
