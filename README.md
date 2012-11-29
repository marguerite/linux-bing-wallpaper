# Linux Bing Wallpaper Shell Scripts

It sets Bing.com wallpaper of the Day as your Linux Desktop

supports GNOME and KDE4.

## Usage

Download these two scripts. 

Put them somewhere 

Change mkt varible in bing_wallpaper.sh to yours.

Give them execution permission.

Make them autostart.(Google is your friend)

So next time you boot your computer for the first time a day, it'll run once. 

Next boots it will run too, but do nothing.

## Easy commands

		mkdir bin
		wget https://raw.github.com/marguerite/linux-bing-wallpaper/master/bing_wallpaper.sh -o bin/bing_wallpaper.sh
		# If you use KDE
		# wget https://raw.github.com/marguerite/linux-bing-wallpaper/master/kde4_set_wallpaper.sh -o bin/kde4_set_wallpaper.sh
		chmod +x bin/*.sh
		./bin/bing_wallpaper.sh

