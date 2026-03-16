export interface Category {
  id: string;
  name: string;
  icon: string;
  subcategories: Subcategory[];
  keywords: string[];
}

export interface Subcategory {
  id: string;
  name: string;
}

export interface EmailNotification {
  id: string;
  timestamp: string;
  from: string;
  subject: string;
  content: string;
  status: 'pending' | 'done';
  rowNumber: number; // To facilitate updates back to the sheet
}
