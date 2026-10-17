package controls

func AdjustGrayFrames(dark, light byte, pressed Buttons) (byte, byte) {
	switch pressed & (Up | Down) {
	case Up:
		if dark < 31 {
			dark++
		}
	case Down:
		if dark > light+1 {
			dark--
		}
	}
	switch pressed & (A | B) {
	case A:
		if light+1 < dark {
			light++
		}
	case B:
		if light > 1 {
			light--
		}
	}
	return dark, light
}
