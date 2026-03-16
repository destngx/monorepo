export interface CategoryTaxonomy {
  id: string;
  name: string;
  icon: string;
  subcategories: string[];
  keywords: string[];
}

export const CATEGORIES: CategoryTaxonomy[] = [
  { id: '1', name: 'Housing', icon: 'Home', subcategories: ['Rent', 'Utilities', 'Internet', 'Maintenance', 'Home Insurance', 'Furnishing'], keywords: ['rent', 'electric', 'water', 'internet'] },
  { id: '2', name: 'Food & Dining', icon: 'Utensils', subcategories: ['Groceries', 'Restaurants', 'Delivery', 'Coffee & Drinks', 'Snacks'], keywords: ['grocery', 'food', 'restaurant', 'coffee'] },
  { id: '3', name: 'Transportation', icon: 'Car', subcategories: ['Fuel/Petrol', 'Grab/Taxi', 'Public Transit', 'Parking', 'Vehicle Maintenance', 'Vehicle Insurance'], keywords: ['grab', 'petrol', 'gas', 'transit', 'parking'] },
  { id: '4', name: 'Healthcare', icon: 'HeartPulse', subcategories: ['Health Insurance', 'Doctor Visits', 'Pharmacy/Medicine', 'Dental', 'Vision', 'Mental Health'], keywords: ['doctor', 'pharmacy', 'hospital', 'dental'] },
  { id: '5', name: 'Personal Care', icon: 'User', subcategories: ['Clothing', 'Grooming/Haircut', 'Gym/Fitness', 'Personal Items'], keywords: ['clothing', 'haircut', 'gym'] },
  { id: '6', name: 'Entertainment', icon: 'Film', subcategories: ['Streaming', 'Games', 'Hobbies', 'Social Events', 'Movies/Concerts'], keywords: ['netflix', 'spotify', 'movie', 'game'] },
  { id: '7', name: 'Education', icon: 'BookOpen', subcategories: ['Online Courses', 'Books', 'Certifications', 'Workshops', 'School/University'], keywords: ['book', 'course', 'tuition'] },
  { id: '8', name: 'Shopping', icon: 'ShoppingBag', subcategories: ['Electronics', 'Household Items', 'Gifts', 'Online Shopping'], keywords: ['shopee', 'lazada', 'amazon', 'gift'] },
  { id: '9', name: 'Bills & Subscriptions', icon: 'Receipt', subcategories: ['Phone Plan', 'SaaS/Software', 'Memberships', 'Recurring Services'], keywords: ['subscription', 'viettel', 'mobifone'] },
  { id: '10', name: 'Financial', icon: 'Landmark', subcategories: ['Bank Fees', 'Crypto Exchange Fees', 'Interest Payments', 'Loan Payments', 'Wire Transfer Fees'], keywords: ['fee', 'interest', 'loan'] },
  { id: '11', name: 'Travel', icon: 'Plane', subcategories: ['Flights', 'Hotels/Accommodation', 'Activities', 'Travel Food', 'Travel Transport'], keywords: ['flight', 'hotel', 'travel'] },
  { id: '12', name: 'Work & Business', icon: 'Briefcase', subcategories: ['Software/Tools', 'Coworking', 'Professional Services', 'Equipment', 'Business Meals'], keywords: ['software', 'co-working', 'business'] },
  { id: '13', name: 'Savings & Investments', icon: 'PiggyBank', subcategories: ['Emergency Fund', 'Retirement', 'Crypto Purchase', 'Stock Purchase', 'Gold'], keywords: ['saving', 'investment', 'crypto', 'stock', 'gold'] },
  { id: '14', name: 'Income', icon: 'ArrowDownLeft', subcategories: ['Salary', 'Freelance', 'Investment Returns', 'Interest', 'Crypto Gains', 'Cashback', 'Gifts Received'], keywords: ['salary', 'income', 'freelance', 'dividend'] },
  { id: '15', name: 'Transfers', icon: 'ArrowRightLeft', subcategories: ['Between Own Accounts', 'To/From Binance', 'Peer-to-Peer'], keywords: ['transfer'] },
  { id: '16', name: 'Other', icon: 'HelpCircle', subcategories: ['Uncategorized', 'Miscellaneous'], keywords: ['misc', 'other'] }
];
