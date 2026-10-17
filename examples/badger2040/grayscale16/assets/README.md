# Sixteen-shade test image

`test-photo.png` is the supplied 296x177 Tommy gun photo. It is center-cropped
to 296x128 without stretching or adding borders. Source orientation is
preserved after applying any EXIF orientation. Transparency is composited
onto white.

Conversion uses luminance weights 77/150/29 and an 8x8 Bayer pattern for
sixteen shades, without a contrast boost. Solid black and white stay solid.
This crop uses shades 0-14; no pure-white padding is added.

`test-photo.bin` has a four-byte little-endian uint16 width/height
header (`28 01 80 00`) followed by row-major packed 4bpp pixels. The high
nibble comes first. Shade 0 is black and shade 15 is white. The complete
asset is 18,948 bytes.

Regenerate from the repository root with Pillow 13.0.0 in an isolated
environment:

```sh
python examples/badger2040/grayscale16/convert.py
python -m unittest discover -s examples/badger2040/grayscale16 -p test_convert.py
```

The firmware embeds only the packed asset. The original PNG and converter
are not needed at runtime. There are no captions or control markers on
the display. Selection and pulse values are printed over serial.
