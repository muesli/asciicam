# asciicam

Displays your webcam... on the terminal

## Usage

```bash
$ asciicam

$ asciicam -device=/dev/video0

# set output width (0 means auto-detection)
$ asciicam -width=80 -height=60

# use ANSI output
$ asciicam -ansi=true

# generate background sample data
$ asciicam -gen=true -sample bgdata/

# enable virtual greenscreen (requires sample data)
$ asciicam -greenscreen=true -sample bgdata/
```
