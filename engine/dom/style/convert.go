package style

import "image/color"

// TODO use standard palette
func (p Property) Color() color.Color {
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
