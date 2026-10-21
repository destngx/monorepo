import type { VisitInput } from './api/types';
import { todayISO } from './format';

/** An empty visit for a person, dated today. */
export function blankVisit(personId: string): VisitInput {
  return { personId, date: todayISO(), type: 'checkup', title: '', note: '', providerName: '', providerKind: '' };
}
