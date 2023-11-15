package style

import "image/color"

// TODO use standard palette
//
// https://pkg.go.dev/github.com/AntoineAugusti/colors#StringToHexColor
//
func (p Property) Color() color.Color {
	if p == "default" {
		//if p == "default" || p == NullStyle {
		return nil
	}
	switch p {
	case "red":
		return color.RGBA{0xff, 0, 0, 0xff}
	case "green":
		return color.RGBA{0, 0xff, 0, 0xff}
	case "blue":
		return color.RGBA{0, 0, 0xff, 0xff}
	case "gray", "grey":
		return color.RGBA{0x80, 0x80, 0x80, 0xff}
	}
	return color.Black
}

func ColorString(c color.Color) string {
	if c == nil {
		return "powderblue" // X11 color and CSS color
	}
	r, g, b, a := c.RGBA()
	if r == a && g == a && b == a {
		return "white"
	}
	if r == 0 && g == 0 && b == 0 {
		return "black"
	}
	if r >= 0x90 {
		return "red"
	} else if g >= 0x90 {
		return "green"
	} else if b >= 0x90 {
		return "blue"
	}
	return "gray"
}
