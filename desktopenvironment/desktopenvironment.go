package desktopenvironment

import (
	"os"
	"regexp"
	"strings"

	"github.com/marguerite/go-stdlib/exec"
)

// Name return the desktop environment name of your system
func Name() string {
	// roughly get the DE name
	env, err := exec.Env("XDG_CURRENT_DESKTOP")
	if err != nil {
		env, err = exec.Env("DESKTOP_SESSION")
		if err != nil {
			env = ""
		}
	}

	// assign env in detail
	switch strings.ToLower(env) {
	case "kde":
		env_ret := getPlasmaVersion()
		if env_ret != "" {
			env = env_ret
			break
		}
		env = "kde4"
	case "deepin":
		env = "dde"
	case "lxde", "lubuntu", "lxqt":
		env = "lxde"
	case "gnome":
		if isGNOME3() {
			env = "gnome3"
		}
	case "xfce", "xfce4", "xfce session":
		env = "xfce"
	default:
		// kde
		_, err := exec.Env("KDE_FULL_SESSION")
		if err != nil {
			env_ret := getPlasmaVersion()
			if env_ret != "" {
				env = env_ret
				break
			}
			env = "kde4"
			break
		}
		// mate
		_, err = exec.Env("MATE_DESKTOP_SESSION_ID")
		if err == nil {
			env = "mate"
			break
		}
		// gnome
		_, err = exec.Env("GNOME_DESKTOP_SESSION_ID")
		_, status, err1 := exec.Exec3("/usr/bin/dbus-send", "--print-reply", "--dest=org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus.GetNameOwner", "string:org.gnome.SessionManager")
		if (status == 0 && err1 == nil) || err == nil {
			if isGNOME3() {
				env = "gnome3"
				break
			}
			env = "gnome"
			break
		}
		// xfce
		out, status, err := exec.Exec3("/usr/bin/xprop", "-root")
		if status == 0 && err == nil {
			if strings.HasPrefix(string(out), "xfce_desktop_window") {
				env = "xfce"
				break
			}
		}
		out, status, err = exec.Exec3("/usr/bin/xprop", "-root", "_DT_SAVE_MODE")
		if status == 0 && err == nil {
			if strings.Contains(string(out), "xfce4") {
				env = "xfce"
				break
			}
		}
		// check via _NET_WM_NAME for other window managers
		env = netWMName()
	}

	return env
}

func getPlasmaVersion() string {
	if _, err := os.Stat("/usr/bin/plasmashell"); os.IsNotExist(err) {
		return ""
	}
	out, status, err := exec.Exec3("/usr/bin/plasmashell", "-v")
	if status != 0 || err != nil {
		return ""
	}
	ver := strings.TrimPrefix(string(out), "plasmashell ")
	if strings.HasPrefix(ver, "5") {
		return "plasma5"
	}
	if strings.HasPrefix(ver, "6") {
		return "plasma6"
	}
	return ""
}

func isGNOME3() bool {
	// gnome-default-applications-properties is only available in GNOME 2.x but not in GNOME 3.x
	_, err := os.Stat("/usr/bin/gnome-default-applications-properties")
	if err != nil {
		return false
	}
	return true
}

func netWMName() (name string) {
	val, err := exec.Env("DISPLAY")
	if err != nil {
		return name
	}
	out, status, err := exec.Exec3("/usr/bin/xprop", "-display", val, "-root")
	if status != 0 || err != nil {
		return name
	}
	re := regexp.MustCompile(`(?m)^_NET_SUPPORTING_WM_CHECK\(WINDOW\): window id # ([^\n]+)\n`)
	if !re.MatchString(string(out)) {
		return name
	}
	winID := re.FindStringSubmatch(string(out))[1]
	out, status, err = exec.Exec3("/usr/bin/xprop", "-id", winID)
	if status != 0 || err != nil {
		return name
	}
	re1 := regexp.MustCompile(`(?m)_NET_WM_NAME.*?= "(.*?)"\n`)
	if !re1.MatchString(string(out)) {
		return name
	}
	return strings.ToLower(re.FindStringSubmatch(string(out))[1])
}
