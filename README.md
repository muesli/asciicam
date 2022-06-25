# asciicam

Displays your webcam... on the terminal.

## Usage

```bash
$ asciicam

# use a specific camera device
$ asciicam -dev=/dev/video0

# set output width (0 means auto-detection)
$ asciicam -width=80 -height=60

# monochrome output
$ asciicam -color "#00ff00"

# use ANSI output
$ asciicam -ansi=true

# generate background sample data
$ asciicam -gen=true -sample bgdata/

# enable virtual greenscreen (requires sample data)
$ asciicam -greenscreen=true -sample bgdata/

# set greenscreen threshold
$ asciicam -greenscreen=true -sample bgdata/ -threshold=0.12

# show FPS counter
$ asciicam -fps=true
```

# Screenshots

ANSI mode:

![ANSI mode](/screenshots/asciicam_ansi.png?raw=true)

ASCII mode:

![ASCII mode](/screenshots/asciicam_ascii.png?raw=true)
