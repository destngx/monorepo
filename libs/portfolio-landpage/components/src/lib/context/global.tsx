'use client';
import { createContext, Dispatch, SetStateAction, useState } from 'react';

interface GlobalContextProps {
  isShowSoundCloudPlayer: boolean;
  setIsShowSoundCloudPlayer: Dispatch<SetStateAction<boolean>>;
  loadModelProgress: number;
  setLoadModelProgress: Dispatch<SetStateAction<number>>;
}

const initialState: GlobalContextProps = {
  isShowSoundCloudPlayer: false,
  setIsShowSoundCloudPlayer: () => {
    throw new Error('setLoadModelProgress function must be overridden');
  },
  loadModelProgress: 0,
  setLoadModelProgress: () => {
    throw new Error('setLoadModelProgress function must be overridden');
  },
};

const GlobalContext = createContext(initialState);

const GlobalProvider = GlobalContext.Provider;
const GlobalConsumer = GlobalContext.Consumer;

const GlobalContextWrapper = ({ children }: { children: React.ReactNode }) => {
  const [isShowSoundCloudPlayer, setIsShowSoundCloudPlayer] = useState<boolean>(false);
  const [loadModelProgress, setLoadModelProgress] = useState<number>(0);

  return (
    <GlobalProvider
      value={{ isShowSoundCloudPlayer, setIsShowSoundCloudPlayer, loadModelProgress, setLoadModelProgress }}
    >
      {children}
    </GlobalProvider>
  );
};

export { GlobalContext, GlobalContextWrapper, GlobalConsumer };
