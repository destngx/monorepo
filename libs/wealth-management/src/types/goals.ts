export type GoalType =
  | 'SAVINGS_TARGET'
  | 'PURCHASE_GOAL'
  | 'DEBT_PAYOFF'
  | 'INVESTMENT_TARGET'
  | 'INCOME_GOAL'
  | 'NET_WORTH_MILESTONE';

export type AIStatus = 'ON_TRACK' | 'AT_RISK' | 'OFF_TRACK';

export interface Milestone {
  percentage: number;
  label: string;
  achievedAt?: string;
}

export interface Contribution {
  id: string;
  date: string;
  amount: number;
  sourceAccount: string;
  note?: string;
}

export interface Goal {
  id: string;
  name: string;
  type: GoalType;
  emoji: string;
  targetAmount: number;
  currentAmount: number;
  deadline: string;
  status: AIStatus;
  linkedAccountId: string;
  contributionType: 'MANUAL' | 'AUTOMATIC' | 'AI_MANAGED';
  streakCount: number;
  milestones: Milestone[];
  history: Contribution[];
}

export interface GoalProjection {
  currentPace: {
    completionDate: string;
    monthlyContribution: number;
  };
  requiredPace: {
    completionDate: string;
    monthlyContribution: number;
  };
  scenarios: {
    label: string;
    monthlyContribution: number;
    completionDate: string;
  }[];
}
