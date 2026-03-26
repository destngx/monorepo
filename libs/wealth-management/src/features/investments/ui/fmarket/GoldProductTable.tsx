import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';
import { TrendingUp } from 'lucide-react';

interface GoldProductTableProps {
  products: any[];
  loading: boolean;
}

export function GoldProductTable({ products, loading }: GoldProductTableProps) {
  return (
    <Table>
      <TableHeader className="bg-muted/30">
        <TableRow>
          <TableHead className="w-[300px] text-[11px] uppercase font-bold text-muted-foreground">Loại vàng</TableHead>
          <TableHead className="text-[11px] uppercase font-bold text-muted-foreground">Công ty</TableHead>
          <TableHead className="text-right text-[11px] uppercase font-bold text-muted-foreground">Mua vào (VND/chỉ)</TableHead>
          <TableHead className="text-right text-[11px] uppercase font-bold text-muted-foreground">Bán ra (VND/chỉ)</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {loading ? (
          Array.from({ length: 5 }).map((_, i) => (
            <TableRow key={i}>
              <TableCell><Skeleton className="h-10 w-full" /></TableCell>
              <TableCell><Skeleton className="h-4 w-20" /></TableCell>
              <TableCell><Skeleton className="h-4 w-24 ml-auto" /></TableCell>
              <TableCell><Skeleton className="h-4 w-24 ml-auto" /></TableCell>
            </TableRow>
          ))
        ) : (
          (products || []).map((product) => {
            const isPaused = product.status !== 'PRODUCT_ACTIVE' || 
                            product.productGold?.isDisable === true || 
                            !product.productGold?.buyPrice;
            return (
              <TableRow key={product.id} className="hover:bg-muted/50 transition-colors group">
                <TableCell>
                  <div className="flex items-center gap-3">
                    <div className="h-8 w-8 rounded bg-amber-100 flex items-center justify-center overflow-hidden border border-amber-200">
                      {product.owner?.avatarUrl ? (
                        <img src={product.owner.avatarUrl} alt={product.shortName} className="h-full w-full object-cover" />
                      ) : (
                        <TrendingUp className="h-4 w-4 text-amber-600" />
                      )}
                    </div>
                    <div>
                      <div className="font-bold text-sm text-amber-900">{product.name}</div>
                      <div className="text-[10px] text-muted-foreground">{product.shortName}</div>
                    </div>
                  </div>
                </TableCell>
                <TableCell>
                  <span className="text-[11px] font-bold text-muted-foreground">{product.owner?.shortName}</span>
                </TableCell>
                <TableCell className="text-right font-mono">
                  {isPaused ? (
                    <span className="text-muted-foreground/50">-</span>
                  ) : (
                    <span className="font-bold text-sm text-emerald-600">
                      {product.productGold?.buyPrice?.toLocaleString('vi-VN')}
                    </span>
                  )}
                </TableCell>
                <TableCell className="text-right font-mono">
                  {isPaused ? (
                    <span className="text-[10px] font-bold text-red-500 bg-red-50 px-2 py-0.5 rounded">Tạm ngừng bán</span>
                  ) : (
                    <span className="font-bold text-sm text-red-600">
                      {product.productGold?.sellPrice?.toLocaleString('vi-VN')}
                    </span>
                  )}
                </TableCell>
              </TableRow>
            );
          })
        )}
      </TableBody>
    </Table>
  );
}
