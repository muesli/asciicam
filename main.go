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
	"time"

	"github.com/blackjack/webcam"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/termenv"
	"github.com/nfnt/resize"
	"golang.org/x/term"
)

const (
// pxWidth  = 320 // requested image width
// pxHeight = 240 // requested image height
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
	screenDist := flag.Float64("threshold", 0.13, "Greenscreen threshold")
	ansi := flag.Bool("ansi", false, "Use ANSI")
	usecol := flag.String("color", "", "Use single color")
	w := flag.Uint("width", 0, "output width")
	h := flag.Uint("height", 0, "output height")
	camWidth := flag.Uint("camWidth", 320, "cam input width")
	camHeight := flag.Uint("camHeight", 180, "cam input height")
	showFPS := flag.Bool("fps", false, "Show FPS")

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
			f, w, h, err := cam.SetImageFormat(k, uint32(*camWidth), uint32(*camHeight))
			if err != nil {
				return fmt.Errorf("failed to set image format: %w", err)
			}
			fmt.Println(f, w, h)
		}
	}

	// start streaming
	_ = cam.SetBufferCount(1)
	err = cam.StartStreaming()
	if err != nil {
		return fmt.Errorf("failed to start streaming: %w", err)
	}
	defer cam.StopStreaming() //nolint:errcheck

	var bg image.Image
	if !*gen && *screen {
		bg, err = loadBgSamples(*sample, width, height)
		if err != nil {
			return fmt.Errorf("could not load background samples: %w", err)
		}
	}

	p := termenv.EnvColorProfile()
	termenv.HideCursor()
	defer termenv.ShowCursor()
	termenv.AltScreen()
	defer termenv.ExitAltScreen()

	// seed fps counter
	var fps []float64
	for i := 0; i < 10; i++ {
		fps = append(fps, 0)
	}

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
		img := frameToImage(frame, *camWidth, *camHeight)

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

		// resize for further processing
		img = resize.Resize(width, height, img, resize.Bilinear).(*image.RGBA)

		// virtual green screen
		if !*gen && *screen {
			greenscreen(img, bg, *screenDist)
		}

		now := time.Now()
		// convert frame to ascii/ansi
		var s string
		if *ansi {
			s = imageToANSI(width, height, p, img)
		} else {
			s = imageToAscii(width, height, p, img)
		}

		// render
		termenv.MoveCursor(0, 0)
		fmt.Fprint(os.Stdout, s)

		if *showFPS {
			for i := len(fps) - 1; i > 0; i-- {
				fps[i] = fps[i-1]
			}
			fps[0] = float64(time.Second / time.Since(now))

			var fpsa float64
			for _, f := range fps {
				fpsa += f
			}

			fmt.Printf("FPS: %.0f", fpsa/float64(len(fps)))
		}
	}
}
