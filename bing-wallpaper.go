// Author: Marguerite Su <i@marguerite.su>
// License: GPL-3.0
// Description: Download Bing Wallpaper of the Day and set it as your Linux Desktop.
// URL: https://github.com/marguerite/linux-bing-wallpaper

package main

import (
	"context"
	"fmt"
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

	"github.com/gookit/color"
	"github.com/marguerite/go-stdlib/slice"
	"github.com/marguerite/linux-bing-wallpaper/desktopenvironment"
	"github.com/urfave/cli"
)

var (
	markets = []string{"en-US", "zh-CN", "ja-JP", "en-AU", "en-UK", "de-DE", "fr-FR", "en-NZ", "en-CA"}
)

func errChk(e error) {
	if e != nil {
		panic(e)
	}
}

func getURLPrefix(url string) string {
	resp, err := http.Get(url)
	errChk(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	errChk(err)
	re := regexp.MustCompile("<urlBase>(.*)</urlBase>")
	return "http://bing.com" + re.FindStringSubmatch(string(body))[1]
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
	resolutions := []string{"_1920x1200", "_1920x1080", "_1366x768", "_1280x768", "_1280x720", "_1024x768"}
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
		if resp.StatusCode == 200 && urlChk(resp, uri) {
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

func setWallpaper(env, pic, picOpts string) {
	fmt.Println("setting wallpaper for " + env)

	switch env {
	case "x-cinnamon":
		os.Setenv("DISPLAY", ":0")
		os.Setenv("GSETTINGS_BACKEND", "dconf")
		_, err := exec.Command("/usr/bin/gsettings", "set", "org.cinnamon.desktop.background", "picture-uri", "file://"+pic).Output()
		errChk(err)
		_, err = exec.Command("/usr/bin/gsettings", "set", "org.cinnamon.desktop.background", "picture-options", picOpts).Output()
		errChk(err)
	case "gnome":
		_, err := exec.Command("/usr/bin/gconftool-2", "-s", "-t", "string", "/desktop/gnome/background/picture_filename", pic).Output()
		errChk(err)
		_, err = exec.Command("/usr/bin/gconftool-2", "-s", "-t", "string", "/desktop/gnome/background/picture_options", picOpts).Output()
		errChk(err)
	case "gnome3":
		os.Setenv("DISPLAY", ":0")
		os.Setenv("GSETTINGS_BACKEND", "dconf")
		_, err := exec.Command("/usr/bin/gsettings", "set", "org.gnome.desktop.background", "picture-uri", "file://"+pic).Output()
		errChk(err)
		_, err = exec.Command("/usr/bin/gsettings", "set", "org.gnome.desktop.background", "picture-options", picOpts).Output()
		errChk(err)
	case "dde":
		os.Setenv("DISPLAY", ":0")
		os.Setenv("GSETTINGS_BACKEND", "dconf")
		_, err := exec.Command("/usr/bin/gsettings", "set", "com.deepin.wrap.gnome.desktop.background", "picture-uri", "file://"+pic).Output()
		errChk(err)
		_, err = exec.Command("/usr/bin/gsettings", "set", "com.deepin.wrap.gnome.desktop.background", "picture-options", picOpts).Output()
		errChk(err)
	case "mate":
		_, err := exec.Command("/usr/bin/dconf", "write", "/org/mate/desktop/background/picture-filename", pic).Output()
		errChk(err)
	case "lxde":
		_, err := exec.Command("/usr/bin/pcmanfm", "-w", pic).Output()
		errChk(err)
		_, err1 := exec.Command("/usr/bin/pcmanfm", "--wallpaper-mode", picOpts).Output()
		errChk(err1)
	case "lxqt":
		_, err := exec.Command("/usr/bin/pcmanfm-qt", "-w", pic).Output()
		errChk(err)
		_, err1 := exec.Command("/usr/bin/pcmanfm-qt", "--wallpaper-mode", picOpts).Output()
		errChk(err1)
	case "xfce":
		setXfceWallpaper(pic)
	case "kde4":
		setKde4Wallpaper(pic)
	case "plasma5":
		setPlasmaWallpaper(pic)
	default:
		// other netWM/EWMH window manager
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
	fmt.Println(string(out))
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

func runDaemon(ctx context.Context, c Config, doJob func(c Config), out io.Writer) error {
	log.SetOutput(out)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.Tick(c.Duration):
			doJob(c)
		}
	}
}

// Config
type Config struct {
	Market   string
	Dir      string
	Desktop  string
	Options  string
	Duration time.Duration
}

func main() {
	duration, _ := time.ParseDuration("1m")

	cli.VersionFlag = cli.BoolFlag{
		Name:  "version",
		Usage: "Display version and exit.",
	}
	app := cli.NewApp()
	app.Usage = "Linux Bing Wallpaper"
	app.Description = "Set Wallpaper of the Day from bing.com as your desktop wallpaper."
	app.Version = "20201023"
	app.Authors = []cli.Author{
		{Name: "Marguerite Su", Email: "marguerite@opensuse.org"},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "market, m",
			Value: "zh-CN",
			Usage: "The region to use",
		},
		cli.BoolFlag{
			Name:  "daemon",
			Usage: "Run as daemon",
		},
		// valid options for gnome and cinnamon are: none, wallpaper, centered, scaled, stretched, zoom, spanned
		// valid options for lxde are: color (that is, disabled), stretch, crop, center, tile, screen
		// valid options for lxqt are: color (that is, disabled), stretch, crop, center, tile, zoom
		cli.StringFlag{
			Name:  "picture-options, o",
			Value: "zoom",
			Usage: "Picture options",
		},
		cli.StringFlag{
			Name:  "desktop",
			Usage: "Specify your destkop environment",
		},
		cli.StringFlag{
			Name:  "dir, d",
			Usage: "The directory to hold the wallpapers",
		},
		cli.DurationFlag{
			Name:  "interval, i",
			Value: duration,
			Usage: "The time interval for another run",
		},
	}

	app.Action = func(c *cli.Context) error {
		if ok, err := slice.Contains(markets, c.String("m")); !ok || err != nil {
			fmt.Printf("market must be one of the following: %s\n", strings.Join(markets, " "))
			os.Exit(1)
		}

		dir := c.String("dir")
		if len(dir) == 0 {
			dir = filepath.Join(os.Getenv("HOME"), "Pictures", "Bing")
		}

		desktop := c.String("desktop")
		if len(desktop) == 0 {
			desktop = desktopenvironment.Name()
		}

		color.Info.Println("Started linux-bing-wallpaper")
		dbusChk()

		config := Config{c.String("m"), dir, desktop, c.String("o"), c.Duration("i")}

		doJob := func(c Config) {
			xml := "http://www.bing.com/HPImageArchive.aspx?format=xml&idx=0&n=1&mkt=" + c.Market
			if len(c.Desktop) > 0 {
				pic := downloadWallpaper(xml, c.Dir)
				setWallpaper(c.Desktop, pic, c.Options)
				color.Info.Printf("Picture saved to: %s\n", pic)
			}
		}

		if c.Bool("daemon") {
			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

			defer func() {
				signal.Stop(signalChan)
				cancel()
			}()

			go func() {
				select {
				case s := <-signalChan:
					switch s {
					case syscall.SIGINT, syscall.SIGTERM:
						color.Warn.Println("Got SIGINT/SIGTERM, exiting.")
						cancel()
						os.Exit(1)
					case syscall.SIGHUP:
						color.Warn.Println("Got SIGHUP, reloading.")
						// FIXME configuration reload code
					}
				case <-ctx.Done():
					color.Info.Println("Done")
					os.Exit(1)
				}
			}()

			if err := runDaemon(ctx, config, doJob, os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				os.Exit(1)
			}
		} else {
			doJob(config)
		}

		return nil
	}

	_ = app.Run(os.Args)
}
