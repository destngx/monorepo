// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore
'use client';

import { isValidMotionProp, motion } from 'framer-motion';
import { chakra } from '@chakra-ui/react';
import React from 'react';

const StyledDiv = chakra(motion.div, {
  shouldForwardProp: isValidMotionProp,
});

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore
const Section = ({ children, delay = 0 }: { children: React.ReactNode; delay?: number }) => (
  <StyledDiv initial={{ y: 10, opacity: 0 }} animate={{ y: 0, opacity: 1 }} transition="0.8s" mb={6}>
    {children}
  </StyledDiv>
);

export { Section };
