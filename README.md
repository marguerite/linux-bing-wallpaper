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

        /usr/bin/bing-wallpaper -market=en-US

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
it after the desktop. So we may not detect the correct desktop thus can't set it to "WM" blindly.

The solution is:

B1: If the desktop information was resolved to null at system start, bing-wallpaper will retry next hour.
If still null, then set the desktop to "WM". So for i3/openbox, it will not set your wallpaper immediately.

B2: use my other systemd user services, like checkprocess and network-real-online, to start bing-wallpaper after kde and after network is up and running.

There's no solution for cron for now. But you will hardly meet this case unless you boot your machine
exactly at the time the cron service runs.

## TODO

allow to specify desktop environment.
