'use client';
import NextLink from 'next/link';
import { Heading, Box, Image, Badge } from '@chakra-ui/react';

export const Title = ({ children }: { children: React.ReactNode }) => (
  <Box>
    <NextLink href="/works">
      <p>Page</p>
    </NextLink>
    <span> {/* <ChevronRightIcon />{' '}*/}</span>
    <Heading display="inline-block" as="h3" fontSize={20} mb={4}>
      {children}
    </Heading>
  </Box>
);

export const WorkImage = ({ src, alt }: { src: string; alt: string }) => (
  <Image borderRadius="lg" w="full" src={src} alt={alt} mb={4} />
);

export const Meta = ({ children }: { children: React.ReactNode }) => (
  <Badge colorScheme="green" mr={2}>
    {children}
  </Badge>
);
