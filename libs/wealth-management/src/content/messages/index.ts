export const MESSAGE_CONTENT = {
  success: {
    account: {
      created: 'Account created successfully',
      updated: 'Account updated',
      deleted: 'Account deleted',
      linked: 'Account linked successfully',
      archived: 'Account archived',
      restored: 'Account restored',
      classified: 'Account type updated',
      synced: 'Account synced with latest data',
    },
    transaction: {
      imported: 'Transactions imported successfully',
      categorized: 'Transaction categorized',
      tagged: 'Tags applied to transaction',
      reconciled: 'Account reconciliation completed',
      bulkCategorized: '{{count}} transactions categorized',
    },
    budget: {
      created: 'Budget created successfully',
      updated: 'Budget updated',
      deleted: 'Budget deleted',
      adjusted: 'Budget limits adjusted',
    },
    goal: {
      created: 'Goal created successfully',
      updated: 'Goal updated',
      achieved: '🎉 Goal achieved! Congratulations!',
      deleted: 'Goal deleted',
    },
  },
  error: {
    account: {
      notFound: 'Account not found',
      nameExists: 'An account with this name already exists',
      invalidType: 'Invalid account type selected',
      deleteFailed: 'Could not delete account. Please try again.',
      hasTransactions: 'Cannot delete account with transactions. Archive instead.',
      insufficientPermissions: 'You do not have permission to modify this account',
      syncFailed: 'Failed to sync account data. Check your connection.',
    },
    transaction: {
      importFailed: 'Failed to import transactions',
      invalidFormat: 'Transaction file format is invalid. Check headers and structure.',
      duplicatesDetected: 'Duplicate transactions detected. Review before importing.',
      categorizationFailed: 'Could not categorize transaction. Try again.',
      invalidAmount: 'Transaction amount is invalid',
      missingAccount: 'Please select a valid account for this transaction',
    },
    budget: {
      overLimit: 'Budget limit exceeded for this category',
      createFailed: 'Could not create budget',
      invalidAmount: 'Budget amount must be greater than zero',
      noCategoriesSelected: 'Please select at least one category for your budget',
    },
    goal: {
      createFailed: 'Could not create goal',
      invalidTarget: 'Target amount must be greater than zero',
      invalidDate: 'Target date must be in the future',
    },
    network: {
      offline: 'No internet connection. Some features may be limited.',
      timeout: 'Request timed out. Please try again.',
      serverError: 'Server error. Please try again later.',
      connectionLost: 'Connection lost. Your data is safe.',
    },
    auth: {
      unauthorized: 'You need to be logged in to perform this action',
      sessionExpired: 'Your session has expired. Please log in again.',
      invalidCredentials: 'Invalid credentials provided',
    },
  },
  alert: {
    budget: {
      nearLimit: '⚠️ {{category}} is near the spending limit ({{spent}}/{{limit}})',
      exceeded: '🚨 {{category}} has exceeded the budget limit by {{amount}}',
      onTrack: '✅ {{category}} is on track',
    },
    account: {
      lowBalance: '⚠️ {{account}} balance is low ({{balance}})',
      unusualActivity: '🔍 Unusual activity detected on {{account}}',
      syncPending: '⏳ Pending sync for {{account}}',
    },
    transaction: {
      largeAmount: '📌 Large transaction detected: {{amount}} on {{date}}',
      uncategorized: '📋 {{count}} uncategorized transactions awaiting review',
    },
  },
  notification: {
    account: {
      syncedSuccessfully: '✓ {{count}} transactions synced for {{account}}',
      syncFailed: '✗ Failed to sync {{account}}. Retry available.',
      newTransactions: 'New transactions available for {{account}}',
      balanceUpdate: '{{account}} balance updated to {{newBalance}}',
    },
    budget: {
      reset: 'Budget period for {{period}} has been reset',
      milestone: '{{percentage}}% of {{category}} budget used',
    },
    goal: {
      progress: '{{goalName}}: {{progress}}% complete ({{current}}/{{target}})',
      almostAchieved: '{{goalName}} is almost complete!',
    },
    system: {
      maintenanceScheduled: 'Scheduled maintenance on {{date}} at {{time}}',
      newFeature: 'Check out our new {{feature}} feature!',
    },
  },
  confirmation: {
    account: {
      delete: 'Are you sure you want to delete this account? This action cannot be undone.',
      archive: 'Archive this account? You can restore it anytime.',
    },
    transaction: {
      delete: 'Delete this transaction? This action cannot be undone.',
      bulkDelete: 'Delete {{count}} transactions? This action cannot be undone.',
    },
    budget: {
      delete: 'Delete this budget? Current data will not be affected.',
    },
  },
  validation: {
    account: {
      nameRequired: 'Account name is required',
      nameMinLength: 'Account name must be at least 2 characters',
      nameMaxLength: 'Account name cannot exceed 50 characters',
      initialBalanceInvalid: 'Initial balance must be a valid number',
    },
    transaction: {
      amountRequired: 'Transaction amount is required',
      amountMustBePositive: 'Amount must be greater than 0',
      dateRequired: 'Transaction date is required',
      dateCannotBeFuture: 'Transaction date cannot be in the future',
      descriptionMaxLength: 'Description cannot exceed 200 characters',
    },
    budget: {
      periodRequired: 'Budget period is required',
      amountRequired: 'Budget amount is required',
      categoriesRequired: 'Select at least one category',
    },
  },
  info: {
    firstTime: {
      welcome: "Welcome to Wealth Management! Let's get you started.",
      noAccounts: 'No accounts yet. Create or link your first account to begin.',
      noTransactions: 'No transactions to display. Start by importing or adding transactions.',
      nobudget: 'Create your first budget to start tracking spending.',
    },
    tips: {
      categorization: 'Tip: Categorize transactions regularly for accurate insights.',
      budgetPlanning: 'Tip: Review past spending before setting budget limits.',
      goalSetting: 'Tip: Set realistic goals with specific dates for better tracking.',
      syncing: 'Tip: Enable auto-sync to keep your accounts up to date automatically.',
    },
  },
};
