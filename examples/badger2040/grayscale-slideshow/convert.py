"""Fit the four supplied photos to Badger's landscape four-shade asset format."""

import argparse
from pathlib import Path
import struct

from PIL import Image, ImageOps


WIDTH, HEIGHT = 296, 128
ASSETS = (
    ("00-macintosh-classic.bin", 1.6),
    ("01-guinea-pig.bin", 1.6),
    ("02-cccp-stamp.bin", 1.0),
    ("03-kanagawa.bin", 1.0),
)


def convert(source, contrast):
    with Image.open(source) as original:
        image = ImageOps.contain(
            ImageOps.exif_transpose(original).convert("RGB"),
            (WIDTH, HEIGHT),
            Image.Resampling.LANCZOS,
        )
        width, height = image.size
        left, top = (WIDTH - width) // 2, (HEIGHT - height) // 2
        canvas = Image.new("RGB", (WIDTH, HEIGHT), "white")
        canvas.paste(image, (left, top))

    pixels = canvas.load()
    values = [
        max(0, min(255, (((77 * r + 150 * g + 29 * b) >> 8) - 128) * contrast + 128))
        for y in range(HEIGHT)
        for x in range(WIDTH)
        for r, g, b in (pixels[x, y],)
    ]
    packed = bytearray(WIDTH * HEIGHT // 4)
    for y in range(HEIGHT):
        for x in range(WIDTH):
            index = y * WIDTH + x
            value = max(0, min(255, values[index]))
            shade = min(3, int(value / 85 + 0.5))
            packed[index // 4] |= shade << (6 - (index % 4) * 2)
            error = value - shade * 85
            for dx, dy, weight in ((1, 0, 7), (-1, 1, 3), (0, 1, 5), (1, 1, 1)):
                nx, ny = x + dx, y + dy
                if left <= nx < left + width and top <= ny < top + height:
                    values[ny * WIDTH + nx] += error * weight / 16
    return struct.pack("<HH", WIDTH, HEIGHT) + packed


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("photos", nargs=4, type=Path, metavar="PHOTO")
    args = parser.parse_args()
    output = Path(__file__).parent / "assets"
    output.mkdir(exist_ok=True)
    for source, (name, contrast) in zip(args.photos, ASSETS):
        data = convert(source, contrast)
        (output / name).write_bytes(data)
        print(f"{name}: {len(data)} bytes")


if __name__ == "__main__":
    main()
