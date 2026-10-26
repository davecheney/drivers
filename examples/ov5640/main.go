package main

import (
	"fmt"
	"machine"

	"tinygo.org/x/drivers/ov5640"
)

func main() {
	if err := verify(); err != nil {
		fmt.Println("OV5640 verification failed:", err)
		return
	}
	fmt.Println("OV5640 register checks passed. Image capture was not checked.")
}

func verify() error {
	fmt.Println("Configure module power, XVCLK, RESETB, and PWDN before this check.")
	bus := machine.I2C0
	if err := bus.Configure(machine.I2CConfig{Frequency: 100_000}); err != nil {
		return err
	}
	camera := ov5640.New(bus)
	id, err := camera.ChipID()
	if err != nil {
		return err
	}
	fmt.Printf("OV5640 chip ID: 0x%04x\n", id)
	if id != ov5640.ID {
		return ov5640.ErrUnexpectedID
	}
	if err := camera.Reset(); err != nil {
		return err
	}
	if err := camera.CheckID(); err != nil {
		return err
	}
	for _, enabled := range []bool{true, false} {
		if err := camera.SetStandby(enabled); err != nil {
			return err
		}
		value, err := camera.ReadRegister(0x3008)
		if err != nil {
			return err
		}
		if (value&0x40 != 0) != enabled {
			return fmt.Errorf("standby check failed with register 0x3008 = 0x%02x", value)
		}
	}
	return nil
}
