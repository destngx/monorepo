'use client';
import { IoLogoSoundcloud } from 'react-icons/io5';
import { useState } from 'react';

const SCToggleButton = () => {
  const [isShowSC, toggleSC] = useState(false);

  return (
    <>
      <div onClick={() => toggleSC(!isShowSC)}>
        <IoLogoSoundcloud />
      </div>
      {
        <div className={`absolute bottom-0 right-0 ${isShowSC ? '' : 'hidden'}`}>
          <iframe
            width="100%"
            height="80"
            allow="autoplay"
            src="https://w.soundcloud.com/player/?url=https%3A//api.soundcloud.com/playlists/1000012141&color=%23ff5500&auto_play=true&hide_related=false&show_comments=true&show_user=true&show_reposts=false&show_teaser=true&visual=true"
          />
        </div>
      }
    </>
  );
};

export { SCToggleButton };
