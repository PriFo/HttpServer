
from __future__ import annotations

import json
from pathlib import Path

import pandas as pd

def generate_json(csv_path: str, json_path: str, version: str = "test_0_0_1", refresh: bool = True) -> None:
    csv_file = Path(csv_path)
    json_file = Path(json_path)

    if not csv_file.exists():
        raise FileNotFoundError(csv_file)

    df = pd.read_csv(csv_file, encoding="utf-8")
    df = df.astype(object).where(pd.notna(df), None)
    payload = {
        "version": version,
        "refresh_baseline": refresh,
        "data": df.to_dict(orient="records"),
    }
    json_file.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"Generated {json_file} from {csv_file}")

if __name__ == "__main__":
    generate_json("datasets/test_dataset.csv", "datasets/test_dataset.json")