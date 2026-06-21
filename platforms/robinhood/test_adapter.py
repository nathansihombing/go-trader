"""Tests for RobinhoodExchangeAdapter — mock robin_stocks and yfinance."""

import sys
import os
import importlib.util
import pytest
from unittest.mock import MagicMock, patch

# Load robinhood adapter by file path to avoid module name collisions
_adapter_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "adapter.py")
_shared_tools = os.path.abspath(os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', '..', 'shared_tools'))
if _shared_tools not in sys.path:
    sys.path.insert(0, _shared_tools)


def _load_rh_module():
    """Load the robinhood adapter module fresh."""
    spec = importlib.util.spec_from_file_location("rh_adapter", _adapter_path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


_mod = _load_rh_module()
RobinhoodExchangeAdapter = _mod.RobinhoodExchangeAdapter
_get_strike_interval = _mod._get_strike_interval


@pytest.fixture
def paper_adapter():
    """Create a paper-mode adapter (no login attempt)."""
    with patch.dict(os.environ, {}, clear=False):
        for key in ("ROBINHOOD_USERNAME", "ROBINHOOD_PASSWORD", "ROBINHOOD_TOTP_SECRET"):
            os.environ.pop(key, None)
        return RobinhoodExchangeAdapter(mode="paper")


# ─── Properties ────────────────────────────────────

class TestProperties:
    def test_name(self, paper_adapter):
        assert paper_adapter.name == "robinhood"

    def test_paper_mode(self, paper_adapter):
        assert paper_adapter.mode == "paper"
        assert paper_adapter.is_live is False

    def test_live_mode_no_creds_raises(self):
        with patch.dict(os.environ, {}, clear=False):
            for key in ("ROBINHOOD_USERNAME", "ROBINHOOD_PASSWORD", "ROBINHOOD_TOTP_SECRET"):
                os.environ.pop(key, None)
            with pytest.raises(RuntimeError, match="Live mode requires"):
                RobinhoodExchangeAdapter(mode="live")


# ─── Symbol Resolution ────────────────────────────

class TestSymbolResolution:
    def test_crypto_resolves_to_yahoo(self, paper_adapter):
        assert paper_adapter._resolve_yahoo_symbol("BTC") == "BTC-USD"
        assert paper_adapter._resolve_yahoo_symbol("ETH") == "ETH-USD"

    def test_stock_passes_through(self, paper_adapter):
        assert paper_adapter._resolve_yahoo_symbol("SPY") == "SPY"
        assert paper_adapter._resolve_yahoo_symbol("AAPL") == "AAPL"


# ─── Market Data ───────────────────────────────────

class TestMarketData:
    def test_get_spot_price_alias(self, paper_adapter):
        """get_spot_price should delegate to get_price."""
        with patch.object(paper_adapter, "get_price", return_value=42000.0):
            assert paper_adapter.get_spot_price("BTC") == 42000.0

    def test_get_ohlcv_delegates_to_yahoo(self, paper_adapter):
        with patch.object(paper_adapter, "_get_yahoo_ohlcv", return_value=[[0, 1, 2, 3, 4, 5]]) as mock:
            result = paper_adapter.get_ohlcv("BTC", "1h", 100)
            mock.assert_called_once_with("BTC", "1h", 100)
            assert result == [[0, 1, 2, 3, 4, 5]]

    def test_get_ohlcv_closes(self, paper_adapter):
        candles = [
            [0, 100, 110, 90, 105, 50],
            [1, 105, 115, 95, 110, 60],
        ]
        with patch.object(paper_adapter, "_get_yahoo_ohlcv", return_value=candles):
            closes = paper_adapter.get_ohlcv_closes("BTC", "4h", 100, min_len=1)
            assert closes == [105, 110]

    def test_get_ohlcv_closes_insufficient(self, paper_adapter):
        with patch.object(paper_adapter, "_get_yahoo_ohlcv", return_value=[]):
            assert paper_adapter.get_ohlcv_closes("BTC", "4h", 100) is None


# ─── Strike Interval ──────────────────────────────

class TestStrikeInterval:
    def test_under_100(self):
        assert _get_strike_interval(50) == 1

    def test_100_to_500(self):
        assert _get_strike_interval(200) == 5

    def test_over_500(self):
        assert _get_strike_interval(600) == 10


# ─── Options Protocol ──────────────────────────────

class TestOptionsProtocol:
    def test_get_real_expiry_paper(self, paper_adapter):
        expiry, dte = paper_adapter.get_real_expiry("SPY", 30)
        assert dte == 30
        # Should be a valid YYYY-MM-DD
        from datetime import datetime
        datetime.strptime(expiry, "%Y-%m-%d")

    def test_get_real_strike_paper_stock(self, paper_adapter):
        # SPY at ~450 => $5 intervals
        strike = paper_adapter.get_real_strike("SPY", "2026-05-01", "call", 453.0)
        assert strike == 455.0  # round to nearest 5

    def test_get_real_strike_paper_low_price(self, paper_adapter):
        # Stock under $100 => $1 intervals
        strike = paper_adapter.get_real_strike("XYZ", "2026-05-01", "call", 42.3)
        assert strike == 42.0

    def test_get_premium_and_greeks_paper_bs(self, paper_adapter):
        """Paper mode uses Black-Scholes."""
        pct, usd, greeks = paper_adapter.get_premium_and_greeks(
            "SPY", "call", 450, "2026-05-01", 30, 445, 0.20
        )
        assert usd > 0
        assert "delta" in greeks
        # For stock options, premium is per-contract (x100)
        assert usd >= pct * 445 * 90  # rough sanity check


# ─── Order Execution ──────────────────────────────

class TestOrderExecution:
    def test_market_buy_paper_raises(self, paper_adapter):
        with pytest.raises(RuntimeError, match="live mode"):
            paper_adapter.market_buy("BTC", 1000)

    def test_market_sell_paper_raises(self, paper_adapter):
        with pytest.raises(RuntimeError, match="live mode"):
            paper_adapter.market_sell("BTC", 0.5)

    def test_get_crypto_positions_not_logged_in(self, paper_adapter):
        assert paper_adapter.get_crypto_positions() == []


def _fake_robinhood_module():
    import types

    return types.SimpleNamespace(
        orders=types.SimpleNamespace(
            order_buy_crypto_by_price=MagicMock(return_value={"id": "crypto-buy"}),
            order_sell_crypto_by_quantity=MagicMock(return_value={"id": "crypto-sell"}),
            order_buy_fractional_by_price=MagicMock(return_value={"id": "stock-buy"}),
            order_sell_fractional_by_quantity=MagicMock(return_value={"id": "stock-sell"}),
        ),
        account=types.SimpleNamespace(
            build_holdings=MagicMock(return_value={
                "AAPL": {"quantity": "1.25", "average_buy_price": "190.50"},
                "CASH": {"quantity": "0", "average_buy_price": "1"},
            })
        ),
        crypto=types.SimpleNamespace(get_crypto_positions=MagicMock(return_value=[])),
    )


def _live_adapter_without_login():
    adapter = RobinhoodExchangeAdapter(mode="paper")
    adapter._mode = "live"
    adapter._logged_in = True
    return adapter


class TestDirectStockShares:
    def test_market_buy_routes_stock_to_fractional_dollar_order(self):
        fake_rh = _fake_robinhood_module()
        adapter = _live_adapter_without_login()
        with patch.dict(sys.modules, {"robin_stocks.robinhood": fake_rh}):
            result = adapter.market_buy("aapl", 250.0)
        assert result == {"id": "stock-buy"}
        fake_rh.orders.order_buy_fractional_by_price.assert_called_once_with("AAPL", 250.0)
        fake_rh.orders.order_buy_crypto_by_price.assert_not_called()

    def test_market_sell_routes_stock_to_fractional_quantity_order(self):
        fake_rh = _fake_robinhood_module()
        adapter = _live_adapter_without_login()
        with patch.dict(sys.modules, {"robin_stocks.robinhood": fake_rh}):
            result = adapter.market_sell("msft", 0.75)
        assert result == {"id": "stock-sell"}
        fake_rh.orders.order_sell_fractional_by_quantity.assert_called_once_with("MSFT", 0.75)
        fake_rh.orders.order_sell_crypto_by_quantity.assert_not_called()

    def test_market_buy_keeps_crypto_on_crypto_endpoint(self):
        fake_rh = _fake_robinhood_module()
        adapter = _live_adapter_without_login()
        with patch.dict(sys.modules, {"robin_stocks.robinhood": fake_rh}):
            result = adapter.market_buy("BTC", 100.0)
        assert result == {"id": "crypto-buy"}
        fake_rh.orders.order_buy_crypto_by_price.assert_called_once_with("BTC", 100.0)
        fake_rh.orders.order_buy_fractional_by_price.assert_not_called()

    def test_get_stock_positions_strict_reads_direct_share_holdings(self):
        fake_rh = _fake_robinhood_module()
        adapter = _live_adapter_without_login()
        with patch.dict(sys.modules, {"robin_stocks.robinhood": fake_rh}):
            positions = adapter.get_stock_positions_strict()
        assert positions == [{"symbol": "AAPL", "quantity": 1.25, "avg_price": 190.5}]
