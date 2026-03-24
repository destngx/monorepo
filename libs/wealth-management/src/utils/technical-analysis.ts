/**
 * Technical Analysis Utilities
 */

export function calculateSMA(data: number[], period: number): number | undefined {
  if (data.length < period) return undefined;
  const lastN = data.slice(-period);
  return lastN.reduce((sum, val) => sum + val, 0) / period;
}

export function calculateEMA(data: number[], period: number): number | undefined {
  if (data.length < period) return undefined;
  const k = 2 / (period + 1);
  let ema = data[0];
  for (let i = 1; i < data.length; i++) {
    ema = data[i] * k + ema * (1 - k);
  }
  return ema;
}

export function calculateRSI(data: number[], period = 14): number | undefined {
  if (data.length <= period) return undefined;

  let gains = 0;
  let losses = 0;

  for (let i = 1; i <= period; i++) {
    const change = data[i] - data[i - 1];
    if (change >= 0) gains += change;
    else losses -= change;
  }

  let avgGain = gains / period;
  let avgLoss = losses / period;

  for (let i = period + 1; i < data.length; i++) {
    const change = data[i] - data[i - 1];
    let currentGain = 0;
    let currentLoss = 0;
    if (change >= 0) currentGain = change;
    else currentLoss = -change;

    avgGain = (avgGain * (period - 1) + currentGain) / period;
    avgLoss = (avgLoss * (period - 1) + currentLoss) / period;
  }

  if (avgLoss === 0) return 100;
  const rs = avgGain / avgLoss;
  return 100 - 100 / (1 + rs);
}

export interface SeasonalityStats {
  rank: number;
  name: string;
  label: string;
  return: number;
  winRate: number;
  pf: number; // Profit Factor
  stdDev: number;
  score: number;
  n: number;
}

export function calculateSeasonality(
  closes: number[],
  dates: Date[],
  type: 'day' | 'week' | 'month' = 'day',
): SeasonalityStats[] {
  if (closes.length < 2 || closes.length !== dates.length) return [];

  const stats: Record<number, { returns: number[]; wins: number; total: number }> = {};

  for (let i = 1; i < closes.length; i++) {
    const ret = (closes[i] - closes[i - 1]) / closes[i - 1];
    let key = 0;

    if (type === 'day') {
      key = dates[i].getDay();
    } else if (type === 'week') {
      // Approximate week of month (1-5)
      const firstDayOfMonth = new Date(dates[i].getFullYear(), dates[i].getMonth(), 1);
      key = Math.ceil((dates[i].getDate() + firstDayOfMonth.getDay()) / 7);
    } else {
      // Month (0-11)
      key = dates[i].getMonth();
    }

    if (!stats[key]) {
      stats[key] = { returns: [], wins: 0, total: 0 };
    }

    stats[key].returns.push(ret);
    stats[key].total++;
    if (ret > 0) stats[key].wins++;
  }

  const result: SeasonalityStats[] = Object.keys(stats).map((keyStr) => {
    const key = parseInt(keyStr);
    const s = stats[key];
    const avgReturn = s.returns.reduce((a, b) => a + b, 0) / s.total;
    const winRate = (s.wins / s.total) * 100;

    const gains = s.returns.filter((r) => r > 0).reduce((a, b) => a + b, 0);
    const losses = Math.abs(s.returns.filter((r) => r < 0).reduce((a, b) => a + b, 0));
    const pf = losses === 0 ? gains * 10 : gains / losses;

    const squareDiffs = s.returns.map((r) => Math.pow(r - avgReturn, 2));
    const stdDev = Math.sqrt(squareDiffs.reduce((a, b) => a + b, 0) / s.total) * 100;

    // Composite score
    const score = (avgReturn * 100 * winRate) / (1 + stdDev);

    let name = '';
    let label = '';

    if (type === 'day') {
      const labels = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
      const shorts = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
      name = labels[key];
      label = shorts[key];
    } else if (type === 'week') {
      name = `Week ${key}`;
      label = `W${key}`;
    } else {
      const labels = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
      name = labels[key];
      label = labels[key];
    }

    return {
      rank: 0,
      name,
      label,
      return: avgReturn * 100,
      winRate,
      pf: parseFloat(pf.toFixed(2)),
      stdDev,
      score,
      n: s.total,
    };
  });

  return result.sort((a, b) => b.score - a.score).map((item, idx) => ({ ...item, rank: idx + 1 }));
}
