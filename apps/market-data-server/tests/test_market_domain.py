from datetime import datetime, timedelta, timezone

import pytest

from src.market.candles import ICT, Timeframe, aggregate_candles, clean_candles
from src.market.chains import Chains, load_chains
from src.market.errors import ValidationError
from src.market.models import BankRate, Candle, ExchangeRate, PriceSeries, RateBasis, RateChannel, SeriesType, Ticker, TickerType, to_json
from src.market.symbols import is_vn_listing, normalise_symbol


def at(*args, tz=timezone.utc):
    return datetime(*args, tzinfo=tz)


class TestSymbols:
    def test_normalise_uppercases_trims_and_drops_caret(self):
        assert normalise_symbol(" ^vnindex ") == "VNINDEX"
        assert normalise_symbol("fpt") == "FPT"
        assert normalise_symbol("") == ""

    def test_fx_gold_and_vix_are_not_vn_listings(self):
        for symbol in ("USDVND", "sjc", "XAUUSD", "^VIX"):
            assert not is_vn_listing(symbol)
        for symbol in ("FPT", "VNINDEX", "VN30"):
            assert is_vn_listing(symbol)

    def test_only_symbols_shaped_like_vn_listings_are_vn_listings(self):
        # Stocks, ETFs, warrants, futures and indices are 3-10 letters and digits.
        for symbol in ("VNM", "E1VFVN30", "FUEVFVND", "CFPT2401", "VN30F1M", "HNX30", "UPCOMINDEX"):
            assert is_vn_listing(symbol)
        # Yahoo-style global symbols, pairs and junk are not, so VN sources skip them.
        for symbol in ("BTC-USD", "EURUSD=X", "FPT.VN", "GC=F", "AB", "", "ABCDEFGHIJK"):
            assert not is_vn_listing(symbol)


class TestTimeframe:
    def test_parse_accepts_known_values_only(self):
        assert Timeframe.parse(" 4H ") is Timeframe.H4
        with pytest.raises(ValidationError):
            Timeframe.parse("2h")

    def test_four_hour_bars_are_fetched_as_hourly(self):
        assert Timeframe.H4.source is Timeframe.H1
        assert Timeframe.D1.source is Timeframe.D1

    def test_lookback_defaults(self):
        assert [tf.analysis_lookback_days for tf in Timeframe] == [60, 180, 730, 1825]


class TestCleanCandles:
    def test_sorts_drops_empty_fills_and_keeps_last_duplicate(self):
        candles = clean_candles(
            [
                Candle(time=at(2026, 10, 2), close=12, open=11, high=13, low=10),
                Candle(time=at(2026, 10, 1), close=10),
                Candle(time=at(2026, 10, 3), close=0),
                Candle(time=None, close=5),
                Candle(time=at(2026, 10, 2), close=14, open=12, high=15, low=11),
            ]
        )
        assert [c.time.day for c in candles] == [1, 2]
        assert (candles[0].open, candles[0].high, candles[0].low) == (10, 10, 10)
        assert candles[1].close == 14


class TestAggregateCandles:
    def hourly(self, start, closes):
        return [
            Candle(time=start + timedelta(hours=i), open=c, high=c + 1, low=c - 1, close=c, volume=10)
            for i, c in enumerate(closes)
        ]

    def test_four_hour_buckets_follow_vietnam_time(self):
        # 02:00 UTC = 09:00 ICT: 09, 10, 11 fall in the 08:00 bucket, 13 and 14 in 12:00.
        bars = self.hourly(at(2026, 10, 12, 2), [10, 11, 12])
        bars += self.hourly(at(2026, 10, 12, 6), [20, 21])
        result = aggregate_candles(bars, Timeframe.H4)
        assert [c.time for c in result] == [at(2026, 10, 12, 8, tz=ICT), at(2026, 10, 12, 12, tz=ICT)]
        first = result[0]
        assert (first.open, first.high, first.low, first.close, first.volume) == (10, 13, 9, 12, 30)

    def test_day_and_week_buckets_use_local_date_and_monday(self):
        # 2026-10-11 18:00 UTC is Monday 2026-10-12 01:00 ICT.
        bars = self.hourly(at(2026, 10, 11, 18), [10]) + self.hourly(at(2026, 10, 14, 3), [12])
        days = aggregate_candles(bars, Timeframe.D1)
        assert [c.time for c in days] == [at(2026, 10, 12, tz=ICT), at(2026, 10, 14, tz=ICT)]
        weeks = aggregate_candles(bars, Timeframe.W1)
        assert [c.time for c in weeks] == [at(2026, 10, 12, tz=ICT)]
        assert weeks[0].close == 12

    def test_hourly_is_unchanged(self):
        bars = self.hourly(at(2026, 10, 12, 2), [10, 11])
        assert aggregate_candles(bars, Timeframe.H1) is bars


class TestChains:
    def test_defaults_match_wealth_management(self):
        chains = Chains()
        assert chains.ticker["equity"] == ["vnstock", "fmarket", "algolab"]
        assert chains.fx_chain("USD_VND") == ["fmarket", "yahoo", "currencyapi"]
        assert chains.fx_chain("EUR_VND") == ["yahoo", "currencyapi"]
        assert chains.candles == ["vnstock", "yahoo", "fmarket", "algolab"]
        assert chains.ticker["crypto"] == ["coingecko", "yahoo"]
        assert chains.bank_rates == ["cafef", "simplize", "fmarket"]

    def test_env_override_replaces_one_entry(self):
        chains = load_chains('{"candles": ["yahoo"]}')
        assert chains.candles == ["yahoo"]
        assert chains.bank_rates == ["cafef", "simplize", "fmarket"]

    @pytest.mark.parametrize("raw", ['{"nope": []}', '{"candles": {"a": 1}}'])
    def test_env_override_rejects_bad_entries(self, raw):
        with pytest.raises(ValueError):
            load_chains(raw)


class TestJson:
    def test_models_serialise_with_wm_field_names(self):
        assert to_json(ExchangeRate(base="USD", quote="VND", rate=25000, provider="yahoo", timestamp="t")) == {
            "from": "USD", "to": "VND", "rate": 25000, "provider": "yahoo", "timestamp": "t",
        }  # fmt: skip
        assert to_json(ExchangeRate(base="USD", quote="VND", rate=25000, provider="yahoo", basis=RateBasis.MARKET))["basis"] == "market"
        assert to_json(Ticker(symbol="FPT", type=TickerType.EQUITY, provider="vnstock", price=1.5)) == {
            "symbol": "FPT", "type": "equity", "provider": "vnstock", "price": 1.5,
        }  # fmt: skip
        series = to_json(PriceSeries(provider="yahoo", series_type=SeriesType.FOREX, data={"points": []}))
        assert series == {"provider": "yahoo", "seriesType": "forex", "data": {"points": []}}
        assert to_json(Candle(time=at(2026, 10, 12, 9, tz=ICT), close=1)) == {
            "time": "2026-10-12T02:00:00Z", "close": 1, "open": 0.0, "high": 0.0, "low": 0.0, "volume": 0.0,
        }  # fmt: skip

    def test_bank_rates_without_a_tenor_keep_the_original_shape(self):
        headline = BankRate(bank="Alpha Bank", rate=6.2, currency="VND", updated_at="2026-10-05")
        assert to_json(headline) == {"bank": "Alpha Bank", "tenor": "", "rate": 6.2, "currency": "VND", "updatedAt": "2026-10-05"}

    def test_bank_rates_by_tenor_carry_months_channel_and_instant_access(self):
        no_term = BankRate.by_tenor("Alpha Bank", 0, 0.2, channel=RateChannel.COUNTER, updated_at="2026-10-17")
        assert to_json(no_term) == {
            "bank": "Alpha Bank", "tenor": "no-term", "rate": 0.2, "currency": "VND", "updatedAt": "2026-10-17",
            "tenorMonths": 0, "channel": "counter", "instantAccess": True,
        }  # fmt: skip
        one_month = to_json(BankRate.by_tenor("Alpha Bank", 1, 3.9))
        assert one_month == {"bank": "Alpha Bank", "tenor": "1m", "rate": 3.9, "currency": "VND", "tenorMonths": 1, "instantAccess": False}
