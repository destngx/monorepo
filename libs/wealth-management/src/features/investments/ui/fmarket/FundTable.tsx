import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, Skeleton } from '@wealth-management/ui';
import { FmarketFund } from '../../model/types';

interface FundTableProps {
  funds: FmarketFund[];
  loading: boolean;
  onSelect?: (ticker: string) => void;
}

export function FundTable({ funds, loading, onSelect }: FundTableProps) {
  const mapAssetType = (type: string) => {
    switch (type) {
      case 'STOCK_FUND': return 'Quỹ cổ phiếu';
      case 'BOND_FUND': return 'Quỹ trái phiếu';
      case 'BALANCED_FUND': return 'Quỹ cân bằng';
      default: return 'Quỹ mở';
    }
  };

  return (
    <Table>
      <TableHeader className="bg-muted/30">
        <TableRow>
          <TableHead className="w-[200px] text-[11px] uppercase font-bold text-muted-foreground">Tên CCQ/Loại quỹ</TableHead>
          <TableHead className="text-[11px] uppercase font-bold text-muted-foreground">Tổ chức phát hành</TableHead>
          <TableHead className="text-right text-[11px] uppercase font-bold text-muted-foreground">Giá gần nhất</TableHead>
          <TableHead className="text-right text-[11px] uppercase font-bold text-muted-foreground">Lợi nhuận 1 năm gần nhất</TableHead>
          <TableHead className="text-right text-[11px] uppercase font-bold text-muted-foreground">LN bình quân (3 năm)</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {loading ? (
          Array.from({ length: 5 }).map((_, i) => (
            <TableRow key={i}>
              <TableCell><Skeleton className="h-10 w-full" /></TableCell>
              <TableCell><Skeleton className="h-4 w-20" /></TableCell>
              <TableCell><Skeleton className="h-8 w-24 ml-auto" /></TableCell>
              <TableCell><Skeleton className="h-4 w-12 ml-auto" /></TableCell>
              <TableCell><Skeleton className="h-4 w-12 ml-auto" /></TableCell>
            </TableRow>
          ))
        ) : (
          (funds || []).map((fund) => (
            <TableRow
              key={fund.id}
              className="hover:bg-muted/50 transition-colors group cursor-pointer"
              onClick={() => onSelect?.(fund.shortName || fund.code)}
            >
              <TableCell>
                <div className="flex items-center gap-3">
                  <div className="h-8 w-8 rounded-full bg-primary/5 flex items-center justify-center overflow-hidden border border-border/50 group-hover:border-primary/30 transition-colors">
                    {fund.owner?.avatarUrl ? (
                      <img src={fund.owner.avatarUrl} alt={fund.code} className="h-full w-full object-cover shadow-sm" />
                    ) : (
                      <div className="text-[10px] font-bold text-primary">{fund.code?.slice(0, 3)}</div>
                    )}
                  </div>
                  <div>
                    <div className="font-bold text-sm text-primary group-hover:text-primary/80">{fund.code}</div>
                    <div className="text-[9px] text-muted-foreground whitespace-nowrap">{mapAssetType(fund.fundAssetType)}</div>
                  </div>
                </div>
              </TableCell>
              <TableCell>
                <span className="text-[11px] font-semibold text-muted-foreground/80">{fund.owner?.shortName}</span>
              </TableCell>
              <TableCell className="text-right font-mono">
                <div className="font-bold text-sm">{fund.nav?.toLocaleString('vi-VN')}</div>
                <div className="text-[9px] text-muted-foreground leading-none mt-1">
                  Theo NAV tại {fund.productNavChange?.updateAt ? new Date(fund.productNavChange.updateAt).toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit' }) : '--/--'}
                </div>
              </TableCell>
              <TableCell className="text-right">
                <span className="text-emerald-500 font-black text-sm">
                  {fund.productNavChange?.navTo12Months?.toFixed(2)}%
                </span>
              </TableCell>
              <TableCell className="text-right">
                <span className="text-indigo-500 font-black text-sm">
                  {fund.productNavChange?.annualizedReturn36Months?.toFixed(2)}%
                </span>
              </TableCell>
            </TableRow>
          ))
        )}
      </TableBody>
    </Table>
  );
}
