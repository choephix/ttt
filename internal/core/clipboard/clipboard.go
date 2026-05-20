package clipboard

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var (
	content      string
	useSystem    = true
)

func DisableSystem() {
	useSystem = false
}

func Set(s string) {
	content = s
	if useSystem {
		writeSystemClipboard(s)
	}
}

func Get() string {
	if useSystem {
		if sys := readSystemClipboard(); sys != "" {
			return sys
		}
	}
	return content
}

func writeSystemClipboard(s string) {
	encoded := base64.StdEncoding.EncodeToString([]byte(s))
	fmt.Fprintf(os.Stderr, "\033]52;c;%s\a", encoded)

	if name, args := findCopyCmd(); name != "" {
		c := exec.Command(name, args...)
		c.Stdin = strings.NewReader(s)
		c.Run()
	}
}

func readSystemClipboard() string {
	name, args := findPasteCmd()
	if name == "" {
		return ""
	}
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func findCopyCmd() (string, []string) {
	if runtime.GOOS == "darwin" {
		return "pbcopy", nil
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if p, err := exec.LookPath("wl-copy"); err == nil {
			return p, nil
		}
	}
	if os.Getenv("DISPLAY") != "" {
		if p, err := exec.LookPath("xclip"); err == nil {
			return p, []string{"-selection", "clipboard"}
		}
		if p, err := exec.LookPath("xsel"); err == nil {
			return p, []string{"--clipboard", "--input"}
		}
	}
	return "", nil
}

func findPasteCmd() (string, []string) {
	if runtime.GOOS == "darwin" {
		return "pbpaste", nil
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if p, err := exec.LookPath("wl-paste"); err == nil {
			return p, []string{"--no-newline"}
		}
	}
	if os.Getenv("DISPLAY") != "" {
		if p, err := exec.LookPath("xclip"); err == nil {
			return p, []string{"-selection", "clipboard", "-o"}
		}
		if p, err := exec.LookPath("xsel"); err == nil {
			return p, []string{"--clipboard", "--output"}
		}
	}
	return "", nil
}
