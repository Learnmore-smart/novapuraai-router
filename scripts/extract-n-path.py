"""Roughly extract the white N silhouette bounds from the brand logo (debug aid)."""
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
src = ROOT / "public" / "logo" / "novapuraai_logo_router_2026-7-14.png"
img = Image.open(src).convert("RGB")
# Work on 32x32 grid for SVG path intuition
small = img.resize((32, 32), Image.Resampling.LANCZOS)
for y in range(32):
    row = []
    for x in range(32):
        r, g, b = small.getpixel((x, y))
        # white N
        if r > 230 and g > 230 and b > 230:
            row.append("#")
        else:
            row.append(".")
    print("".join(row))
