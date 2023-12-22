'use client';

import { SCToggleButton } from './soundcloud-button';
import React, { ComponentProps, useContext } from 'react';
import { motion } from 'framer-motion';
import { GlobalContext } from './context/global';

const Navbar = (props: ComponentProps<'div'>) => {
  // const { path } = props;
  const { isShowSoundCloudPlayer } = useContext(GlobalContext);

  return (
    <motion.div
      animate={isShowSoundCloudPlayer ? 'open' : 'closed'}
      variants={{
        open: { width: 'fit-content', height: 'fit-content' },
        closed: { width: '2.5rem', height: '2.5rem' },
      }}
      className="overflow-hidden fixed flex flex-row bottom-6 backdrop-blur bg-white/20 rounded p-1 m-1 "
    >
      <div className={'w-10'}>
        <SCToggleButton />
      </div>
      <motion.nav
        animate={isShowSoundCloudPlayer ? 'open' : 'closed'}
        variants={{ open: { opacity: 1, x: 0 }, closed: { opacity: 0, x: '-200%' } }}
        className={`rounded overflow-hidden shadow`}
      >
        <iframe
          title={'sc-player'}
          width="100%"
          height="100%"
          allow="autoplay"
          src="https://w.soundcloud.com/player/?url=https%3A//api.soundcloud.com/playlists/1000012141&color=%23ff5500&auto_play=true&hide_related=false&show_comments=false&show_user=true&show_reposts=false&show_teaser=true&visual=true"
        />
      </motion.nav>
    </motion.div>
  );
};

export { Navbar };
