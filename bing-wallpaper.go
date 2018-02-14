// Author: Marguerite Su <i@marguerite.su>
// License: GPL-3.0
// Description: Download Bing Wallpaper of the Day and set it as your Linux Desktop.
// URL: https://github.com/marguerite/linux-bing-wallpaper

package main

import (
  "fmt"
  "io"
  "io/ioutil"
  "net/http"
  "os"
  "os/exec"
  "path/filepath"
  "regexp"
  "strconv"
  "strings"
)

func check(e error) {
  if e != nil {
    panic(e)
  }
}

func include(s []string, e string) bool {
  for _, a := range s {
    if a == e {
      return true
    }
  }
  return false
}

func join(arr []string) string {
  var s string
  for _, a := range arr {
    s += a + " "
  }
  return s
}

func toString(b []byte) string {
  return string(b[:])
}

func detect_de() string {
  var de string
  de = strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))

  // classic fallbacks
  if de == "" {
    kde, err := strconv.ParseBool(os.Getenv("KDE_FULL_SESSION"))
    check(err)
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

  if de == "" {
    return "WM"
  }

  return de
}

func get_url_prefix(url string) string {
  resp, err := http.Get(url)
  check(err)
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)
  check(err)
  res := toString(body)
  re := regexp.MustCompile("<urlBase>(.*)</urlBase>")
  return "http://bing.com" + re.FindStringSubmatch(res)[1]
}

// download the highest resolution
func download_pictures(xml string, dir string) string {
  prefix := get_url_prefix(xml)
  resolutions := []string{"_1920x1200", "_1920x1080", "_1366x768", "_1280x720", "_1024x768"}
  var file string
  var downloadResult []string
  picExt := ".jpg"
  // create picture diretory if does not already exist
  if _, err := os.Stat(dir); os.IsNotExist(err) {
    err = os.MkdirAll(dir, 0755)
    check(err)
  }

  for _, res := range resolutions {
    url := prefix + res + picExt
    arr := strings.Split(url, "/")
    pic := arr[len(arr)-1]
    file = dir + "/" + pic

    if _, err := os.Stat(file); os.IsNotExist(err) {
      out, err := os.Create(file)
      check(err)
      defer out.Close()

      resp, err := http.Get(url)
      check(err)
      defer resp.Body.Close()

      _, err = io.Copy(out, resp.Body)
      check(err)
    }

    if out, err := exec.Command("/usr/bin/file", "--mime-type", "-b", file).Output(); err == nil {
      re := regexp.MustCompile("^image/")
      if re.MatchString(string(out)) {
        downloadResult = append(downloadResult, "0")
        break
      } else {
        err = os.Remove(file)
        check(err)
        downloadResult = append(downloadResult, "1")
      }
    }
  }

  if !include(downloadResult, "0") {
    panic("Couldn't download any picture")
  }

  return file
}

// cron needs the DBUS_SESSION_BUS_ADDRESS env set
func check_dbus() {
  if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
    path, err := filepath.Glob("/home/" + os.Getenv("LOGNAME") + "/.dbus/session-bus/*")
    check(err)
    file, err := ioutil.ReadFile(path[0])
    check(err)

    re := regexp.MustCompile("DBUS_SESSION_BUS_ADDRESS='(.*)'")
    dbus := re.FindStringSubmatch(string(file))[1]
    os.Setenv("DBUS_SESSION_BUS_ADDRESS", dbus)
  }
}

func set_wallpaper(de, pic string) {
  // valid options are: none, wallpaper, centered, scaled, stretched, zoom, spanned
  picOpts := "zoom"

  if de == "x-cinnamon" {
    os.Setenv("DISPLAY", ":0")
    os.Setenv("GSETTINGS_BACKEND", "dconf")
    _, err := exec.Command("/usr/bin/gsettings", "set", "org.cinnamon.desktop.background", "picture-uri", "file://" + pic).Output()
    check(err)
    _, err1 := exec.Command("/usr/bin/gsettings", "set", "org.cinnamon.desktop.background", "picture-options", picOpts).Output()
    check(err1)
  }

  if de == "gnome" {
    _, err := exec.Command("/usr/bin/gconftool-2", "-s", "-t", "string", "/desktop/gnome/background/picture_filename", pic).Output()
    check(err)
    _, err1 := exec.Command("/usr/bin/gconftool-2", "-s", "-t", "string", "/desktop/gnome/background/picture_options", picOpts).Output()
    check(err1)
  }

  if de == "gnome3" {
    os.Setenv("DISPLAY", ":0")
    os.Setenv("GSETTINGS_BACKEND", "dconf")
    _, err := exec.Command("/usr/bin/gsettings", "set", "org.gnome.desktop.background", "picture-uri", "file://" + pic).Output()
    check(err)
    _, err1 := exec.Command("/usr/bin/gsettings", "set", "org.gnome.desktop.background", "picture-options", picOpts).Output()
    check(err1)
  }

  if de == "mate" {
    _, err := exec.Command("/usr/bin/dconf", "write", "/org/mate/desktop/background/picture-filename", pic).Output()
    check(err)
  }

  if de == "lxqt" {
    _, err := exec.Command("/usr/bin/pcmanfm-qt", "-w", pic).Output()
    check(err)
  }

  if de == "xfce" {
    set_xfce_wallpaper(pic)
  }

  if de == "kde" {
    set_plasma_wallpaper(pic)
  }

  if de == "WM" {
    _, err := exec.Command("/usr/bin/feh", "--bg-tile", pic).Output()
    check(err)
  }
}

func set_plasma_wallpaper(pic string) string {
  if _, err := os.Stat("/usr/bin/xdotool"); os.IsNotExist(err) {
    panic("please install xdotool")
  }

  if _, err := os.Stat("/usr/bin/gettext"); os.IsNotExist(err) {
    panic("please install gettext-runtime")
  }
  return ""
}

func set_xfce_wallpaper(pic string) string {
  return ""
}

func main() {
  opts := os.Args
  size := len(opts)
  markets := []string{"en-US", "zh-CN", "ja-JP", "en-AU", "en-UK", "de-DE", "en-NZ", "en-CA"}
  de := detect_de()
  var mkt string
  var err error
  var exitAfterRunning bool
  idx := "0"
  // dir is used to set the location where Bing pictures of the day
  // are stored. HOME holds the path of the current user's home directory
  dir := "/home/" + os.Getenv("LOGNAME") + "/Pictures/Bing"

  if size == 1 {
    mkt = "zh-CN"
    exitAfterRunning = false
  }

  if size == 3 {
    if !include(markets, opts[1]) {
      panic("mkt must be of the following: " + join(markets))
    }
    mkt = opts[1]

    exitAfterRunning, err = strconv.ParseBool(opts[2])
    check(err)
  }

  if size > 3 || size == 2 {
    panic("Usage: bing_wallpaper mkt[" + join(markets) + "] exitAfterRunning[true,false]")
  }

  xml := "http://www.bing.com/HPImageArchive.aspx?format=xml&idx=" + idx + "&n=1&mkt=" + mkt
  pic := download_pictures(xml, dir)

  check_dbus()
  set_wallpaper(de, pic)

  fmt.Println(exitAfterRunning)
}
