'use client';
import NextLink from 'next/link';
import { Heading, Box, Badge } from '@chakra-ui/react';

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
  <Box
    borderRadius="lg"
    w="full"
    p={8}
    mb={4}
    bg="gray.100"
    display="flex"
    alignItems="center"
    justifyContent="center"
    minH="200px"
  >
    <Box fontSize="xl" fontWeight="bold" color="gray.400">
      {alt}
    </Box>
  </Box>
);

export const Meta = ({ children }: { children: React.ReactNode }) => (
  <Badge colorScheme="green" mr={2}>
    {children}
  </Badge>
);
