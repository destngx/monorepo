import { AnimatePresence } from 'framer-motion';
import { Chakra, Fonts, MainLayout } from '@portfolio-landpage/components';
import { NextRouter } from 'next/router';
import { ElementType, ReactNode } from 'react';

if (typeof window !== 'undefined') {
  window.history.scrollRestoration = 'manual';
}

interface Props {
  Component: ElementType;
  pageProps: Record<string, string>;
  router: NextRouter;
}
const Website = ({ Component, pageProps, router }: Props) => {
  return (
    <Chakra cookies={pageProps.cookies}>
      <Fonts />
      <MainLayout router={router}>
        <AnimatePresence
          mode={'wait'}
          initial={true}
          onExitComplete={() => {
            if (typeof window !== 'undefined') {
              window.scrollTo({ top: 0 });
            }
          }}
        >
          <Component {...pageProps} key={router.route} />
        </AnimatePresence>
      </MainLayout>
    </Chakra>
  );
};

export default Website;
