package main

import (
	"machine"
	"strconv"

	"github.com/soypat/fat"
	"tinygo.org/x/drivers/sd"
)

type storage struct {
	card *sd.SPICard
	fs   fat.FS
	file fat.File
	hz   uint32
	next int
}

// mountSD mounts the FAT filesystem on the microSD card of the Sense board.
func mountSD(spi *machine.SPI, cs machine.Pin) (*storage, error) {
	cs.Configure(machine.PinConfig{Mode: machine.PinOutput})
	cs.High()
	setSPI(250_000)
	s := &storage{card: sd.NewSPICard(spi, cs.Set), next: 1}
	if err := s.card.Init(); err != nil {
		return nil, err
	}
	csd := s.card.CSD()
	s.hz = min(uint32(csd.TransferSpeed().RateKilobits())*1000, 20_000_000)
	setSPI(s.hz)
	if err := s.fs.Mount(s.card, int(csd.ReadBlockLen()), fat.ModeRW); err != nil {
		return nil, err
	}
	return s, nil
}

func photoName(n int) string {
	s := strconv.Itoa(n)
	for len(s) < 5 {
		s = "0" + s
	}
	return "IMG" + s + ".BMP"
}

// create opens the next unused photo file.
func (s *storage) create() (string, error) {
	var info fat.FileInfo
	for {
		name := photoName(s.next)
		s.next++
		if s.fs.Stat(name, &info) == nil {
			continue
		}
		return name, s.fs.OpenFile(&s.file, name, fat.ModeCreateNew|fat.ModeWrite)
	}
}
