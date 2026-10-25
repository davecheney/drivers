# Adafruit 2.2" TFT with Arduino Due

This example shows red, green, blue, and white bars on the
[Adafruit 2.2" 240x320 TFT display, product 1480](https://www.adafruit.com/product/1480).
It uses the `ili9341` driver with the display's ILI9340C controller.

The example was tested on an Arduino Due.

## Wiring

Disconnect USB power before you connect the wires.

| Display pin | Due pin |
|---|---|
| `Vin` | `5V` (SPI header pin 2) |
| `GND` | `GND` (SPI header pin 6) |
| `CLK` | `SCK` (SPI header pin 3) |
| `MOSI` | `MOSI` (SPI header pin 4) |
| `LCD_CS` | `D10` |
| `D/C` | `D9` |
| `RESET` | `D8` |

The SPI header is the 6 pin header in the middle of the Due. Pin 1 has a dot
next to it.

```
1 MISO   2 5V
3 SCK    4 MOSI
5 RESET  6 GND
```

`SCK` and `MOSI` are only on the SPI header. Do not connect anything to pin 5.
Leave `Backlite`, `MISO`, and `SDCS` unconnected. This example does not use the
microSD card slot.

See the [Adafruit pin descriptions](https://learn.adafruit.com/2-2-tft-display/pinouts).

## Flash

Connect USB to the programming port, which is the port next to the power jack.
From the repository root, run:

```
tinygo flash -target=arduino-due ./examples/ili9341/due
```

The display should show four horizontal bars, with red at the top and white at
the bottom.
