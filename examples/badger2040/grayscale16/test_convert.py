import struct
import tempfile
import unittest
from pathlib import Path

from PIL import Image

from convert import HEIGHT, WIDTH, bayer_rank, convert


def shade(data, x, y):
    index = y * WIDTH + x
    return (data[4 + index // 2] >> (4 - index % 2 * 4)) & 15


class ConversionTests(unittest.TestCase):
    def converted(self, image):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "test.png"
            image.save(source)
            return convert(source)

    def test_bayer_ranks(self):
        self.assertEqual({bayer_rank(x, y) for y in range(8) for x in range(8)}, set(range(64)))
        self.assertEqual([bayer_rank(x, 0) for x in range(8)], [0, 48, 12, 60, 3, 51, 15, 63])

    def test_dimensions_and_full_bleed_black(self):
        data = self.converted(Image.new("RGB", (100, 100), "black"))
        self.assertEqual(len(data), 18948)
        self.assertEqual(struct.unpack("<HH", data[:4]), (296, 128))
        self.assertEqual(data[4:], bytes(18944))

    def test_center_crop_not_stretch_or_pad(self):
        image = Image.new("RGB", (400, 400), (85, 85, 85))
        image.paste("black", (0, 0, 400, 80))
        image.paste("black", (0, 320, 400, 400))
        data = self.converted(image)
        self.assertEqual(data[4:], bytes([0x55]) * 18944)

    def test_transparency_is_white(self):
        data = self.converted(Image.new("RGBA", (100, 100), (0, 0, 0, 0)))
        self.assertEqual(data[4:], bytes([255]) * 18944)

    def test_photo_regeneration_and_full_bleed(self):
        assets = Path(__file__).parent / "assets"
        data = convert(assets / "test-photo.jpg")
        self.assertEqual(data, (assets / "test-photo.bin").read_bytes())
        used = set()
        for y in range(HEIGHT):
            for x in range(WIDTH):
                value = shade(data, x, y)
                used.add(value)
        self.assertEqual(used, set(range(16)))
        for x in (0, WIDTH - 1):
            self.assertTrue(any(shade(data, x, y) != 15 for y in range(HEIGHT)))


if __name__ == "__main__":
    unittest.main()
