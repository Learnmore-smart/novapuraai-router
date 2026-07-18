"""Generate rounded favicons that fill the browser tab icon area.

Pipeline:
1. Load source logo
2. Alpha-trim transparent padding
3. Center on a solid copper field so the mark fills the canvas
4. Scale to ~94% of the canvas (avoids empty ring / over-crop)
5. Soft rounded mask (~20% radius) — single corner radius only
"""
from __future__ import annotations

from pathlib import Path

from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "web" / "default" / "public"
SRC = ROOT / "public" / "logo" / "novapuraai_logo_router_2026-7-14.png"
if not SRC.exists():
    SRC = OUT / "logo.png"

# Match monogram / primary brand copper used in assets/logo.tsx
COPPER = (196, 90, 18, 255)  # #c45a12


def alpha_bbox(im: Image.Image, threshold: int = 12) -> tuple[int, int, int, int]:
    """Bounding box of non-transparent pixels, with small threshold for soft edges."""
    if im.mode != "RGBA":
        im = im.convert("RGBA")
    alpha = im.split()[-1]
    # Binarize soft alpha so anti-aliased edges still count
    mask = alpha.point(lambda a: 255 if a > threshold else 0)
    box = mask.getbbox()
    if box is None:
        return (0, 0, im.width, im.height)
    return box


def fit_on_copper(size: int, fill_ratio: float = 0.94) -> Image.Image:
    """Place the trimmed mark on a solid copper square, filling most of the canvas."""
    src = Image.open(SRC).convert("RGBA")
    left, top, right, bottom = alpha_bbox(src)
    cropped = src.crop((left, top, right, bottom))

    # Square canvas, solid copper (tab icons look empty when mark sits on transparency)
    canvas = Image.new("RGBA", (size, size), COPPER)

    target = max(1, int(size * fill_ratio))
    cw, ch = cropped.size
    scale = min(target / cw, target / ch)
    nw = max(1, int(cw * scale))
    nh = max(1, int(ch * scale))
    resized = cropped.resize((nw, nh), Image.Resampling.LANCZOS)

    # If the source already is a full copper square mark, just scale it
    # Otherwise composite the glyph (preserve alpha) onto copper.
    ox = (size - nw) // 2
    oy = (size - nh) // 2
    canvas.alpha_composite(resized, (ox, oy))
    return canvas


def rounded(size: int, radius_ratio: float = 0.20, fill_ratio: float = 0.94) -> Image.Image:
    work = fit_on_copper(size, fill_ratio=fill_ratio)
    mask = Image.new("L", (size, size), 0)
    draw = ImageDraw.Draw(mask)
    r = max(2, int(size * radius_ratio))
    draw.rounded_rectangle((0, 0, size - 1, size - 1), radius=r, fill=255)
    out = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    out.paste(work, (0, 0))
    out.putalpha(mask)
    return out


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    print(f"source: {SRC}")

    for size, name in (
        (16, "favicon-16x16.png"),
        (32, "favicon-32x32.png"),
        (180, "apple-touch-icon.png"),
        (192, "android-chrome-192x192.png"),
        (512, "android-chrome-512x512.png"),
    ):
        # Smaller sizes: slightly higher fill so 16px doesn't look sparse
        fill = 0.96 if size <= 32 else 0.94
        path = OUT / name
        rounded(size, fill_ratio=fill).save(path, format="PNG", optimize=True)
        print(f"  {name}: {path.stat().st_size / 1024:.1f} KB")

    ico_sizes = [16, 32, 48]
    ico_images = [rounded(s, fill_ratio=0.96 if s <= 32 else 0.94) for s in ico_sizes]
    ico_path = OUT / "favicon.ico"
    ico_images[0].save(
        ico_path,
        format="ICO",
        sizes=[(s, s) for s in ico_sizes],
        append_images=ico_images[1:],
    )
    print(f"  favicon.ico: {ico_path.stat().st_size / 1024:.1f} KB")
    (OUT / "favicon-20260715.ico").write_bytes(ico_path.read_bytes())

    # Raster logos used when SVG monogram is not available (admin/custom fallbacks)
    logo = rounded(512, radius_ratio=0.18, fill_ratio=0.94)
    logo.save(OUT / "logo.png", format="PNG", optimize=True)
    logo.resize((256, 256), Image.Resampling.LANCZOS).save(
        OUT / "logo-256.png", format="PNG", optimize=True
    )
    print("  logo.png / logo-256.png updated")

    print("done")


if __name__ == "__main__":
    main()
