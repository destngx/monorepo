export const TASK_CONTENT = {
  account: {
    creation: {
      title: 'Create New Account',
      description: 'Add a new financial account to your portfolio',
      instructions: [
        'Provide account name (e.g., "Primary Checking")',
        'Select account type from the list',
        'Enter initial balance if applicable',
        'Optionally set a goal amount if this is a savings account',
        'Review and save',
      ],
      contextTips: [
        'Account names should be descriptive and unique within your portfolio',
        'Active Use accounts are for regular daily transactions',
        'Long Holding accounts are for assets held for extended periods',
        'Set the initial balance to match your current balance for accurate tracking',
      ],
    },
    linking: {
      title: 'Link Existing Account',
      description: 'Connect an existing financial account to track',
      instructions: [
        'Search for your bank or financial institution',
        'Authenticate with your credentials',
        'Select which accounts to import',
        'Review account mapping',
        'Confirm and sync transactions',
      ],
      contextTips: [
        'Linking requires secure authentication with your financial institution',
        'Your credentials are encrypted and never stored directly',
        'Transaction sync happens automatically after linking',
        'You can link multiple accounts from the same institution',
      ],
      prerequisites: ['Valid banking credentials', 'Internet connection'],
    },
    classification: {
      title: 'Classify Your Accounts',
      description: 'Organize accounts by type for better analytics',
      instructions: [
        'Review each account in your portfolio',
        'Assign the most appropriate account type',
        'Consider: frequency of use, holding period, purpose',
        'Verify crypto/bank/investment classifications',
        'Save classifications',
      ],
      contextTips: [
        'Account type affects budget calculations and reporting',
        'Crypto accounts are excluded from traditional budget analysis',
        'Bank accounts typically include checking and savings',
        'Investment accounts track long-term holdings separately',
      ],
      relatedContent: ['Account Types Reference', 'Portfolio Classification Guide'],
    },
    archival: {
      title: 'Archive Inactive Accounts',
      description: 'Move unused accounts to archive for cleaner dashboard',
      instructions: [
        'Review accounts with no activity in the last 90 days',
        'Check if account should truly be archived',
        'Archived accounts can be restored anytime',
        'Confirm archival',
      ],
      contextTips: [
        'Archival does not delete data - it just hides inactive accounts',
        'Historical transactions remain in reports',
        'You can restore archived accounts at any time',
      ],
    },
  },
  transaction: {
    import: {
      title: 'Import Transactions',
      description: 'Add historical transaction data',
      instructions: [
        'Export your transaction data as CSV from your bank',
        'Select the account to import into',
        'Upload the CSV file',
        'Map columns to transaction fields',
        'Review for duplicates',
        'Confirm import',
      ],
      supportedFormats: ['CSV', 'OFX', 'Excel'],
      contextTips: [
        'Import preserves original transaction dates and amounts',
        'Large imports may take a few moments to process',
        'Duplicate detection prevents double-counting transactions',
      ],
    },
    categorization: {
      title: 'Categorize Transactions',
      description: 'Organize transactions into categories for reporting',
      instructions: [
        'Review uncategorized transactions',
        'Select a category from the list',
        'Use auto-suggestions when available',
        'Add tags for advanced filtering',
        'Save categorization',
      ],
      contextTips: [
        'Categories help track spending patterns by type',
        'Tags enable custom filtering beyond predefined categories',
        'AI suggestions improve accuracy with each manual categorization',
        'Bulk categorization available for similar transactions',
      ],
      relatedContent: ['Transaction Categories', 'Smart Tagging Guide'],
    },
    reconciliation: {
      title: 'Reconcile Account',
      description: 'Verify imported and manual transactions match bank records',
      instructions: [
        'Gather your bank statement for the period',
        'Compare beginning balance',
        'Mark transactions as cleared or pending',
        'Identify any discrepancies',
        'Adjust or add missing transactions',
        'Verify ending balance matches',
        'Complete reconciliation',
      ],
      contextTips: [
        'Regular reconciliation catches data entry errors early',
        'Pending transactions will clear automatically when they post',
        'Bank fees and interest are usually final transactions to reconcile',
      ],
    },
  },
  budget: {
    creation: {
      title: 'Create Budget',
      description: 'Set spending limits for budget categories',
      instructions: [
        'Choose budget period (monthly, quarterly, annually)',
        'Select budget categories',
        'Set spending limit for each category',
        'Review total vs. income',
        'Enable notifications for overspend alerts',
        'Save budget',
      ],
      contextTips: [
        'Monthly budgets are most effective for tracking habits',
        'Budget limits should be realistic based on past spending',
        'Categories with no spending can be excluded',
        'Alerts help prevent overspending before it happens',
      ],
    },
    adjustment: {
      title: 'Adjust Budget Limits',
      description: 'Modify spending limits mid-period',
      instructions: [
        'Review current budget progress',
        'Identify categories needing adjustment',
        'Update spending limits',
        'Consider seasonal variations',
        'Save changes',
      ],
      contextTips: [
        'Mid-period adjustments help account for unexpected expenses',
        'Track reasons for large adjustments for future planning',
        'Some categories may need permanent vs. temporary changes',
      ],
    },
  },
  goalSetting: {
    creation: {
      title: 'Set Financial Goal',
      description: 'Define savings or investment goals',
      instructions: [
        'Choose goal type (savings, investment, debt payoff)',
        'Set target amount',
        'Set target date',
        'Link to specific accounts',
        'Enable progress tracking',
        'Save goal',
      ],
      contextTips: [
        'SMART goals (Specific, Measurable, Achievable, Relevant, Time-bound) work best',
        'Goals help visualize progress and maintain motivation',
        'Multiple goals can be tracked simultaneously',
      ],
    },
  },
};
