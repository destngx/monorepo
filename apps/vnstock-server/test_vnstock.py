from vnstock import Vnstock
try:
    stock = Vnstock().stock(symbol='VND', source='VCI')
    df_hist = stock.quote.history(start='2024-01-01', end='2024-01-10')
    print("History:")
    print(df_hist.head())
    
    df_list = stock.listing.all_symbols()
    print("Listing:")
    print(df_list.head())
except Exception as e:
    print("Error:", e)
