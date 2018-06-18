# Linux Bing Wallpaper

It sets Bing.com wallpaper of the Day as your Linux Desktop

It supports GNOME (2 and 3), KDE 4 / Plasma 5, XFCE4, MATE, Cinnamon, LXDE(LXQT).

## Usage

Install [golang](https://golang.org).

    git clone https://github.com/marguerite/linux-bing-wallpaper
    cd linux-bing-wallpaper
    go build bing-wallpaper.go

Copy the generated `bing-wallpaper` somewhere (/usr/bin for example)

Run it using cron or systemd user service.

So next time you boot your computer for the first time in a day, it'll update your wallpaper.

## Easy commands

        # The first param is Market
        # The second param should be false to not loop infinitely (for cron)
        # (otherwise, script will keep running and checking for the next update)
        /usr/bin/bing-wallpaper -market=en-US -loop=true

## Example cron usage (crontab -e for your user)
```
# m h dom mon dow command
* * * * * /usr/bin/bing-wallpaper -market=en-US
```

## Example systemd user service usage

    mkdir -p ~/.config/systemd/user
    cp -r bing-wallpaper.service ~/.config/systemd/user
    systemctl --user enable bing-wallpaper
    systemctl --user start bing-wallpaper

## Known problems

A: On KDE Plasma 5, you have to unlock your desktop to receive wallpaper updates, there's no other way.

B: There is a racing problem when running bing-wallpaper with systemd. systemd can't guarantee to start
us after the desktop. So we may not detect the correct desktop thus can't set it to "WM" blindly.

The solution is:

If the desktop information was resolved to null at system start, bing-wallpaper will retry next hour.
If still null, then set the desktop to "WM". So for i3/openbox, it will not set your wallpaper immediately.

There's no solution for cron for now. But you will hardly meet this case unless you boot your machine
exactly at the time the cron service runs.
