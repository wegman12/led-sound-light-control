package led

import "strings"

type Color int

const (
	RedLedColor Color = iota
	GreenLedColor
	BlueLedColor
	WhiteLedColor
)

func LookupColor(colorName string) Color {
	switch strings.ToLower(colorName) {
	case "red":
		return RedLedColor
	case "green":
		return GreenLedColor
	case "blue":
		return BlueLedColor
	case "white":
		return WhiteLedColor
	default:
		return WhiteLedColor
	}
}

var (
	RedConfig = ledConfig{
		chipId: 4,
		pwmId:  1,
	}
	BlueConfig = ledConfig{
		chipId: 2,
		pwmId:  0,
	}
	GreenConfig = ledConfig{
		chipId: 4,
		pwmId:  0,
	}
	WhiteConfig = ledConfig{
		chipId: 2,
		pwmId:  1,
	}
)
