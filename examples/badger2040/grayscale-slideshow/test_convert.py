import struct
import tempfile
import unittest
from pathlib import Path

from PIL import Image

from convert import ASSETS, HEIGHT, WIDTH, convert


def shade(data, x, y):
    index = y * WIDTH + x
    return (data[4 + index // 4] >> (6 - index % 4 * 2)) & 3


class ConversionTests(unittest.TestCase):
    def converted(self, image, contrast=1.0):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "photo.png"
            image.save(source)
            return convert(source, contrast)

    def test_dimensions_packing_and_aspect_ratio(self):
        data = self.converted(Image.new("RGB", (88, 44), "black"))
        self.assertEqual(len(data), 9476)
        self.assertEqual(struct.unpack("<HH", data[:4]), (296, 128))
        for y in range(HEIGHT):
            for x in range(WIDTH):
                self.assertEqual(shade(data, x, y), 0 if 20 <= x < 276 else 3)

    def test_white_and_grayscale_padding(self):
        data = self.converted(Image.new("RGB", (100, 100), (100, 100, 100)))
        used = set()
        for y in range(HEIGHT):
            for x in range(WIDTH):
                value = shade(data, x, y)
                if 84 <= x < 212:
                    used.add(value)
                else:
                    self.assertEqual(value, 3)
        self.assertEqual(used, {1, 2})
        white = self.converted(Image.new("RGB", (100, 100), "white"))
        self.assertEqual(white[4:], bytes([255]) * 9472)

    def test_contrast_and_photo_order(self):
        image = Image.new("RGB", (128, 128), (70, 70, 70))
        normal = self.converted(image)
        boosted = self.converted(image, 1.6)
        normal_sum = sum(shade(normal, x, y) for y in range(128) for x in range(84, 212))
        boosted_sum = sum(shade(boosted, x, y) for y in range(128) for x in range(84, 212))
        self.assertLess(boosted_sum, normal_sum)
        self.assertEqual(
            ASSETS,
            (
                ("00-macintosh-classic.bin", 1.6),
                ("01-guinea-pig.bin", 1.6),
                ("02-cccp-stamp.bin", 1.0),
                ("03-kanagawa.bin", 1.0),
            ),
        )


if __name__ == "__main__":
    unittest.main()
