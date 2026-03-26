import { CardStat } from '@wealth-management/types';

interface CreditCardHistoryProps {
  allCardStats: CardStat[];
}

export function CreditCardHistory({ allCardStats }: CreditCardHistoryProps) {
  return (
    <div className="space-y-4">
      {allCardStats.map((card) => (
        <div key={card.name} className="p-4 border rounded-lg">
          <div className="flex justify-between">
            <div>
              <p className="font-semibold">{card.name}</p>
              <p className="text-sm text-gray-400">{card.bank}</p>
            </div>
            <div className="text-right">
              <p className="text-sm">
                ${card.totalUsage}/${card.limit}
              </p>
              <p className="text-xs text-gray-400">{card.transactionCount} transactions</p>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
