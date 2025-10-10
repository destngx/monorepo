'use client';

import NextLink from 'next/link';
import { Box, Text, LinkBox, LinkOverlay } from '@chakra-ui/react';
import { Global } from '@emotion/react';

interface GridItemProps {
  children: React.ReactNode;
  href: string;
  title: string;
  thumbnail: any;
}
export const GridItem: React.FC<GridItemProps> = ({ children, href, title, thumbnail }) => (
  <Box w="100%" textAlign="center">
    <LinkBox cursor="pointer">
      <Box
        className="grid-item-thumbnail"
        bg="gray.100"
        p={8}
        display="flex"
        alignItems="center"
        justifyContent="center"
        minH="200px"
      >
        <Text fontSize="xl" fontWeight="bold" color="gray.400">
          {title}
        </Text>
      </Box>
      <LinkOverlay href={href} target="_blank">
        <Text mt={2}>{title}</Text>
      </LinkOverlay>
      <Text fontSize={14}>{children}</Text>
    </LinkBox>
  </Box>
);

interface WorkGridItemProps {
  children: React.ReactNode;
  id: string;
  title: string;
  thumbnail: string;
}
export const WorkGridItem = ({ children, id, title, thumbnail }: WorkGridItemProps) => (
  <Box w="100%" textAlign="center">
    {/* <NextLink href={`/works/${id}`} scroll={false}>*/}
    {/*  <LinkBox cursor="pointer">*/}
    {/*    <Image src={thumbnail} alt={title} className="grid-item-thumbnail" placeholder="blur" />*/}
    {/*    <LinkOverlay href={`/works/${id}`}>*/}
    {/*      <Text mt={2} fontSize={20}>*/}
    {/*        {title}*/}
    {/*      </Text>*/}
    {/*    </LinkOverlay>*/}
    {/*    <Text fontSize={14}>{children}</Text>*/}
    {/*  </LinkBox>*/}
    {/* </NextLink>*/}
  </Box>
);

export const GridItemStyle = () => (
  <Global
    styles={`
      .grid-item-thumbnail {
        border-radius: 12px;
      }
    `}
  />
);
