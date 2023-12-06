import { BioSection, BioYear, GridItem, LazyRobotModel, Paragraph, Section } from '@portfolio-landpage/components';
import NextLink from 'next/link';
import { IoLogoGithub, IoLogoYoutube } from 'react-icons/io5';
import thumbYouTube from '../../public/images/links/youtube.jpeg';
import thumbYTLilKitty from '../../public/images/links/ytlilkitty.png';
import NextImage from 'next/image';

if (typeof window !== 'undefined') {
  window.history.scrollRestoration = 'manual';
}

const Home = () => {
  return (
    <div className={'bg-pink-400'}>
      {/* <AnimatePresence
        mode={'wait'}
        initial={true}
        onExitComplete={() => {
          if (typeof window !== 'undefined') {
            window.scrollTo({ top: 0 });
          }
        }}
      <LazyRobotModel />
      >*/}
      <div className={'bg-slate-400 rounded-sm h-10'}>
        Hello I&apos;m a full-stack developer based in Viet Nam! Nguyen Pham Quang Dinh
      </div>
      <p>Digital Craftman ( Artist / Developer / Designer )</p>
      <NextImage src="/images/me.jpg" alt="Profile image" width={'100'} height="100" />
      <Section delay={0.1}>
        Work
        <Paragraph>
          I am a freelance and a full-stack developer based in Ho Chi Minh, Viet Nam with a passion for building digital
          services/stuffs.
        </Paragraph>
        <NextLink href="/works">
          <p>My Portfolio</p>
        </NextLink>
      </Section>
      <Section delay={0.2}>
        Bio
        <BioSection>
          <BioYear>1999</BioYear>
          Born in Lam Dong, Viet Nam.
        </BioSection>
        <BioSection>
          <BioYear>2022</BioYear>
          Completed the Master&apos;s Program in the Graduate School of Information Science of Ho Chi Minh University of
          Science
        </BioSection>
      </Section>
      <Section delay={0.3}>
        I ♥
        <Paragraph>
          Art, Drawing,{' '}
          <NextLink href="https://open.spotify.com/user/22uqfqyasviijy57kq2fnxv3a?si=591a37358eb34a77" target="_blank">
            Music (here my spotify)
          </NextLink>
          , Playing Guitar,{' '}
          <NextLink href="https://www.eyeem.com/u/destnguyxn" target="_blank">
            Photography
          </NextLink>
          , Machine Learning
        </Paragraph>
      </Section>
      On the web
      <NextLink href="https://github.com/destnguyxn" target="_blank">
        @dest.nguyxn
      </NextLink>
      <IoLogoYoutube />
      <GridItem
        href="https://www.youtube.com/channel/UCa7k2ySSe-9fkNV3WcoGDgA"
        title="hoping to make more videos soon..."
        thumbnail={thumbYouTube}
      >
        <div></div>
      </GridItem>
      <GridItem href="https://www.youtube.com/shorts/TVXjyCdU7rk" title="pet lil kitty" thumbnail={thumbYTLilKitty}>
        <div></div>
      </GridItem>
      <NextLink href="/posts" scroll={false}>
        Popular posts
      </NextLink>
      {/*  </AnimatePresence>*/}
    </div>
  );
};

export default Home;
