package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/blackjack/webcam"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

const (
	pxWidth  = 320  // requested image width
	pxHeight = 240  // requested image height
	bgDist   = 0.10 // distance threshold to background
)

var (
	col    = color.Color(color.RGBA{0, 0, 0, 0}) // if alpha is 0, use truecolor
	pixels = []rune{' ', '.', ',', ':', ';', 'i', '1', 't', 'f', 'L', 'C', 'G', '0', '8', '@'}
)

func main() {
	// graceful shutdown on SIGINT, SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs

		fmt.Println("\nShutting down...")
		cancel()
	}()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	dev := flag.String("dev", "/dev/video0", "video device")
	sample := flag.String("sample", "bgsample", "Where to find/store the sample data")
	gen := flag.Bool("gen", false, "Generate a new background")
	screen := flag.Bool("greenscreen", false, "Use greenscreen")
	ansi := flag.Bool("ansi", false, "Use ANSI")
	usecol := flag.String("color", "", "Use single color")
	w := flag.Uint("width", 0, "output width")
	h := flag.Uint("height", 0, "output height")
	flag.Parse()

	if *usecol != "" {
		c, err := colorful.Hex(*usecol)
		if err != nil {
			return fmt.Errorf("invalid color: %v", err)
		}

		col = c
	}
	height := *h // height of the terminal output
	width := *w  // width of the terminal output

	// detect terminal width
	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	if isTerminal {
		w, h, err := term.GetSize(int(os.Stdout.Fd()))
		if err == nil {
			if width == 0 {
				width = uint(w)
			}
			if height == 0 {
				height = uint(h)
			}
		}
	}
	if width == 0 {
		width = 125
	}
	if height == 0 {
		height = 50
	}

	// ANSI rendering uses half-height blocks
	if *ansi {
		height *= 2
	}

	cam, err := webcam.Open(*dev)
	if err != nil {
		return err
	}
	defer cam.Close()

	// find available yuyv format
	formats := cam.GetSupportedFormats()
	for k, v := range formats {
		fmt.Println(k, v)
		if strings.Contains(v, "YUYV") {
			f, w, h, err := cam.SetImageFormat(k, uint32(pxWidth), uint32(pxHeight))
			if err != nil {
				return fmt.Errorf("failed to set image format: %w", err)
			}
			fmt.Println(f, w, h)
		}
	}

	// start streaming
	err = cam.StartStreaming()
	if err != nil {
		return fmt.Errorf("failed to start streaming: %w", err)
	}

	var bg []image.Image
	if !*gen && *screen {
		bg, err = loadBgSamples(*sample)
		if err != nil {
			return fmt.Errorf("could not load background samples: %w", err)
		}
	}

	p := termenv.EnvColorProfile()
	termenv.HideCursor()
	defer termenv.ShowCursor()
	termenv.AltScreen()
	defer termenv.ExitAltScreen()

	i := 0
	for {
		if ctx.Err() != nil {
			return nil
		}

		err = cam.WaitForFrame(1)
		switch err.(type) {
		case nil:
		case *webcam.Timeout:
			fmt.Fprintln(os.Stderr, err.Error())
			continue

		default:
			return fmt.Errorf("failed waiting for frame: %w", err)
		}

		frame, err := cam.ReadFrame()
		if err != nil {
			return fmt.Errorf("failed to read frame: %w", err)
		}
		if len(frame) == 0 {
			continue
		}
		img := frameToImage(frame, width, height)

		// generate background sample data
		if *gen {
			f, err := os.Create(fmt.Sprintf("%s/%d.png", *sample, i))
			if err != nil {
				return fmt.Errorf("failed to create sample file: %w", err)
			}
			if err := png.Encode(f, img); err != nil {
				return fmt.Errorf("failed to encode sample frame: %w", err)
			}
			f.Close()

			i++
			if i > 100 {
				os.Exit(0)
			}
		}

		// virtual green screen
		if !*gen && *screen {
			greenscreen(img, bg)
		}

		// convert frame to ascii/ansi
		var s string
		if *ansi {
			s = imageToANSI(width, height, img)
		} else {
			s = imageToAscii(width, height, p, img)
		}
		termenv.MoveCursor(0, 0)
		fmt.Fprint(os.Stdout, s)
	}
}
