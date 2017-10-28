# Linux Bing Wallpaper Shell Scripts

Downloading the latest Picture of the Day from Bing and sets it as the Wallpaper.

## Compatibility
### Supported DE's
* XFCE4
* GNOME 2 & 3
* KDE4
* i3wm (feh)

**Pull requests for more DE's are welcome!**
### Dependencies
* curl
* feh (only for i3wm)

## Usage
### Installation
#### Manual Installation
Download the main script **bing_wallpaper.sh**.

Download **kde4_set_wallpaper.sh** or **xfce4_set_wallpaper.sh** if needed.

Put the file(s) wherever you like.

I recommend:

"**~/.config/linux-bing-wallpaper**"


or the folder with your WM's configuration.
e.g.:


"**~/.config/i3/bing/bing_wallpaper.sh**"

Give the scripts execution permissions (*chmod +x ...*).
#### Easy Installation
```
mkdir -p ~/.config
cd .config
git clone https://github.com/Marco98/linux-bing-wallpaper.git
```
### Syntax
```
./bing_wallpaper.sh [mkt] [exitAfterRunning]
```
**Both arguments are optional**

**[mkt]** = ***en-US*** / zh-CN / ja-JP / en-AU / en-UK / de-DE / en-NZ / en-CA

The mkt parameter determines which Bing market you would like to obtain your images from.

**exitAfterRunning**

**[exitAfterRunning]** = ***true*** / false

If false the script will wait for the next Day to download a new Picture.

If true the script will exit after the first run.
### Usage Example for i3wm
**~/.config/i3/config** snippet:
```
exec ~/.config/linux-bing-wallpaper/bing_wallpaper.sh de-DE false
```
