'use client';

import NextLink from 'next/link';
import styled from '@emotion/styled';
import NextImage from 'next/image';

const LogoBox = styled.div`
  img {
    transition: 200ms ease;
  }

  &:hover img {
    transform: rotate(180deg);
  }
`;

const Logo = () => {
  const footPrintImg = `/images/footprint${''}.png`;

  return (
    <NextLink href="/" scroll={false}>
      <LogoBox className={'cursor-pointer'}>
        <NextImage src={footPrintImg} width={128} height={128} alt="logo" />
      </LogoBox>
    </NextLink>
  );
};

export { Logo };
