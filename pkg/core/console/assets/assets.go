package assets

import (
	"embed"
	"encoding/base64"
	"log"
)

var (
	//go:embed images/OpenAudioProtocol-Logo-inverse-v1.0.svg
	imagesFs embed.FS
	//go:embed js/main.js
	mainJS []byte
)

var (
	OAPLogoInverse string
	MainJS         string
)

func init() {
	svgContent, err := imagesFs.ReadFile("images/OpenAudioProtocol-Logo-inverse-v1.0.svg")
	if err != nil {
		log.Fatalf("SVG not found: %v", err)
	}
	encodedSVG := base64.StdEncoding.EncodeToString(svgContent)
	OAPLogoInverse = "data:image/svg+xml;base64," + encodedSVG

	MainJS = string(mainJS)
}
