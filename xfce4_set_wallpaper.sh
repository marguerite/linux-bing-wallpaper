#!/bin/bash

# Author: Weitian Leung <weitianleung@gmail.com>
# Version: 2.0
# License: GPL-3.0
# Description: set a picture as xfce4 wallpaper

tools=(xfconf-query)

for tool in ${tools[@]} ; do
	if ! which $tool &> /dev/null ; then
		echo "Missing $tool"
		exit 1
	fi
done

wallpaper=$1

# check image
mime_type=`file --mime-type -b "$wallpaper"`
if [[ ! "$mime_type" == image/* ]]; then
	echo "Invalid image"
	exit 1
fi

# set to every monitor that contains image-path/last-image
properties=$(xfconf-query -c xfce4-desktop -p /backdrop -l | grep -e "screen.*/monitor.*image-path$" -e "screen.*/monitor.*/last-image$")

for property in $properties; do
	xfconf-query -c xfce4-desktop -p $property -s "$wallpaper"
done
