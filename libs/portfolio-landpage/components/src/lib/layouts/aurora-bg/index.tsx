import './style.css';
import { ReactNode } from 'react';
export const AuroraBg = ({ children }: { children: ReactNode }) => {
  return (
    <div className={'relative h-screen w-full'}>
      {children}
      <div id={'background'} className="absolute">
        <section id="up"></section>
        <section id="down"></section>
        <section id="left"></section>
        <section id="right"></section>
      </div>
    </div>
  );
};
