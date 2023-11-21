'use server';
import { MongoClient, ObjectId } from 'mongodb';
import { cookies } from 'next/headers';

const ONE_DAY_IN_MILISECONDS = 1000 * 60 * 60 * 24;

type SystemSecret = {
  _id: ObjectId;
  password: string;
};
type Session = {
  _id: ObjectId;
  date: Date;
};

export default async function checkSecret(password: string) {
  const requestSession = cookies().get('session')?.value;
  const client = new MongoClient(process.env.MONGODB_URI || '', {});
  try {
    await client.connect();
    const database = client.db('cloud-photos-app'); // Choose a name for your database

    const sessionCollection = database.collection('session');
    let session = undefined;
    if (requestSession) {
      session = await sessionCollection.findOne<Session>({ _id: new ObjectId(requestSession) });
    }

    if (session) {
      if (Date.now() - session.date.getTime() <= ONE_DAY_IN_MILISECONDS) {
        return true;
      }
      await sessionCollection.deleteOne({ _id: session._id });
      return false;
    }
    const systemSecretCollection = database.collection('system-secret');
    const systemSecret = await systemSecretCollection.findOne<SystemSecret>({});

    const sessionToken = new ObjectId();
    if (systemSecret) {
      if (password === systemSecret?.password) {
        await sessionCollection.insertOne({
          _id: sessionToken,
          date: new Date(),
        });
        cookies().set('session', sessionToken.toString());
        return true;
      } else {
        return false;
      }
    }
  } catch (error) {
    console.error(error);
  } finally {
    await client.close();
  }
  return false;
}
