# Tainigo test image

The image comes from Conejo's
[Tainigo sticker](https://code.madriguera.me/conejo/tinyart/raw/branch/main/stickers/tainigo.svg).

The SVG's 60x33 viewBox is rendered on white at 1200x660, then fitted without
cropping or stretching to 233x128 on a white 296x128 landscape canvas.
It uses the same luminance weights and four-shade Floyd-Steinberg conversion
as the photo examples, with no contrast adjustment.

`tainigo.bin` is 9,476 bytes. Its four-byte header is little-endian uint16
width and height (`28 01 80 00`). The remainder is row-major 2bpp data, most
significant pixel first. Shade values are 0 black, 1 dark gray, 2 light gray,
and 3 white.

To regenerate, install Pillow and resvg-py in an isolated Python environment,
download the SVG as `tainigo.svg`, then run from the repository root:

```python
from pathlib import Path
import runpy
import resvg_py

Path("tainigo.png").write_bytes(resvg_py.svg_to_bytes(
    svg_path="tainigo.svg", width=1200, height=660, dpi=96, background="white",
))
convert = runpy.run_path("examples/badger2040/grayscale-slideshow/convert.py")["convert"]
Path("examples/badger2040/grayscale-image/assets/tainigo.bin").write_bytes(
    convert("tainigo.png", 1.0)
)
```

Build the single-image example:

```sh
tinygo build -target=badger2040 -o badger-tainigo.uf2 ./examples/badger2040/grayscale-image
```

It shows the image once using the accepted 10/4-frame dark/light gray
calibration, then enters deep sleep. There is no slideshow or stage pause.
