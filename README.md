# Linux Bing Wallpaper

It sets Bing.com wallpaper of the Day as your Linux Desktop

supports XFCE4, GNOME (2 and 3) and KDE4, as well fallback to feh.

## Usage

Install [golang](https://golang.org).

    git clone https://github.com/marguerite/linux-bing-wallpaper
    cd linux-bing-wallpaper
    go build bing-wallpaper.go

Copy the generated `bing-wallpaper` somewhere (~/bin for example)

Make it autostart. (Google is your friend)

So next time you boot your computer for the first time a day, it'll run once.

Next boots it will run too, but do nothing.

## Easy commands

        # First param is Market
        # Second param is true to exit immediately if you want to use a cron
        # (otherwise, script will sleep 24 hrs)
        ~/bin/bing-wallpaper en-US true

## Example cron usage (crontab -e for your user)
```
# m h dom mon dow command
* * * * * ~/bin/bing-wallpaper en-US true
```
