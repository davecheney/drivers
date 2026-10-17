# Tainigo test image

The image uses Conejo's transparent, no-bleed
[Tainigo sticker](https://code.madriguera.me/conejo/tinyart/raw/branch/main/stickers/tainigo_nobleed_nobg.svg).
The unmodified source is included as `tainigo_nobleed_nobg.svg`.
`convert.py` generates `tainigo.svg` with a radial starburst behind it.
It maps the blue lettering and shadow to solid black for this display.
The original white outline and all path geometry are preserved. There is
no opaque sticker background, bleed or cut outline hiding the rays.

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

To regenerate, install Pillow 13.0.0 and resvg-py 0.5.0 in an isolated environment,
then run from the repository root:

```sh
python examples/badger2040/grayscale-image/convert.py
python -m unittest discover -s examples/badger2040/grayscale-image -p test_black_image.py
```

Build the single-image example:

```sh
tinygo build -target=badger2040 -o badger-tainigo.uf2 ./examples/badger2040/grayscale-image
```

It shows the image once using the accepted 9/5-frame dark/light gray
calibration, then enters deep sleep. There is no slideshow or stage pause.

`test_black_image.py` checks source geometry, starburst coverage through the
transparent areas and exact asset regeneration. Every solid black source
pixel remains black, while the image still contains all four shades.
