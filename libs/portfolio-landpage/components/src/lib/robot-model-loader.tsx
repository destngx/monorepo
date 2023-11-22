import { forwardRef, LegacyRef, ReactNode } from 'react';
import { Box, Spinner } from '@chakra-ui/react';

export const DogSpinner = () => (
  <Spinner
    size="xl"
    position="absolute"
    left="50%"
    top="50%"
    ml="calc(0px - var(--spinner-size) / 2)"
    mt="calc(0px - var(--spinner-size))"
  />
);

export const DogContainer = forwardRef(({ children }: { children: ReactNode }, ref: LegacyRef<HTMLDivElement>) => (
  <Box
    _hover={{ cursor: '-webkit-grab' }}
    ref={ref}
    className="robot-model"
    m="auto"
    mt={['-20px', '-60px', '-120px']}
    mb={['-40px', '-140px', '-192px']}
    w={[280, 480, 640]}
    h={[280, 480, 640]}
    position="relative"
  >
    {children}
  </Box>
));

const Loader = () => {
  return (
    <DogContainer>
      <DogSpinner />
    </DogContainer>
  );
};

export default Loader;
