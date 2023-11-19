import { Loader } from '@nx-pnpm-monorepo/cloudinary-photos-app/components/ui';
import React from 'react';

const Loading = () => {
  return (
    <div className="h-screen flex flex-col justify-center items-start flex-wrap content-around">
      <Loader />
    </div>
  );
};
export default Loading;
