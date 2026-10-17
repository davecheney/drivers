# Tainigo test image

The image uses the supplied black-lettering revision of Conejo's
[Tainigo sticker](https://code.madriguera.me/conejo/tinyart/raw/branch/main/stickers/tainigo.svg).
The revised source with a radial sunburst is included as `tainigo.svg`.
The published original has blue lettering and should not be substituted for
this black version.

The SVG viewBox is widened to 76.3125x33 to fill the 296x128 landscape display.
The original logo stays centered at its previous size without cropping or
stretching. Twenty-four alternating white and light-gray radial sectors sit
behind it and extend to the display edges. The SVG is rendered at 1184x512,
then reduced to 296x128.
It uses the same luminance weights and four-shade Floyd-Steinberg conversion
as the photo examples, with no contrast adjustment.
Solid black source pixels remain shade 0. Only the other areas are dithered.

`tainigo.bin` is 9,476 bytes. Its four-byte header is little-endian uint16
width and height (`28 01 80 00`). The remainder is row-major 2bpp data, most
significant pixel first. Shade values are 0 black, 1 dark gray, 2 light gray,
and 3 white.

To regenerate, install Pillow and resvg-py in an isolated Python environment,
then run from the repository root:

```python
from pathlib import Path
import runpy
import resvg_py

Path("tainigo.png").write_bytes(resvg_py.svg_to_bytes(
    svg_path="examples/badger2040/grayscale-image/assets/tainigo.svg",
    width=1184, height=512, dpi=96, background="white",
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

`test_black_image.py` regenerates the asset and checks every solid black
source pixel remains black, while the image still contains all four shades.
