package main

import (
	"log"

	"github.com/signintech/gopdf"
)

const heightOffset = 260
const fontSize = 48

const defaultFont = "Fonts/Inter_18pt-SemiBold.ttf"

var PageSizeLetterLandscape = gopdf.Rect{
	W: gopdf.PageSizeLetter.H,
	H: gopdf.PageSizeLetter.W,
}

func generate(names []string, inputPath string, outputPath string) {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: PageSizeLetterLandscape})

	template := pdf.ImportPage(inputPath, 1, "/MediaBox")

	err := pdf.AddTTFFont("inter", defaultFont)
	if err != nil {
		log.Printf("Error adding font: %v", err)
		return
	}

	err = pdf.SetFont("inter", "", 14)
	if err != nil {
		log.Printf("Error setting font: %v", err)
		return
	}

	// These are flipped since we are using landscape mode
	width, height := PageSizeLetterLandscape.W, PageSizeLetterLandscape.H

	for _, name := range names {
		pdf.AddPage()
		pdf.UseImportedTemplate(template, 0, 0, width, height)

		pdf.SetFontSize(fontSize)

		textwidth, err := pdf.MeasureTextWidth(name)
		if err != nil {
			log.Printf("Error measuring text width: %v", err)
			return
		}

		pdf.SetX((width - textwidth) / 2)
		pdf.SetY(height - heightOffset)
		pdf.Cell(nil, name)
	}

	err = pdf.WritePdf(outputPath)
	if err != nil {
		log.Printf("Error writing PDF: %v", err)
		return
	}

}
