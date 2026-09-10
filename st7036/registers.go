package st7036

// Command bits and masks for the ST7036 character LCD controller.
//
// Datasheet: https://www.crystalfontz.com/controllers/Sitronix/ST7036/
const (
	commandClear          = 0b00000001
	commandHome           = 0b00000010
	commandScroll         = 0b00010000
	commandDouble         = 0b00010000
	commandBias           = 0b00010100
	commandSetDisplayMode = 0b00001000

	blinkOn    = 0b00000001
	cursorOn   = 0b00000010
	displayOn  = 0b00000100
	setDDRAM   = 0b10000000
	setCGRAM   = 0b01000000
	entryShift = 0b00000010
	entryMove  = 0b00000100
)

// defaultInstructionSetTemplate is the base opcode for the function set
// command, or'd with the instruction set bits (0, 1 or 2) and the double
// height bit.
const defaultInstructionSetTemplate = 0b00111000
