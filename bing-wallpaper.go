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
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	gettext "github.com/chai2010/gettext-go"
	"github.com/fatih/camelcase"
	"github.com/gookit/color"
	"github.com/marguerite/go-stdlib/dir"
	"github.com/marguerite/go-stdlib/exec"
	"github.com/marguerite/go-stdlib/fileutils"
	"github.com/marguerite/go-stdlib/slice"
	"github.com/marguerite/linux-bing-wallpaper/desktopenvironment"
	"github.com/urfave/cli"
	yaml "gopkg.in/yaml.v2"
)

var (
	markets     = []string{"en-US", "zh-CN", "ja-JP", "en-AU", "en-UK", "de-DE", "fr-FR", "en-NZ", "en-CA" "es-ES" "es-XL" "pt-BR" "pt-PT"}
	resolutions = []string{"1920x1200", "1920x1800", "1366x768", "1280x768", "1280x720", "1024x768"}
)

func errChk(status int, e error) {
	if e != nil {
		panic(e)
	}
	if status != 0 {
		panic(fmt.Errorf("exit status is %d", status))
	}
}

func getWallpaperURL(uri string) string {
	var resp *http.Response
	var err error

	for {
		resp, err = http.Get(uri)
		if err != nil {
			if strings.Contains(err.Error(), "network is unreachable") {
				time.Sleep(5 * time.Second)
				continue
			} else {
				panic(err)
			}
		} else {
			break
		}
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
	f, err := os.Stat(image)
	if os.IsNotExist(err) {
		return false
	}
	out, status, err := exec.Exec3("/usr/bin/file", "-L", "--mime-type", "-b", image)
	errChk(status, err)
	return strings.HasPrefix(string(out), "image") && f.Size() == int64(length)
}

func uriPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		panic(err)
	}
	return u.Path
}

func urlChk(resp *http.Response, uri string) bool {
	return uriPath(resp.Request.URL.String()) == uriPath(uri)
}

// download the highest resolution
func downloadWallpaper(xml, directory string) string {
	var file string
	prefix := getWallpaperURL(xml)

	// create picture diretory if does not already exist
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		fmt.Println("creating " + directory)
		dir.MkdirP(directory)
	}

	for _, res := range resolutions {
		uri := prefix + "_" + res + ".jpg"

		fmt.Println("upstream uri:" + uri)

		resp, err := http.Get(uri)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		contentLength, _ := strconv.Atoi(resp.Header.Get("Content-Length"))

		// bing will not return 301 for redirect
		if resp.StatusCode == 200 && urlChk(resp, uri) {
			// KDE can't recognize image file name with "th?id="
			file = filepath.Join(directory, strings.TrimPrefix(filepath.Base(uri), "th?id="))
			if !imageChk(file, contentLength) {
				os.Remove(file)
				b, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					panic(err)
				}
				ioutil.WriteFile(file, b, 0644)
				if imageChk(file, contentLength) {
					fmt.Println("downloaded to:" + file)
					break
				} else {
					os.Remove(file)
					file = ""
					continue
				}
			} else {
				break
			}
		}
	}
	return file
}

// cron needs the DBUS_SESSION_BUS_ADDRESS env set
func dbusChk() {
	if len(os.Getenv("DBUS_SESSION_BUS_ADDRESS")) > 0 {
		return
	}
	fmt.Println("setting DBUS_SESSION_BUS_ADDRESS")
	paths, err := filepath.Glob(filepath.Join("/home", os.Getenv("LOGNAME"), "/.dbus/session-bus/*"))
	if err != nil {
		panic(err)
	}
	if len(paths) == 0 {
		return
	}
	b, err := ioutil.ReadFile(paths[0])
	if err != nil {
		panic(err)
	}
	dbus := strings.TrimSpace(strings.TrimPrefix(string(b), "DBUS_SESSION_BUS_ADDRESS="))
	os.Setenv("DBUS_SESSION_BUS_ADDRESS", dbus)
}

func gsettingsSetWallpaper(name, pic, opts string) {
	os.Setenv("DISPLAY", ":0")
	os.Setenv("GSETTINGS_BACKEND", "dconf")
	_, stat, err := exec.Exec3("/usr/bin/gsettings", "set", "org."+name+".desktop.background", "picture-uri", "file://"+pic)
	errChk(stat, err)
	_, stat, err = exec.Exec3("/usr/bin/gsettings", "set", "org."+name+".desktop.background", "picture-options", opts)
	errChk(stat, err)
}

func setWallpaper(desktop, pic, opts, cmd string) {
	fmt.Println("setting wallpaper for " + desktop)
	var status int
	var err error

	switch desktop {
	case "x-cinnamon", "gnome3", "dde":
		var name string
		switch desktop {
		case "dde":
			name = "deepin.wrap.gnome"
		case "x-cinnamon":
			name = "cinnamon"
		default:
			name = "gnome"
		}
		gsettingsSetWallpaper(name, pic, opts)
	case "gnome":
		_, status, err = exec.Exec3("/usr/bin/gconftool-2", "-s", "-t", "string", "/desktop/gnome/background/picture_filename", pic)
		errChk(status, err)
		_, status, err = exec.Exec3("/usr/bin/gconftool-2", "-s", "-t", "string", "/desktop/gnome/background/picture_options", opts)
		errChk(status, err)
	case "mate":
		_, status, err = exec.Exec3("/usr/bin/dconf", "write", "/org/mate/desktop/background/picture-filename", "'"+pic+"'")
		errChk(status, err)
	case "lxde", "lxqt":
		cmd := "/usr/bin/pcmanfm"
		if desktop == "lxqt" {
			cmd += "-qt"
		}
		_, status, err = exec.Exec3(cmd, "-w", pic)
		errChk(status, err)
		_, status, err := exec.Exec3(cmd, "--wallpaper-mode", opts)
		errChk(status, err)
	case "xfce":
		setXfceWallpaper(pic)
	case "kde4", "plasma5":
		setPlasmaWallpaper(pic, desktop)
	case "none":
	default:
		// other netWM/EWMH window manager
		arr := strings.Split(cmd, " ")
		slice.Concat(&arr, pic)
		_, status, err = exec.Exec3(arr[0], arr[1:]...)
		errChk(status, err)
	}
}

func setPlasmaWallpaper(pic, env string) {
	var window, suffix, script string
	prefix := filepath.Join("/home", os.Getenv("LOGNAME"), ".local/share/plasmashell")
	dir.MkdirP(prefix)
	file := filepath.Join(prefix, "interactiveconsoleautosave.js")

	switch env {
	case "kde4":
		if _, err := exec.Search("/usr/bin/xdotool"); err != nil {
			panic("please install xdotool")
		}
		if _, err := exec.Search("/usr/bin/gettext"); err != nil {
			panic("please install gettext-runtime")
		}
	
		lang, _ := exec.Env("LANG")
		lang = strings.Split(lang, ".")[0]
		console := "Desktop Shell Scripting Console"

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

	os.Remove(file)
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

func runDaemon(ctx context.Context, c *Config, doJob func(c *Config), out io.Writer) {
	log.SetOutput(out)
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.Tick(c.UpdateInterval):
			doJob(c)
		}
	}
}

// Config
type Config struct {
	BingMarket         string
	WallpaperDir       string
	DesktopEnvironment string
	PictureOptions     string
	UpdateInterval     time.Duration
	DefaultCommand     string
}

func (c *Config) Load(fail bool, context *cli.Context, typ ...string) {
	file := filepath.Join("/home", os.Getenv("LOGNAME"), ".config", "linux-bing-wallpaper", "config.yaml")

	if _, err := os.Stat(filepath.Dir(file)); os.IsNotExist(err) {
		dir.MkdirP(filepath.Dir(file))
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		if _, err := os.Stat(filepath.Join(cwd, "config.yaml")); !os.IsNotExist(err) {
			fileutils.Copy(filepath.Join(cwd, "config.yaml"), file)
		}
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println("can not read config.yaml")
		if fail {
			os.Exit(1)
		} else {
			return
		}
	}

	if ok, err := slice.Contains(typ, "file"); ok && err == nil {
		// load from file
		err = yaml.Unmarshal(b, &c)
		if err != nil {
			fmt.Println(err)
			if fail {
				os.Exit(1)
			} else {
				return
			}
		}
	}
	if ok, err := slice.Contains(typ, "env"); ok && err == nil {
		// environment variable can overwrite the configs from file
		cv := reflect.Indirect(reflect.ValueOf(c))
		for i := 0; i < cv.NumField(); i++ {
			fieldName := cv.Type().Field(i).Name
			env := os.Getenv(strings.ToUpper(strings.Join(camelcase.Split(fieldName), "_")))
			if len(env) > 0 {
				cv.Field(i).Set(reflect.ValueOf(env))
			}
		}
	}
	if ok, err := slice.Contains(typ, "cmd"); ok && err == nil {
		// command-line args overwrite last
		if len(context.String("market")) > 0 {
			c.BingMarket = context.String("market")
		}
		if len(context.String("dir")) > 0 {
			c.WallpaperDir = context.String("dir")
		}
		if len(context.String("desktop")) > 0 {
			c.DesktopEnvironment = context.String("desktop")
		}
		if len(context.String("picture-options")) > 0 {
			c.PictureOptions = context.String("picture-options")
		}
		if context.Duration("interval") > 0 {
			c.UpdateInterval = context.Duration("interval")
		}
		if len(context.String("command")) > 0 {
			c.DefaultCommand = context.String("command")
		}
	}
	// set default values
	if len(c.DesktopEnvironment) == 0 {
		c.DesktopEnvironment = desktopenvironment.Name()
	}
	if len(c.WallpaperDir) == 0 {
		c.WallpaperDir = filepath.Join("/home", os.Getenv("LOGNAME"), "Pictures/Bing")
	}
	if c.UpdateInterval == 0 {
		duration, _ := time.ParseDuration("6h")
		c.UpdateInterval = duration
	}
}

func main() {
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
			Usage: "Picture options",
		},
		cli.StringFlag{
			Name:  "desktop",
			Usage: "Specify your desktop environment",
		},
		cli.StringFlag{
			Name:  "dir, d",
			Usage: "The directory to hold the wallpapers",
		},
		cli.DurationFlag{
			Name:  "interval, i",
			Usage: "The time interval for another run",
		},
		cli.StringFlag{
			Name:  "command, c",
			Usage: "The command to set wallpaper when no desktop environment was detected",
		},
	}

	app.Action = func(c *cli.Context) error {
		if ok, err := slice.Contains(markets, c.String("market")); len(c.String("market")) > 0 && (!ok || err != nil) {
			fmt.Printf("market must be one of the following: %s\n", strings.Join(markets, " "))
			os.Exit(1)
		}

		color.Info.Println("Started linux-bing-wallpaper")
		dbusChk()

		globalCfg := new(Config)
		cfgLock := new(sync.RWMutex)

		cfg := new(Config)
		cfg.Load(true, c, "file", "env", "cmd")
		cfgLock.Lock()
		globalCfg = cfg
		cfgLock.Unlock()

		doJob := func(cfg *Config) {
			xml := "http://www.bing.com/HPImageArchive.aspx?format=xml&idx=0&n=1&mkt=" + cfg.BingMarket
			pic := downloadWallpaper(xml, cfg.WallpaperDir)
			if len(pic) > 0 {
				fmt.Printf("Downloaded %s\n", pic)
				setWallpaper(cfg.DesktopEnvironment, pic, cfg.PictureOptions, cfg.DefaultCommand)
				fmt.Printf("Set wallpaper for %s\n", cfg.DesktopEnvironment)
			}
		}

		doJob(globalCfg)

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
						os.Exit(0)
					case syscall.SIGHUP:
						color.Warn.Println("Got SIGHUP, reloading.")
						cfg := new(Config)
						cfg.Load(false, c, "file", "env")
						cfgLock.Lock()
						globalCfg = cfg
						cfgLock.Unlock()
					}
				case <-ctx.Done():
					color.Info.Println("Done")
					os.Exit(0)
				}
			}()

			runDaemon(ctx, globalCfg, doJob, os.Stdout)
		}

		return nil
	}

	_ = app.Run(os.Args)
}
