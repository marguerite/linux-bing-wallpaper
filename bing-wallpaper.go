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
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	gettext "github.com/chai2010/gettext-go"
	"github.com/gookit/color"
	"github.com/marguerite/go-stdlib/dir"
	"github.com/marguerite/go-stdlib/exec"
	"github.com/marguerite/go-stdlib/slice"
	"github.com/marguerite/linux-bing-wallpaper/desktopenvironment"
	"github.com/urfave/cli"
)

var (
	markets = []string{"en-US", "zh-CN", "ja-JP", "en-AU", "en-UK", "de-DE", "fr-FR", "en-NZ", "en-CA"}
)

func errChk(status int, e error) {
	if e != nil {
		panic(e)
	}
	if status != 0 {
		panic(fmt.Errorf("exit status is %d", status))
	}
}

func getURLPrefix(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	re := regexp.MustCompile("<urlBase>(.*)</urlBase>")
	return "http://bing.com" + re.FindStringSubmatch(string(body))[1]
}

func imageChk(image string, length int) bool {
	re := regexp.MustCompile(`^image/`)
	out, status, err := exec.Exec3("/usr/bin/file", "-L", "--mime-type", "-b", image)
	errChk(status, err)
	info, err := os.Stat(image)
	if err != nil {
		panic(err)
	}
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
		if err != nil {
			panic(err)
		}
	}

	for _, res := range resolutions {
		uri := prefix + res + ".jpg"

		fmt.Println("upstream uri:" + uri)

		resp, err := http.Get(uri)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		contentLength, _ := strconv.Atoi(resp.Header.Get("Content-Length"))

		// bing will not return 301 for redirect
		if resp.StatusCode == 200 && urlChk(resp, uri) {
			file = filepath.Join(dir, filepath.Base(uri))
			if _, err := os.Stat(file); os.IsNotExist(err) {
				out, err := os.Create(file)
				if err != nil {
					panic(err)
				}

				_, err = io.Copy(out, resp.Body)
				if err != nil {
					panic(err)
				}

				out.Sync()
				out.Close()

				if imageChk(file, contentLength) {
					fmt.Println("downloaded to:" + file)
					break
				} else {
					err = os.Remove(file)
					if err != nil {
						panic(err)
					}
					file = ""
					continue
				}
			} else {
				if imageChk(file, contentLength) {
					break
				} else {
					err = os.Remove(file)
					if err != nil {
						panic(err)
					}
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
		if err != nil {
			panic(err)
		}
		file, err := ioutil.ReadFile(path[0])
		if err != nil {
			panic(err)
		}

		re := regexp.MustCompile("DBUS_SESSION_BUS_ADDRESS='(.*)'")
		dbus := re.FindStringSubmatch(string(file))[1]
		os.Setenv("DBUS_SESSION_BUS_ADDRESS", dbus)
	}
}

func setWallpaper(env, pic, picOpts string) {
	fmt.Println("setting wallpaper for " + env)
	var status int
	var err error

	switch env {
	case "x-cinnamon":
		os.Setenv("DISPLAY", ":0")
		os.Setenv("GSETTINGS_BACKEND", "dconf")
		_, status, err = exec.Exec3("/usr/bin/gsettings", "set", "org.cinnamon.desktop.background", "picture-uri", "file://"+pic)
		errChk(status, err)
		_, status, err = exec.Exec3("/usr/bin/gsettings", "set", "org.cinnamon.desktop.background", "picture-options", picOpts)
		errChk(status, err)
	case "gnome":
		_, status, err = exec.Exec3("/usr/bin/gconftool-2", "-s", "-t", "string", "/desktop/gnome/background/picture_filename", pic)
		errChk(status, err)
		_, status, err = exec.Exec3("/usr/bin/gconftool-2", "-s", "-t", "string", "/desktop/gnome/background/picture_options", picOpts)
		errChk(status, err)
	case "gnome3":
		os.Setenv("DISPLAY", ":0")
		os.Setenv("GSETTINGS_BACKEND", "dconf")
		_, status, err := exec.Exec3("/usr/bin/gsettings", "set", "org.gnome.desktop.background", "picture-uri", "file://"+pic)
		errChk(status, err)
		_, status, err = exec.Exec3("/usr/bin/gsettings", "set", "org.gnome.desktop.background", "picture-options", picOpts)
		errChk(status, err)
	case "dde":
		os.Setenv("DISPLAY", ":0")
		os.Setenv("GSETTINGS_BACKEND", "dconf")
		_, status, err = exec.Exec3("/usr/bin/gsettings", "set", "com.deepin.wrap.gnome.desktop.background", "picture-uri", "file://"+pic)
		errChk(status, err)
		_, status, err = exec.Exec3("/usr/bin/gsettings", "set", "com.deepin.wrap.gnome.desktop.background", "picture-options", picOpts)
		errChk(status, err)
	case "mate":
		_, status, err = exec.Exec3("/usr/bin/dconf", "write", "/org/mate/desktop/background/picture-filename", pic)
		errChk(status, err)
	case "lxde", "lxqt":
		cmd := "/usr/bin/pcmanfm"
		if env == "lxqt" {
			cmd += "-qt"
		}
		_, status, err = exec.Exec3(cmd, "-w", pic)
		errChk(status, err)
		_, status, err := exec.Exec3(cmd, "--wallpaper-mode", picOpts)
		errChk(status, err)
	case "xfce":
		setXfceWallpaper(pic)
	case "kde4":
		setPlasmaWallpaper(pic, env)
	case "plasma5":
		setPlasmaWallpaper(pic, env)
	default:
		// other netWM/EWMH window manager
		_, status, err = exec.Exec3("/usr/bin/feh", "--bg-tile", pic)
		errChk(status, err)
	}
}

func setPlasmaWallpaper(pic, env string) {
	if _, err := exec.Search("/usr/bin/xdotool"); err != nil {
		panic("please install xdotool")
	}
	if _, err := exec.Search("/usr/bin/gettext"); err != nil {
		panic("please install gettext-runtime")
	}

	lang, _ := exec.Env("LANG")
	lang = strings.Split(lang, ".")[0]
	console := "Desktop Shell Scripting Console"
	var window, suffix, script string
	prefix := filepath.Join(os.Getenv("HOME"), ".local/share/plasmashell")
	dir.MkdirP(prefix)
	file := filepath.Join(prefix, "interactiveconsoleautosave.js")

	switch env {
	case "kde4":
		suffix = "Plasma Desktop Shell"
		window = console + " - " + suffix
		if len(lang) > 0 {
			gettext := gettext.New("plasma-desktop", "/usr/share/locale").SetLanguage("zh_CN")
			window = gettext.Gettext(console) + " - " + gettext.Gettext(suffix)
		}
		script = "var wallpaper = " + pic + "; var activity = activities()[0]; activity.currentConfigGroup = new Array(\"wallpaper\", \"image\"); activity.writeConfig(\"wallpaper\", wallpaper); activity.writeConfig(\"userswallpaper\", wallpaper); activity.reloadConfig();\n"
		ioutil.WriteFile(file, []byte(script), 0644)
		_, status, err := exec.Exec3("/usr/bin/qdbus", "org.kde.plasma-desktop", "/App", "local.PlasmaApp.loadScriptInInteractiveConsole", file)
		errChk(status, err)
		_, status, err = exec.Exec3("/usr/bin/xdotool", "search", "--name", window, "windowactivate", "key", "ctrl+e", "key", "ctrl+w")
		errChk(status, err)
	case "plasma5":
    // https://gist.github.com/marguerite/34d687cfaa88888f17bc0777a1c40509
		script = "for (i in activities()) { activityID = activities()[i]; desktops = desktopsForActivity(activityID); for (j in desktops) { desktop = desktops[j]; desktop.wallpaperPlugin = \"org.kde.image\"; desktop.wallpaperMode = \"Scaled and Cropped\"; desktop.currentConfigGroup = new Array(\"Wallpaper\", \"org.kde.image\", \"General\"); desktop.writeConfig(\"Image\", \"file://" + pic + "\");}}"
		out, status, err := exec.Exec3("/usr/bin/qdbus-qt5", "org.kde.plasmashell", "/PlasmaShell", "org.kde.PlasmaShell.evaluateScript", script)
		errChk(status, err)
		if strings.Contains(string(out), "Widgets are locked") {
			fmt.Println("Can't set wallpaper for Plasma because widgets are locked!")
		}
	}

	err := os.Remove(file)
	if err != nil {
		panic(err)
	}
}

func setXfceWallpaper(pic string) {
	if _, err := os.Stat("/usr/bin/xfconf-query"); os.IsNotExist(err) {
		panic("please install xfconf-query")
	}

	out, status, err := exec.Exec3("/usr/bin/xfconf-query", "--channel", "xfce4-desktop", "--property", "/backdrop", "-l")
	errChk(status, err)

	re := regexp.MustCompile(`(?m)^.*screen.*/monitor.*(image-path|last-image)$`)
	paths := re.FindAllString(string(out), -1)

	for _, p := range paths {
		_, status, err := exec.Exec3("/usr/bin/xfconf-query", "--channel", "xfce4-desktop", "--property", p, "-s", pic)
		errChk(status, err)
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
