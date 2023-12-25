'use client';
import { IoLogoSoundcloud } from 'react-icons/io5';
import { useContext } from 'react';
import { GlobalContext } from './context/global';
import { motion } from 'framer-motion';
import { Music } from 'lucide-react';
import { FaCompactDisc } from 'react-icons/fa';
import { IconContext } from 'react-icons/lib';

const SCToggleButton = () => {
  const { isShowSoundCloudPlayer: isShowSC, setIsShowSoundCloudPlayer: toggleSC } = useContext(GlobalContext);

  return (
    <motion.button
      whileHover={{ scale: 1.5 }}
      whileTap={{ scale: 0.9 }}
      className={'fixed bottom-2 left-2 animate-spin'}
      onClick={() => toggleSC(!isShowSC)}
    >
      <IconContext.Provider value={{ size: '1.4em' }}>
        <FaCompactDisc />
      </IconContext.Provider>
    </motion.button>
  );
};

export { SCToggleButton };
