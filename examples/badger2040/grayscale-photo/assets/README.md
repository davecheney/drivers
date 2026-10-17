# Four-shade test photo

This is the same supplied Tommy gun photo as the
[sixteen-shade tester](../../grayscale16/assets/test-photo.png), center-cropped
to 296x128 without stretching or padding. It uses Floyd-Steinberg error diffusion
into black, dark gray, light gray and white, with no contrast boost.

The asset has a four-byte little-endian uint16 width/height header, followed by
row-major packed 2bpp pixels, MSB first. Values 0, 1, 2 and 3 correspond to
luminance 0, 85, 170 and 255. The complete asset is 9,476 bytes.

Regenerate from the repository root with Pillow 13.0.0 in an isolated environment:

```sh
python examples/badger2040/grayscale16/convert.py --bits 2 -o examples/badger2040/grayscale-photo/assets/test-photo.bin
python -m unittest discover -s examples/badger2040/grayscale16 -p test_convert.py
```

Only the packed image is embedded in firmware. The display has no captions
or markers. Up/Down adjust dark gray, A/B adjust light gray, and the starting
timings are D9/L5. Values are printed over serial and reset after a restart.
