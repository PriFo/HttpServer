import asyncio
import json
from pathlib import Path
from typing import Any, Dict

from aiohttp import ClientSession, ClientTimeout, ClientResponse

DATASETS_DIR = Path(__file__).resolve().parent.parent / "datasets"
REQUEST_TIMEOUT = ClientTimeout(total=30)


def _load_payload(path: Path, *, keep_data: bool = False) -> Dict[str, Any]:
    raw = json.loads(path.read_text(encoding="utf-8"))
    if isinstance(raw, list):
        payload: Dict[str, Any] = {"items": raw}
    elif isinstance(raw, dict):
        payload = dict(raw)
        if "items" not in payload and "data" in payload:
            payload["items"] = payload["data"]
        if not keep_data:
            payload.pop("data", None)
    else:
        raise ValueError(f"Unsupported payload format in {path}")
    return payload


async def _read_response(response: ClientResponse) -> dict:
    try:
        return await response.json(content_type=None)
    except Exception:
        text = await response.text()
        raise RuntimeError(f"Unexpected response ({response.status}): {text}") from None


async def send_request(session: ClientSession, url: str, payload: dict | None = None) -> dict:
    print(f"→ {url}")
    if payload is None:
        async with session.get(url, timeout=REQUEST_TIMEOUT) as resp:
            return await _read_response(resp)
    async with session.post(url, json=payload, timeout=REQUEST_TIMEOUT) as resp:
        return await _read_response(resp)


async def main() -> None:
    base_url = "http://localhost:8085"
    endpoints = {
        "features": "/features",
        "metadata": "/metadata",
        "predict": "/predict",
        "quality": "/quality",
        "drift_check": "/drift/check",
        "refresh_baseline": "/drift/baseline",
        "health": "/health",
        "train": "/train",
    }

    train_payload = _load_payload(DATASETS_DIR / "test_dataset.json", keep_data=True)
    quality_payload = _load_payload(DATASETS_DIR / "test_quality.json", keep_data=True)
    predict_payload = _load_payload(DATASETS_DIR / "test_predict.json")

    final_result = {
        "health": None,
        "quality": None,
        "train": None,
        "predict": None,
        "features": None,
        "metadata": None,
    }

    async with ClientSession() as session:
        print("== health ==")
        final_result["health"] = await send_request(session, base_url + endpoints["health"])

        print("== quality ==")
        final_result["quality"] = await send_request(session, base_url + endpoints["quality"], quality_payload)

        print("== train ==")
        final_result["train"] = await send_request(session, base_url + endpoints["train"], train_payload)

        print("== predict ==")
        predict_body = {"items": predict_payload["items"], "top_k": 2, "explain": False}
        final_result["predict"] = await send_request(session, base_url + endpoints["predict"], predict_body)

        print("== features ==")
        final_result["features"] = await send_request(session, base_url + endpoints["features"])

        print("== metadata ==")
        final_result["metadata"] = await send_request(session, base_url + endpoints["metadata"])

        print("== health ==")
        final_result["health"] = await send_request(session, base_url + endpoints["health"])


        output_path = "ml_service/scripts/final_result.json"
        with open(output_path, "w", encoding="utf-8") as f:
            json.dump(final_result, f, ensure_ascii=False, indent=2)
        print(f"Итоговый результат сохранен в {output_path}")


if __name__ == "__main__":
    asyncio.run(main())
