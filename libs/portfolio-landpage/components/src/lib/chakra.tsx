'use client';
import { ChakraProvider, createCookieStorageManager, localStorageManager } from '@chakra-ui/react';
import { theme } from './lib/theme';
import { NextRequest } from 'next/server';
// import { CyHttpMessages } from 'cypress/types/net-stubbing';
// import IncomingHttpRequest = CyHttpMessages.IncomingHttpRequest;

export function Chakra({ cookies, children }: { cookies: string; children: React.ReactNode }) {
  const colorModeManager = typeof cookies === 'string' ? createCookieStorageManager(cookies) : localStorageManager;

  return (
    <ChakraProvider theme={theme} colorModeManager={colorModeManager}>
      {children}
    </ChakraProvider>
  );
}

export function getServerSideProps({ req }: { req: NextRequest & { headers: { cookie?: string } } }) {
  return {
    props: {
      cookies: req.headers.cookie ?? '',
    },
  };
}
