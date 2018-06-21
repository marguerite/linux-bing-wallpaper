// Author: Marguerite Su <i@marguerite.su>
// License: GPL-3.0
// Description: Download Bing Wallpaper of the Day and set it as your Linux Desktop.
// URL: https://github.com/marguerite/linux-bing-wallpaper

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func errChk(e error) {
	if e != nil {
		panic(e)
	}
}

func sliceContains(arr []string, s string) bool {
	for _, i := range arr {
		if i == s {
			return true
		}
	}
	return false
}

func sliceJoin(arr []string) string {
	var s string
	for _, i := range arr {
		s += i + " "
	}
	return s
}

func toString(b []byte) string {
	return string(b[:])
}

func sessionEnv() string {
	// use $XDG_CURRENT_DESKTOP, then $DESKTOP_SESSION
	if e := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP")); len(e) != 0 {
		return e
	}
	if e := strings.ToLower(os.Getenv("DESKTOP_SESSION")); len(e) != 0 {
		return e
	}
	return ""
}

func isPlasma5() (bool, error) {
	re := regexp.MustCompile(`\s(\d+)\..*?`)
	// check plasmashell's version
	if _, err := os.Stat("/usr/bin/plasmashell"); !os.IsNotExist(err) {
		version, err := exec.Command("/usr/bin/plasmashell", "-v").Output()
		if err != nil {
			return false, err
		}
		if re.MatchString(string(version)) && re.FindStringSubmatch(string(version))[1] == "5" {
			return true, nil
		}
		return false, errors.New("can't find valid version from plasmashell -v")
	}
	return false, nil
}

func kdeSession(session string) string {
	env := os.Getenv("KDE_FULL_SESSION")
	// use sessionEnv, then $KDE_FULL_SESSION
	if session == "kde" || len(env) != 0 {
		plasma5, err := isPlasma5()
		errChk(err)
		if plasma5 {
			return "plasma5"
		}
		return "kde4"
	}
	return ""
}

func isGnome3() bool {
	// gnome-default-applications-properties is only available in GNOME 2.x but not in GNOME 3.x
	if _, err := os.Stat("/usr/bin/gnome-default-applications-properties"); !os.IsNotExist(err) {
		return true
	}
	return false
}

func gnomeSession(session string) string {
	_, err := exec.Command("/usr/bin/dbus-send", "--print-reply", "--dest=org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus.GetNameOwner", "string:org.gnome.SessionManager").Output()
	if session == "gnome" || len(os.Getenv("GNOME_DESKTOP_SESSION_ID")) != 0 || err == nil {
		if isGnome3() {
			return "gnome3"
		}
		return "gnome"
	}
	return ""
}

func mateSession(session string) string {
	if session == "mate" || len(os.Getenv("MATE_DESKTOP_SESSION_ID")) != 0 {
		return "mate"
	}
	return ""
}

func xfceSession(session string) string {
	saveMode, _ := exec.Command("/usr/bin/xprop", "-root", "_DT_SAVE_MODE").Output()
	window, _ := exec.Command("/usr/bin/xprop", "-root").Output()
	saveModeRe := regexp.MustCompile(" - \"xfce4\"$")
	windowRe := regexp.MustCompile("^xfce_desktop_window")

	if session == "xfce" || session == "xfce4" || session == "xfce session" || saveModeRe.MatchString(string(saveMode)) || windowRe.MatchString(string(window)) {
		return "xfce"
	}
	return ""
}

func lxdeSession(session string) string {
	if session == "lxde" || session == "lxqt" || session == "lubuntu" {
		return "lxde"
	}
	return ""
}

func desktopEnv() string {
	env := sessionEnv()
	if kde := kdeSession(env); len(kde) != 0 {
		return kde
	}
	if gnome := gnomeSession(env); len(gnome) != 0 {
		return gnome
	}
	if mate := mateSession(env); len(mate) != 0 {
		return mate
	}
	if xfce := xfceSession(env); len(xfce) != 0 {
		return xfce
	}
	if lxde := lxdeSession(env); len(lxde) != 0 {
		return lxde
	}
	return env
}

func getURLPrefix(url string) string {
	resp, err := http.Get(url)
	errChk(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	errChk(err)
	res := toString(body)
	re := regexp.MustCompile("<urlBase>(.*)</urlBase>")
	return "http://bing.com" + re.FindStringSubmatch(res)[1]
}

func imageChk(image string, length int) bool {
	re := regexp.MustCompile(`^image/`)
	out, err := exec.Command("/usr/bin/file", "-L", "--mime-type", "-b", image).Output()
	errChk(err)
	info, err := os.Stat(image)
	errChk(err)
	return re.MatchString(string(out)) && info.Size() == int64(length)
}

func uriPath(uri string) string {
	re := regexp.MustCompile(`http(s)?:\/\/[^\/]+(.*)`)
	if re.MatchString(uri) {
		return re.FindStringSubmatch(uri)[2]
	}
	return ""
}

func urlChk(resp *http.Response, uri string) bool {
	return uriPath(resp.Request.URL.String()) == uriPath(uri)
}

// download the highest resolution
func downloadWallpaper(xml string, dir string) string {
	file := ""
	prefix := getURLPrefix(xml)
	resolutions := []string{"_1920x1200", "_1920x1080", "_1366x768", "_1280x720", "_1024x768"}
	// create picture diretory if does not already exist
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Println("creating " + dir)
		err = os.MkdirAll(dir, 0755)
		errChk(err)
	}

	for _, res := range resolutions {
		uri := prefix + res + ".jpg"

		fmt.Println("upstream uri:" + uri)

		resp, err := http.Get(uri)
		errChk(err)
		defer resp.Body.Close()

		contentLength, _ := strconv.Atoi(resp.Header.Get("Content-Length"))

		// bing will not return 301 for redirect
		if resp.StatusCode != 200 || !urlChk(resp, uri) {
			continue
		} else {
			file = filepath.Join(dir, filepath.Base(uri))
			if _, err := os.Stat(file); os.IsNotExist(err) {
				out, err := os.Create(file)
				errChk(err)

				_, err = io.Copy(out, resp.Body)
				errChk(err)

				out.Sync()
				out.Close()

				if imageChk(file, contentLength) {
					fmt.Println("downloaded to:" + file)
					break
				} else {
					err = os.Remove(file)
					errChk(err)
					file = ""
					continue
				}
			} else {
				if imageChk(file, contentLength) {
					break
				} else {
					err = os.Remove(file)
					errChk(err)
					file = ""
					continue
				}
			}
		}
	}
	return file
}

// cron needs the DBUS_SESSION_BUS_ADDRESS env set
func dbusChk() {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		fmt.Println("setting DBUS_SESSION_BUS_ADDRESS")
		path, err := filepath.Glob("/home/" + os.Getenv("LOGNAME") + "/.dbus/session-bus/*")
		errChk(err)
		file, err := ioutil.ReadFile(path[0])
		errChk(err)

		re := regexp.MustCompile("DBUS_SESSION_BUS_ADDRESS='(.*)'")
		dbus := re.FindStringSubmatch(string(file))[1]
		os.Setenv("DBUS_SESSION_BUS_ADDRESS", dbus)
	}
}

func setWallpaper(de, pic, picOpts string) {
	fmt.Println("setting wallpaper for " + de)

	if de == "x-cinnamon" {
		os.Setenv("DISPLAY", ":0")
		os.Setenv("GSETTINGS_BACKEND", "dconf")
		_, err := exec.Command("/usr/bin/gsettings", "set", "org.cinnamon.desktop.background", "picture-uri", "file://"+pic).Output()
		errChk(err)
		_, err = exec.Command("/usr/bin/gsettings", "set", "org.cinnamon.desktop.background", "picture-options", picOpts).Output()
		errChk(err)
	}

	if de == "gnome" {
		_, err := exec.Command("/usr/bin/gconftool-2", "-s", "-t", "string", "/desktop/gnome/background/picture_filename", pic).Output()
		errChk(err)
		_, err = exec.Command("/usr/bin/gconftool-2", "-s", "-t", "string", "/desktop/gnome/background/picture_options", picOpts).Output()
		errChk(err)
	}

	if de == "gnome3" {
		os.Setenv("DISPLAY", ":0")
		os.Setenv("GSETTINGS_BACKEND", "dconf")
		_, err := exec.Command("/usr/bin/gsettings", "set", "org.gnome.desktop.background", "picture-uri", "file://"+pic).Output()
		errChk(err)
		_, err = exec.Command("/usr/bin/gsettings", "set", "org.gnome.desktop.background", "picture-options", picOpts).Output()
		errChk(err)
	}

	if de == "mate" {
		_, err := exec.Command("/usr/bin/dconf", "write", "/org/mate/desktop/background/picture-filename", pic).Output()
		errChk(err)
	}

	if de == "lxde" {
		_, err := exec.Command("/usr/bin/pcmanfm", "-w", pic).Output()
		errChk(err)
		_, err1 := exec.Command("/usr/bin/pcmanfm", "--wallpaper-mode", picOpts).Output()
		errChk(err1)
	}

	if de == "lxqt" {
		_, err := exec.Command("/usr/bin/pcmanfm-qt", "-w", pic).Output()
		errChk(err)
		_, err1 := exec.Command("/usr/bin/pcmanfm-qt", "--wallpaper-mode", picOpts).Output()
		errChk(err1)
	}

	if de == "xfce" {
		setXfceWallpaper(pic)
	}

	if de == "kde4" {
		setKde4Wallpaper(pic)
	}

	if de == "plasma5" {
		setPlasmaWallpaper(pic)
	}

	if de == "WM" {
		_, err := exec.Command("/usr/bin/feh", "--bg-tile", pic).Output()
		errChk(err)
	}
}

func setKde4Wallpaper(pic string) {
	if _, err := os.Stat("/usr/bin/xdotool"); os.IsNotExist(err) {
		panic("please install xdotool")
	}

	if _, err := os.Stat("/usr/bin/gettext"); os.IsNotExist(err) {
		panic("please install gettext-runtime")
	}

	re := regexp.MustCompile(`^(.*?)\..*$`)
	locale := re.FindStringSubmatch(os.Getenv("LANG"))[1]

	console1 := "Desktop Shell Scripting Console"
	console2 := "Plasma Desktop Shell"

	var jsconsole string
	if locale != "" {
		os.Setenv("LANGUAGE", locale)
		out, err := exec.Command("/usr/bin/gettext", "-d", "plasma-desktop", "-s", console1).Output()
		errChk(err)
		out1, err1 := exec.Command("/usr/bin/gettext", "-d", "plasma-desktop", "-s", console2).Output()
		errChk(err1)
		jsconsole = string(out) + " - " + string(out1)
	} else {
		jsconsole = console1 + " - " + console2
	}

	file, err := os.Create("/tmp/jsconsole")
	errChk(err)

	str := []string{"var wallpaper = " + pic + ";\n",
		"var activity = activities()[0];\n",
		"activity.currentConfigGroup = new Array(\"wallpaper\", \"image\");\n",
		"activity.writeConfig(\"wallpaper\", wallpaper);\n",
		"activity.writeConfig(\"userswallpaper\", wallpaper);\n",
		"activity.reloadConfig();\n"}

	for _, s := range str {
		_, err = file.WriteString(s)
		errChk(err)
	}

	err = file.Sync()
	errChk(err)

	file.Close()

	_, err = exec.Command("/usr/bin/qdbus", "org.kde.plasma-desktop", "/App", "local.PlasmaApp.loadScriptInInteractiveConsole", "/tmp/jsconsole").Output()
	errChk(err)

	_, err = exec.Command("/usr/bin/xdotool", "search", "--name", jsconsole, "windowactivate", "key", "ctrl+e", "key", "ctrl+w").Output()
	errChk(err)

	err = os.Remove("/tmp/jsconsole")
	errChk(err)
}

func setPlasmaWallpaper(pic string) {
	str := `string:
var all = desktops();
for (i=0;i<all.length;i++) {
  d = all[i];
  d.wallpaperPlugin = "org.kde.image";
  d.currentConfigGroup = Array("Wallpaper", "org.kde.image", "General");
  d.writeConfig("Image", "file://` + pic + `");
}`

	out, err := exec.Command("/usr/bin/dbus-send", "--session", "--dest=org.kde.plasmashell", "--type=method_call", "--print-reply", "/PlasmaShell", "org.kde.PlasmaShell.evaluateScript", str).Output()
	errChk(err)

	if strings.Contains(string(out), "Widgets are locked") {
		fmt.Println("Can't set wallpaper for Plasma because widgets are locked!")
	}
}

func setXfceWallpaper(pic string) {
	if _, err := os.Stat("/usr/bin/xfconf-query"); os.IsNotExist(err) {
		panic("please install xfconf-query")
	}

	out, err := exec.Command("/usr/bin/xfconf-query", "--channel", "xfce4-desktop", "--property", "/backdrop", "-l").Output()
	errChk(err)

	re := regexp.MustCompile(`(?m)^.*screen.*/monitor.*(image-path|last-image)$`)
	paths := re.FindAllString(string(out), -1)

	for _, p := range paths {
		_, err := exec.Command("/usr/bin/xfconf-query", "--channel", "xfce4-desktop", "--property", p, "-s", pic).Output()
		errChk(err)
	}
}

func main() {
	var mkt string
	var pic string
	var picOpts string
	var loop bool
	de := desktopEnv()
	markets := []string{"en-US", "zh-CN", "ja-JP", "en-AU", "en-UK", "de-DE", "en-NZ", "en-CA"}
	idx := "0"
	// dir is used to set the location where Bing pictures of the day
	// are stored. HOME holds the path of the current user's home directory
	dir := "/home/" + os.Getenv("LOGNAME") + "/Pictures/Bing"
	// valid options for gnome and cinnamon are: none, wallpaper, centered, scaled, stretched, zoom, spanned
	// valid options for lxde are: color (that is, disabled), stretch, crop, center, tile, screen
	// valid options for lxqt are: color (that is, disabled), stretch, crop, center, tile, zoom
	flag.StringVar(&mkt, "market", "zh-CN", "the region to use. available: "+sliceJoin(markets))
	flag.BoolVar(&loop, "loop", false, "whether to loop or not")
	flag.StringVar(&picOpts, "picopts", "zoom", "picture options")
	flag.Parse()

	if !sliceContains(markets, mkt) {
		panic("market must be one of the following: " + sliceJoin(markets))
	}

	fmt.Println("started bing-wallpaper")
	dbusChk()

	xml := "http://www.bing.com/HPImageArchive.aspx?format=xml&idx=" + idx + "&n=1&mkt=" + mkt

	if len(de) != 0 {
		pic = downloadWallpaper(xml, dir)
		setWallpaper(de, pic, picOpts)
		fmt.Println("the picture location:" + pic)
	}

	ticker := time.NewTicker(time.Hour * 1)

	if loop {
		for range ticker.C {
			// there's a racing problem between bing-wallpaper and the desktop.
			// if bing-wallpaper was started before the desktop by systemd, we'll fail to get any desktop information
			// so we try an hour later, if the destkop variable is still null, then we treat it as WM
			de = desktopEnv()
			if len(de) == 0 {
				de = "WM"
			}
			newPic := downloadWallpaper(xml, dir)
			if newPic != pic {
				setWallpaper(de, newPic, picOpts)
				fmt.Println("the new picture:" + newPic)
			}
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println("received signal:")
		fmt.Println(sig)
		fmt.Println("quiting...")
		ticker.Stop()
		os.Exit(0)
	}()

	ticker.Stop() // memory leak
}
