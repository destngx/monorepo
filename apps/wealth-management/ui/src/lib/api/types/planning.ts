export interface Goal {
  id: string;
  name: string;
  type: string;
  emoji: string;
  targetAmount: number;
  currentAmount: number;
  deadline: string;
  status: 'ON_TRACK' | 'AT_RISK' | 'OFF_TRACK';
}

export interface Loan {
  name: string;
  monthlyDebt: number;
  monthlyPaid: number;
  monthlyRemaining: number;
  extra: string;
  yearlyDebt: number;
  yearlyPaid: number;
  yearlyRemaining: number;
}
