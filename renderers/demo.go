package renderers

import (
	"fmt"
	"strings"

	"github.com/dockttier/dockttier/style"
)

// Demo renders representative sample output for every renderer using fake data,
// so users can preview the active theme without running real docker commands.
// Triggered by `dockttier --dockttier-demo` (optionally with DOCKTTIER_THEME).
func Demo() {
	w := style.Width()
	div := "  " + style.Border.Render(strings.Repeat("─", w-2))

	// Header.
	out(style.Header("build", []string{"-t", "myapp:latest", "."}, lipColor(accentBuild()), "[BUILDKIT]"))
	out("")
	out(style.SectionLabel("build stages"))
	out("")
	out(" " + style.Dim.Render(" 1") + "  " + style.RenderIcon(style.IconDone) + "  " +
		style.Text.Render(padDemo("FROM node:20-alpine", 40)) + style.Yellow.Render("2.1s"))
	out(" " + style.Dim.Render(" 2") + "  " + style.RenderIcon(style.IconCached) + "  " +
		style.Dim.Render(padDemo("COPY package*.json ./", 40)) + style.Dim.Render("cached"))
	out(" " + style.Dim.Render(" 3") + "  " + style.SpinnerFrame(2) + "  " +
		style.Text.Render(padDemo("RUN npm ci", 40)) +
		style.SlimBar(0.46, 18, lipColor(style.HexCyan)) + " " + style.Cyan.Render(" 46%") +
		"  " + style.Yellow.Render("8.4s"))
	out(" " + style.Dim.Render(" 4") + "  " + style.RenderIcon(style.IconStopped) + "  " +
		style.Dim.Render(padDemo("RUN npm run build", 40)))

	out("")
	out(style.SectionLabel("push layers"))
	out("")
	demoLayer("7f3a91b0", "2.1 MB", 1.0, style.IconDone, style.HexGreen, "pushed", true)
	demoLayer("e5f6a7b8", "14.3 MB", 0.62, style.IconUp, style.HexOrange, "pushing", true)
	demoLayer("a1b2c3d4", "71.2 MB", 1.0, style.IconCached, style.HexTextDim, "layer exists", false)
	demoLayer("90c9a7bc", "3.4 MB", 0.0, style.IconStopped, style.HexTextDim, "waiting", false)

	out("")
	out(style.SectionLabel("images"))
	out("")
	demoImage("myapp", "latest", "87.6 MB", 0.9, style.HexGreen)
	demoImage("postgres", "15", "379.2 MB", 1.0, style.HexRed)
	demoImage("redis", "7-alpine", "40.4 MB", 0.25, style.HexGreen)

	out("")
	out(style.SectionLabel("status icons"))
	out("")
	out("  " + style.RenderIcon(style.IconDone) + " done   " +
		style.RenderIcon(style.IconCached) + " cached   " +
		style.RenderIcon(style.IconRemoved) + " removed   " +
		style.RenderIcon(style.IconWarn) + " warn   " +
		style.RenderIcon(style.IconRunning) + " running   " +
		style.RenderIcon(style.IconPaused) + " paused")
	out("  " + style.Badge("CACHED", lipColor(style.HexTextDim)) + "  " +
		style.Badge("RUNNING", lipColor(style.HexGreen)) + "  " +
		style.Badge("REMOVED", lipColor(style.HexRed)))

	// Sample distilled error panel.
	renderErrorPanel(errorInfo{
		title:  "Access denied",
		detail: "denied: requested access to the resource is denied for repository acme/app",
		hint:   "run `docker login` for this registry, then retry",
	})

	out("")
	out(div)
	out(style.Footer(style.IconRunning, "Theme preview",
		style.KV("spinner", style.SpinnerFrame(0)+style.SpinnerFrame(1)+style.SpinnerFrame(2)+style.SpinnerFrame(3), lipColor(style.HexBrand)),
	))
}

func padDemo(s string, n int) string { return style.PadRight(s, n) }

func demoLayer(id, size string, frac float64, icon style.Icon, color string, status string, showPct bool) {
	barW := style.ResponsiveBarWidth(46)
	bar := style.SlimBar(frac, barW, lipColor(color))
	pct := "    "
	if showPct {
		pct = fmt.Sprintf("%3.0f%%", frac*100)
	}
	out("  " + style.RenderIcon(icon) + "  " +
		style.Brand.Render(style.Cell(id, 12)) +
		style.PadLeft(style.Muted.Render(size), 9) + "  " +
		bar + " " + lipStyle(color, pct) + "  " + style.Dim.Render(status))
}

func demoImage(repo, tag, size string, ratio float64, color string) {
	bar := style.Bar(int(ratio*6), 6, lipColor(color))
	out("  " + style.Blue.Render(style.Cell(repo, 14)) +
		style.Brand.Render(style.Cell(tag, 12)) +
		fmt.Sprintf("%s", lipStyle(color, style.PadLeft(size, 10))) + "  " +
		bar)
}

func lipStyle(hex, s string) string {
	return style.Text.Foreground(lipColor(hex)).Render(s)
}
