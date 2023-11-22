import { Box } from '@chakra-ui/react';

const Footer = () => {
  return (
    <Box alignItems="center" opacity={0.4} fontSize="sm">
      &copy; {new Date().getFullYear()} Nguyen Pham Quang Dinh :"{'>'}
    </Box>
  );
};

export default Footer;
