"""
Errors of the /api/v2 market layer. Each maps to one HTTP status and error code
(MARKET_PLATFORM.md section 3.1).
"""


class MarketError(Exception):
    status = 500
    code = "internal"


class ValidationError(MarketError):
    """The request is malformed, or no provider in the chain covers it."""

    status = 400
    code = "validation"


class Unsupported(MarketError):
    """
    Raised by an adapter for a symbol, pair, type or timeframe outside its
    upstream. The router skips the adapter instead of counting a failure.
    """

    status = 400
    code = "unsupported"


class NotFound(MarketError):
    status = 404
    code = "not_found"


class QuotaExceeded(MarketError):
    """Our own rate limit, or the upstream's quota, is exhausted."""

    status = 429
    code = "quota_exceeded"


class UpstreamError(MarketError):
    """Every provider failed and there was no last-good copy to serve."""

    status = 502
    code = "upstream"
