'use client';

import NextLink from 'next/link';
import styled from '@emotion/styled';
import NextImage from 'next/image';

const LogoBox = styled.div`
  font-weight: bold;
  font-size: 18px;
  display: inline-flex;
  align-items: center;
  height: 30px;
  line-height: 20px;
  padding: 10px;

  img {
    transition: 200ms ease;
  }

  &:hover img {
    transform: rotate(20deg);
  }
`;

const Logo = () => {
  const footPrintImg = `/images/footprint${''}.png`;

  return (
    <NextLink href="/" scroll={false}>
      <LogoBox className={'cursor-pointer flex-col justify-around'}>
        <NextImage src={footPrintImg} width={20} height={20} alt="logo" />
        <div className={'ml-3 font-bold font-m-plus-rounded-1c'}>Dest Nguyen</div>
      </LogoBox>
    </NextLink>
  );
};

export { Logo };
