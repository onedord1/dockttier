# dockttier themes

dockttier ships with **5 built-in themes**. Each theme is a complete look: its
own 14-color palette, its own progress/load-bar glyphs, and its own spinner
animation — so switching themes changes the *feel*, not just the hue.

| Preset | Vibe | Bar glyphs | Spinner |
|---|---|---|---|
| `midnight` | GitHub-dark, mint accent (default) | `█` / `░` solid blocks | braille `⠋⠙⠹…` |
| `neon` | cyberpunk magenta + cyan on black | `▰` / `▱` ribbon | rotating arc `◜◠◝◞◡◟` |
| `dracula` | the classic Dracula palette | `▓` / `░` shaded | braille spin `⣾⣽⣻…` |
| `solarized` | Solarized Dark, calm teal/blue | `━` / `─` lines | quadrant `▖▘▝▗` |
| `matrix` | green monochrome terminal | `▮` / `▯` | classic `\|/-\` |

---

## Preview them right now (no docker needed)

```bash
DOCKTTIER_THEME=midnight  ./bin/dockttier --dockttier-demo
DOCKTTIER_THEME=neon      ./bin/dockttier --dockttier-demo
DOCKTTIER_THEME=dracula   ./bin/dockttier --dockttier-demo
DOCKTTIER_THEME=solarized ./bin/dockttier --dockttier-demo
DOCKTTIER_THEME=matrix    ./bin/dockttier --dockttier-demo
```

`DOCKTTIER_THEME` also works on real commands, handy for A/B testing:

```bash
DOCKTTIER_THEME=neon ./bin/dockttier images
```

---

## Make a theme permanent (config.toml)

Pick a theme by name in `~/.config/dockttier/config.toml`:

```toml
[theme]
preset = "neon"
```

That's it — every command now uses that theme. The ready-made files in the
[`themes/`](themes/) folder are full configs you can copy:

```bash
mkdir -p ~/.config/dockttier
cp themes/neon.toml ~/.config/dockttier/config.toml
```

### Override on top of a preset

You can still tweak individual options on top of a chosen preset:

```toml
[theme]
preset        = "dracula"
brand_color   = "#ff79c6"   # override just the signature/brand accent
disable_emoji = false       # true => ASCII glyphs (good for `matrix`)
```

Precedence: built-in defaults → `preset` → `brand_color`/`disable_emoji`
overrides → `DOCKTTIER_THEME` env var (highest, for quick testing).

An unknown `preset` name prints a warning and falls back to the default.

---

## The 5 palettes

### midnight (default)
```
brand #00d4aa  green #3fb950  blue #58a6ff  yellow #d29922  red #f85149  purple #bc8cff
```

### neon
```
brand #ff2e97  green #39ff14  cyan #00f0ff  yellow #ffe600  red #ff3860  purple #b14bff
```

### dracula
```
brand #bd93f9  green #50fa7b  cyan #8be9fd  yellow #f1fa8c  red #ff5555  pink #ff79c6
```

### solarized
```
brand #2aa198  green #859900  blue #268bd2  yellow #b58900  red #dc322f  violet #6c71c4
```

### matrix
```
brand #39ff14  green #39ff14  lime #aaff00  (green monochrome)  red #ff5f5f
```

---

## Want your own theme?

The presets live in [`style/presets.go`](style/presets.go). Copy one entry in
the `Presets` map, rename it, change the colors/glyphs/spinner, add the name to
`PresetNames()` and to `knownPreset()` in `config/config.go`, then rebuild with
`make build`. Each `Theme` field maps directly to what you see on screen:

```go
"mytheme": {
    Name:           "mytheme",
    Brand:          "#rrggbb",
    Green:          "#rrggbb",
    // … the rest of the 14-color palette …
    BarFilledGlyph: "█",      // filled progress cell
    BarEmptyGlyph:  "░",      // empty progress cell
    Spinner:        []string{"⠋", "⠙", "⠹"}, // animation frames
},
```
