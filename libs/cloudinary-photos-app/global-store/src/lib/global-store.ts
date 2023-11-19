import { atom } from 'jotai';

// Create your atoms and derivatives
import { atomWithStorage, createJSONStorage } from 'jotai/utils';

const sessionAtomStorage = createJSONStorage(() => sessionStorage);
const isAuthenticatedAtom = atomWithStorage('isAuthenticated', null, sessionAtomStorage);
const cloudinaryProject = atom<string>('');

const errorNotificationAtom = atom<string>('');

export { isAuthenticatedAtom, cloudinaryProject, errorNotificationAtom };
