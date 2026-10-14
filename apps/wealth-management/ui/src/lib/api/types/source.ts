/** Where a market answer came from (domain.DataSource, WM-INV-15). */
export interface DataSource {
  provider: string;
  /** Providers tried before `provider`, as "name: reason". */
  fallback?: string[];
  /** Every provider failed: this is the market data server's last-good copy as of `asOf`. */
  stale?: boolean;
  /** When the data was read from the provider (RFC 3339 UTC). */
  asOf?: string;
}
