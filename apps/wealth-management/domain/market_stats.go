package domain

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"slices"
	"time"
)

// Seasonality groupings (WM-INV-14).
const (
	SeasonalityDay   = "day"
	SeasonalityWeek  = "week"
	SeasonalityMonth = "month"
)

// SeasonalityBucket summarises the bar-to-bar returns that fall in one
// weekday, week of month or month. Returns, win rate and deviation are in
// percent; Score = avgReturn × 100 × winRate / (1 + stdDev) with the legacy
// units (return as a fraction, win rate and deviation in percent).
type SeasonalityBucket struct {
	Type         string  `json:"type"`
	Key          int     `json:"key"`
	Label        string  `json:"label"`
	Rank         int     `json:"rank"`
	AvgReturn    float64 `json:"avgReturn"`
	WinRate      float64 `json:"winRate"`
	ProfitFactor float64 `json:"profitFactor"`
	StdDev       float64 `json:"stdDev"`
	Score        float64 `json:"score"`
	N            int     `json:"n"`
}

var weekdayLabels = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

// Seasonality ranks buckets of daily returns by score within each type.
// Buckets with fewer than two returns are dropped. Bar dates are read in
// location.
func Seasonality(candles []Candle, location *time.Location) []SeasonalityBucket {
	type group struct{ returns []float64 }
	groups := map[string]map[int]*group{SeasonalityDay: {}, SeasonalityWeek: {}, SeasonalityMonth: {}}
	for i := 1; i < len(candles); i++ {
		ret := candles[i].Close/candles[i-1].Close - 1
		local := candles[i].Time.In(location)
		firstOfMonth := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location)
		keys := map[string]int{
			SeasonalityDay:   int(local.Weekday()),
			SeasonalityWeek:  (local.Day()+int(firstOfMonth.Weekday())-1)/7 + 1,
			SeasonalityMonth: int(local.Month()),
		}
		for kind, key := range keys {
			if groups[kind][key] == nil {
				groups[kind][key] = &group{}
			}
			groups[kind][key].returns = append(groups[kind][key].returns, ret)
		}
	}

	result := []SeasonalityBucket{}
	for _, kind := range []string{SeasonalityDay, SeasonalityWeek, SeasonalityMonth} {
		buckets := []SeasonalityBucket{}
		for key, g := range groups[kind] {
			if len(g.returns) < 2 {
				continue
			}
			buckets = append(buckets, seasonalityBucket(kind, key, g.returns))
		}
		slices.SortFunc(buckets, func(a, b SeasonalityBucket) int {
			if a.Score != b.Score {
				return compareDesc(a.Score, b.Score)
			}
			return a.Key - b.Key
		})
		for i := range buckets {
			buckets[i].Rank = i + 1
		}
		result = append(result, buckets...)
	}
	return result
}

func seasonalityBucket(kind string, key int, returns []float64) SeasonalityBucket {
	avg := mean(returns)
	var wins int
	var gains, losses float64
	for _, ret := range returns {
		if ret > 0 {
			wins++
			gains += ret
		} else {
			losses -= ret
		}
	}
	winRate := float64(wins) / float64(len(returns)) * 100
	deviation := stdDev(returns, avg) * 100
	profitFactor := 0.0
	if losses > 0 {
		profitFactor = gains / losses
	}
	label := ""
	switch kind {
	case SeasonalityDay:
		label = weekdayLabels[key]
	case SeasonalityWeek:
		label = fmt.Sprintf("W%d", key)
	default:
		label = time.Month(key).String()[:3]
	}
	return SeasonalityBucket{
		Type:         kind,
		Key:          key,
		Label:        label,
		AvgReturn:    avg * 100,
		WinRate:      winRate,
		ProfitFactor: profitFactor,
		StdDev:       deviation,
		Score:        avg * 100 * winRate / (1 + deviation),
		N:            len(returns),
	}
}

func compareDesc(a, b float64) int {
	switch {
	case a > b:
		return -1
	case a < b:
		return 1
	default:
		return 0
	}
}

// TradingDaysPerYear annualises daily statistics.
const TradingDaysPerYear = 252

// MonteCarloPaths is the number of simulated paths (WM-INV-16).
const MonteCarloPaths = 10000

// RiskStats are honest, history-only statistics (WM-INV-16): no fair value
// is claimed. The simulation resamples historical daily returns; it shows
// the spread history implies, not a forecast.
type RiskStats struct {
	From             string      `json:"from"`
	To               string      `json:"to"`
	Returns          int         `json:"returns"`
	PeriodReturn     float64     `json:"periodReturn"`
	AnnualisedReturn *float64    `json:"annualisedReturn"`
	AnnualisedVol    float64     `json:"annualisedVolatility"`
	MaxDrawdown      float64     `json:"maxDrawdown"`
	CurrentDrawdown  float64     `json:"currentDrawdown"`
	Simulation       *Simulation `json:"simulation"`
	Method           string      `json:"method"`
}

// Simulation is a bootstrap of historical daily returns over Horizon
// trading days. Prices are percentiles of the simulated final price.
type Simulation struct {
	Horizon     int     `json:"horizonDays"`
	Paths       int     `json:"paths"`
	P10         float64 `json:"p10"`
	P50         float64 `json:"p50"`
	P90         float64 `json:"p90"`
	ProbLossPct float64 `json:"probabilityOfLoss"`
	Seed        uint64  `json:"seed"`
}

// minSimulationReturns is the history needed before resampling it means
// anything (about six months of sessions).
const minSimulationReturns = 120

// ComputeRiskStats needs daily candles (oldest first). Percent fields are in
// percent; drawdowns are negative. The simulation seed is derived from the
// symbol and last bar so the same data gives the same answer.
func ComputeRiskStats(symbol string, daily []Candle, horizon int) (RiskStats, error) {
	if len(daily) < 2 {
		return RiskStats{}, fmt.Errorf("%w: need at least two daily candles", ErrValidation)
	}
	first, last := daily[0], daily[len(daily)-1]
	logReturns := make([]float64, 0, len(daily)-1)
	peak, maxDrawdown := daily[0].Close, 0.0
	for i := 1; i < len(daily); i++ {
		logReturns = append(logReturns, math.Log(daily[i].Close/daily[i-1].Close))
		peak = math.Max(peak, daily[i].Close)
		maxDrawdown = math.Min(maxDrawdown, daily[i].Close/peak-1)
	}
	stats := RiskStats{
		From:            first.Time.Format(time.RFC3339),
		To:              last.Time.Format(time.RFC3339),
		Returns:         len(logReturns),
		PeriodReturn:    (last.Close/first.Close - 1) * 100,
		AnnualisedVol:   sampleStdDev(logReturns) * math.Sqrt(TradingDaysPerYear) * 100,
		MaxDrawdown:     maxDrawdown * 100,
		CurrentDrawdown: (last.Close/peak - 1) * 100,
		Method:          "History only: annualised figures use 252 sessions; the simulation resamples past daily returns and is not a forecast.",
	}
	if years := last.Time.Sub(first.Time).Hours() / 24 / 365.25; years >= 1 {
		annualised := (math.Pow(last.Close/first.Close, 1/years) - 1) * 100
		stats.AnnualisedReturn = &annualised
	}
	if horizon > 0 && len(logReturns) >= minSimulationReturns {
		simulation := bootstrap(logReturns, last.Close, horizon, MonteCarloPaths, simulationSeed(symbol, last.Time))
		stats.Simulation = &simulation
	}
	return stats, nil
}

func bootstrap(logReturns []float64, price float64, horizon int, paths int, seed uint64) Simulation {
	random := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	finals := make([]float64, paths)
	var losses int
	for p := range finals {
		var total float64
		for range horizon {
			total += logReturns[random.IntN(len(logReturns))]
		}
		finals[p] = price * math.Exp(total)
		if total < 0 {
			losses++
		}
	}
	slices.Sort(finals)
	percentile := func(q float64) float64 { return finals[int(q*float64(paths-1))] }
	return Simulation{
		Horizon:     horizon,
		Paths:       paths,
		P10:         percentile(0.10),
		P50:         percentile(0.50),
		P90:         percentile(0.90),
		ProbLossPct: float64(losses) / float64(paths) * 100,
		Seed:        seed,
	}
}

func simulationSeed(symbol string, last time.Time) uint64 {
	hash := fnv.New64a()
	_, _ = fmt.Fprintf(hash, "%s|%d", symbol, last.Unix())
	return hash.Sum64()
}

func sampleStdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	avg := mean(values)
	var sum float64
	for _, value := range values {
		sum += (value - avg) * (value - avg)
	}
	return math.Sqrt(sum / float64(len(values)-1))
}

// PearsonCorrelation of two equally long series; ok is false when either is
// constant or shorter than three points.
func PearsonCorrelation(x, y []float64) (float64, bool) {
	if len(x) != len(y) || len(x) < 3 {
		return 0, false
	}
	xMean, yMean := mean(x), mean(y)
	var num, xDen, yDen float64
	for i := range x {
		dx, dy := x[i]-xMean, y[i]-yMean
		num += dx * dy
		xDen += dx * dx
		yDen += dy * dy
	}
	if xDen == 0 || yDen == 0 {
		return 0, false
	}
	return num / math.Sqrt(xDen*yDen), true
}
