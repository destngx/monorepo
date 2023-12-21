'use client';
import React, { useContext, useEffect } from 'react';
import { AuroraBg, DestnguyxnTitle, GlobalContext, GlobalContextWrapper } from '@portfolio-landpage/components';

const InitLoading = ({ className, isLoading }: { className?: string; isLoading: boolean }) => {
  return <DestnguyxnTitle className={className} isAnimated={isLoading} />;
};

const Loading = ({ children }: { children: React.ReactNode }) => {
  const [isLoading, setIsLoading] = React.useState(true);
  const { loadModelProgress } = useContext(GlobalContext);
  useEffect(() => {
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return,@typescript-eslint/no-unsafe-call
    setTimeout(() => setIsLoading(false), 3000);
  });

  return (
    <GlobalContextWrapper>
      <AuroraBg>
        {
          <>
            <InitLoading
              className={
                'h-34 rounded p-2 w-fit transition-all duration-1000 ' +
                (isLoading && loadModelProgress !== 100 ? 'translate-x-[30vw] translate-y-[40vh]' : '')
              }
              isLoading={isLoading && loadModelProgress !== 100}
            />
            <div
              className={
                (isLoading && loadModelProgress !== 100 ? 'opacity-0' : 'opacity-100') +
                ' ease-in-out transition-all delay-[5000ms] duration-3000'
              }
            >
              {children}
            </div>
          </>
        }
      </AuroraBg>
    </GlobalContextWrapper>
  );
};
export default Loading;
