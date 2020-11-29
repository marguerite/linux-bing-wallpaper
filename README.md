# Linux Bing Wallpaper

It sets Bing.com wallpaper of the Day as your Linux Desktop

It supports GNOME (2 and 3), KDE 4 / Plasma 5, XFCE4, MATE, Cinnamon, LXDE(LXQT).

## Usage

Install [golang](https://golang.org).

    git clone https://github.com/marguerite/linux-bing-wallpaper
    cd linux-bing-wallpaper
    export GO111MODULE=ON
    go mod download
    go mod vendor
    go build

Copy the generated `linux-bing-wallpaper` somewhere (/usr/bin for example)

Fill the config.yaml and copy it to ~/.config/linux-bing-wallpaper/

Run it using cron or systemd user service.

So next time you boot your computer for the first time in a day, it'll update your wallpaper.

## Easy commands

        /usr/bin/linux-bing-wallpaper -market=en-US

## Example cron usage (crontab -e for your user)

```
# m h dom mon dow command
* * * * * /usr/bin/linux-bing-wallpaper -market=en-US
```

## Example systemd user service usage

    mkdir -p ~/.config/systemd/user
    cp -r bing-wallpaper.service ~/.config/systemd/user
    systemctl --user enable bing-wallpaper
    systemctl --user start bing-wallpaper

## Known problems

On KDE Plasma 5(until 5.18), you have to unlock your desktop to receive wallpaper updates, there's no other way.

