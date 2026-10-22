from datetime import datetime, timedelta, timezone

import pytest
from fastapi.testclient import TestClient

from src.cache import CacheManager
from src.main import app
from src.market.candles import Timeframe
from src.market.chains import Chains, MarketTTL
from src.market.errors import NotFound, Unsupported, UpstreamError, ValidationError
from src.market.models import BankRate, Candle, CandleSeries, EarningStart, ExchangeRate, FlexibleSaving, RateChannel, SavingsBacking, Ticker, utc_iso
from src.market.router import Router
from src.market.runtime import get_market_service
from src.market.service import MarketService
from src.market.store import LastGoodStore

pytestmark = pytest.mark.anyio

NOW = datetime(2026, 10, 12, 20, 0, tzinfo=timezone.utc)


class FakeSource:
    """A source whose capabilities are set per test; records every call."""

    def __init__(self, name, **answers):
        self.name = name
        self.calls = []
        for method, answer in answers.items():
            setattr(self, method, self._capability(method, answer))

    def _capability(self, method, answer):
        async def run(*args):
            self.calls.append((method, args))
            value = answer(*args) if callable(answer) else answer
            if isinstance(value, Exception):
                raise value
            return value

        return run

    async def health(self):
        return "ok"

    async def aclose(self):
        pass


def ticker(provider, price=10.0):
    return Ticker(symbol="FPT", type="equity", provider=provider, price=price, currency="VND")


def hourly_series(provider, *_):
    start = NOW - timedelta(hours=8)
    candles = [Candle(time=start + timedelta(hours=i), open=i, high=i + 1, low=i, close=i + 0.5, volume=1) for i in range(1, 7)]
    return CandleSeries(symbol="FPT", timeframe="1h", provider=provider, candles=candles, currency="VND")


def tenor_rates(*tenors, bank="Alpha Bank"):
    return [BankRate.by_tenor(bank, months, rate, channel=RateChannel.COUNTER) for months, rate in tenors]


HEADLINE = [BankRate(bank="Alpha Bank", rate=6.2, currency="VND", updated_at="2026-10-05")]


@pytest.fixture
def store():
    return LastGoodStore(":memory:")


def service(*sources, cache=None, store=None, chains=None):
    return MarketService({s.name: s for s in sources}, chains=chains or Chains(), cache=cache, last_good=store, now=lambda: NOW)


class TestRouter:
    async def test_skips_unsupported_and_missing_capabilities_but_reports_failures(self):
        router = Router({
            "a": FakeSource("a"),
            "b": FakeSource("b", get_ticker=Unsupported("not covered")),
            "c": FakeSource("c", get_ticker=RuntimeError("timeout")),
            "d": FakeSource("d", get_ticker="answer"),
        })  # fmt: skip
        routed = await router.route("op", ["a", "b", "c", "d"], "get_ticker", lambda s: s.get_ticker())
        assert (routed.value, routed.provider, routed.fallback) == ("answer", "d", ["c: timeout"])

    async def test_all_skipped_is_a_validation_error(self):
        router = Router({"a": FakeSource("a", get_ticker=Unsupported("nope"))})
        with pytest.raises(ValidationError, match="no provider covers"):
            await router.route("op", ["a"], "get_ticker", lambda s: s.get_ticker())

    async def test_failures_are_listed_before_skips(self):
        router = Router({"a": FakeSource("a", get_ticker=Unsupported("nope")), "b": FakeSource("b", get_ticker=RuntimeError())})
        with pytest.raises(UpstreamError, match=r"b: RuntimeError; unknown market provider: ghost; a: nope"):
            await router.route("op", ["a", "b", "ghost"], "get_ticker", lambda s: s.get_ticker())

    async def test_misses_fall_through_and_are_recorded(self):
        router = Router({"a": FakeSource("a", get_fund=NotFound("no fund X")), "b": FakeSource("b", get_fund="fund")})
        routed = await router.route("op", ["a", "b"], "get_fund", lambda s: s.get_fund())
        assert (routed.value, routed.provider, routed.fallback) == ("fund", "b", ["a: no fund X"])

    async def test_only_misses_and_skips_is_not_found(self):
        router = Router({"a": FakeSource("a", get_fund=Unsupported("nope")), "b": FakeSource("b", get_fund=NotFound("no fund X"))})
        with pytest.raises(NotFound, match="b: no fund X"):
            await router.route("op", ["a", "b"], "get_fund", lambda s: s.get_fund())

    async def test_a_failure_beats_a_miss(self):
        router = Router({"a": FakeSource("a", get_fund=NotFound("no fund X")), "b": FakeSource("b", get_fund=RuntimeError("down"))})
        with pytest.raises(UpstreamError, match="a: no fund X; b: down"):
            await router.route("op", ["a", "b"], "get_fund", lambda s: s.get_fund())

    async def test_empty_chain_is_a_validation_error(self):
        with pytest.raises(ValidationError):
            await Router({}).route("op", [], "get_ticker", lambda s: s.get_ticker())


class TestService:
    async def test_fallback_records_what_failed(self, store):
        primary = FakeSource("vnstock", get_ticker=RuntimeError("quota"))
        backup = FakeSource("fmarket", get_ticker=lambda *a: ticker("fmarket"))
        answer = await service(primary, backup, store=store).ticker(" fpt ", "EQUITY")
        assert (answer.provider, answer.fallback, answer.stale) == ("fmarket", ["vnstock: quota"], False)
        assert answer.data["price"] == 10.0
        assert answer.as_of == utc_iso(NOW)
        assert primary.calls[0][1][0] == "FPT"

    async def test_bad_input_is_rejected_before_routing(self):
        source = FakeSource("vnstock", get_ticker=lambda *a: ticker("vnstock"))
        market = service(source)
        for call in (
            lambda: market.ticker("", "equity"),
            lambda: market.ticker("FPT", "bond"),
            lambda: market.exchange_rate("US D", "VND"),
            lambda: market.candles("FPT", "2h"),
            lambda: market.candles("FPT", "1d", days=0),
            lambda: market.candles("FPT", "1d", start="2026-10-10", end="2026-10-01"),
            lambda: market.candles("FPT", "1d", end="2026-10-01"),
            lambda: market.funds([], False, 0, 10),
            lambda: market.fx_history("EUR_VND", None, None),
        ):
            with pytest.raises(ValidationError):
                await call()
        assert source.calls == []

    async def test_cache_hit_and_skip_cache(self):
        source = FakeSource("vnstock", get_ticker=lambda *a: ticker("vnstock", price=len(source.calls)))
        market = service(source, cache=CacheManager(enabled=False))
        first = await market.ticker("FPT", "equity")
        second = await market.ticker("FPT", "equity")
        assert (first.cached, second.cached, second.data["price"]) == (False, True, 1)
        refreshed = await market.ticker("FPT", "equity", skip_cache=True)
        assert (refreshed.cached, refreshed.data["price"]) == (False, 2)
        # The refreshed answer replaced the cached one.
        assert (await market.ticker("FPT", "equity")).data["price"] == 2

    @pytest.mark.parametrize(
        "call, ttl",
        [
            (lambda m: m.ticker("FPT", "equity"), MarketTTL.TICKER),
            (lambda m: m.bank_rates(), MarketTTL.BANK_RATES),
            (lambda m: m.candles("FPT", "1d"), MarketTTL.CANDLES),
        ],
    )
    async def test_each_capability_has_its_ttl(self, monkeypatch, call, ttl):
        cache = CacheManager(enabled=False)
        ttls = []
        original = cache.set

        async def spy(endpoint, params, data, ttl):
            ttls.append(ttl)
            return await original(endpoint, params, data, ttl=ttl)

        monkeypatch.setattr(cache, "set", spy)
        source = FakeSource(
            "vnstock",
            get_ticker=lambda *a: ticker("vnstock"),
            get_bank_rates=[],
            get_candles=lambda *a: hourly_series("vnstock"),
        )
        await call(service(source, cache=cache, chains=Chains(bank_rates=["vnstock"])))
        assert len(ttls) == 1 and 0 < ttls[0] <= ttl

    async def test_last_good_copy_is_served_stale_when_every_provider_fails(self, store):
        answers = iter([ticker("vnstock"), RuntimeError("down")])
        source = FakeSource("vnstock", get_ticker=lambda *a: next(answers))
        market = service(source, store=store)
        fresh = await market.ticker("FPT", "equity")
        stale = await market.ticker("FPT", "equity", skip_cache=True)
        assert (stale.stale, stale.as_of, stale.data) == (True, fresh.as_of, fresh.data)

    async def test_an_item_that_is_gone_is_not_served_stale(self, store):
        answers = iter([{"code": "DCDS"}, NotFound("fmarket has no fund DCDS")])
        market = service(FakeSource("fmarket", get_fund=lambda *a: next(answers)), store=store)
        await market.fund("DCDS")
        with pytest.raises(NotFound):
            await market.fund("DCDS", skip_cache=True)

    async def test_without_a_copy_the_failure_surfaces(self, store):
        market = service(FakeSource("vnstock", get_ticker=RuntimeError("down")), store=store)
        with pytest.raises(UpstreamError, match="vnstock: down"):
            await market.ticker("FPT", "equity")

    async def test_validation_errors_are_not_served_stale(self, store):
        answers = iter([ticker("vnstock"), Unsupported("no")])
        market = service(FakeSource("vnstock", get_ticker=lambda *a: next(answers)), store=store, chains=Chains(ticker={"equity": ["vnstock"]}))
        await market.ticker("FPT", "equity")
        with pytest.raises(ValidationError):
            await market.ticker("FPT", "equity", skip_cache=True)

    async def test_stale_or_bad_rates_fall_through(self):
        old = ExchangeRate(base="USD", quote="VND", rate=25000, provider="fmarket", timestamp=utc_iso(NOW - timedelta(days=5)))
        current = ExchangeRate(base="USD", quote="VND", rate=26300, provider="yahoo", timestamp=utc_iso(NOW))
        zero = ExchangeRate(base="USD", quote="VND", rate=0, provider="currencyapi", timestamp=utc_iso(NOW))
        market = service(
            FakeSource("fmarket", get_exchange_rate=old),
            FakeSource("yahoo", get_exchange_rate=current),
            FakeSource("currencyapi", get_exchange_rate=zero),
        )
        answer = await market.exchange_rate("usd", "vnd")
        assert answer.provider == "yahoo" and answer.data["from"] == "USD"
        assert answer.fallback[0].startswith("fmarket: rate is stale")
        with pytest.raises(UpstreamError, match="non-positive"):
            await service(FakeSource("currencyapi", get_exchange_rate=zero)).exchange_rate("USDT", "VND")

    async def test_four_hour_candles_are_built_from_hourly_and_empty_answers_fall_through(self):
        empty = FakeSource("vnstock", get_candles=lambda *a: CandleSeries(symbol="FPT", timeframe="1h", provider="vnstock", candles=[]))
        yahoo = FakeSource("yahoo", get_candles=lambda *a: hourly_series("yahoo"))
        answer = await service(empty, yahoo).candles("fpt", "4h", days=10)
        assert answer.fallback == ["vnstock: returned no candles"]
        assert yahoo.calls[0][1][1] is Timeframe.H1
        _, _, start, end = yahoo.calls[0][1]
        assert (end - start, end) == (timedelta(days=10), NOW)
        data = answer.data
        assert (data["symbol"], data["timeframe"], data["provider"]) == ("FPT", "4h", "yahoo")
        # 13:00..18:00 UTC = 20:00..01:00 ICT: buckets 20:00 (4 bars) and 00:00 (2 bars).
        assert [c["time"] for c in data["candles"]] == ["2026-10-12T13:00:00Z", "2026-10-12T17:00:00Z"]
        assert [c["volume"] for c in data["candles"]] == [4, 2]

    async def test_candle_range_is_whole_ict_days(self):
        source = FakeSource("vnstock", get_candles=lambda *a: hourly_series("vnstock"))
        await service(source).candles("FPT", "1d", start="2026-10-01", end="20261005")
        _, _, start, end = source.calls[0][1]
        assert utc_iso(start) == "2026-09-30T17:00:00Z"
        assert utc_iso(end) == "2026-10-05T16:59:59Z"

    async def test_fund_nav_resolves_a_code_to_the_product_id(self):
        source = FakeSource("fmarket", get_fund={"id": 28, "code": "DCDS"}, get_fund_nav=[{"nav": 1}])
        market = service(source)
        assert (await market.fund_nav("dcds")).data == [{"nav": 1}]
        assert (await market.fund_nav("28")).data == [{"nav": 1}]
        assert [c for c in source.calls if c[0] == "get_fund_nav"] == [("get_fund_nav", (28,)), ("get_fund_nav", (28,))]


class TestBankRates:
    async def test_falls_back_to_the_next_source_with_tenors(self, store):
        cafef = FakeSource("cafef", get_bank_rates=RuntimeError("cdn down"))
        simplize = FakeSource("simplize", get_bank_rates=tenor_rates((0, 0.1), (1, 2.2)))
        fmarket = FakeSource("fmarket", get_bank_rates=HEADLINE)
        answer = await service(cafef, simplize, fmarket, store=store).bank_rates()
        assert (answer.provider, answer.fallback) == ("simplize", ["cafef: cdn down"])
        assert [(r["tenor"], r["tenorMonths"], r["instantAccess"]) for r in answer.data] == [("no-term", 0, True), ("1m", 1, False)]
        assert fmarket.calls == []

    async def test_headline_rates_are_the_last_fallback_without_a_filter(self):
        cafef = FakeSource("cafef", get_bank_rates=RuntimeError("cdn down"))
        simplize = FakeSource("simplize", get_bank_rates=ValueError("page changed"))
        fmarket = FakeSource("fmarket", get_bank_rates=HEADLINE)
        answer = await service(cafef, simplize, fmarket).bank_rates()
        assert answer.provider == "fmarket"
        assert answer.data == [{"bank": "Alpha Bank", "tenor": "", "rate": 6.2, "currency": "VND", "updatedAt": "2026-10-05"}]

    async def test_tenor_filter_keeps_one_tenor(self):
        cafef = FakeSource("cafef", get_bank_rates=tenor_rates((0, 0.2), (1, 3.9), (12, 6.1)))
        answer = await service(cafef).bank_rates(tenor_months=1)
        assert [(r["bank"], r["tenorMonths"], r["rate"]) for r in answer.data] == [("Alpha Bank", 1, 3.9)]

    async def test_max_tenor_filter_keeps_instant_access_and_short_terms(self):
        cafef = FakeSource("cafef", get_bank_rates=tenor_rates((0, 0.2), (1, 3.9), (3, 4.1), (12, 6.1)))
        answer = await service(cafef).bank_rates(max_tenor_months=1)
        assert [r["tenorMonths"] for r in answer.data] == [0, 1]

    async def test_filters_skip_sources_without_tenors(self):
        fmarket = FakeSource("fmarket", get_bank_rates=HEADLINE)
        cafef = FakeSource("cafef", get_bank_rates=tenor_rates((1, 3.9)))
        answer = await service(fmarket, cafef, chains=Chains(bank_rates=["fmarket", "cafef"])).bank_rates(tenor_months=1)
        assert (answer.provider, answer.fallback) == ("cafef", [])
        with pytest.raises(ValidationError, match="no rates by tenor"):
            await service(fmarket, chains=Chains(bank_rates=["fmarket"])).bank_rates(max_tenor_months=1)

    async def test_a_tenor_one_source_lacks_moves_to_the_next(self):
        cafef = FakeSource("cafef", get_bank_rates=tenor_rates((1, 3.9), (24, 6.5)))
        simplize = FakeSource("simplize", get_bank_rates=tenor_rates((1, 2.2), (36, 5.5)))
        chains = Chains(bank_rates=["cafef", "simplize"])
        answer = await service(cafef, simplize, chains=chains).bank_rates(tenor_months=36)
        assert (answer.provider, answer.fallback) == ("simplize", ["cafef: publishes no 36-month rates"])
        with pytest.raises(NotFound):
            await service(cafef, simplize, chains=chains).bank_rates(tenor_months=48)

    async def test_filtered_and_full_answers_are_cached_apart(self):
        cafef = FakeSource("cafef", get_bank_rates=lambda: tenor_rates((0, 0.2), (1, 3.9)))
        market = service(cafef, cache=CacheManager(enabled=False))
        assert len((await market.bank_rates()).data) == 2
        assert len((await market.bank_rates(tenor_months=1)).data) == 1
        assert (await market.bank_rates()).cached is True
        assert len(cafef.calls) == 2

    @pytest.mark.parametrize("kwargs", [{"tenor_months": -1}, {"max_tenor_months": 61}, {"tenor_months": 1, "max_tenor_months": 1}])
    async def test_bad_filters_are_rejected_before_routing(self, kwargs):
        cafef = FakeSource("cafef", get_bank_rates=tenor_rates((1, 3.9)))
        with pytest.raises(ValidationError):
            await service(cafef).bank_rates(**kwargs)
        assert cafef.calls == []


def product(provider, rate):
    return FlexibleSaving(
        id=f"{provider.lower()}:p", provider=provider, product="P", rate=rate, backing=SavingsBacking.DEPOSIT,
        earning_starts=EarningStart.NEXT_WORKING_DAY, url=f"https://{provider.lower()}.test/p",
    )  # fmt: skip


class TestFlexibleSavings:
    async def test_every_source_answers_for_its_own_products(self):
        zalopay = FakeSource("zalopay", get_flexible_savings=[product("Zalopay", 4.5)])
        momo = FakeSource("momo", get_flexible_savings=[product("MoMo", 4.0)])
        answer = await service(zalopay, momo).flexible_savings()
        assert [(p["provider"], p["rate"], p["stale"], p["asOf"]) for p in answer.data["products"]] == [
            ("Zalopay", 4.5, False, utc_iso(NOW)),
            ("MoMo", 4.0, False, utc_iso(NOW)),
        ]
        assert (answer.provider, answer.data["failed"], answer.stale) == ("zalopay, momo", [], False)

    async def test_a_failing_source_is_listed_not_fatal(self):
        zalopay = FakeSource("zalopay", get_flexible_savings=ValueError("page changed"))
        momo = FakeSource("momo", get_flexible_savings=[product("MoMo", 4.0)])
        answer = await service(zalopay, momo).flexible_savings()
        assert [p["provider"] for p in answer.data["products"]] == ["MoMo"]
        assert answer.data["failed"] == ["get flexible savings from zalopay failed: zalopay: page changed"]
        assert answer.provider == "momo"

    async def test_a_failing_source_serves_its_own_last_good_copy(self, store):
        answers = iter([[product("Zalopay", 4.5)], RuntimeError("down")])
        zalopay = FakeSource("zalopay", get_flexible_savings=lambda: next(answers))
        momo = FakeSource("momo", get_flexible_savings=[product("MoMo", 4.0)])
        market = service(zalopay, momo, store=store)
        await market.flexible_savings()
        answer = await market.flexible_savings(skip_cache=True)
        assert [(p["provider"], p["stale"]) for p in answer.data["products"]] == [("Zalopay", True), ("MoMo", False)]
        assert (answer.stale, answer.data["failed"]) == (True, [])

    async def test_every_source_failing_is_an_upstream_error(self):
        zalopay = FakeSource("zalopay", get_flexible_savings=RuntimeError("down"))
        momo = FakeSource("momo", get_flexible_savings=RuntimeError("timeout"))
        with pytest.raises(UpstreamError, match="zalopay: down.*momo: timeout"):
            await service(zalopay, momo).flexible_savings()

    async def test_each_source_is_cached_apart(self):
        zalopay = FakeSource("zalopay", get_flexible_savings=lambda: [product("Zalopay", 4.5)])
        momo = FakeSource("momo", get_flexible_savings=lambda: [product("MoMo", 4.0)])
        market = service(zalopay, momo, cache=CacheManager(enabled=False))
        await market.flexible_savings()
        assert (await market.flexible_savings()).cached is True
        assert (len(zalopay.calls), len(momo.calls)) == (1, 1)

    async def test_no_configured_source_is_a_validation_error(self):
        with pytest.raises(ValidationError):
            await service(chains=Chains(flexible_savings=[])).flexible_savings()


class TestApi:
    @pytest.fixture
    def client(self, store):
        sources = [
            FakeSource("vnstock", get_ticker=RuntimeError("quota"), get_candles=lambda *a: hourly_series("vnstock")),
            FakeSource("fmarket", get_ticker=lambda *a: ticker("fmarket"), get_issuers=[{"id": 1}]),
            FakeSource("cafef", get_bank_rates=tenor_rates((0, 0.2), (1, 3.9), (12, 6.1))),
            FakeSource("zalopay", get_flexible_savings=[product("Zalopay", 4.5)]),
        ]
        market = service(*sources, store=store, chains=Chains(flexible_savings=["zalopay"]))
        app.dependency_overrides[get_market_service] = lambda: market
        yield TestClient(app), sources
        app.dependency_overrides.clear()

    def test_envelope_carries_provider_and_fallback(self, client):
        response = client[0].get("/api/v2/ticker", params={"symbol": "fpt", "type": "equity"})
        assert response.status_code == 200
        body = response.json()
        assert body["data"]["provider"] == "fmarket"
        assert body["meta"] == {"provider": "fmarket", "fallback": ["vnstock: quota"], "cached": False, "stale": False, "as_of": utc_iso(NOW)}
        assert "X-Data-Stale" not in response.headers

    def test_stale_answers_set_the_header(self, client):
        http, sources = client
        http.get("/api/v2/ticker", params={"symbol": "FPT", "type": "equity"})
        sources[1].get_ticker = sources[1]._capability("get_ticker", RuntimeError("down"))
        response = http.get("/api/v2/ticker", params={"symbol": "FPT", "type": "equity", "skip_cache": True})
        assert response.status_code == 200
        assert response.json()["meta"]["stale"] is True
        assert response.headers["X-Data-Stale"] == utc_iso(NOW)

    def test_errors_use_the_v2_shape(self, client):
        http, _ = client
        bad = http.get("/api/v2/ticker", params={"symbol": "FPT", "type": "bond"})
        assert bad.status_code == 400 and bad.json()["error"]["code"] == "validation"
        missing = http.get("/api/v2/ticker", params={"type": "equity"})
        assert missing.status_code == 400 and "symbol" in missing.json()["error"]["message"]
        down = http.get("/api/v2/ticker", params={"symbol": "BTC", "type": "crypto"})
        assert down.status_code == 502 and down.json()["error"]["code"] == "upstream"
        unknown = http.get("/api/v2/providers/nope/health")
        assert unknown.status_code == 404

    def test_v1_validation_errors_keep_the_fastapi_shape(self, client):
        response = client[0].get("/api/v1/stocks/quote")
        assert response.status_code == 422 and "detail" in response.json()

    def test_static_fund_paths_win_over_codes(self, client):
        assert client[0].get("/api/v2/funds/issuers").json()["data"] == [{"id": 1}]

    def test_candles_and_providers(self, client):
        http, _ = client
        candles = http.get("/api/v2/candles", params={"symbol": "FPT", "timeframe": "1h", "days": 2}).json()
        assert candles["data"]["timeframe"] == "1h" and len(candles["data"]["candles"]) == 6
        providers = http.get("/api/v2/providers").json()["data"]
        assert providers == [
            {"name": "vnstock", "capabilities": ["ticker", "candles"]},
            {"name": "fmarket", "capabilities": ["ticker"]},
            {"name": "cafef", "capabilities": ["bank_rates"]},
            {"name": "zalopay", "capabilities": ["flexible_savings"]},
        ]
        assert http.get("/api/v2/providers/vnstock/health").json() == {"data": {"provider": "vnstock", "status": "ok"}}

    def test_bank_rates_filter_by_tenor(self, client):
        http, _ = client
        body = http.get("/api/v2/bank-rates", params={"max_tenor_months": 1}).json()
        assert body["meta"]["provider"] == "cafef"
        assert body["data"] == [
            {"bank": "Alpha Bank", "tenor": "no-term", "rate": 0.2, "currency": "VND", "tenorMonths": 0, "channel": "counter", "instantAccess": True},
            {"bank": "Alpha Bank", "tenor": "1m", "rate": 3.9, "currency": "VND", "tenorMonths": 1, "channel": "counter", "instantAccess": False},
        ]
        assert len(http.get("/api/v2/bank-rates").json()["data"]) == 3
        for params in ({"tenor_months": "one"}, {"tenor_months": -1}, {"tenor_months": 1, "max_tenor_months": 3}):
            bad = http.get("/api/v2/bank-rates", params=params)
            assert bad.status_code == 400 and bad.json()["error"]["code"] == "validation"

    def test_flexible_savings(self, client):
        body = client[0].get("/api/v2/flexible-savings").json()
        assert body["meta"]["provider"] == "zalopay"
        assert body["data"] == {
            "products": [
                {
                    "id": "zalopay:p", "provider": "Zalopay", "product": "P", "rate": 4.5, "backing": "deposit",
                    "earningStarts": "next_working_day", "currency": "VND", "url": "https://zalopay.test/p",
                    "asOf": utc_iso(NOW), "stale": False,
                }
            ],
            "failed": [],
        }  # fmt: skip
