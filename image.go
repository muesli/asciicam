package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"math"
	"strings"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/termenv"
	"github.com/nfnt/resize"
)

func frameToImage(frame []byte, width, height uint) *image.RGBA {
	yuyv := image.NewYCbCr(image.Rect(0, 0, int(width), int(height)), image.YCbCrSubsampleRatio422)
	for i := range yuyv.Cb {
		ii := i * 4
		yuyv.Y[i*2] = frame[ii]
		yuyv.Y[i*2+1] = frame[ii+2]
		yuyv.Cb[i] = frame[ii+1]
		yuyv.Cr[i] = frame[ii+3]

	}

	b := yuyv.Bounds()
	img := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(img, img.Bounds(), yuyv, b.Min, draw.Src)

	return img
}

func pixelToASCII(pixel color.Color) rune {
	r2, g2, b2, a2 := pixel.RGBA()
	r := uint(r2 / 257)
	g := uint(g2 / 257)
	b := uint(b2 / 257)
	a := uint(a2 / 257)

	intensity := (r + g + b) * a / 255
	precision := float64(255 * 3 / (len(pixels) - 1))

	v := int(math.Floor(float64(intensity)/precision + 0.5))
	return pixels[v]
}

func imageToAscii(width, height uint, p termenv.Profile, img image.Image) string {
	str := strings.Builder{}

	for i := 0; i < int(height); i++ {
		for j := 0; j < int(width); j++ {
			pixel := color.NRGBAModel.Convert(img.At(j, i))
			s := termenv.String(string(pixelToASCII(pixel)))

			_, _, _, a := col.RGBA()
			if a > 0 {
				s = s.Foreground(p.FromColor(col))
			} else {
				s = s.Foreground(p.FromColor(pixel))
			}
			str.WriteString(s.String())
		}
		str.WriteString("\n")
	}

	return str.String()
}

func imageToANSI(width, height uint, p termenv.Profile, img image.Image) string {
	b := img.Bounds()
	w := b.Max.X
	h := b.Max.Y

	str := strings.Builder{}
	for y := 0; y < h; y += 2 {
		for x := w; x < int(width); x += 2 {
			str.WriteString(" ")
		}
		for x := 0; x < w; x++ {
			str.WriteString(termenv.String("▀").
				Foreground(p.FromColor(img.At(x, y))).
				Background(p.FromColor(img.At(x, y+1))).
				String())
		}
		str.WriteString("\n")
	}

	return str.String()
}

func greenscreen(img *image.RGBA, bg []image.Image, dist float64) {
	for _, v := range bg {
		/*
			if img.Bounds().Size().Y != v.Bounds().Size().Y {
				panic(nil)
			}
			if img.Bounds().Size().X != v.Bounds().Size().X {
				panic(nil)
			}
		*/

		for y := 0; y < img.Bounds().Size().Y; y++ {
			for x := 0; x < img.Bounds().Size().X; x++ {
				c1, _ := colorful.MakeColor(img.At(x, y))
				c2, _ := colorful.MakeColor(v.At(x, y))

				/*
					add face detection?
					if (x > 42 && x < 78) && (y > 5 && y < 40) {
						continue
					}
				*/

				if c1.DistanceLab(c2) < dist {
					img.Set(x, y, image.Transparent)
				}
			}
		}
	}
}

func loadBgSamples(path string, width, height uint) (image.Image, error) {
	//TODO: take average of sample set
	// for i := 40; i < 41; i++ {
	i := 40
	b, err := ioutil.ReadFile(fmt.Sprintf("%s/%d.png", path, i))
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	return resize.Resize(width, height, img, resize.Bilinear).(*image.RGBA), nil
	// }
}
