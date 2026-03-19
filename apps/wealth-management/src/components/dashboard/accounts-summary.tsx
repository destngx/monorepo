import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Account } from '@wealth-management/types';
import { ACCOUNT_TYPES } from '@wealth-management/features/accounts/model/types';
import { MaskedBalance } from '@/components/ui/masked-balance';
import { Landmark, Wallet, BarChart3, Archive, Bitcoin, DollarSign } from 'lucide-react';

const getIcon = (type: string) => {
  const accountType = ACCOUNT_TYPES[type as keyof typeof ACCOUNT_TYPES];
  if (!accountType) return <Landmark className="h-4 w-4 text-blue-500" />;

  const iconMap: Record<string, React.ReactNode> = {
    wallet: <Wallet className="h-4 w-4 text-emerald-500" />,
    'piggy-bank': <Landmark className="h-4 w-4 text-blue-500" />,
    'trending-down': <BarChart3 className="h-4 w-4 text-red-500" />,
    'trending-up': <BarChart3 className="h-4 w-4 text-indigo-500" />,
    archive: <Archive className="h-4 w-4 text-gray-400" />,
    bitcoin: <Bitcoin className="h-4 w-4 text-yellow-500" />,
    'dollar-sign': <DollarSign className="h-4 w-4 text-green-500" />,
  };

  const icon = iconMap[accountType.icon];
  return icon || <Wallet className="h-4 w-4 text-emerald-500" />;
};

export function AccountsSummary({ accounts }: { accounts: Account[] }) {
  const sortedAccounts = [...accounts].sort((a, b) => Math.abs(b.balance) - Math.abs(a.balance)).slice(0, 5);

  return (
    <Card className="col-span-1 shadow-sm h-full flex flex-col">
      <CardHeader>
        <CardTitle className="text-lg">Top Accounts</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {sortedAccounts.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">No accounts data found</p>
        ) : (
          sortedAccounts.map((account, idx) => (
            <div key={idx} className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-muted rounded-full">{getIcon(account.type)}</div>
                <div>
                  <p className="text-sm font-medium leading-none">{account.name}</p>
                  <p className="text-xs text-muted-foreground mt-1 capitalize">{account.type}</p>
                </div>
              </div>
              <div className="text-right">
                {account.type === 'negative active use' ? (
                  <>
                    <div className="font-semibold text-sm text-emerald-500">
                      <MaskedBalance
                        amount={
                          account.note && /^\d+/.test(account.note)
                            ? parseFloat(account.note)
                            : Math.abs(account.balance)
                        }
                      />
                    </div>
                    <p className="text-[10px] text-muted-foreground">remaining</p>
                  </>
                ) : (
                  <div className={`font-semibold text-sm ${account.balance < 0 ? 'text-red-500' : ''}`}>
                    <MaskedBalance amount={account.balance} />
                  </div>
                )}
              </div>
            </div>
          ))
        )}
      </CardContent>
    </Card>
  );
}
