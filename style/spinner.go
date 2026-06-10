package style

// SpinnerFrame returns the brand-colored spinner glyph for the given tick,
// using the active theme's spinner frames. A lightweight, allocation-free
// alternative to a full Bubbletea loop.
func SpinnerFrame(tick int) string {
	if disableEmoji {
		switch tick % 4 {
		case 0:
			return Brand.Render("|")
		case 1:
			return Brand.Render("/")
		case 2:
			return Brand.Render("-")
		default:
			return Brand.Render("\\")
		}
	}
	if len(spinnerFrames) == 0 {
		return Brand.Render("*")
	}
	return Brand.Render(spinnerFrames[tick%len(spinnerFrames)])
}
