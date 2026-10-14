from typing import Any, Dict, List

import pandas as pd


def to_records(data: Any) -> List[Dict[str, Any]]:
    """
    Convert a vnstock result into JSON-safe records.

    Handles the shapes vnstock 4.x returns: DataFrames (including MultiIndex
    columns such as VCI's price board, flattened to "group_field"), Series
    (symbol lists) and NaN/NaT values (returned as null).
    """
    if data is None:
        return []
    if isinstance(data, pd.Series):
        data = data.to_frame(name=data.name or "value")
    if not isinstance(data, pd.DataFrame):
        raise TypeError(f"Unexpected vnstock result type: {type(data).__name__}")

    df = data.copy()
    if isinstance(df.columns, pd.MultiIndex):
        df.columns = ["_".join(str(part) for part in col if str(part)) for col in df.columns]
    for col in df.columns:
        if pd.api.types.is_datetime64_any_dtype(df[col]):
            df[col] = df[col].dt.strftime("%Y-%m-%d %H:%M:%S")
    df = df.astype(object).where(pd.notnull(df), None)
    return df.to_dict(orient="records")
