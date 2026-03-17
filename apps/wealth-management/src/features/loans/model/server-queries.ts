/**
 * Loans Feature - Server-side Query Layer
 * These queries run on the server and can access sheet data directly
 */

import { getLoans as getLoansFromSheets } from "@wealth-management/services/server";
import { Loan } from './types';

export async function getLoans(forceFresh = false): Promise<Loan[]> {
  return getLoansFromSheets(forceFresh);
}
