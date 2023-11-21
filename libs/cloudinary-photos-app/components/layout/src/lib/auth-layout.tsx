'use client';
import { useAtom } from 'jotai';
import { errorNotificationAtom, isAuthenticatedAtom } from '@cloudinary-photos-app/global-store';
import React, { useEffect, useState } from 'react';
import checkSecret from './checkSecret';

export interface LayoutProps {
  children: React.ReactNode;
}

export function AuthLayout(props: LayoutProps) {
  const [isAuthenticated, setIsAuthenticated] = useAtom(isAuthenticatedAtom);
  const [errorNotification, setErrorNotification] = useAtom(errorNotificationAtom);
  const [input, setInput] = useState<string>('');
  const [isChecking, setIsChecking] = useState<boolean>(true);

  useEffect(() => {
    handleCheck();
  }, []);
  useEffect(() => {
    if (errorNotification.length) {
      setTimeout(() => {
        setErrorNotification('');
      }, 3000);
    }
  }, [errorNotification, setErrorNotification]);

  const handleCheck = () => {
    const sessionAuthenticated = sessionStorage.getItem('isAuthenticated');
    let isCorrectPassword = false;
    void (async () => {
      if (sessionAuthenticated !== 'true') {
        setIsChecking(true);
        isCorrectPassword = await checkSecret(input);
        setIsChecking(false);
        if (!isCorrectPassword) {
          setErrorNotification("You don't have a valid authenticate");
          return;
        }
      }
      setIsAuthenticated(isCorrectPassword);
    })();
  };
  if (!isAuthenticated)
    return (
      <div className="h-screen flex flex-col justify-center items-start">
        <h1 className="pl-8 pt-8 text-xl leading-4 font-semibold text-red-500">
          {isChecking ? 'Checking' : 'No'} authenticate
        </h1>
        <div className="p-8 pt-4 justify-center items-center flex ">
          <input
            disabled={isChecking}
            className="bg-gray-200 shadow-inner rounded-l p-2 flex-1 text-black active:border-0"
            id="password"
            aria-label="secret"
            placeholder="Enter your password"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                void handleCheck();
              }
            }}
          />
          <button
            className={
              'inline-flex items-center px-4 py-2 font-semibold leading-6 text-sm shadow rounded-r-md text-white bg-indigo-500 hover:bg-indigo-400 transition ease-in-out duration-150'
            }
            onClick={handleCheck}
          >
            {isChecking && (
              <svg
                className="animate-spin -ml-1 mr-3 h-5 w-5 text-white"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                ></path>
              </svg>
            )}
            Check
          </button>
        </div>
        <div className="border-red-950 text-red-500 animate-accordion-down duration-300 transition">
          {errorNotification}
        </div>
      </div>
    );
  else return <div>{props.children}</div>;
}

export default AuthLayout;
