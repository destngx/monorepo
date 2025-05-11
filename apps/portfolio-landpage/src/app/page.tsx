'use client';

import React from 'react';

if (typeof window !== 'undefined') {
  window.history.scrollRestoration = 'manual';
}


const Home = () => {
  return (
    <div className={' flex '}>
      Hello
    </div>
  );
};

export default Home;
