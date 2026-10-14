"""Candle timeframes, cleaning and aggregation."""

from dataclasses import replace
from datetime import datetime, timedelta, timezone
from enum import Enum
from typing import List

from .errors import ValidationError
from .models import Candle

# Vietnam time (UTC+7, no DST): bars are bucketed on HOSE trading days.
ICT = timezone(timedelta(hours=7), "ICT")

MAX_LOOKBACK_DAYS = 10 * 365


class Timeframe(str, Enum):
    H1 = "1h"
    H4 = "4h"
    D1 = "1d"
    W1 = "1w"

    @classmethod
    def parse(cls, raw: str) -> "Timeframe":
        value = (raw or "").strip().lower()
        for known in cls:
            if known.value == value:
                return known
        raise ValidationError("timeframe must be one of 1h, 4h, 1d, 1w")

    @property
    def analysis_lookback_days(self) -> int:
        """Long enough for EMA200 and seasonality: about 200+ bars."""
        return {Timeframe.H1: 60, Timeframe.H4: 180, Timeframe.D1: 2 * 365, Timeframe.W1: 5 * 365}[self]

    @property
    def source(self) -> "Timeframe":
        """The resolution fetched upstream: 4h bars are built from 1h because no source serves them."""
        return Timeframe.H1 if self is Timeframe.H4 else self


def clean_candles(candles: List[Candle]) -> List[Candle]:
    """
    Sort by time, drop bars without a positive close or a time, fill a
    missing open/high/low from the close, and keep the last bar of
    duplicate times.
    """
    cleaned = []
    for candle in candles:
        if candle.time is None or not candle.close or candle.close <= 0:
            continue
        candle = replace(candle)
        if candle.open <= 0:
            candle.open = candle.close
        if candle.high <= 0:
            candle.high = max(candle.open, candle.close)
        if candle.low <= 0:
            candle.low = min(candle.open, candle.close)
        cleaned.append(candle)
    cleaned.sort(key=lambda c: c.time)
    unique: List[Candle] = []
    for candle in cleaned:
        if unique and candle.time == unique[-1].time:
            unique[-1] = candle
        else:
            unique.append(candle)
    return unique


def _bucket_start(local: datetime, timeframe: Timeframe) -> datetime:
    day = local.replace(hour=0, minute=0, second=0, microsecond=0)
    if timeframe is Timeframe.H4:
        return day + timedelta(hours=local.hour // 4 * 4)
    if timeframe is Timeframe.W1:
        return day - timedelta(days=day.weekday())
    return day


def aggregate_candles(candles: List[Candle], timeframe: Timeframe, tz=ICT) -> List[Candle]:
    """
    Merge shorter bars into timeframe bars aligned in tz: 4h buckets start at
    00/04/08/12/16/20 (one morning and one afternoon bar on HOSE), 1d at
    midnight and 1w on Monday. 1h input is returned unchanged.
    """
    if timeframe is Timeframe.H1:
        return candles
    result: List[Candle] = []
    for candle in candles:
        start = _bucket_start(candle.time.astimezone(tz), timeframe)
        if not result or result[-1].time != start:
            result.append(replace(candle, time=start))
            continue
        last = result[-1]
        last.high = max(last.high, candle.high)
        last.low = min(last.low, candle.low)
        last.close = candle.close
        last.volume += candle.volume
    return result
