from __future__ import annotations

import hashlib

import pandas as pd


def data_fingerprint(series: pd.Series) -> str:
    normalized = series.sort_index()
    digest = hashlib.sha256()
    for timestamp, value in normalized.items():
        digest.update(pd.Timestamp(timestamp).isoformat().encode("utf-8"))
        digest.update(b"\0")
        digest.update(format(float(value), ".12g").encode("ascii"))
        digest.update(b"\n")
    return (
        f"{normalized.index[0].isoformat()}|{normalized.index[-1].isoformat()}|"
        f"{len(normalized)}|{digest.hexdigest()}"
    )
