import { generateText } from 'ai';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { loadPrompt, replacePlaceholders } from '@wealth-management/ai/server';
import { getErrorMessage } from '@wealth-management/utils';
import { MarketAsset, MarketState } from './types';

/**
 * Uses AI to synthesize market data into a cohesive scenario and capital flow signals.
 */
export async function generateAiMarketAnalysis(us: MarketState, vn: MarketState, timeframe: string) {
  const model = getLanguageModel('github-gpt-4o');

  const template = await loadPrompt('market', 'generate-ai-analysis');
  if (!template) {
    throw new Error(
      'Missing prompt template: market/generate-ai-analysis. Please ensure it is present in Google Sheets.',
    );
  }

  const prompt = replacePlaceholders(template, {
    timeframe,
    usAssets: us.assets.map((a) => `${a.name}: ${a.price} (${a.percentChange.toFixed(2)}% on ${timeframe})`).join(', '),
    usPhase: us.technicals?.cycle.phase || 'N/A',
    usDescription: us.technicals?.cycle.description || 'N/A',
    usSupportResistance:
      us.technicals?.supportResistance
        .slice(0, 3)
        .map((sr) => `${sr.symbol}: S:[${sr.support.join(',')}] R:[${sr.resistance.join(',')}]`)
        .join(' | ') || 'N/A',
    usDcf:
      us.valuation?.dcf
        .map((v) => `${v.symbol}: FV $${v.fairValue.toFixed(2)} (${v.upside.toFixed(1)}% upside)`)
        .join(', ') || 'N/A',
    usMonteCarlo: us.valuation?.monteCarlo.map((m) => `${m.symbol}: $${m.p50.toFixed(2)}`).join(', ') || 'N/A',
    usAssetList: us.assetList.join(', '),
    usCorrelationMatrix: JSON.stringify(us.correlationMatrix),
    vnAssets: vn.assets.map((a) => `${a.name}: ${a.price} (${a.percentChange.toFixed(2)}% on ${timeframe})`).join(', '),
    vnPhase: vn.technicals?.cycle.phase || 'N/A',
    vnDescription: vn.technicals?.cycle.description || 'N/A',
    vnSupportResistance:
      vn.technicals?.supportResistance
        .slice(0, 3)
        .map((sr) => `${sr.symbol}: S:[${sr.support.join(',')}] R:[${sr.resistance.join(',')}]`)
        .join(' | ') || 'N/A',
    vnDcf:
      vn.valuation?.dcf
        .map((v) => `${v.symbol}: FV ${v.fairValue.toLocaleString()} (${v.upside.toFixed(1)}% upside)`)
        .join(', ') || 'N/A',
    vnSectorMultiples:
      vn.valuation?.sectorComparison
        .map((s) => `${s.symbol} P/E: ${s.metrics.find((m) => m.label.includes('P/E'))?.asset}`)
        .join(', ') || 'N/A',
    vnAssetList: vn.assetList.join(', '),
    vnCorrelationMatrix: JSON.stringify(vn.correlationMatrix),
  });

  const { text } = await generateText({
    model,
    prompt,
  });

  try {
    // Clean potential markdown wrap
    const cleanJson = text.replace(/```json|```/g, '').trim();
    return JSON.parse(cleanJson);
  } catch (e) {
    const message = getErrorMessage(e);
    console.error('[MarketPulse:AI] Failed to parse AI market analysis JSON:', message);
    return null;
  }
}

export function computeCapitalFlow(assets: MarketAsset[], market: 'US' | 'VN'): MarketState['capitalFlow'] {
  const gold = assets.find((a) => a.name.includes('Gold'));
  const sp500 = assets.find((a) => a.name === 'S&P500' || a.name === 'VN-Index');
  const vix = assets.find((a) => a.name === 'VIX');

  if (gold && gold.percentChange > 0.5 && ((sp500 && sp500.percentChange < 0) || (vix && vix.percentChange > 2))) {
    return {
      signal: 'DEFENSIVE',
      summaryEn: `Defensive signal led by Gold (+${gold.percentChange.toFixed(2)}%). Equities under pressure.`,
      summaryVi: `Tín hiệu phòng thủ đang nghiêng về Vàng (+${gold.percentChange.toFixed(2)}%). Cổ phiếu đang gặp áp lực.`,
    };
  }

  if (sp500 && sp500.percentChange > 0.5 && vix && vix.percentChange < 0) {
    return {
      signal: 'RISK-ON',
      summaryEn: `Risk-on environment detected. Equities gaining (+${sp500.percentChange.toFixed(2)}%).`,
      summaryVi: `Tín hiệu tăng trưởng được ghi nhận. Cổ phiếu đang tăng (+${sp500.percentChange.toFixed(2)}%).`,
    };
  }

  return {
    signal: 'MIXED',
    summaryEn: `Market signals are mixed. No clear direction in capital flow.`,
    summaryVi: `Tín hiệu thị trường đang hỗn hợp. Chưa có xu hướng rõ ràng của dòng tiền.`,
  };
}

export function detectScenario(us: MarketState, vn: MarketState): NonNullable<MarketState['scenarios']>[0] {
  const vix = us.assets.find((a) => a.name === 'VIX');
  const sp500 = us.assets.find((a) => a.name === 'S&P500');
  const vn30 = vn.assets.find((a) => a.name === 'VN30');
  const oil = us.assets.find((a) => a.name === 'WTI');

  // BunnyQuant: Falling Knife Detection for VN30
  if (vn30 && vn30.percentChange < -1.5) {
    return {
      name: 'Falling Knife Warning',
      regime: 'Crisis',
      confidence: 90,
      summaryEn: 'Severe institutional dumping phase. RSI/Oversold indicators are failing. Do NOT average down.',
      summaryVi:
        'Cảnh báo Bắt Dao Rơi. Chu kỳ giảm mạnh, các chỉ báo như RSI hiện vô dụng. Tuyệt đối KHÔNG "cứ đỏ là mua" hay DCA.',
      actionEn: 'Halt all buying immediately. Wait for bottom accumulation confirmation.',
      actionVi: 'Dừng hoàn toàn việc mua vào. Đợi tín hiệu xác nhận tạo đáy tích lũy.',
    };
  }

  if (vix && vix.price > 30) {
    return {
      name: 'Crisis mode',
      regime: 'Crisis',
      confidence: 85,
      summaryEn: 'High volatility detected. Extreme panic across markets.',
      summaryVi: 'Biến động mạnh. Các thị trường đang ở mức hoảng loạn cao.',
      actionEn: 'Shift to cash and defensive assets. Hedge long positions.',
      actionVi: 'Chuyển sang tiền mặt và tài sản phòng thủ. Hedging các vị thế mua.',
    };
  }

  if (vix && vix.price < 15 && sp500 && sp500.percentChange > 0) {
    return {
      name: 'Normal Growth',
      regime: 'Risk-ON',
      confidence: 75,
      summaryEn: 'Scenario detected based on current correlations and price action.',
      summaryVi: 'Kịch bản được nhận diện dựa trên tương quan và biến động giá hiện tại.',
      actionEn: 'Maintain long exposure to risk assets and strong momentum sectors.',
      actionVi: 'Duy trì nắm giữ các tài sản rủi ro và các nhóm ngành có dòng tiền mạnh.',
    };
  }

  if (oil && oil.percentChange > 3 && sp500 && sp500.percentChange < -1) {
    return {
      name: 'Stagflation Risk',
      regime: 'Stagflation',
      confidence: 65,
      summaryEn: 'Oil surging while equities fall suggests supply-side pressure.',
      summaryVi: 'Giá dầu tăng mạnh trong khi cổ phiếu giảm cho thấy áp lực từ nguồn cung.',
      actionEn: 'Increase exposure to commodities/energy. Reduce cyclical stocks.',
      actionVi: 'Tăng cường tỷ trọng hàng hóa/năng lượng. Giảm cổ phiếu chu kỳ.',
    };
  }

  return {
    name: 'Neutral / Mixed',
    regime: 'Risk-OFF',
    confidence: 50,
    summaryEn: 'Market is in a transitional or uncertain phase.',
    summaryVi: 'Thị trường đang ở giai đoạn chuyển giao hoặc không chắc chắn.',
    actionEn: 'Hold current positions. Wait for a clear directional breakout.',
    actionVi: 'Giữ nguyên vị thế hiện tại. Chờ đợi xu hướng định hướng rõ ràng.',
  };
}
