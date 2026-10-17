"""Fit the supplied test photo to Badger's sixteen-shade packed image format."""

import argparse
from pathlib import Path
import struct

from PIL import Image, ImageOps


WIDTH, HEIGHT = 296, 128


def bayer_rank(x, y):
    tile = ((0, 3), (2, 1))
    rank = 0
    for bit in range(3):
        rank = rank * 4 + tile[(y >> bit) & 1][(x >> bit) & 1]
    return rank


def convert(source):
    with Image.open(source) as original:
        image = ImageOps.contain(
            ImageOps.exif_transpose(original).convert("RGBA"),
            (WIDTH, HEIGHT),
            Image.Resampling.LANCZOS,
        )
        left, top = (WIDTH - image.width) // 2, (HEIGHT - image.height) // 2
        canvas = Image.new("RGBA", (WIDTH, HEIGHT), "white")
        canvas.alpha_composite(image, (left, top))
        canvas = canvas.convert("RGB")

    pixels = canvas.load()
    packed = bytearray(WIDTH * HEIGHT // 2)
    for y in range(HEIGHT):
        for x in range(WIDTH):
            r, g, b = pixels[x, y]
            gray = (77 * r + 150 * g + 29 * b) >> 8
            level = gray // 17
            if gray % 17 * 128 > (2 * bayer_rank(x, y) + 1) * 17:
                level += 1
            index = y * WIDTH + x
            packed[index // 2] |= level << (4 - (index % 2) * 4)
    return struct.pack("<HH", WIDTH, HEIGHT) + packed


def main():
    assets = Path(__file__).parent / "assets"
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", nargs="?", type=Path, default=assets / "person-with-pipe.png")
    parser.add_argument("-o", "--output", type=Path, default=assets / "person-with-pipe.bin")
    args = parser.parse_args()
    data = convert(args.source)
    args.output.write_bytes(data)
    print(f"{args.output.name}: {len(data)} bytes")


if __name__ == "__main__":
    main()
