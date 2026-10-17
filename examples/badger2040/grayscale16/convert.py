"""Center-crop the test photo to Badger's sixteen-shade packed image format."""

import argparse
from pathlib import Path
import runpy
import struct

from PIL import Image, ImageOps


WIDTH, HEIGHT = 296, 128
quantize = runpy.run_path(
    str(Path(__file__).resolve().parents[1] / "internal/dither/floyd_steinberg.py")
)["quantize"]


def convert(source):
    with Image.open(source) as original:
        image = ImageOps.fit(
            ImageOps.exif_transpose(original).convert("RGBA"),
            (WIDTH, HEIGHT),
            Image.Resampling.LANCZOS,
        )
        canvas = Image.new("RGBA", (WIDTH, HEIGHT), "white")
        canvas.alpha_composite(image)
        canvas = canvas.convert("RGB")

    pixels = canvas.load()
    values = [
        (77 * r + 150 * g + 29 * b) >> 8
        for y in range(HEIGHT)
        for x in range(WIDTH)
        for r, g, b in (pixels[x, y],)
    ]
    return struct.pack("<HH", WIDTH, HEIGHT) + quantize(values, WIDTH, HEIGHT, 4)


def main():
    assets = Path(__file__).parent / "assets"
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", nargs="?", type=Path, default=assets / "test-photo.png")
    parser.add_argument("-o", "--output", type=Path, default=assets / "test-photo.bin")
    args = parser.parse_args()
    data = convert(args.source)
    args.output.write_bytes(data)
    print(f"{args.output.name}: {len(data)} bytes")


if __name__ == "__main__":
    main()
