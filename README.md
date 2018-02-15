# Linux Bing Wallpaper

It sets Bing.com wallpaper of the Day as your Linux Desktop

supports GNOME (2 and 3), KDE 4 / Plasma 5, XFCE4, MATE, Cinnamon, LXDE(LXQT), as well fallback to feh.

## Usage

Install [golang](https://golang.org).

    git clone https://github.com/marguerite/linux-bing-wallpaper
    cd linux-bing-wallpaper
    go build bing-wallpaper.go

Copy the generated `bing-wallpaper` somewhere (~/bin for example)

Run it using cron or systemd user service.

So next time you boot your computer for the first time a day, it'll run once.

Next boots it will run too, but do nothing.

## Easy commands

        # The first param is Market
        # The second param should be false to not loop infinitely (for cron)
        # (otherwise, script will keep running and checking for the next update)
        ~/bin/bing-wallpaper en-US true

## Example cron usage (crontab -e for your user)
```
# m h dom mon dow command
* * * * * ~/bin/bing-wallpaper en-US false
```

## Example systemd user service usage

    mkdir -p ~/.config/systemd/user
    cp -r bing-wallpaper.service ~/.config/systemd/user
    systemctl --user enable bing-wallpaper
    systemctl --user start bing-wallpaper
