package style

// Presets holds the built-in named themes selectable via config.toml
// ([theme] preset = "..."). Each theme has its own color scheme, progress-bar
// glyphs and spinner animation so they look and feel distinct.
var Presets = map[string]Theme{

	// midnight — the default GitHub-dark scheme with a mint accent. Solid block
	// progress bars and a smooth braille spinner.
	"midnight": {
		Name:           "midnight",
		Brand:          "#00d4aa",
		Text:           "#e6edf3",
		TextMuted:      "#7d8590",
		TextDim:        "#484f58",
		Green:          "#3fb950",
		Cyan:           "#39d353",
		Blue:           "#58a6ff",
		Teal:           "#2ea5a0",
		Yellow:         "#d29922",
		Orange:         "#f0883e",
		Red:            "#f85149",
		Purple:         "#bc8cff",
		Border:         "#30363d",
		BgPanel:        "#161b22",
		BarEmpty:       "#1c2128",
		BarFilledGlyph: "█",
		BarEmptyGlyph:  "░",
		Spinner:        []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	},

	// neon — vibrant cyberpunk magenta/cyan on near-black. Half-block "ribbon"
	// bars and a rotating arc spinner.
	"neon": {
		Name:           "neon",
		Brand:          "#ff2e97",
		Text:           "#f5f5ff",
		TextMuted:      "#8b8ba7",
		TextDim:        "#4b4b6b",
		Green:          "#39ff14",
		Cyan:           "#00f0ff",
		Blue:           "#2b8eff",
		Teal:           "#00ffd0",
		Yellow:         "#ffe600",
		Orange:         "#ff9f1c",
		Red:            "#ff3860",
		Purple:         "#b14bff",
		Border:         "#2a2a40",
		BgPanel:        "#12121f",
		BarEmpty:       "#1f1f33",
		BarFilledGlyph: "▰",
		BarEmptyGlyph:  "▱",
		Spinner:        []string{"◜", "◠", "◝", "◞", "◡", "◟"},
	},

	// dracula — the popular Dracula palette. Shaded bars and a braille spinner.
	"dracula": {
		Name:           "dracula",
		Brand:          "#bd93f9",
		Text:           "#f8f8f2",
		TextMuted:      "#969ab5",
		TextDim:        "#565a73",
		Green:          "#50fa7b",
		Cyan:           "#8be9fd",
		Blue:           "#6272a4",
		Teal:           "#54e6c5",
		Yellow:         "#f1fa8c",
		Orange:         "#ffb86c",
		Red:            "#ff5555",
		Purple:         "#ff79c6",
		Border:         "#44475a",
		BgPanel:        "#282a36",
		BarEmpty:       "#343746",
		BarFilledGlyph: "▓",
		BarEmptyGlyph:  "░",
		Spinner:        []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
	},

	// solarized — Solarized Dark. Heavy/light line bars and a quadrant spinner.
	"solarized": {
		Name:           "solarized",
		Brand:          "#2aa198",
		Text:           "#eee8d5",
		TextMuted:      "#93a1a1",
		TextDim:        "#586e75",
		Green:          "#859900",
		Cyan:           "#2aa198",
		Blue:           "#268bd2",
		Teal:           "#2aa198",
		Yellow:         "#b58900",
		Orange:         "#cb4b16",
		Red:            "#dc322f",
		Purple:         "#6c71c4",
		Border:         "#073642",
		BgPanel:        "#002b36",
		BarEmpty:       "#073642",
		BarFilledGlyph: "━",
		BarEmptyGlyph:  "─",
		Spinner:        []string{"▖", "▘", "▝", "▗"},
	},

	// matrix — green monochrome terminal. ASCII-flavored hash bars and a classic
	// spinning bar. Pairs well with disable_emoji = true.
	"matrix": {
		Name:           "matrix",
		Brand:          "#39ff14",
		Text:           "#9bff9b",
		TextMuted:      "#36a336",
		TextDim:        "#1f5f1f",
		Green:          "#39ff14",
		Cyan:           "#54ff8a",
		Blue:           "#37d67a",
		Teal:           "#2ee6a6",
		Yellow:         "#aaff00",
		Orange:         "#7fff00",
		Red:            "#ff5f5f",
		Purple:         "#73ffb0",
		Border:         "#114411",
		BgPanel:        "#001500",
		BarEmpty:       "#0a2a0a",
		BarFilledGlyph: "▮",
		BarEmptyGlyph:  "▯",
		Spinner:        []string{"|", "/", "-", "\\"},
	},
}

// PresetNames returns the available preset names (for help text/validation).
func PresetNames() []string {
	return []string{"midnight", "neon", "dracula", "solarized", "matrix"}
}
