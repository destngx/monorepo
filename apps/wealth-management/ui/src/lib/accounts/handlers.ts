import type { Account } from '$lib/api/types';

// Callback shape shared by the accounts components.
export type AccountHandler = (account: Account) => void;
export type SubmitHandler = (event: SubmitEvent) => void;
