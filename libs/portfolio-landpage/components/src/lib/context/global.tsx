'use client';
import { createContext, useState } from 'react';

const initialState = {
  isShowSoundCloudPlayer: false,
  setIsShowSoundCloudPlayer: (value: boolean) => {},
  loadModelProgress: 0,
  setLoadModelProgress: (value: number) => {},
};
const GlobalContext = createContext(initialState);

const GlobalProvider = GlobalContext.Provider;
const GlobalConsumer = GlobalContext.Consumer;

const GlobalContextWrapper = ({ children }: { children: React.ReactNode }) => {
  const [isShowSoundCloudPlayer, setIsShowSoundCloudPlayer] = useState(false);
  const [loadModelProgress, setLoadModelProgress] = useState(0);
  return (
    <GlobalProvider
      value={{ isShowSoundCloudPlayer, setIsShowSoundCloudPlayer, loadModelProgress, setLoadModelProgress }}
    >
      {children}
    </GlobalProvider>
  );
};

export { GlobalContext, GlobalContextWrapper };
