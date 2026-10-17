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
| Dark gray | 1 | 1 | 1 | BB | 9 |
| Light gray | 2 | 0 | 0 | WW | 5 |
| White | 3 | 1 | 0 | BW | 0 |

The waveform clock is 100Hz, the VCOM LUT is 44 bytes, and each pixel LUT is
42 bytes. The reset waveforms and grayscale algorithm follow the Badger-tested
[MicroPython implementation](https://github.com/antirez/uc8151_micropython/blob/cc2a0de91386f08ee6c08e6e315bcfef850ffc49/uc8151.py).
Its MIT license is retained in `LICENSE-grayscale`.

The source's 21/10-frame dark/light pulses looked too dark on the operator's
Badger. The defaults above shorten them to 9/5 frames. Use `DarkGrayFrames`
and `LightGrayFrames` in `GrayscaleConfig` to tune another panel. Zero selects
the default. The light duration must be less than dark, and dark must be
less than the fixed 32-frame black duration. White receives no black pulse.
The operator selected the 9/5-frame calibration using the interactive Bayer
gradient on their Badger 2040.

**This mode uses experimental custom waveforms, not vendor-supported
temperature-compensated grayscale.** The timings were developed for room
temperature. Contrast, gray separation, ghosting, and long-term panel behavior
require physical testing. Host tests cover encoding and the controller
protocol. The operator confirmed the diagnostic on a Badger 2040. The
slideshow has only been built offline. This does not establish long-term
panel behavior or replace the monochrome speed, flicker-free, or
partial-update API.

## Experimental sixteen shades

`NewGrayscale16` and `Grayscale16Config` provide a separate experimental 4bpp
path. The four-shade API and its L5/D9 defaults are unchanged. The 18,944-byte
framebuffer uses two packed 2bpp planes. RGB luminance selects shade 0 (black)
through 15 (white), with the same rotations and error handling as the
four-shade driver.

After a full white reset, five masked passes each drive three non-white
shades. The fourth waveform is a no-op for every other pixel, so each
non-white pixel receives one blackening pulse, not a sum of pulses.
Empty passes are skipped. This follows
[`update_greyscale` in the reference driver](https://github.com/antirez/uc8151_micropython/blob/cc2a0de91386f08ee6c08e6e315bcfef850ffc49/uc8151.py).
The existing `LICENSE-grayscale` also applies to this adaptation.

`Grayscale16Config.Frames` contains 15 pulse durations from lightest non-white
to black. An all-zero array selects these initial values at 100Hz:

```text
1 2 3 4 5 6 7 8 9 10 11 12 14 17 32
```

Custom durations must be positive and strictly increasing, ending at the
fixed 32-frame black. The initial linear ramp made dark tones turn black
too quickly on the operator's panel. This lighter curve keeps all gray
pulses within 1-17 frames and reserves 32 frames for full black. It retains
distinct timings but is not calibrated for equal perceived brightness.
Sixteen addressable shades do not guarantee sixteen visually distinct tones
on a particular panel.
Temperature sensitivity, ghosting and long-term behavior remain unvalidated.

The [sixteen-shade tester](../examples/badger2040/grayscale16) shows sixteen
**undithered solid swatches**, white to black, above a sixteen-shade Bayer
gradient. A selects the next lighter swatch, B the next darker swatch.
Up increases the selected shade's duration (darker); Down decreases it
(lighter), without crossing adjacent durations. Black and white stay fixed.
The tester starts at shade 2, near black. The marker identifies the selected
swatch. `S` is its shade number and `F` its pulse frames. The bracketed range
shows the allowed duration between neighboring shades. A range such as
`[7-7]` means there is no room to adjust that shade independently at 100Hz
without repeating a neighboring duration. Serial output prints the complete
timing table.

Every redraw hard-resets the controller and runs the same white, black,
white cleaning sequence as the four-shade tester, followed by the masked
passes. Wait until the activity LED goes out before pressing again.
Settings are not saved across restarts. This tester is slower than four
shades and stays awake for input.

## Badger examples

The examples use the board EPD pin constants, SPI0 mode 0 at 12MHz,
`ENABLE_3V3` power hold, and Rotation270.

```sh
tinygo build -target=badger2040 -o badger-gray.uf2 ./examples/badger2040/grayscale
tinygo build -target=badger2040 -o badger-gradient.uf2 ./examples/badger2040/grayscale-gradient
tinygo build -target=badger2040 -o badger-gray16.uf2 ./examples/badger2040/grayscale16
tinygo build -target=badger2040 -o badger-tainigo.uf2 ./examples/badger2040/grayscale-image
tinygo build -target=badger2040 -o badger-slideshow.uf2 ./examples/badger2040/grayscale-slideshow
```

The diagnostic prints each initialization stage and pauses for five seconds.
It clears to white, shows four bordered swatches with an orientation marker,
then enters deep sleep. The [gradient test](../examples/badger2040/grayscale-gradient)
draws a 280x64 horizontal bar from white on the left to black on the right.
It uses an 8x8 Bayer matrix to dither between adjacent shades of the four-shade
palette and preserves solid endpoints. Up increases dark-gray pulse frames
(darker), Down decreases them (lighter). A increases light-gray frames (darker),
B decreases them (lighter). Each debounced press changes one frame at 100Hz.
The front buttons use pull-down inputs and read HIGH when pressed.
The screen and serial output show the current dark (`D`) and light (`L`) values.
Adjustments retain `0 < light < dark < 32`; opposing buttons cancel.

Each gradient update hard-resets and reinitializes the controller, then displays
solid black before the gradient. Both refreshes include the driver's full white
reset, giving a white, black, white, gradient sequence to clear the previous image.
Wait for the activity LED to turn off before the next press. Holding a button
does not repeat. The example stays awake for input. Values reset to 9/5 after
a restart and are not saved to flash.

The [single-image test](../examples/badger2040/grayscale-image)
shows the supplied Tainigo sticker once, then enters deep sleep without
stage pauses. The optional slideshow displays the four supplied photographs
with ten seconds between completed refreshes. Every frame includes a white reset.

The [asset format and conversion instructions](../examples/badger2040/grayscale-slideshow/assets/README.md)
describe the 296x128 images. These examples do not flash hardware as part of
the build.

## References

- [Pimoroni Badger controller and pins](https://github.com/pimoroni/pimoroni-pico/blob/bbb1792856c7c8f9939a28e134021447b32f5bbb/libraries/badger2040/badger2040.hpp)
- [Pimoroni power hold and initialization](https://github.com/pimoroni/pimoroni-pico/blob/bbb1792856c7c8f9939a28e134021447b32f5bbb/libraries/badger2040/badger2040.cpp)
- [UC8151C datasheet](https://www.buydisplay.com/download/ic/UC8151C.pdf), commands 00h, 01h, 10h, 13h, 20h-24h, 30h, and 50h
- [Badger grayscale algorithm and waveform explanation](https://github.com/antirez/uc8151_micropython/tree/cc2a0de91386f08ee6c08e6e315bcfef850ffc49)
