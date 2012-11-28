#!/bin/sh
# Author: Marguerite Su <i@marguerite.su>
# Version: 1.0
# License: GPL-3.0
# Description: Download Bing Wallpaper of the Day and set it as your Linux Desktop.

# $bing is needed to form the fully qualified URL for
# the Bing pic of the day
bing="www.bing.com"

# The mkt parameter determines which Bing market you would like to
# obtain your images from.
# Valid values are: en-US, zh-CN, ja-JP, en-AU, en-UK, de-DE, en-NZ, en-CA.
mkt="zh-CN"

# The idx parameter determines where to start from. 0 is the current day,
# 1 the previous day, etc.
idx="0"

# $xmlURL is needed to get the xml data from which
# the relative URL for the Bing pic of the day is extracted
xmlURL="http://www.bing.com/HPImageArchive.aspx?format=xml&idx=$idx&n=1&mkt=$mkt"

# $saveDir is used to set the location where Bing pics of the day
# are stored.  $HOME holds the path of the current user's home directory
saveDir=$HOME'/Pictures/Bing/'

# Create saveDir if it does not already exist
mkdir -p $saveDir

# Set picture options
# Valid options are: none,wallpaper,centered,scaled,stretched,zoom,spanned
picOpts="zoom"

# The file extension for the Bing pic
picExt=".jpg"

# Download the highest resolution

for picRes in _1920x1200 _1366x768 _1280x720 _1024x768; do

# Extract the relative URL of the Bing pic of the day from
# the XML data retrieved from xmlURL, form the fully qualified
# URL for the pic of the day, and store it in $picURL
picURL=$bing$(echo $(curl -s $xmlURL) | grep -oP "<urlBase>(.*)</urlBase>" | cut -d ">" -f 2 | cut -d "<" -f 1)$picRes$picExt

# $picName contains the filename of the Bing pic of the day
picName=${picURL#*2f}

# Download the Bing pic of the day
curl -s -o $saveDir$picName $picURL

# Test if it's a pic
file $saveDir$picName | grep HTML && rm -rf $saveDir$picName && continue

break
done

if [[ `rpm -qa gnome-session` != "" ]]; then
# Set the GNOME3 wallpaper
DISPLAY=:0 GSETTINGS_BACKEND=dconf gsettings set org.gnome.desktop.background picture-uri '"file://'$saveDir$picName'"'

# Set the GNOME 3 wallpaper picture options
DISPLAY=:0 GSETTINGS_BACKEND=dconf gsettings set org.gnome.desktop.background picture-options $picOpts
fi

if [[ `rpm -qa kdebase4-runtime` != "" ]]; then
test -e /usr/bin/xdotool || sudo zypper --no-refresh install xdotool
test -e /usr/bin/gettext || sudo zypper --no-refresh install gettext-runtime
wget https://raw.github.com/marguerite/linux-bing-wallpaper/master/kde4_set_wallpaper.sh
chmod +x kde4_set_wallpaper.sh
./kde4_set_wallpaper.sh $saveDir$picName
fi

# Exit the script
exit 0
