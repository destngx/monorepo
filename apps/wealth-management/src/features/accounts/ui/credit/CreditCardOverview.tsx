import { BankSummary } from '@wealth-management/types';

interface CreditCardOverviewProps {
  banks: BankSummary[];
}

export function CreditCardOverview({ banks }: CreditCardOverviewProps) {
  return (
    <div className="space-y-6">
      {banks.map((bank) => (
        <div key={bank.name} className="p-4 border rounded-lg">
          <h3 className="text-lg font-semibold">{bank.name}</h3>
          <p className="text-sm text-gray-400">{bank.accounts.length} accounts</p>
        </div>
      ))}
    </div>
  );
}
