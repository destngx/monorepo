'use client';
import { IoLogoSoundcloud } from 'react-icons/io5';
import { useContext } from 'react';
import { GlobalContext } from './context/global';

const SCToggleButton = () => {
  const { isShowSoundCloudPlayer: isShowSC, setIsShowSoundCloudPlayer: toggleSC } = useContext(GlobalContext);

  return (
    <div onClick={() => toggleSC(!isShowSC)}>
      <IoLogoSoundcloud />
    </div>
  );
};

export { SCToggleButton };
