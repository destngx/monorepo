import {
  Container,
  Badge,
  Link,
  List,
  ListItem,
  SimpleGrid,
  UnorderedList,
  Heading,
  Center,
  Image,
} from '@chakra-ui/react';
import { ExternalLinkIcon } from '@chakra-ui/icons';
import { Paragraph as P, ArticleLayout as Layout, Title, WorkImage, Meta } from '@portfolio-landpage/components';

const Work = () => (
  <Layout title="eHealicords">
    <Container>
      <Title>
        eHealicords <Badge>2021 - 2022</Badge>
      </Title>
      <Center my={6}>
        <Image src="/images/works/ehealicords.ico" alt="icon" />
      </Center>
      <P>Provides a patient record management interface for the system of clinics.</P>

      <UnorderedList ml={4} my={4}>
        <ListItem>Description.</ListItem>
      </UnorderedList>

      <List ml={4} my={4}>
        <ListItem>
          <Meta>Platform</Meta>
          <span>Web</span>
        </ListItem>
        <ListItem>
          <Meta>Stack</Meta>
          <span>NextJS, CharkaUI</span>
        </ListItem>
        <ListItem>
          <Meta>Link</Meta>
          <span>
            <Link href="https://ehealicords.vercel.app/" target={'_blank'}>
              eHealicords
              <ExternalLinkIcon mx="2px" />
            </Link>
          </span>
        </ListItem>
      </List>

      <Heading as="h4" fontSize={16} my={6}>
        <Center>Media coverage</Center>
      </Heading>

      <SimpleGrid columns={2} gap={2}>
        <WorkImage src="/images/works/ehealicords_01.png" alt="ehealicords" />
      </SimpleGrid>
    </Container>
  </Layout>
);

export default Work;
export { getServerSideProps } from '@portfolio-landpage/components';
