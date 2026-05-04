#!/usr/bin/env python3
"""
Generate all required favicon and PWA icons from a source PNG.
Usage: python3 scripts/generate-icons.py public/app-icon-source.png
"""
import sys
import os
from pathlib import Path
from PIL import Image

BRAND_COLOR = (59, 183, 65, 255)  # #3bb741


def make_maskable(img: Image.Image, size: int = 512, safe_zone_pct: float = 0.1) -> Image.Image:
    """
    Build a maskable icon: brand-colour background with the logo scaled to
    fit within the safe zone (central 80 % of the canvas).
    """
    inner = int(size * (1 - 2 * safe_zone_pct))
    logo = img.copy()
    logo.thumbnail((inner, inner), Image.LANCZOS)

    canvas = Image.new("RGBA", (size, size), BRAND_COLOR)
    offset = ((size - logo.width) // 2, (size - logo.height) // 2)
    canvas.paste(logo, offset, logo if logo.mode == "RGBA" else None)
    return canvas

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 scripts/generate-icons.py <source-image.png>")
        sys.exit(1)

    src_path = Path(sys.argv[1])
    if not src_path.exists():
        print(f"Source image not found: {src_path}")
        sys.exit(1)

    out_dir = Path("public/icons")
    out_dir.mkdir(parents=True, exist_ok=True)

    src = Image.open(src_path).convert("RGBA")

    sizes = {
        "favicon-16.png":        (16, 16),
        "favicon-32.png":        (32, 32),
        "apple-touch-icon.png":  (180, 180),
        "icon-192.png":          (192, 192),
        "icon-512.png":          (512, 512),
    }

    for name, size in sizes.items():
        img = src.copy()
        img.thumbnail(size, Image.LANCZOS)

        canvas = Image.new("RGBA", size, BRAND_COLOR)
        offset = ((size[0] - img.width) // 2, (size[1] - img.height) // 2)
        canvas.paste(img, offset, img if img.mode == "RGBA" else None)
        canvas.save(out_dir / name, "PNG")
        print(f"  created {out_dir / name}")

    # Maskable icon: brand-colour background, logo inside the safe zone
    maskable = make_maskable(src, size=512, safe_zone_pct=0.1)
    maskable.save(out_dir / "icon-512-maskable.png", "PNG")
    print(f"  created {out_dir / 'icon-512-maskable.png'}")

    print("\nAll icons generated successfully.")

if __name__ == "__main__":
    main()
