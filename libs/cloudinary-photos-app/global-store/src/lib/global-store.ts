import { atom } from 'jotai';

// Create your atoms and derivatives
const isAuthenticatedAtom = atom(false);
const cloudinaryProject = atom('');

export { isAuthenticatedAtom, cloudinaryProject };
