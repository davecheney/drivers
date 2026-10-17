import struct
import tempfile
import unittest
from pathlib import Path

from PIL import Image

from convert import HEIGHT, WIDTH, convert, quantize


def shade(data, x, y):
    index = y * WIDTH + x
    return (data[4 + index // 2] >> (4 - index % 2 * 4)) & 15


class ConversionTests(unittest.TestCase):
    def converted(self, image, bits=4):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "test.png"
            image.save(source)
            return convert(source, bits)

    def test_error_diffusion(self):
        self.assertEqual(quantize([8] * 8, 4, 2, 4), bytes([0x01, 0x01, 0x10, 0x10]))
        self.assertEqual(quantize([42] * 8, 4, 2, 2), bytes([0x11, 0x44]))
        self.assertEqual(
            quantize([0, 8, 16, 24, 32, 40, 48, 56], 4, 2, 4),
            bytes([0x00, 0x11, 0x23, 0x33]),
        )

    def test_palette_and_odd_packing(self):
        self.assertEqual(quantize([i * 17 for i in range(16)], 16, 1, 4), bytes.fromhex("0123456789abcdef"))
        self.assertEqual(quantize([0, 85, 170, 255], 4, 1, 2), bytes([0x1B]))
        self.assertEqual(quantize([255], 1, 1, 4), bytes([0xF0]))
        self.assertEqual(quantize([255], 1, 1, 2), bytes([0xC0]))

    def test_diffusion_stays_in_bounds(self):
        values = [255] * 9
        values[4] = 8
        original = values.copy()
        quantize(values, 3, 3, 4, (1, 1, 2, 2))
        self.assertEqual(values, original)

    def test_invalid_quantizer_input(self):
        for values, width, height, bits, bounds in (
            ([], 0, 1, 4, None),
            ([], 1, 1, 4, None),
            ([0], 1, 1, 3, None),
            ([0], 1, 1, 4, (-1, 0, 1, 1)),
            ([0], 1, 1, 4, (0, 0, 2, 1)),
        ):
            with self.assertRaises(ValueError):
                quantize(values, width, height, bits, bounds)

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

    def test_four_shade_conversion(self):
        for color, packed in (("black", 0), ("white", 255), ((85, 85, 85), 0x55)):
            data = self.converted(Image.new("RGB", (400, 400), color), 2)
            self.assertEqual(len(data), 9476)
            self.assertEqual(struct.unpack("<HH", data[:4]), (296, 128))
            self.assertEqual(data[4:], bytes([packed]) * 9472)

    def test_four_shade_crop_and_transparency(self):
        image = Image.new("RGB", (400, 400), (85, 85, 85))
        image.paste("black", (0, 0, 400, 80))
        image.paste("black", (0, 320, 400, 400))
        self.assertEqual(self.converted(image, 2)[4:], bytes([0x55]) * 9472)
        transparent = Image.new("RGBA", (100, 100), (0, 0, 0, 0))
        self.assertEqual(self.converted(transparent, 2)[4:], bytes([255]) * 9472)

    def test_four_shade_photo_regeneration(self):
        assets = Path(__file__).parent / "assets"
        data = convert(assets / "test-photo.png", 2)
        packed = assets.parents[1] / "grayscale-photo/assets/test-photo.bin"
        self.assertEqual(data, packed.read_bytes())
        self.assertEqual(len(data), 9476)
        used = {(value >> shift) & 3 for value in data[4:] for shift in (6, 4, 2, 0)}
        self.assertEqual(used, {0, 1, 2, 3})

    def test_photo_regeneration_and_full_bleed(self):
        assets = Path(__file__).parent / "assets"
        data = convert(assets / "test-photo.png")
        self.assertEqual(data, (assets / "test-photo.bin").read_bytes())
        used = set()
        for y in range(HEIGHT):
            for x in range(WIDTH):
                value = shade(data, x, y)
                used.add(value)
        self.assertEqual(used, set(range(15)))
        for x in (0, WIDTH - 1):
            self.assertTrue(any(shade(data, x, y) != 15 for y in range(HEIGHT)))


if __name__ == "__main__":
    unittest.main()
