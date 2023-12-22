'use client';

import { AnimatePresence, motion } from 'framer-motion';
import { MoonIcon } from 'lucide-react';
// import { SunIcon, MoonIcon } from '@chakra-ui/icons';

const ThemeToggleButton = () => {
  return (
    <AnimatePresence mode={'wait'} initial={false}>
      <motion.div
        className={'fixed bottom-6 left-2 '}
        style={{ display: 'inline-block' }}
        initial={{ y: -20, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        exit={{ y: 20, opacity: 0 }}
        transition={{ duration: 0.2 }}
      >
        <div
          aria-label="Toggle theme"
          onClick={() => {
            console.error('toggle theme');
          }}
        >
          <MoonIcon />
        </div>
      </motion.div>
    </AnimatePresence>
  );
};

export default ThemeToggleButton;
