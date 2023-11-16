'use client';
import styles from './layout.module.css';
import { useAtom } from 'jotai';
import { isAuthenticatedAtom } from '@cloudinary-photos-app/global-store';
import { useState } from 'react';
import checkSecret from './checkSecret';

/* eslint-disable-next-line */
export interface LayoutProps {
  children: React.ReactNode;
}

export function AuthLayout(props: LayoutProps) {
  const [isAuthenticated, setIsAuthenticated] = useAtom(isAuthenticatedAtom);
  const [input, setInput] = useState<string>('');

  const handleCheck = async () => {
    const isCorrectPassword = await checkSecret(input);
    setIsAuthenticated(isCorrectPassword);
  };

  if (!isAuthenticated)
    return (
      <div className={styles['container']}>
        <h1>No authenticate</h1>
        <div className="p-8 justify-center items-center h-screen flex ">
          <input
            className="bg-gray-200 shadow-inner rounded-l p-2 flex-1 text-black"
            id="password"
            aria-label="secret"
            placeholder="Enter your secret password to access"
            value={input}
            onChange={(e) => setInput(e.target.value)}
          />
          <button
            className="bg-blue-600 hover:bg-blue-700 duration-300 text-white shadow p-2 rounded-r"
            onClick={handleCheck}
          >
            Check
          </button>
        </div>
      </div>
    );
  else return <>{props.children}</>;
}

export default AuthLayout;
