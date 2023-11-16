'use server';
// React server components are async so you make database or API calls.
import { MongoClient } from 'mongodb';

type SystemSecret = {
  _id: string;
  password: string;
};

export default async function checkSecret(password: string) {
  const client = new MongoClient(process.env.MONGODB_URI || '', {
    // useNewUrlParser: true,
    // useUnifiedTopology: true,
  });

  try {
    await client.connect();
    const database = client.db('cloud-photos-app'); // Choose a name for your database
    const collection = database.collection('system-secret'); // Choose a name for your collection
    const result = await collection.findOne<SystemSecret>({});

    return password === result?.password;
  } catch (error) {
    console.error(error);
  } finally {
    await client.close();
  }

  return false;
}
