#!/bin/sh

# Author: Weitian Leung <weitianleung@gmail.com>
# Version: 1.0
# License: GPL-3.0
# Description: set a picture as xfce4 wallpaper

tools=(xrandr xfconf-query)

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

# TODO: when monitor name is NULL, changed to number
monitor=($(xrandr | grep " connected" | cut -d' ' -f1))

if [ ${#monitor[@]} -gt 1 ] ; then
	exp="\("
	for m in ${monitor[@]} ; do
		exp+="$m\|"
	done
	monitor="${exp::-2}\)"
fi

properties=$(xfconf-query -c xfce4-desktop -p /backdrop -l | grep -e "screen.*/monitor${monitor}.*image-path$" -e "screen.*/monitor${monitor}.*/last-image$")

for property in $properties; do
	xfconf-query -c xfce4-desktop -p $property -s "$wallpaper"
done
