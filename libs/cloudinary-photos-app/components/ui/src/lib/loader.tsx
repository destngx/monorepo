export const Loader = () => {
  const circleCommonClasses = 'h-2.5 w-2.5 bg-current rounded-full';

  return (
    <div className="flex">
      <div className={`${circleCommonClasses} mr-1 animate-bounce delay-100`}></div>
      <div className={`${circleCommonClasses} mr-1 animate-bounce delay-200`}></div>
      <div className={`${circleCommonClasses} animate-bounce`}></div>
    </div>
  );
};
