# Linux Bing Wallpaper Shell Scripts

It sets Bing.com wallpaper of the Day as your Linux Desktop

supports GNOME (2 and 3) and KDE4.

## Usage

Download these two scripts.

Put them somewhere (~/bin for example)

Change mkt varible in bing_wallpaper.sh to your market (valid values are: en-US, zh-CN, ja-JP, en-AU, en-UK, de-DE, en-NZ, en-CA)

Give the scripts execution permissions.

Make them autostart. (Google is your friend)

So next time you boot your computer for the first time a day, it'll run once.

Next boots it will run too, but do nothing.

## Easy commands

        cd ~
        mkdir bin
        wget https://raw.githubusercontent.com/marguerite/linux-bing-wallpaper/master/bing_wallpaper.sh -o bin/bing_wallpaper.sh
        # If you use KDE
        wget https://raw.githubusercontent.com/marguerite/linux-bing-wallpaper/master/kde4_set_wallpaper.sh -o bin/kde4_set_wallpaper.sh
        chmod +x bin/*.sh

        # Default behavior
        ./bin/bing_wallpaper.sh

        # First param is Market
        # Second param is true to exit immediately if you want to use a cron
        # (otherwise, script will sleep 24 hrs)
        ./bin/bing_wallpaper.sh en-US true

