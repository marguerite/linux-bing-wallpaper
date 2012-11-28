#!/bin/sh
# Author: Marguerite Su <i@marguerite.su>
# Version: 1.0
# License: GPL-3.0
# Description: set a picture as kde4 wallpaper.

LOCALE=`echo $LANG | sed 's/\..*$//'`

EN_CONSOLE1="Desktop Shell Scripting Console"
EN_CONSOLE2="Plasma Desktop Shell"

if [[ $LOCALE != "" ]]; then
        JS_CONSOLE1=`LANGUAGE=$LOCALE gettext -d plasma-desktop -s "$EN_CONSOLE1"`
        JS_CONSOLE2=`LANGUAGE=$LOCALE gettext -d plasma-desktop -s "$EN_CONSOLE2"`
        JS_CONSOLE="$JS_CONSOLE1 - $JS_CONSOLE2"
else
        JS_CONSOLE="$EN_CONSOLE1 - $EN_CONSOLE2"
fi

js=$(mktemp)
cat > $js <<_EOF
var wallpaper = "$1";
var activity = activities()[0];
activity.currentConfigGroup = new Array("Wallpaper", "image");
activity.writeConfig("wallpaper", wallpaper);
activity.writeConfig("userswallpaper", wallpaper);
activity.reloadConfig();
_EOF
qdbus org.kde.plasma-desktop /App local.PlasmaApp.loadScriptInInteractiveConsole "$js" > /dev/null
xdotool search --name "$JS_CONSOLE" windowactivate key ctrl+e key ctrl+w
rm -f "$js"
exit 0
