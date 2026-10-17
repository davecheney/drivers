# UC8151

The existing `New` and `Config` API remains monochrome. `NewGrayscale` is a
separate four-shade driver for the **128x296 UC8151/IL0373 panel on the original
Pimoroni Badger 2040**. It implements `drivers.Displayer` with a 9,472-byte
packed 2bpp framebuffer. Rotation90 and Rotation270 expose 296x128 landscape.

This is not the SSD1680 used by Waveshare's `epd2in9v2` driver. A build for the
`badger2040` target does not identify the connected controller. Do not use this
driver on an SSD1680, a replacement panel, or a different Badger revision
without checking its controller. No controller auto-detection is performed.

## Four shades

```go
display := uc8151.NewGrayscale(spi, cs, dc, rst, busy)
if err := display.Configure(uc8151.GrayscaleConfig{
    Rotation: drivers.Rotation270,
}); err != nil {
    return err
}
display.SetPixel(10, 10, color.RGBA{85, 85, 85, 255})
return display.Display()
```

Configure the SPI bus and pins before use. BUSY is active LOW. The default
timeout is 30 seconds per wait and can be changed with `BusyTimeout`. SPI
failures and BUSY timeouts are returned. `OnInitStage` optionally traces
initialization.

RGB luminance is quantized to black, dark gray, light gray, and white.
Alpha is ignored. `ClearBuffer` clears only the framebuffer. `ClearDisplay`
refreshes the panel to white without changing the framebuffer.
`DeepSleep` retains the image and requires `Configure` before another refresh.

Each `Display` first performs a full white reset, then applies three
blackening pulses selected by the old/new RAM bits:

| Shade | Value | Old RAM bit | New RAM bit | LUT | Blackening frames |
| --- | --- | --- | --- | --- | --- |
| Black | 0 | 0 | 1 | WB | 32 |
| Dark gray | 1 | 1 | 1 | BB | 10 |
| Light gray | 2 | 0 | 0 | WW | 4 |
| White | 3 | 1 | 0 | BW | 0 |

The waveform clock is 100Hz, the VCOM LUT is 44 bytes, and each pixel LUT is
42 bytes. The reset waveforms and grayscale algorithm follow the Badger-tested
[MicroPython implementation](https://github.com/antirez/uc8151_micropython/blob/cc2a0de91386f08ee6c08e6e315bcfef850ffc49/uc8151.py).
Its MIT license is retained in `LICENSE-grayscale`.

The source's 21/10-frame dark/light pulses looked too dark on the operator's
Badger. The defaults above shorten them to 10/4 frames. Use `DarkGrayFrames`
and `LightGrayFrames` in `GrayscaleConfig` to tune another panel. Zero selects
the default. The light duration must be less than dark, and dark must be
less than the fixed 32-frame black duration. White receives no black pulse.
The operator accepted the 10/4-frame calibration on their Badger 2040.

**This mode uses experimental custom waveforms, not vendor-supported
temperature-compensated grayscale.** The timings were developed for room
temperature. Contrast, gray separation, ghosting, and long-term panel behavior
require physical testing. Host tests cover encoding and the controller
protocol. The operator confirmed the diagnostic on a Badger 2040. The
slideshow has only been built offline. This does not establish long-term
panel behavior or replace the monochrome speed, flicker-free, or
partial-update API.

## Badger examples

Both examples use the board EPD pin constants, SPI0 mode 0 at 12MHz,
`ENABLE_3V3` power hold, and Rotation270.

```sh
tinygo build -target=badger2040 -o badger-gray.uf2 ./examples/badger2040/grayscale
tinygo build -target=badger2040 -o badger-slideshow.uf2 ./examples/badger2040/grayscale-slideshow
```

The diagnostic prints each initialization stage and pauses for five seconds.
It clears to white, shows four bordered swatches with an orientation marker,
then enters deep sleep. The slideshow displays the four supplied photographs
with ten seconds between completed refreshes. Every frame includes a white
reset.

The [asset format and conversion instructions](../examples/badger2040/grayscale-slideshow/assets/README.md)
describe the 296x128 images. These examples do not flash hardware as part of
the build.

## References

- [Pimoroni Badger controller and pins](https://github.com/pimoroni/pimoroni-pico/blob/bbb1792856c7c8f9939a28e134021447b32f5bbb/libraries/badger2040/badger2040.hpp)
- [Pimoroni power hold and initialization](https://github.com/pimoroni/pimoroni-pico/blob/bbb1792856c7c8f9939a28e134021447b32f5bbb/libraries/badger2040/badger2040.cpp)
- [UC8151C datasheet](https://www.buydisplay.com/download/ic/UC8151C.pdf), commands 00h, 01h, 10h, 13h, 20h-24h, 30h, and 50h
- [Badger grayscale algorithm and waveform explanation](https://github.com/antirez/uc8151_micropython/tree/cc2a0de91386f08ee6c08e6e315bcfef850ffc49)
