# Lorenz attractor on Arduino Due

This example draws the Lorenz attractor on the
[Adafruit 2.2" 240x320 TFT display, product 1480](https://www.adafruit.com/product/1480).
A 3000 point trail traces the attractor while the view slowly rotates around
the z axis.

The example was tested on an Arduino Due. It uses SPI at 42 MHz. This is
faster than the ILI9341 datasheet rating of 10 MHz. If the display shows noise,
use shorter wires or set a lower `Frequency`.

## Wiring

Use the same wiring as the [Arduino Due example](../due/README.md).

## Flash

Connect USB to the programming port, which is the port next to the power jack.
From the repository root, run:

```
tinygo flash -target=arduino-due ./examples/ili9341/duelorenz
```
