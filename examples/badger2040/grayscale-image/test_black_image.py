from io import BytesIO
from pathlib import Path
import runpy
import tempfile
import unittest

from PIL import Image, ImageOps
import resvg_py


HERE = Path(__file__).resolve().parent
convert = runpy.run_path(str(HERE.parent / "grayscale-slideshow" / "convert.py"))["convert"]


class BlackImageTests(unittest.TestCase):
    def test_solid_black_is_not_dithered(self):
        png = resvg_py.svg_to_bytes(
            svg_path=str(HERE / "assets/tainigo.svg"),
            width=1200, height=660, dpi=96, background="white",
        )
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "tainigo.png"
            source.write_bytes(png)
            data = convert(source, 1.0)
        self.assertEqual(data, (HERE / "assets/tainigo.bin").read_bytes())

        image = ImageOps.contain(
            Image.open(BytesIO(png)).convert("RGB"),
            (296, 128), Image.Resampling.LANCZOS,
        )
        canvas = Image.new("RGB", (296, 128), "white")
        canvas.paste(image, ((296 - image.width) // 2, (128 - image.height) // 2))
        pixels = canvas.load()
        black_count = 0
        shades = set()
        for y in range(128):
            for x in range(296):
                index = y * 296 + x
                shade = (data[4 + index // 4] >> (6 - index % 4 * 2)) & 3
                shades.add(shade)
                if pixels[x, y] == (0, 0, 0):
                    self.assertEqual(shade, 0, f"solid black changed at {x},{y}")
                    black_count += 1
        self.assertGreater(black_count, 0)
        self.assertEqual(shades, {0, 1, 2, 3})


if __name__ == "__main__":
    unittest.main()
