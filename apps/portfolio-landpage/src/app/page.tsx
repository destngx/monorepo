'use client';
import { BioSection, BioYear, GlobalContext, Navbar, Paragraph, ThreeModel } from '@portfolio-landpage/components';
import NextLink from 'next/link';
import { IoLogoGithub, IoLogoYoutube } from 'react-icons/io5';
import React, { useContext } from 'react';

if (typeof window !== 'undefined') {
  window.history.scrollRestoration = 'manual';
}

const projectList = [
  { projectName: 'Photo Gallery', url: 'https://destnguyxn-photos.vercel.app/', year: 2023 },
  { projectName: 'ProjectName', url: '', year: 2023 },
  { projectName: 'ProjectName', url: '', year: 2023 },
  { projectName: 'ProjectName', url: '', year: 2023 },
];
const Home = () => {
  const { isShowSoundCloudPlayer } = useContext(GlobalContext);
  return (
    <div className={' flex '}>
      <Navbar />
      <div className={'m-2 grow grid grid-cols-1 md:grid-cols-2 gap-6'}>
        <div className={'mt-60 m-2 p-1 flex-col flex gap-6'}>
          <div className="w-full font-thin backdrop-blur text-gray-50 bg-black/10 p-2 rounded shadow">
            Hi! I&apos;m Nguyen Pham Quang Dinh, a full-stack developer based in Ho Chi Minh, Viet Nam
          </div>
          <div className={'py-2'}>
            <BioSection>
              <BioYear>1999</BioYear>
              Born in Lam Dong, Viet Nam.
            </BioSection>
            <BioSection>
              <BioYear>2022</BioYear>
              Completed the Master&apos;s Program in the Graduate School of Information Science of Ho Chi Minh
              University of Science
            </BioSection>
            <BioSection>
              <BioYear>2023</BioYear>
              Working at a top company of Korea
            </BioSection>
            <BioSection>
              <BioYear>2024</BioYear>
              Concurrently being a freelancer and learning new things
            </BioSection>
          </div>
          <Paragraph>
            I ♥ Art, Drawing, Music{' '}
            <NextLink
              href="https://open.spotify.com/user/22uqfqyasviijy57kq2fnxv3a?si=591a37358eb34a77"
              target="_blank"
              className={'font-bold'}
            >
              (here my spotify)
            </NextLink>
            , Playing Guitar,{' '}
            <NextLink href="https://www.eyeem.com/u/destnguyxn" target="_blank">
              Photography
            </NextLink>
            , Machine Learning
          </Paragraph>
          <div>
            Find me on the web
            <div className={'flex flex-col gap-1'}>
              <NextLink
                className={'flex flex-row font-bold gap-2 align-middle items-center'}
                href={'https://github.com/destnguyxn'}
              >
                <IoLogoGithub />
                Github ↗
              </NextLink>
              <NextLink
                className={'flex flex-row font-bold gap-2 align-middle items-center'}
                href={'https://www.youtube.com/'}
              >
                <IoLogoYoutube />
                Youtube ↗
              </NextLink>
            </div>
          </div>
        </div>
        <div className={'flex flex-col text-right gap-4'}>
          <ThreeModel />
          <div>Projects</div>
          {projectList.map((project) => {
            return (
              <NextLink
                key={Math.random()}
                href={project.url}
                className={'flex flex-row-reverse justify-items-start gap-4'}
              >
                <div className={'text-5xl'}>{project.projectName}</div>
                <div className={'self-end'}>{project.year} /</div>
              </NextLink>
            );
          })}
        </div>
      </div>

      <div
        className={`border-yellow-500 border-[1px] absolute rounded overflow-hidden shadow bottom-6 mb-4 left-16 transition-all duration-300 ease-in-out ${
          isShowSoundCloudPlayer ? '' : 'hidden'
        }`}
      >
        <iframe
          width="100%"
          height="100%"
          allow="autoplay"
          src="https://w.soundcloud.com/player/?url=https%3A//api.soundcloud.com/playlists/1000012141&color=%23ff5500&auto_play=true&hide_related=false&show_comments=false&show_user=true&show_reposts=false&show_teaser=true&visual=true"
        />
      </div>
    </div>
  );
};

export default Home;
