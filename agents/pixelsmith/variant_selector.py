"""Select the best variant from multiple generated images by comparing to reference."""

from __future__ import annotations

import base64
import logging
import urllib.request
from io import BytesIO

from PIL import Image
from langchain_core.messages import HumanMessage, SystemMessage
from langchain_openai import ChatOpenAI
from pydantic import BaseModel, Field

logger = logging.getLogger(__name__)


class _VariantSelection(BaseModel):
    index: int = Field(description="1-based index of the best candidate (e.g. 1, 2, or 3)")


_SELECTOR_SYSTEM = """\
You are selecting the best pixel art sprite from {n} candidates.

You will see:
1. A reference image showing the original weapon/item
2. {n} generated pixel art sprite candidates (labeled 1 through {n})

Pick the candidate whose SHAPE and PROPORTIONS best match the reference weapon. \
Consider:
- Blade shape (curved vs straight, thin vs thick, correct proportions)
- Guard/crossguard shape and size
- Handle length and style
- Overall silhouette similarity to the reference

All candidates should have similar colors. Focus on SHAPE quality.

Return the number of the best candidate (1 through {n}) in the `index` field."""


def _image_to_b64(img: Image.Image) -> str:
    buf = BytesIO()
    img.save(buf, format="PNG")
    return base64.b64encode(buf.getvalue()).decode("ascii")


def _url_to_b64(url: str) -> str:
    req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
    with urllib.request.urlopen(req, timeout=15) as resp:
        return base64.b64encode(resp.read()).decode("ascii")


def select_best_variant(
    candidates: list[Image.Image],
    reference_url: str,
    *,
    model: str = "gpt-4o",
) -> int:
    """Pick the best candidate index (0-based) by comparing to reference.

    Returns the index of the best candidate in the list.
    """
    n = len(candidates)
    if n <= 1:
        return 0

    logger.info("Selecting best variant from %d candidates", n)

    ref_b64 = _url_to_b64(reference_url)

    content: list[dict] = [
        {"type": "text", "text": "Reference image:"},
        {"type": "image_url", "image_url": {"url": f"data:image/png;base64,{ref_b64}", "detail": "high"}},
    ]

    for i, img in enumerate(candidates, 1):
        b64 = _image_to_b64(img)
        content.append({"type": "text", "text": f"Candidate {i}:"})
        content.append({"type": "image_url", "image_url": {"url": f"data:image/png;base64,{b64}", "detail": "high"}})

    llm = ChatOpenAI(model=model, timeout=60).with_structured_output(
        _VariantSelection
    )
    messages = [
        SystemMessage(content=_SELECTOR_SYSTEM.format(n=n)),
        HumanMessage(content=content),
    ]

    result: _VariantSelection = llm.invoke(messages)
    if 1 <= result.index <= n:
        logger.info("LLM selected candidate %d of %d", result.index, n)
        return result.index - 1

    logger.warning("LLM returned out-of-range index %d (n=%d), defaulting to candidate 1", result.index, n)
    return 0
