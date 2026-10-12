package e2e

import (
	"net/http"
	"testing"
)

// Market routes call fmarket live, so the checks are structural and loose.

func TestE2EMarketSmoke(t *testing.T) {
	h := engine(t)
	t.Run("fund ticker", func(t *testing.T) {
		var ticker map[string]any
		h.request(t, http.MethodGet, "/api/market/ticker?symbol=VFMVF1&type=ifc", nil, http.StatusOK, &ticker)
		if ticker["symbol"] != "VFMVF1" || ticker["currency"] != "VND" || number(ticker["price"]) <= 0 {
			t.Fatalf("unexpected fund ticker %+v", ticker)
		}
	})
	t.Run("SJC gold", func(t *testing.T) {
		var ticker map[string]any
		h.request(t, http.MethodGet, "/api/market/ticker?symbol=SJC&type=gold", nil, http.StatusOK, &ticker)
		bid, ask := number(ticker["bid"]), number(ticker["ask"])
		if ticker["currency"] != "VND" || bid <= 0 || ask < bid {
			t.Fatalf("unexpected SJC ticker %+v", ticker)
		}
	})
	t.Run("USD/VND", func(t *testing.T) {
		var rate map[string]any
		h.request(t, http.MethodGet, "/api/market/exchange-rate?from=USD&to=VND", nil, http.StatusOK, &rate)
		if value := number(rate["rate"]); value < 10000 || value > 50000 {
			t.Fatalf("implausible USD/VND rate %+v", rate)
		}
	})
	t.Run("world gold series", func(t *testing.T) {
		var series struct {
			Currency string `json:"currency"`
			Data     struct {
				Points []struct {
					Price float64 `json:"price"`
				} `json:"points"`
			} `json:"data"`
		}
		h.request(t, http.MethodGet, "/api/market/price-series?symbol=GOLD&type=gold_usd", nil, http.StatusOK, &series)
		if series.Currency != "USD" || len(series.Data.Points) == 0 || series.Data.Points[0].Price <= 0 {
			t.Fatalf("unexpected gold series %+v", series)
		}
	})
	t.Run("bank rates", func(t *testing.T) {
		var rates []struct {
			Bank string  `json:"bank"`
			Rate float64 `json:"rate"`
		}
		h.request(t, http.MethodGet, "/api/market/bank-rates", nil, http.StatusOK, &rates)
		if len(rates) == 0 || rates[0].Bank == "" || rates[0].Rate <= 0 {
			t.Fatalf("unexpected bank rates %+v", rates)
		}
	})
}

func number(value any) float64 {
	f, _ := value.(float64)
	return f
}
