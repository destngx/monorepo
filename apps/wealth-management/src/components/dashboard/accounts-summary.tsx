import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Account } from "@wealth-management/types";
import { MaskedBalance } from "@/components/ui/masked-balance";
import { Landmark, Wallet, BarChart3, Archive, CreditCard } from "lucide-react";

const getIcon = (type: string) => {
  switch (type.toLowerCase()) {
    case 'active use': return <Wallet className="h-4 w-4 text-emerald-500" />;
    case 'long holding': return <BarChart3 className="h-4 w-4 text-indigo-500" />;
    case 'deprecated': return <Archive className="h-4 w-4 text-gray-400" />;
    case 'negative active use': return <CreditCard className="h-4 w-4 text-red-500" />;
    case 'rarely use':
    default: return <Landmark className="h-4 w-4 text-blue-500" />;
  }
};

export function AccountsSummary({ accounts }: { accounts: Account[] }) {
  // Sort by absolute balance descending, take top 5
  const sortedAccounts = [...accounts]
    .sort((a, b) => Math.abs(b.balance) - Math.abs(a.balance))
    .slice(0, 5);

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
                <div className="p-2 bg-muted rounded-full">
                  {getIcon(account.type)}
                </div>
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
                        amount={account.note && /^\d+/.test(account.note)
                          ? parseFloat(account.note)
                          : Math.abs(account.balance)} 
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

