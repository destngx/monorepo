'use client';

import NextLink from 'next/link';
import styled from '@emotion/styled';

const LogoBox = styled.div`
  font-size: 24px;
  font-weight: bold;
  transition: 200ms ease;
  text-shadow:
    0 0 20px rgba(0, 0, 0, 0.8),
    0 0 40px rgba(0, 0, 0, 0.6);

  &:hover {
    transform: scale(1.1);
  }
`;

const Logo = () => {
  return (
    <NextLink href="/" scroll={false}>
      <LogoBox className={'cursor-pointer'}>destnguyxn</LogoBox>
    </NextLink>
  );
};

export { Logo };
