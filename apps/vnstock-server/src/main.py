from fastapi import FastAPI, HTTPException, Query
from vnstock import Vnstock
from typing import Optional, List, Dict, Any
import pandas as pd
from datetime import datetime, timedelta

app = FastAPI(
    title="Vnstock API Wrapper",
    description="A FastAPI wrapper for the vnstock library to provide Vietnam stock market data.",
    version="0.1.0"
)

@app.get("/health")
def health_check():
    return {"status": "ok", "service": "vnstock-server", "timestamp": datetime.now().isoformat()}

@app.get("/")
def read_root():
    return {"message": "Welcome to Vnstock API Wrapper. Access /docs for API documentation."}

@app.get("/api/v1/stocks/listing")
def get_listing():
    """
    Get a list of all companies listed on the Vietnamese stock market.
    """
    try:
        df = Vnstock().stock(symbol='VND', source='VCI').listing.all_symbols()
        return {"success": True, "data": df.to_dict(orient='records')}
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error fetching stock listing: {str(e)}")

@app.get("/api/v1/stocks/quote")
def get_quote(symbol: str = Query(..., description="Ticker symbol")):
    """
    Get the latest price and metadata for a specific symbol.
    """
    try:
        # Get historical data for the last 1 day to fetch the most recent price
        df = Vnstock().stock(symbol=symbol, source='VCI').quote.history(
            start=(datetime.now() - timedelta(days=7)).strftime("%Y-%m-%d"),
            end=datetime.now().strftime("%Y-%m-%d")
        )
        
        if df.empty:
            raise HTTPException(status_code=404, detail=f"No data found for symbol {symbol}")
            
        latest = df.iloc[-1].to_dict()
        
        # Ensure time column is string
        if 'time' in latest and isinstance(latest['time'], (pd.Timestamp, datetime)):
            latest['time'] = latest['time'].strftime('%Y-%m-%d %H:%M:%S')
            
        return {
            "success": True, 
            "data": {
                "symbol": symbol,
                "price": latest.get('close'),
                "change": 0, # Could be calculated if needed
                "time": latest.get('time'),
                "name": symbol # listing_companies() could be used here but it's slow to call every time
            }
        }
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error fetching quote for {symbol}: {str(e)}")

@app.get("/api/v1/stocks/historical")
def get_historical_data(
    symbol: str = Query(..., description="Ticker symbol (e.g., GMD, VCB)"),
    start_date: Optional[str] = Query(None, description="Start date in YYYY-MM-DD format"),
    end_date: Optional[str] = Query(None, description="End date in YYYY-MM-DD format"),
    resolution: str = Query("1D", description="Data resolution (1M, 5M, 15M, 30M, 1H, 1D)")
):
    """
    Get historical stock data for a specific symbol.
    """
    if not end_date:
        end_date = datetime.now().strftime("%Y-%m-%d")
    if not start_date:
        start_date = (datetime.now() - timedelta(days=30)).strftime("%Y-%m-%d")

    try:
        df = Vnstock().stock(symbol=symbol, source='VCI').quote.history(
            start=start_date,
            end=end_date
        )
        
        if df.empty:
            return {"success": True, "data": []}
            
        # Ensure time column is string
        if 'time' in df.columns:
            if pd.api.types.is_datetime64_any_dtype(df['time']):
                df['time'] = df['time'].dt.strftime('%Y-%m-%d %H:%M:%S')
            else:
                # In some versions, 'time' is already a string but could be in different format
                # We normalize it here
                pass
            
        return {"success": True, "data": df.to_dict(orient='records')}
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error fetching historical data for {symbol}: {str(e)}")
