'use client';
import { IoLogoSoundcloud } from 'react-icons/io5';
import { useContext } from 'react';
import { GlobalContext } from './context/global';
import { motion } from 'framer-motion';

const SCToggleButton = () => {
  const { isShowSoundCloudPlayer: isShowSC, setIsShowSoundCloudPlayer: toggleSC } = useContext(GlobalContext);

  return (
    <motion.button
      whileHover={{ scale: 1.5 }}
      whileTap={{ scale: 0.9 }}
      className={'fixed bottom-2 left-3'}
      onClick={() => toggleSC(!isShowSC)}
    >
      <IoLogoSoundcloud />
    </motion.button>
  );
};

export { SCToggleButton };
