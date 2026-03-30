export type TransactionCategory =
  | 'groceries'
  | 'restaurants'
  | 'fuel'
  | 'parking'
  | 'insurance'
  | 'utilities'
  | 'rent'
  | 'mortgage'
  | 'phone'
  | 'internet'
  | 'subscriptions'
  | 'gym'
  | 'medical'
  | 'pharmacy'
  | 'entertainment'
  | 'travel'
  | 'shopping'
  | 'gifts'
  | 'charity'
  | 'education'
  | 'books'
  | 'salary'
  | 'bonus'
  | 'investment income'
  | 'refund'
  | 'transfer'
  | 'fee'
  | 'interest'
  | 'tax'
  | 'other';

export interface TransactionCategoryMetadata {
  id: TransactionCategory;
  label: string;
  icon: string;
  color: string;
  type: 'income' | 'expense' | 'non-budget';
  description: string;
}

export interface CategoryTaggingRule {
  category: TransactionCategory;
  keywords: string[];
  merchants?: string[];
  autoTag?: boolean;
}

export const TRANSACTION_CATEGORIES: Record<TransactionCategory, TransactionCategoryMetadata> = {
  groceries: {
    id: 'groceries',
    label: 'Groceries',
    icon: 'shopping-cart',
    color: 'bg-green-500',
    type: 'expense',
    description: 'Food and grocery store purchases',
  },
  restaurants: {
    id: 'restaurants',
    label: 'Restaurants & Dining',
    icon: 'utensils',
    color: 'bg-orange-500',
    type: 'expense',
    description: 'Dining out, takeout, and food delivery',
  },
  fuel: {
    id: 'fuel',
    label: 'Fuel',
    icon: 'fuel',
    color: 'bg-yellow-600',
    type: 'expense',
    description: 'Gasoline and fuel purchases',
  },
  parking: {
    id: 'parking',
    label: 'Parking',
    icon: 'square',
    color: 'bg-slate-500',
    type: 'expense',
    description: 'Parking fees and tolls',
  },
  insurance: {
    id: 'insurance',
    label: 'Insurance',
    icon: 'shield',
    color: 'bg-blue-600',
    type: 'expense',
    description: 'Insurance premiums (health, auto, home)',
  },
  utilities: {
    id: 'utilities',
    label: 'Utilities',
    icon: 'lightbulb',
    color: 'bg-amber-500',
    type: 'expense',
    description: 'Electricity, water, gas, waste',
  },
  rent: {
    id: 'rent',
    label: 'Rent',
    icon: 'home',
    color: 'bg-red-600',
    type: 'expense',
    description: 'Rental payments',
  },
  mortgage: {
    id: 'mortgage',
    label: 'Mortgage',
    icon: 'building',
    color: 'bg-red-700',
    type: 'expense',
    description: 'Mortgage payments',
  },
  phone: {
    id: 'phone',
    label: 'Phone',
    icon: 'phone',
    color: 'bg-indigo-500',
    type: 'expense',
    description: 'Mobile phone service',
  },
  internet: {
    id: 'internet',
    label: 'Internet',
    icon: 'wifi',
    color: 'bg-cyan-500',
    type: 'expense',
    description: 'Internet service provider',
  },
  subscriptions: {
    id: 'subscriptions',
    label: 'Subscriptions',
    icon: 'repeat',
    color: 'bg-purple-500',
    type: 'expense',
    description: 'Recurring subscription services',
  },
  gym: {
    id: 'gym',
    label: 'Gym & Fitness',
    icon: 'dumbbell',
    color: 'bg-pink-500',
    type: 'expense',
    description: 'Gym membership and fitness classes',
  },
  medical: {
    id: 'medical',
    label: 'Medical',
    icon: 'heart',
    color: 'bg-rose-500',
    type: 'expense',
    description: 'Doctor visits and medical services',
  },
  pharmacy: {
    id: 'pharmacy',
    label: 'Pharmacy',
    icon: 'pill',
    color: 'bg-teal-500',
    type: 'expense',
    description: 'Medications and pharmaceutical purchases',
  },
  entertainment: {
    id: 'entertainment',
    label: 'Entertainment',
    icon: 'film',
    color: 'bg-fuchsia-500',
    type: 'expense',
    description: 'Movies, concerts, games, hobbies',
  },
  travel: {
    id: 'travel',
    label: 'Travel',
    icon: 'plane',
    color: 'bg-sky-500',
    type: 'expense',
    description: 'Flights, hotels, vacation expenses',
  },
  shopping: {
    id: 'shopping',
    label: 'Shopping',
    icon: 'shopping-bag',
    color: 'bg-pink-600',
    type: 'expense',
    description: 'Retail and clothing purchases',
  },
  gifts: {
    id: 'gifts',
    label: 'Gifts',
    icon: 'gift',
    color: 'bg-red-500',
    type: 'expense',
    description: 'Gifts and charitable giving',
  },
  charity: {
    id: 'charity',
    label: 'Charity',
    icon: 'heart-handshake',
    color: 'bg-lime-500',
    type: 'expense',
    description: 'Charitable donations and contributions',
  },
  education: {
    id: 'education',
    label: 'Education',
    icon: 'book-open',
    color: 'bg-indigo-600',
    type: 'expense',
    description: 'Tuition, courses, and educational materials',
  },
  books: {
    id: 'books',
    label: 'Books',
    icon: 'book',
    color: 'bg-amber-700',
    type: 'expense',
    description: 'Book purchases',
  },
  salary: {
    id: 'salary',
    label: 'Salary',
    icon: 'briefcase',
    color: 'bg-emerald-600',
    type: 'income',
    description: 'Salary and regular employment income',
  },
  bonus: {
    id: 'bonus',
    label: 'Bonus',
    icon: 'star',
    color: 'bg-yellow-500',
    type: 'income',
    description: 'Bonuses and incentive payments',
  },
  'investment income': {
    id: 'investment income',
    label: 'Investment Income',
    icon: 'trending-up',
    color: 'bg-green-600',
    type: 'income',
    description: 'Dividends, interest, and investment returns',
  },
  refund: {
    id: 'refund',
    label: 'Refund',
    icon: 'arrow-left',
    color: 'bg-blue-500',
    type: 'income',
    description: 'Refunds and reversals',
  },
  transfer: {
    id: 'transfer',
    label: 'Transfer',
    icon: 'arrow-right-left',
    color: 'bg-gray-500',
    type: 'non-budget',
    description: 'Transfers between accounts',
  },
  fee: {
    id: 'fee',
    label: 'Fee',
    icon: 'zap',
    color: 'bg-red-600',
    type: 'expense',
    description: 'Bank fees and service charges',
  },
  interest: {
    id: 'interest',
    label: 'Interest',
    icon: 'percent',
    color: 'bg-green-400',
    type: 'income',
    description: 'Interest earned on deposits',
  },
  tax: {
    id: 'tax',
    label: 'Tax',
    icon: 'receipt',
    color: 'bg-orange-700',
    type: 'expense',
    description: 'Tax payments and withholding',
  },
  other: {
    id: 'other',
    label: 'Other',
    icon: 'help-circle',
    color: 'bg-gray-400',
    type: 'non-budget',
    description: 'Uncategorized transactions',
  },
};

/**
 * Auto-tagging rules for transaction categorization
 * These patterns help automatically suggest or categorize transactions based on keywords
 */
export const CATEGORY_TAGGING_RULES: CategoryTaggingRule[] = [
  {
    category: 'groceries',
    keywords: [
      'grocery',
      'supermarket',
      'whole foods',
      'trader joe',
      'safeway',
      'kroger',
      'walmart',
      'costco',
      'market',
    ],
    merchants: ['WHOLE FOODS', 'TRADER JOE', 'SAFEWAY', 'KROGER', 'COSTCO'],
    autoTag: true,
  },
  {
    category: 'restaurants',
    keywords: ['restaurant', 'dining', 'cafe', 'coffee', 'pizza', 'burger', 'taco', 'sushi', 'ramen'],
    merchants: ['DOORDASH', 'UBER EATS', 'GRUBHUB', 'FOODPANDA'],
    autoTag: true,
  },
  {
    category: 'fuel',
    keywords: ['gas', 'fuel', 'chevron', 'shell', 'chevron', 'bp', 'exxon', 'mobil'],
    merchants: ['CHEVRON', 'SHELL', 'BP', 'EXXON', 'MOBIL'],
    autoTag: true,
  },
  {
    category: 'parking',
    keywords: ['parking', 'valet', 'toll', 'meter'],
    merchants: ['PARKWHIZ', 'SPOTHERO', 'PARKINGPANDA'],
    autoTag: true,
  },
  {
    category: 'insurance',
    keywords: ['insurance', 'premium', 'geico', 'state farm', 'allstate'],
    merchants: ['GEICO', 'STATE FARM', 'ALLSTATE'],
    autoTag: false,
  },
  {
    category: 'utilities',
    keywords: ['electricity', 'water', 'gas', 'utility', 'pg&e', 'consolidated edison'],
    merchants: ['PG&E', 'CON EDISON', 'DWP'],
    autoTag: true,
  },
  {
    category: 'rent',
    keywords: ['rent', 'rental payment', 'lease'],
    merchants: [],
    autoTag: false,
  },
  {
    category: 'mortgage',
    keywords: ['mortgage', 'home loan', 'principal', 'interest payment'],
    merchants: [],
    autoTag: false,
  },
  {
    category: 'phone',
    keywords: ['phone', 'mobile', 'verizon', 'at&t', 't-mobile', 'sprint'],
    merchants: ['VERIZON', 'AT&T', 'T-MOBILE', 'SPRINT'],
    autoTag: true,
  },
  {
    category: 'internet',
    keywords: ['internet', 'isp', 'broadband', 'comcast', 'verizon fios', 'spectrum'],
    merchants: ['COMCAST', 'VERIZON FIOS', 'SPECTRUM'],
    autoTag: true,
  },
  {
    category: 'subscriptions',
    keywords: ['subscription', 'annual', 'monthly', 'netflix', 'spotify', 'adobe', 'microsoft'],
    merchants: ['NETFLIX', 'SPOTIFY', 'ADOBE', 'MICROSOFT'],
    autoTag: true,
  },
  {
    category: 'gym',
    keywords: ['gym', 'fitness', 'yoga', 'peloton', 'lululemon'],
    merchants: ['PELOTON', 'LULULEMON', 'EQUINOX'],
    autoTag: true,
  },
  {
    category: 'medical',
    keywords: ['doctor', 'hospital', 'medical', 'clinic', 'surgeon'],
    merchants: ['KAISER', 'CEDARS SINAI', 'MOUNT SINAI'],
    autoTag: false,
  },
  {
    category: 'pharmacy',
    keywords: ['pharmacy', 'cvs', 'walgreens', 'medicine', 'prescripti'],
    merchants: ['CVS', 'WALGREENS', 'RITE AID'],
    autoTag: true,
  },
  {
    category: 'entertainment',
    keywords: ['movie', 'theater', 'concert', 'ticket', 'game', 'steam', 'playstation'],
    merchants: ['TICKETMASTER', 'REGAL CINEMA', 'STEAM'],
    autoTag: true,
  },
  {
    category: 'travel',
    keywords: ['airline', 'hotel', 'flight', 'booking', 'expedia', 'airbnb'],
    merchants: ['UNITED AIRLINES', 'MARRIOTT', 'AIRBNB', 'BOOKING.COM'],
    autoTag: true,
  },
  {
    category: 'shopping',
    keywords: ['amazon', 'ebay', 'mall', 'retail', 'clothing', 'apparel', 'fashion'],
    merchants: ['AMAZON', 'EBAY', 'TARGET', 'MACY'],
    autoTag: true,
  },
  {
    category: 'gifts',
    keywords: ['gift', 'present', 'birthday', 'anniversary'],
    merchants: [],
    autoTag: false,
  },
  {
    category: 'charity',
    keywords: ['donation', 'charity', 'nonprofit', 'relief', 'foundation'],
    merchants: [],
    autoTag: false,
  },
  {
    category: 'education',
    keywords: ['tuition', 'course', 'university', 'college', 'school', 'udemy', 'coursera'],
    merchants: ['UDEMY', 'COURSERA', 'MASTERCLASS'],
    autoTag: true,
  },
  {
    category: 'books',
    keywords: ['book', 'kindle', 'audible'],
    merchants: ['AMAZON BOOKS', 'BARNES NOBLE', 'AUDIBLE'],
    autoTag: true,
  },
  {
    category: 'salary',
    keywords: ['salary', 'payroll', 'wage', 'direct deposit'],
    merchants: [],
    autoTag: false,
  },
  {
    category: 'bonus',
    keywords: ['bonus', 'incentive', 'commission'],
    merchants: [],
    autoTag: false,
  },
  {
    category: 'investment income',
    keywords: ['dividend', 'investment', 'interest income', 'capital gains'],
    merchants: [],
    autoTag: false,
  },
  {
    category: 'refund',
    keywords: ['refund', 'credit', 'reversal', 'return'],
    merchants: [],
    autoTag: true,
  },
  {
    category: 'transfer',
    keywords: ['transfer', 'xfer', 'payment'],
    merchants: [],
    autoTag: false,
  },
  {
    category: 'fee',
    keywords: ['fee', 'charge', 'overdraft', 'atm'],
    merchants: [],
    autoTag: true,
  },
  {
    category: 'interest',
    keywords: ['interest earned', 'interest paid', 'yield'],
    merchants: [],
    autoTag: true,
  },
  {
    category: 'tax',
    keywords: ['tax', 'irs', 'withholding', 'estimated tax'],
    merchants: [],
    autoTag: false,
  },
];
