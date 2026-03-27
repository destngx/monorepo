'use server';

import { getSheetsClient } from './auth';
import { AppError } from '../../utils/errors';
import { env } from '@wealth-management/config';

const SHEET_ID = env.sheets.id;

export async function readSheet(range: string): Promise<string[][]> {
  if (!SHEET_ID) throw new Error('GOOGLE_SHEETS_ID is not configured');

  try {
    const sheets = await getSheetsClient();
    const response = await sheets.spreadsheets.values.get({
      spreadsheetId: SHEET_ID,
      range,
      valueRenderOption: 'UNFORMATTED_VALUE',
    });

    // Cast all values to strings for uniform parsing downstream
    const rows = response.data.values || [];
    return rows.map((row) => row.map((cell) => (cell === null || cell === undefined ? '' : String(cell))));
  } catch (error: any) {
    console.error(`Sheets Read error [${range}]:`, error);
    const message = error.message || 'Unknown Sheets error';
    const isInvalidGrant = message.includes('invalid_grant');

    throw new AppError(`Failed to read from Google Sheets: ${message}`, undefined, error.code || 500, undefined, {
      userMessage: isInvalidGrant
        ? 'Google access session has expired. Please sign in again.'
        : 'Failed to access Google Sheets. Please check your configuration.',
      code: isInvalidGrant ? 'invalid_grant' : undefined,
    });
  }
}

export async function appendRow(range: string, values: any[]): Promise<boolean> {
  if (!SHEET_ID) throw new Error('GOOGLE_SHEETS_ID is not configured');

  try {
    const sheets = await getSheetsClient();
    await sheets.spreadsheets.values.append({
      spreadsheetId: SHEET_ID,
      range,
      valueInputOption: 'USER_ENTERED',
      requestBody: {
        values: [values],
      },
    });

    return true;
  } catch (error: any) {
    console.error(`Sheets Append error [${range}]:`, error);
    throw new AppError(`Failed to write to Google Sheets: ${error.message}`, undefined, error.code || 500, undefined, {
      userMessage: error.message?.includes('invalid_grant')
        ? 'Google access session has expired. Please sign in again.'
        : 'Failed to write data to Google Sheets.',
    });
  }
}

/**
 * Writes a row to the first empty row in column A of the given sheet.
 * This is safer than appendRow when the sheet has pre-defined formula rows
 * (e.g. running balance formulas) that extend beyond the actual data — the
 * standard Sheets append API would land the new row after those formulas.
 *
 * @param sheetName  The sheet tab name, e.g. 'Transactions'
 * @param columnARange  The full column-A range to scan, e.g. 'Transactions!A2:A'
 * @param values  Row values to write (columns A, B, C …)
 */
export async function writeToFirstEmptyRow(sheetName: string, columnARange: string, values: any[]): Promise<boolean> {
  if (!SHEET_ID) throw new Error('GOOGLE_SHEETS_ID is not configured');

  try {
    const sheets = await getSheetsClient();

    // Read only column A to find the last row that has an account name
    const readRes = await sheets.spreadsheets.values.get({
      spreadsheetId: SHEET_ID,
      range: columnARange,
      valueRenderOption: 'UNFORMATTED_VALUE',
    });

    const colA = readRes.data.values || [];
    // Find the last row index (0-based) that has a non-empty value in col A
    let lastDataIdx = -1;
    for (let i = 0; i < colA.length; i++) {
      if (colA[i] && colA[i][0] !== '' && colA[i][0] !== null && colA[i][0] !== undefined) {
        lastDataIdx = i;
      }
    }

    // columnARange starts at row 2 (index 0 = sheet row 2), so:
    const targetRow = 2 + lastDataIdx + 1; // next row after last data row

    const targetRange = `${sheetName}!A${targetRow}`;

    await sheets.spreadsheets.values.update({
      spreadsheetId: SHEET_ID,
      range: targetRange,
      valueInputOption: 'USER_ENTERED',
      requestBody: {
        values: [values],
      },
    });

    return true;
  } catch (error: any) {
    console.error(`Sheets Update (First Empty) error [${columnARange}]:`, error);
    throw new AppError(`Failed to update Google Sheets: ${error.message}`, undefined, error.code || 500, undefined, {
      userMessage: error.message?.includes('invalid_grant')
        ? 'Google access session has expired. Please sign in again.'
        : 'Failed to find/update empty row in Google Sheets.',
    });
  }
}

export async function updateRow(range: string, values: any[]): Promise<boolean> {
  if (!SHEET_ID) throw new Error('GOOGLE_SHEETS_ID is not configured');

  try {
    const sheets = await getSheetsClient();
    await sheets.spreadsheets.values.update({
      spreadsheetId: SHEET_ID,
      range,
      valueInputOption: 'USER_ENTERED',
      requestBody: {
        values: [values],
      },
    });

    return true;
  } catch (error: any) {
    console.error(`Sheets Update error [${range}]:`, error);
    throw new AppError(`Failed to update Google Sheets: ${error.message}`, undefined, error.code || 500, undefined, {
      userMessage: error.message?.includes('invalid_grant')
        ? 'Google access session has expired. Please sign in again.'
        : 'Failed to update data in Google Sheets.',
    });
  }
}
