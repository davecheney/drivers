from io import BytesIO
from pathlib import Path
import runpy
import tempfile
import unittest
import xml.etree.ElementTree as ET

from PIL import Image, ImageOps
import resvg_py


HERE = Path(__file__).resolve().parent
convert = runpy.run_path(str(HERE.parent / "grayscale-slideshow" / "convert.py"))["convert"]
compose = runpy.run_path(str(HERE / "convert.py"))["compose"]
SVG = "{http://www.w3.org/2000/svg}"


class BlackImageTests(unittest.TestCase):
    def test_transparent_source_and_starburst(self):
        source = HERE / "assets/tainigo_nobleed_nobg.svg"
        svg = compose(source)
        self.assertEqual(svg, (HERE / "assets/tainigo.svg").read_text())
        original = ET.parse(source).getroot()
        root = ET.fromstring(svg)
        self.assertEqual((root.get("width"), root.get("height")), ("296", "128"))
        self.assertEqual(root.get("viewBox"), "-8.15625 0 76.3125 33")
        elements = {element.get("id"): element for element in root.iter()}
        for path in original.iter(f"{SVG}path"):
            result = elements[path.get("id")]
            expected = dict(path.attrib)
            expected["style"] = expected["style"].replace("fill:#00459c", "fill:#000000")
            self.assertEqual(result.attrib, expected)
        for layer in ("layer2", "layer3", "layer4"):
            self.assertEqual(len(elements[layer]), 0, f"{layer} added a background or bleed")
        burst = elements["sunburst"]
        self.assertLess(list(root).index(burst), list(root).index(elements["layer1"]))
        rays = burst.find(f"{SVG}g")
        self.assertEqual(rays.get("fill"), "#aaaaaa")
        self.assertEqual(
            [ray.get("transform") for ray in rays],
            [f"rotate({angle})" for angle in range(0, 360, 30)],
        )

    def test_starburst_shows_through_transparent_areas(self):
        root = ET.fromstring(compose(HERE / "assets/tainigo_nobleed_nobg.svg"))
        images = []
        for omit in (None, "sunburst", "layer1"):
            tree = ET.fromstring(ET.tostring(root))
            if omit is not None:
                tree.remove(next(element for element in tree if element.get("id") == omit))
            png = resvg_py.svg_to_bytes(
                svg_string=ET.tostring(tree, encoding="unicode"),
                width=296, height=128, dpi=96,
            )
            images.append(Image.open(BytesIO(png)).convert("RGBA"))
        combined, logo, background = images
        gray_count = 0
        for y in range(128):
            for x in range(40, 256):
                if logo.getpixel((x, y))[3] == 0:
                    self.assertEqual(combined.getpixel((x, y)), background.getpixel((x, y)))
                    if background.getpixel((x, y)) == (170, 170, 170, 255):
                        gray_count += 1
        self.assertGreater(gray_count, 1000)

    def test_solid_black_is_not_dithered(self):
        png = resvg_py.svg_to_bytes(
            svg_path=str(HERE / "assets/tainigo.svg"),
            width=1184, height=512, dpi=96, background="white",
        )
        self.assertEqual(Image.open(BytesIO(png)).size, (1184, 512))
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
