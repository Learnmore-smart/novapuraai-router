"""Generate optimized NovaPuraAI logo/favicon assets for web/default/public."""
from __future__ import annotations

import shutil
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
SRC_LOGO = ROOT / "public" / "logo" / "novapuraai_logo_router_2026-7-14.png"
SRC_FAV = ROOT / "public" / "logo" / "novapuraai_router_favicon"
OUT_WEB = ROOT / "web" / "default" / "public"
OUT_LOGO_DIR = ROOT / "public" / "logo"


def save_png(im: Image.Image, path: Path, size: int | None = None) -> None:
    work = im.copy()
    if size:
        work = work.resize((size, size), Image.Resampling.LANCZOS)
    if work.mode == "RGBA":
        alpha = work.split()[-1]
        if alpha.getextrema()[0] >= 250:
            work = work.convert("RGB")
    work.save(path, format="PNG", optimize=True)
    print(f"  {path.name}: {work.size} {path.stat().st_size / 1024:.1f} KB")


def main() -> None:
    OUT_WEB.mkdir(parents=True, exist_ok=True)
    img = Image.open(SRC_LOGO).convert("RGBA")

    print("Generating optimized logos...")
    save_png(img, OUT_WEB / "logo.png", 512)
    save_png(img, OUT_WEB / "logo-256.png", 256)
    save_png(img, OUT_LOGO_DIR / "novapuraai_logo_512.png", 512)
    save_png(img, OUT_LOGO_DIR / "novapuraai_logo_256.png", 256)

    try:
        for size, name in ((512, "logo.webp"), (256, "logo-256.webp")):
            w = img.resize((size, size), Image.Resampling.LANCZOS).convert("RGB")
            p = OUT_WEB / name
            w.save(p, format="WEBP", quality=88, method=6)
            print(f"  {name}: {size}x{size} {p.stat().st_size / 1024:.1f} KB")
    except Exception as e:  # noqa: BLE001
        print("webp skip", e)

    print("Copying favicon.ico + small PNGs...")
    for name in ("favicon.ico", "favicon-16x16.png", "favicon-32x32.png"):
        s = SRC_FAV / name
        d = OUT_WEB / name
        shutil.copy2(s, d)
        print(f"  {name}: {d.stat().st_size / 1024:.1f} KB")

    # Re-encode touch/PWA icons from master for smaller PNG size
    print("Re-encoding touch / PWA icons from master logo...")
    for size, name in (
        (180, "apple-touch-icon.png"),
        (192, "android-chrome-192x192.png"),
        (512, "android-chrome-512x512.png"),
    ):
        save_png(img, OUT_WEB / name, size)

    print("done")


if __name__ == "__main__":
    main()
