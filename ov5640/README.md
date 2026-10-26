# OV5640 camera controls

This driver provides I2C access to the OV5640 control registers for board
verification. It does not initialize image output or capture images.

## Hardware requirements

Check the camera module schematic before connecting it. The module can have
regulators, level converters, or an oscillator that the bare sensor does not
have. Do not assume that its power or signal pins accept 3.3 V or 5 V.

The caller must configure power, the XVCLK input clock, RESETB, and PWDN before
using the driver. Follow the module power sequence and the OV5640 datasheet,
section 2.7. Keep RESETB high and PWDN low for register access. Apply a hardware
reset at power up with a low pulse of at least 1 ms, as required by section 2.8.
Allow at least 20 ms after RESETB goes high and at least 1 ms of XVCLK before
register access, as required by section 2.7.

Connect a common ground, SIOC to I2C SCL, and SIOD to I2C SDA. Use pull-up
resistors connected to the correct module I/O voltage. Configure the bus at
100 kHz for initial checks. The sensor supports up to 400 kHz.

No DVP signal connections are needed for these register checks.

## API

Call `New(bus)` with a configured `drivers.I2C` bus. This does not access the
device. The default 7-bit address is `0x3c`. `Device.Address` can be changed
if the sensor address was changed externally.

| Method | Result |
|---|---|
| `ChipID()` | Returns the 16-bit chip ID and any bus error. The expected ID is `ov5640.ID`, or `0x5640`. |
| `CheckID()` | Returns a bus error or `ov5640.ErrUnexpectedID` if the ID is incorrect. Does not change registers. |
| `ReadRegister(register)` | Reads one byte from a 16-bit register address. |
| `WriteRegister(register, value)` | Writes one byte to a 16-bit register address. |
| `Reset()` | Starts a software reset and waits 1 ms for it to settle. Does not replace the hardware reset at power up. |
| `SetStandby(enabled)` | Sets or clears software standby and preserves the other system control bits. |

Software reset restores sensor defaults. Leaving standby does not initialize
image output. Raw register writes bypass the reset delay and all other
high-level behavior. Use them only with the register reference.

Register reads use a register address write, a STOP, and then a separate
one-byte read. The I2C implementation must support read-only transactions.
The driver returns I2C errors without suppressing missing acknowledgements.
Do not call methods on the same device concurrently. Do not interleave other
access to the same sensor between the address and data transactions.

## Board verification

Use `examples/ov5640` after configuring the module power and clock. The example
uses the board's default I2C0 pins and does not set up power or clock signals.
Change the I2C configuration to match the actual wiring.

1. Read the chip ID and confirm `0x5640`.
2. Reset the sensor and confirm the chip ID again.
3. Enter software standby and read register `0x3008`. Bit 6 must be set.
4. Leave software standby and read the same register. Bit 6 must be clear.

The example performs these checks and reports errors. They verify register
communication only. DVP capture and image settings remain out of scope.
The driver has not yet been verified on hardware.

## References

- [OV5640 datasheet, version 2.03](https://cdn-learn.adafruit.com/assets/assets/000/118/994/original/OV5640_datasheet.pdf?1677598686).
  See sections 2.6 through 2.9, table 7-1 for registers `0x3008`, `0x300a`,
  and `0x300b`, table 8-4 for reset settling time, and table 8-6 for SCCB timing.
- [SCCB specification, version 2.1](https://www.waveshare.com/w/upload/1/14/OmniVision_Technologies_Seril_Camera_Control_Bus%28SCCB%29_Specification.pdf).
  See sections 3.2.1.2 and 3.2.1.3 for register read transactions.
