# Sixteen-shade test image

`person-with-pipe.png` is the supplied "Person with Pipe" image. Its
296x177 frame is fitted without cropping or stretching to 214x128, centered
on a white 296x128 canvas. Source orientation is preserved after applying
any EXIF orientation. Transparency is composited onto white.

Conversion uses luminance weights 77/150/29 and an 8x8 Bayer pattern for
sixteen shades, without a contrast boost. Solid black and white stay solid.

`person-with-pipe.bin` has a four-byte little-endian uint16 width/height
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
