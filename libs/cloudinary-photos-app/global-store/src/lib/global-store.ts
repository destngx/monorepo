import { atom } from 'jotai';

// Define atoms and derivatives
const isAuthenticatedAtom = atom(false);
const cloudinaryProject = atom<string>('');

const errorNotificationAtom = atom<string>('');

export { isAuthenticatedAtom, cloudinaryProject, errorNotificationAtom };
