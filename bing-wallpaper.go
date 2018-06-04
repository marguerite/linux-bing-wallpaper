// Author: Marguerite Su <i@marguerite.su>
// License: GPL-3.0
// Description: Download Bing Wallpaper of the Day and set it as your Linux Desktop.
// URL: https://github.com/marguerite/linux-bing-wallpaper

package main

import (
	"io"
	"io/ioutil"
	"log"
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

func detectDesktopEnv() string {
	de := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))

	// classic fallbacks
	if de == "" {
		kde, err := strconv.ParseBool(os.Getenv("KDE_FULL_SESSION"))
		errChk(err)
		if kde {
			return "kde"
		}
		if os.Getenv("GNOME_DESKTOP_SESSION_ID") != "" {
			de = "gnome"
		}
		if _, err := exec.Command("/usr/bin/dbus-send", "--print-reply", "--dest=org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus.GetNameOwner", "string:org.gnome.SessionManager").Output(); err == nil {
			de = "gnome"
		}
		if os.Getenv("MATE_DESKTOP_SESSION_ID") != "" {
			return "mate"
		}
		if out, err := exec.Command("/usr/bin/xprop", "-root", "_DT_SAVE_MODE").Output(); err == nil {
			re := regexp.MustCompile(" - \"xfce4\"$")
			if re.MatchString(string(out)) {
				return "xfce"
			}
		}
		if out, err := exec.Command("/usr/bin/xprop", "-root").Output(); err == nil {
			re := regexp.MustCompile("^xfce_desktop_window")
			if re.MatchString(string(out)) {
				return "xfce"
			}
		}
	}

	// fallback to checking $DESKTOP_SESSION
	if de == "" {
		de = strings.ToLower(os.Getenv("DESKTOP_SESSION"))
	}

	if de != "" {
		if de == "lubuntu" {
			return "lxde"
		}
		if de == "xfce4" || de == "xfce session" {
			return "xfce"
		}
	}

	// gnome-default-applications-properties is only available in GNOME 2.x but not in GNOME 3.x
	if de == "gnome" {
		if _, err := exec.Command("which", "gnome-default-applications-properties").Output(); err == nil {
			return "gnome3"
		}
	}

	// check plasmashell's version
	if de == "kde" {
		if _, err := os.Stat("/usr/bin/plasmashell"); !os.IsNotExist(err) {
			out, err := exec.Command("/usr/bin/plasmashell", "-v").Output()
			errChk(err)
			re := regexp.MustCompile(`\s(\d+)\..*?`)
			v := re.FindStringSubmatch(string(out))[1]
			if v == "5" {
				return "kde5"
			}
		}
	}

	if de == "" {
		return "WM"
	}

	return de
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

// download the highest resolution
func downloadWallpaper(xml string, dir string) string {
	prefix := getURLPrefix(xml)
	resolutions := []string{"_1920x1200", "_1920x1080", "_1366x768", "_1280x720", "_1024x768"}
	var file string
	var downloadResult []string
	picExt := ".jpg"
	// create picture diretory if does not already exist
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Println("creating " + dir)
		err = os.MkdirAll(dir, 0755)
		errChk(err)
	}

	for _, res := range resolutions {
		url := prefix + res + picExt
		arr := strings.Split(url, "/")
		pic := arr[len(arr)-1]
		file = dir + "/" + pic

		log.Println("upstream url:" + url)

		if _, err := os.Stat(file); os.IsNotExist(err) {
			out, err := os.Create(file)
			errChk(err)
			defer out.Close()

			resp, err := http.Get(url)
			errChk(err)
			defer resp.Body.Close()

			if resp.StatusCode == 404 {
				panic(url + "could not found")
			}

			_, err = io.Copy(out, resp.Body)
			errChk(err)

			log.Println("downloaded to:" + file)
		}

		if out, err := exec.Command("/usr/bin/file", "-L", "--mime-type", "-b", file).Output(); err == nil {
			re := regexp.MustCompile("^image/")
			if re.MatchString(string(out)) {
				downloadResult = append(downloadResult, "0")
				break
			} else {
				err = os.Remove(file)
				errChk(err)
				downloadResult = append(downloadResult, "1")
			}
		}
	}

	if !sliceContains(downloadResult, "0") {
		panic("Couldn't download any picture")
	}

	return file
}

// cron needs the DBUS_SESSION_BUS_ADDRESS env set
func dbusChk() {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		log.Println("setting DBUS_SESSION_BUS_ADDRESS")
		path, err := filepath.Glob("/home/" + os.Getenv("LOGNAME") + "/.dbus/session-bus/*")
		errChk(err)
		file, err := ioutil.ReadFile(path[0])
		errChk(err)

		re := regexp.MustCompile("DBUS_SESSION_BUS_ADDRESS='(.*)'")
		dbus := re.FindStringSubmatch(string(file))[1]
		os.Setenv("DBUS_SESSION_BUS_ADDRESS", dbus)
	}
}

func setWallpaper(de, pic string) {
	// valid options are: none, wallpaper, centered, scaled, stretched, zoom, spanned
	picOpts := "zoom"

	log.Println("setting wallpaper for " + de)

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

	if de == "lxqt" {
		_, err := exec.Command("/usr/bin/pcmanfm-qt", "-w", pic).Output()
		errChk(err)
	}

	if de == "xfce" {
		setXfceWallpaper(pic)
	}

	if de == "kde" {
		setKde4Wallpaper(pic)
	}

	if de == "kde5" {
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

	_, err := exec.Command("/usr/bin/dbus-send", "--session", "--dest=org.kde.plasmashell", "--type=method_call", "/PlasmaShell", "org.kde.PlasmaShell.evaluateScript", str).Output()
	errChk(err)
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
	opts := os.Args
	size := len(opts)
	markets := []string{"en-US", "zh-CN", "ja-JP", "en-AU", "en-UK", "de-DE", "en-NZ", "en-CA"}
	de := detectDesktopEnv()
	var mkt string
	var loop bool
	var err error
	idx := "0"
	// dir is used to set the location where Bing pictures of the day
	// are stored. HOME holds the path of the current user's home directory
	dir := "/home/" + os.Getenv("LOGNAME") + "/Pictures/Bing"

	if size == 1 {
		mkt = "zh-CN"
		loop = true
	}

	if size == 3 {
		if !sliceContains(markets, opts[1]) {
			panic("mkt must be of the following: " + sliceJoin(markets))
		}
		mkt = opts[1]
		loop, err = strconv.ParseBool(opts[2])
		errChk(err)
	}

	if size > 3 || size == 2 {
		panic("Usage: bing_wallpaper mkt[" + sliceJoin(markets) + "] loop[true,false]")
	}

	log.Println("started bing-wallpaper")
	dbusChk()

	xml := "http://www.bing.com/HPImageArchive.aspx?format=xml&idx=" + idx + "&n=1&mkt=" + mkt
	pic := downloadWallpaper(xml, dir)
        setWallpaper(de, pic)
	log.Println(pic)
        ticker := time.NewTicker(time.Hour*1)

	if loop {
		for range ticker.C {
			if time.Now().Format("15") == "00" {
				pic := downloadWallpaper(xml, dir)
				setWallpaper(de, pic)
				log.Println(pic)
			}
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Println("received signal:")
		log.Println(sig)
		log.Println("quiting...")
		ticker.Stop()
		os.Exit(0)
	}()
}
