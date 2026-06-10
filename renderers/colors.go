package renderers

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/dockttier/dockttier/style"
)

// Per-command header accent colors. These are functions (not constants) so they
// always reflect the active theme's palette, which may be swapped at startup.
func accentBuild() string     { return style.HexBlue }      // docker build
func accentPush() string      { return style.HexOrange }    // docker push
func accentPull() string      { return style.HexCyan }      // docker pull
func accentRemove() string    { return style.HexRed }       // docker rm / rmi
func accentImages() string    { return style.HexPurple }    // docker images
func accentContainer() string { return style.HexTeal }      // docker ps / container ls
func accentDF() string        { return style.HexYellow }    // docker system df
func accentPrune() string     { return style.HexOrange }    // docker system prune
func accentLogs() string      { return style.HexTextMuted } // docker logs
func accentTag() string       { return style.HexBrand }     // docker tag
func accentFallback() string  { return style.HexTextMuted } // everything else

func lipColor(hex string) lipgloss.Color { return lipgloss.Color(hex) }
