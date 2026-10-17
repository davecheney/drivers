# Badger four-shade images

The original user-supplied photographs are fitted without cropping or
stretching to a white **296x128** landscape canvas. They are not resized from
the existing 264x176 Waveshare assets.

| Order | File | Image | Contrast |
| --- | --- | --- | --- |
| 1 | `00-macintosh-classic.bin` | Macintosh Classic | 1.6x |
| 2 | `01-guinea-pig.bin` | Guinea pig | 1.6x |
| 3 | `02-cccp-stamp.bin` | Soviet space stamp | Unchanged |
| 4 | `03-kanagawa.bin` | The Great Wave off Kanagawa | Unchanged |

Each file is 9,476 bytes. The first four bytes are width and height as
little-endian uint16 values (`28 01 80 00`). The remaining 9,472 bytes are
row-major 2bpp pixels, four per byte, most significant pixel first.
Values are 0 black, 1 dark gray, 2 light gray, 3 white. For example, `1b`
encodes those four shades in order.

The loader checks dimensions and exact payload length before changing any
pixel or refreshing the screen. Assets are embedded and shown in filename
order, with ten seconds between refreshes. Missing images, malformed assets,
and display errors are reported rather than falling back to monochrome.

## Regenerating

`convert.py` requires Pillow. Install it in an isolated environment, then
provide the four original photographs in the order above:

```sh
python3 -m venv /tmp/badger-images
/tmp/badger-images/bin/pip install Pillow
/tmp/badger-images/bin/python examples/badger2040/grayscale-slideshow/convert.py \
    macintosh.jpeg guinea-pig.jpeg stamp.jpeg kanagawa.jpeg
```

The converter preserves aspect ratio, uses luminance weights 77/150/29,
applies the listed contrast around mid-gray, and uses Floyd-Steinberg
dithering to fixed levels 0/85/170/255. White padding stays white.
The fitted content widths are 169, 192, 178, and 185 pixels, respectively,
all with height 128.

Build offline:

```sh
tinygo build -target=badger2040 -o badger-slideshow.uf2 ./examples/badger2040/grayscale-slideshow
```

This example assumes the original UC8151/IL0373 Badger 2040 panel.
See the [driver notes](../../../../uc8151/README.md) for custom waveform
limitations. The diagnostic was confirmed on a Badger, but this slideshow
has only been built offline.
