# Video on a 16x4 HD44780 LCD

This example plays a video on a 16x4 HD44780 LCD with a PCF8574 I2C
backpack at address 0x26. Each frame uses the 8 custom characters, and goes
out in one I2C transaction at 400 kHz.

## Make video.bin

`video.bin` in this directory is an empty placeholder. Convert a video with
`tools/lcdvideogen`, which needs ffmpeg. From the repository root, run:

```
go run ./examples/hd44780i2c/video/tools/lcdvideogen -input clip.mp4 -output examples/hd44780i2c/video/video.bin
```

Use `-fps` to set the frame rate and `-hold` to make the file smaller. The
output must fit in flash with the player. On an Arduino Due, a 30 fps file
of 400 KB plays for about 3.5 minutes.

## Arduino Due

| Backpack pin | Due pin |
|---|---|
| `VCC` | `5V` |
| `GND` | `GND` |
| `SDA` | `20 SDA` |
| `SCL` | `21 SCL` |

The Due pins are not 5 V tolerant. Use 5V for `VCC` only if the backpack has
no pull-up resistors from `SDA` and `SCL` to `VCC`. Measure this with the
power off. Otherwise use 3.3V or a level shifter.

From the repository root, run:

```
tinygo flash -target=arduino-due ./examples/hd44780i2c/video
```

The LED on D13 is on while the video plays. If it blinks fast, `video.bin`
has no valid header.

This was tested on an Arduino Due with a 30 fps file of 400 KB. The
arduino-due target is not in a TinyGo release yet.
