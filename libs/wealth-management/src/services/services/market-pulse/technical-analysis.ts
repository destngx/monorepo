import { calculateEMA, calculateSMA, calculateRSI, calculateSeasonality } from '@wealth-management/utils';
import { MarketAsset, Technicals } from './types';

export function computeReturns(closes: number[]): number[] {
  const returns = [];
  for (let i = 1; i < closes.length; i++) {
    returns.push((closes[i] - closes[i - 1]) / closes[i - 1]);
  }
  return returns;
}

export function calculatePearson(x: number[], y: number[]): number {
  if (x.length === 0 || y.length === 0) return 0;
  const len = Math.min(x.length, y.length);
  const x1 = x.slice(-len);
  const y1 = y.slice(-len);

  const xMean = x1.reduce((a, b) => a + b, 0) / len;
  const yMean = y1.reduce((a, b) => a + b, 0) / len;

  let num = 0;
  let den1 = 0;
  let den2 = 0;

  for (let i = 0; i < len; i++) {
    const xDiff = x1[i] - xMean;
    const yDiff = y1[i] - yMean;
    num += xDiff * yDiff;
    den1 += xDiff * xDiff;
    den2 += yDiff * yDiff;
  }

  if (den1 === 0 || den2 === 0) return 1;
  return num / Math.sqrt(den1 * den2);
}

export function generateRealCorrelationMatrix(assets: MarketAsset[]): number[][] {
  const matrix: number[][] = [];
  const returnsArray = assets.map((a) => computeReturns(a.closes || []));

  for (let i = 0; i < assets.length; i++) {
    const row: number[] = [];
    for (let j = 0; j < assets.length; j++) {
      if (i === j) {
        row.push(1);
      } else {
        const val = calculatePearson(returnsArray[i], returnsArray[j]);
        // Handle NaN/Infinity
        row.push(Number.isNaN(val) ? 0 : Number(val.toFixed(3)));
      }
    }
    matrix.push(row);
  }
  return matrix;
}

export function generateTechnicals(assets: MarketAsset[], market: 'US' | 'VN'): Technicals {
  const primaryAsset =
    assets.find((a) => a.name === 'S&P500' || a.name === 'VN-Index' || a.name === 'VN30') || assets[0];
  const closes = primaryAsset?.closes || [];
  const avgChange = assets.reduce((sum, a) => sum + a.percentChange, 0) / (assets.length || 1);

  // Indicators for the primary asset
  const rsi = calculateRSI(closes);
  const ema20 = calculateEMA(closes, 20);
  const ema50 = calculateEMA(closes, 50);
  const ema200 = calculateEMA(closes, 200);
  const sma20 = calculateSMA(closes, 20);
  const sma50 = calculateSMA(closes, 50);

  // Real ATR Calculation
  const highs = primaryAsset.highs || [];
  const lows = primaryAsset.lows || [];
  let atr = primaryAsset.price * 0.02; // Fallback

  if (closes.length > 14 && highs.length > 14 && lows.length > 14) {
    let trSum = 0;
    for (let i = closes.length - 14; i < closes.length; i++) {
      const h = highs[i] ?? closes[i];
      const l = lows[i] ?? closes[i];
      const pc = closes[i - 1] ?? closes[i];
      const tr = Math.max(h - l, Math.abs(h - pc), Math.abs(l - pc));
      trSum += tr;
    }
    atr = trSum / 14;
  }

  // Cycle Phase Logic (Wyckoff Heuristics)
  let phase: Technicals['cycle']['phase'] = 'Accumulation';
  let desc = '';
  let descVi = '';

  const currentPrice = primaryAsset.price;
  const isAboveEma20 = ema20 ? currentPrice > ema20 : false;
  const isAboveEma50 = ema50 ? currentPrice > ema50 : false;
  const isAboveEma200 = ema200 ? currentPrice > ema200 : true; // Trend fallback

  if (isAboveEma20 && isAboveEma50 && isAboveEma200) {
    phase = 'Markup';
    desc = 'Expansion phase with strong institutional demand.';
    descVi = 'Giai đoạn Tăng giá (Mark-up). Xu hướng tăng mạnh với hỗ trợ từ dòng tiền lớn.';
  } else if (!isAboveEma20 && !isAboveEma50 && !isAboveEma200) {
    phase = 'Mark-Down';
    desc = 'Distribution complete. Aggressive sell-off phase in progress.';
    descVi = 'Giai đoạn Giảm giá (Mark-down). Chu kỳ bán tháo mạnh mẽ sau khi phân phối.';
  } else if (isAboveEma50 && rsi && rsi > 70) {
    phase = 'Distribution';
    desc = 'Supply starting to overwhelm demand. Smart money is exiting.';
    descVi = 'Giai đoạn Phân phối. Nguồn cung bắt đầu áp đảo, dòng tiền lớn đang thoát hàng.';
  } else if (!isAboveEma200 && rsi && rsi < 30) {
    phase = 'Accumulation';
    desc = 'Bottoming process. Institutional players absorbing supply.';
    descVi = 'Giai đoạn Tích lũy. Giá đang tạo đáy, các tổ chức đang âm thầm gom hàng.';
  } else if (isAboveEma50 && !isAboveEma20) {
    phase = 'Decline';
    desc = 'Momentum slowing down. Initial signs of trend reversal.';
    descVi = 'Đà tăng đang chững lại. Các dấu hiệu đầu tiên của sự đảo chiều xu hướng.';
  } else {
    phase = 'Accumulation';
    desc = 'Consolidation phase. Market searching for new equilibrium.';
    descVi = 'Giai đoạn đi ngang tích lũy. Thị trường đang tìm kiếm điểm cân bằng mới.';
  }

  // Phases for Donut chart (Heuristic distribution)
  const phases = [
    { label: 'Accumulation', value: phase === 'Accumulation' ? 80 : 5 },
    { label: 'Mark-Up', value: phase === 'Markup' ? 80 : 5 },
    { label: 'Distribution', value: phase === 'Distribution' ? 80 : 5 },
    { label: 'Mark-Down', value: phase === 'Mark-Down' ? 80 : 5 },
  ];

  // Signal Generation (Expanded Actions)
  let action: 'SHORT' | 'LONG' | 'AVOID' | 'EXIT' | 'REDUCE' | 'TAKE PROFIT' | 'HOLD/WATCH' = 'HOLD/WATCH';
  let actionVi = 'NẮM GIỮ/QUAN SÁT';
  let confidence = 50;
  let reasons: string[] = [];
  let reasonsVi: string[] = [];

  if (phase === 'Markup') {
    if (rsi && rsi < 65) {
      action = 'LONG';
      actionVi = 'LỆNH LONG';
      confidence = 85;
      reasons = ['Strong uptrend confirmed', 'Healthy RSI levels', 'Breakout momentum active'];
      reasonsVi = ['Xác nhận xu hướng tăng mạnh', 'Chỉ báo RSI ổn định', 'Đà bùng nổ đang tiếp diễn'];
    } else {
      action = 'TAKE PROFIT';
      actionVi = 'CHỐT LỜI';
      confidence = 75;
      reasons = ['Overextended Markup phase', 'RSI entering overbought territory', 'Protecting gains recommended'];
      reasonsVi = ['Giai đoạn tăng giá quá đà', 'RSI đi vào vùng quá mua', 'Khuyến nghị chốt lời bảo vệ thành quả'];
    }
  } else if (phase === 'Mark-Down') {
    if (rsi && rsi > 35) {
      action = 'SHORT';
      actionVi = 'LỆNH SHORT';
      confidence = 80;
      reasons = ['Aggressive Markdown phase', 'Trend resistance holding', 'Bearish volume expanding'];
      reasonsVi = ['Giai đoạn Giảm giá mạnh', 'Kháng cự xu hướng được giữ vững', 'Khối lượng bán đang gia tăng'];
    } else {
      action = 'EXIT';
      actionVi = 'THOÁT VỊ THẾ';
      confidence = 90;
      reasons = ['Extreme bearish momentum', 'Final washout phase', 'Oversold bounce likely - Exit shorts'];
      reasonsVi = ['Đà giảm cực đại', 'Giai đoạn rũ bỏ cuối cùng', 'Dễ có nhịp hồi kỹ thuật - Ưu tiên thoát lệnh'];
    }
  } else if (phase === 'Distribution') {
    if (rsi && rsi > 60) {
      action = 'REDUCE';
      actionVi = 'GIẢM TỶ TRỌNG';
      confidence = 70;
      reasons = ['Distribution signs detected', 'Momentum divergent from price', 'De-risking portfolio recommended'];
      reasonsVi = ['Phát hiện dấu hiệu phân phối', 'Đà tăng phân kỳ với giá', 'Khuyến nghị hạ tỷ trọng danh mục'];
    } else {
      action = 'EXIT';
      actionVi = 'TẤT TOÁN VỊ THẾ';
      confidence = 65;
      reasons = ['Trend integrity lost', 'Smart money exiting', 'High risk of cascading decline'];
      reasonsVi = ['Mất xu hướng tăng', 'Dòng tiền lớn đã thoát', 'Rủi ro cao xảy ra nhịp giảm mạnh'];
    }
  } else if (phase === 'Accumulation') {
    if (rsi && rsi < 30) {
      action = 'LONG';
      actionVi = 'MUA TÍCH LŨY';
      confidence = 60;
      reasons = ['Value area reached', 'Supply exhaustion signs', 'Institutional buying detected'];
      reasonsVi = ['Đã chạm vùng giá trị', 'Dấu hiệu cạn kiệt nguồn cung', 'Phát hiện lực mua từ tổ chức'];
    } else {
      action = 'HOLD/WATCH';
      actionVi = 'QUAN SÁT/CHỜ ĐỢI';
      confidence = 55;
      reasons = ['Sideways range active', 'Waiting for clear breakout', 'Volatility compression in progress'];
      reasonsVi = ['Đang đi ngang tích lũy', 'Chờ đợi điểm bùng nổ xác nhận', 'Biến động đang bị nén chặt'];
    }
  }

  // ATR-based SL/TP (Real Volatility)
  const entry = currentPrice;
  const stopLoss = action === 'LONG' || action === 'HOLD/WATCH' ? entry - atr * 1.5 : entry + atr * 1.5;
  const takeProfit = action === 'LONG' || action === 'HOLD/WATCH' ? entry + atr * 3.5 : entry - atr * 3.5;
  const rr = Math.abs((takeProfit - entry) / (entry - stopLoss));

  // S/R logic
  const supportResistance = assets.slice(0, 5).map((a) => {
    const aCloses = a.closes || [];
    let sma = a.price;
    let stdDev = a.price * 0.05;

    let localMins = [a.price * 0.95, a.price * 0.92, a.price * 0.88];
    let localMaxs = [a.price * 1.05, a.price * 1.08, a.price * 1.12];

    if (aCloses.length > 20) {
      const last20 = aCloses.slice(-20);
      sma = last20.reduce((s, v) => s + v, 0) / last20.length;
      const variance = last20.reduce((s, v) => s + Math.pow(v - sma, 2), 0) / last20.length;
      stdDev = Math.sqrt(variance);

      localMins = [...last20].sort((x, y) => x - y).slice(0, 3);
      localMaxs = [...last20].sort((x, y) => y - x).slice(0, 3);
    }

    return {
      symbol: a.name,
      support: localMins,
      resistance: localMaxs,
      bollingerUpper: sma + stdDev * 2,
      bollingerLower: sma - stdDev * 2,
      bollingerMid: sma,
    };
  });

  // Seasonality calculation
  const dates = (primaryAsset.timestamps || []).map((ts) => new Date(ts * 1000));
  const dayStats = calculateSeasonality(closes, dates, 'day');
  const weekStats = calculateSeasonality(closes, dates, 'week');
  const monthStats = calculateSeasonality(closes, dates, 'month');

  const combinedSeasonality = [
    ...dayStats.map((s) => ({ ...s, type: 'day' as const })),
    ...weekStats.map((s) => ({ ...s, type: 'week' as const })),
    ...monthStats.map((s) => ({ ...s, type: 'month' as const })),
  ];

  // ICT Signature Detection (FVG / Order Blocks)
  const fvgs = [];
  if (highs.length >= 3) {
    for (let i = highs.length - 1; i >= highs.length - 10 && i >= 2; i--) {
      // Bullish FVG: Low of candle 0 > High of candle -2
      if (lows[i] > highs[i - 2]) {
        fvgs.push({ type: 'BULLISH', top: lows[i], bottom: highs[i - 2], gap: lows[i] - highs[i - 2] });
      }
      // Bearish FVG: High of candle 0 < Low of candle -2
      else if (highs[i] < lows[i - 2]) {
        fvgs.push({ type: 'BEARISH', top: lows[i - 2], bottom: highs[i], gap: lows[i - 2] - highs[i] });
      }
    }
  }

  // Trend Analysis Metrics (matching UI)
  const isUp = currentPrice > (ema20 || 0) && (ema20 || 0) > (ema50 || 0);
  const isDown = currentPrice < (ema20 || 0) && (ema20 || 0) < (ema50 || 0);
  const trendDirection = isUp ? 'Tăng' : isDown ? 'Giảm' : 'Đi ngang';
  const trendStrength = Math.min(100, Math.max(0, (rsi || 50) + (isUp ? 20 : isDown ? -20 : 0)));
  const trendConfidence = Math.min(100, Math.floor(rsi ? (rsi > 40 && rsi < 60 ? 40 : 80) : 50));

  // Cycle Probabilities for Ring Chart
  const phaseWeights = [
    { label: 'Tích lũy', value: phase === 'Accumulation' ? 60 : 10, color: '#6366f1' },
    { label: 'Tăng giá', value: phase === 'Markup' ? 70 : 10, color: '#10b981' },
    { label: 'Phân phối', value: phase === 'Distribution' ? 50 : 5, color: '#fbbf24' },
    { label: 'Giảm giá', value: phase === 'Mark-Down' ? 80 : 5, color: '#f43f5e' },
  ];
  const totalWeight = phaseWeights.reduce((s, w) => s + w.value, 0);
  const normalizedPhases = phaseWeights.map((w) => ({ ...w, value: Math.round((w.value / totalWeight) * 100) }));

  // Sentiment Analysis
  const vixAsset = assets.find((a) => a.name === 'VIX' || a.symbol === '^VIX');
  const vixVal = vixAsset?.price || 20;
  const sentimentScore = Math.max(0, Math.min(100, 100 - vixVal * 2.5 + (rsi ? rsi - 50 : 0)));
  const sentimentLabel = sentimentScore > 65 ? 'Greed' : sentimentScore < 35 ? 'Fear' : 'Neutral';

  return {
    cycle: {
      phase,
      description: desc,
      descriptionVi: descVi,
      strength: trendStrength,
      confidence: trendConfidence,
      phases: normalizedPhases,
    },
    trend: {
      direction: trendDirection,
      directionEn: isUp ? 'Up' : isDown ? 'Down' : 'Sideways',
      strength: trendStrength,
      confidence: trendConfidence,
    },
    ict: {
      fvgs: fvgs.slice(0, 3),
      orderBlocks: [], // Placeholder for future logic
    },
    sentiment: {
      score: Math.round(sentimentScore),
      label: sentimentLabel,
      labelVi: sentimentLabel === 'Greed' ? 'Tham lam' : sentimentLabel === 'Fear' ? 'Sợ hãi' : 'Trung lập',
    },
    indicators: { rsi, ema20, ema50, ema200, sma20, sma50 },
    signals: {
      action,
      actionVi,
      entry,
      stopLoss,
      takeProfit,
      rr: parseFloat(rr.toFixed(2)),
      confidence,
      reasons,
      reasonsVi,
      optimalEntry: {
        price:
          action === 'LONG'
            ? ema20 && ema20 < entry
              ? ema20
              : entry * 0.985
            : ema20 && ema20 > entry
              ? ema20
              : entry * 1.015,
        pullbackPercent: Math.abs(((entry - (ema20 || entry)) / entry) * 100),
        pullbackAmount: Math.abs(entry - (ema20 || entry)),
      },
    },
    timeframeRelationships: [
      {
        pair: '1wk -> 1d',
        status: isAboveEma50 && isAboveEma200 ? 'ALIGNED' : 'CHOPIED',
        relationship: isAboveEma50 && isAboveEma200 ? 'Bullish Dominance' : 'Regime Transition',
        relationshipVi: isAboveEma50 && isAboveEma200 ? 'Thế trận giá tăng' : 'Giai đoạn chuyển đổi',
        advice: isAboveEma50 ? 'Buy dips on support' : 'Wait for bottom formation',
        adviceVi: isAboveEma50 ? 'Mua tại vùng hỗ trợ' : 'Chờ đợi xác nhận tạo đáy',
      },
      {
        pair: '1d -> 4h',
        status: isAboveEma20 && isAboveEma50 ? 'STRONG' : 'WEAK',
        relationship: isAboveEma20 && isAboveEma50 ? 'Aggressive Trend' : 'Mean Reversion',
        relationshipVi: isAboveEma20 && isAboveEma50 ? 'Xu hướng quyết liệt' : 'Hồi quy về giá trị trung bình',
        advice: isAboveEma20 ? 'Active Scaling' : 'Wait for SMA Alignment',
        adviceVi: isAboveEma20 ? 'Chủ động gia tăng vị thế' : 'Đợi xác nhận từ đường SMA',
      },
      {
        pair: '4h -> 1h',
        status: isAboveEma20 ? 'ALIGNED' : 'WEAK',
        relationship: isAboveEma20 ? 'Momentum Push' : 'Short-term consolidation',
        relationshipVi: isAboveEma20 ? 'Đà tăng mạnh' : 'Tích lũy ngắn hạn',
        advice: isAboveEma20 ? 'Momentum Entry' : 'Wait for EMA20 breakout',
        adviceVi: isAboveEma20 ? 'Vào lệnh theo đà' : 'Đợi giá vượt EMA20',
      },
    ],
    entryTimingScore: {
      overall: Math.min(10, Math.floor((rsi ? (rsi < 35 ? 8 : rsi > 75 ? 2 : 5) : 5) + (isAboveEma20 ? 2 : 0))),
      higherTfSupport: isAboveEma50 ? 5 : 2,
      lowerTfConfirm: isAboveEma20 ? 4 : 1,
    },
    supportResistance,
    seasonality: combinedSeasonality,
    n: closes.length,
    dateRange: `${closes.length > 0 ? new Date((primaryAsset.timestamps?.[0] || 0) * 1000).toLocaleDateString('vi-VN') : 'N/A'} -> ${closes.length > 0 ? new Date((primaryAsset.timestamps?.[closes.length - 1] || 0) * 1000).toLocaleDateString('vi-VN') : 'N/A'}`,
  };
}
