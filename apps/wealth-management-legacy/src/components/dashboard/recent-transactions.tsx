import { Card, CardContent, CardHeader, CardTitle } from '@wealth-management/ui/card';
import { Transaction } from '@wealth-management/types';
import { MaskedBalance } from '@wealth-management/ui/masked-balance';
import { formatDate } from '@wealth-management/utils';
import { ArrowUpRight, ArrowDownLeft } from 'lucide-react';

export function RecentTransactions({ transactions }: { transactions: Transaction[] }) {
  // Sort transactions by date descending
  const recent = [...transactions].sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime()).slice(0, 10);

  return (
    <Card className="col-span-1 shadow-sm">
      <CardHeader>
        <CardTitle className="text-lg">Recent Transactions</CardTitle>
      </CardHeader>
      <CardContent>
        {recent.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">No recent transactions</p>
        ) : (
          <div className="space-y-4">
            {recent.map((txn, idx) => {
              const isPayment = txn.payment !== null;
              return (
                <div key={idx} className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div
                      className={`p-2 rounded-full ${isPayment ? 'bg-orange-100 text-orange-600' : 'bg-green-100 text-green-600'}`}
                    >
                      {isPayment ? <ArrowUpRight className="h-4 w-4" /> : <ArrowDownLeft className="h-4 w-4" />}
                    </div>
                    <div>
                      <p className="text-sm font-medium">{txn.payee || 'Unknown Payee'}</p>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        {txn.category} • {formatDate(txn.date)}
                      </p>
                    </div>
                  </div>
                  <div className={`font-semibold text-sm ${isPayment ? '' : 'text-green-600'}`}>
                    {isPayment ? (
                      <>
                        <span className="mr-0.5">-</span>
                        <MaskedBalance amount={txn.payment ?? 0} />
                      </>
                    ) : (
                      <>
                        <span className="mr-0.5">+</span>
                        <MaskedBalance amount={txn.deposit ?? 0} />
                      </>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
