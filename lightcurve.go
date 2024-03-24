package main

import (
	"encoding/binary"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/astrogo/fitsio"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"math"
)

func showFlashLightcurve() {

	buildPlot() // Writes flashLightcurve.png in current working directory

	pngWin := myWin.App.NewWindow("'flash' lightcurve")
	pngWin.Resize(fyne.Size{Height: 450, Width: 1400})

	testImage := canvas.NewImageFromFile("flashLightcurve.png")
	pngWin.SetContent(testImage)
	pngWin.CenterOnScreen()
	pngWin.Show()
}

func buildPlot() {

	n := len(myWin.lightcurve)
	myPts := make(plotter.XYs, n)
	for i := range myPts {
		myPts[i].X = float64(i + myWin.lightCurveStartIndex)
		myPts[i].Y = myWin.lightcurve[i]
	}

	plot.DefaultFont = font.Font{Typeface: "Liberation", Variant: "Sans", Style: 0, Weight: 3, Size: font.Points(20)}

	plt := plot.New()
	plt.X.Min = 0
	plt.X.Max = float64(myWin.numFiles + 5)
	plt.Title.Text = "'flash' lightcurve'"
	plt.X.Label.Text = "frame index"
	plt.Y.Label.Text = "intensity"

	plotutil.DefaultGlyphShapes[0] = plotutil.Shape(5) // set point shape to filled circle

	err := plotutil.AddScatters(plt, myPts)
	if err != nil {
		panic(err)
	}

	err = plt.Save(21*vg.Inch, 6*vg.Inch, "flashLightcurve.png")
	if err != nil {
		panic(err)
	}
}

func askIfLoopPointsAreToBeUsed() {
	startFrameWidget := widget.NewEntry()
	endFrameWidget := widget.NewEntry()
	startFrameWidget.Text = fmt.Sprintf("%d", myWin.loopStartIndex)
	endFrameWidget.Text = fmt.Sprintf("%d", myWin.loopEndIndex)
	item1 := widget.NewFormItem("start index", startFrameWidget)
	item2 := widget.NewFormItem("end index", endFrameWidget)
	items := []*widget.FormItem{item1, item2}
	loopPointQuery := dialog.NewForm("Should loop start and end indices be used\n"+
		"to bracket lightcurve?", "Use", "Don't use", items,
		func(useLoopPoints bool) { processLoopPointUsageAnswer(useLoopPoints) }, myWin.parentWindow)
	loopPointQuery.Show()
}

func processLoopPointUsageAnswer(useLoopPoints bool) {
	if useLoopPoints {
		myWin.lightCurveStartIndex = myWin.loopStartIndex
		myWin.lightCurveEndIndex = myWin.loopEndIndex
	} else {
		myWin.lightCurveStartIndex = 0
		myWin.lightCurveEndIndex = myWin.numFiles - 1
	}
	runLightcurveAcquisition()
}

func pixelSum() float64 {
	kind := myWin.imageKind
	var pixelSum float64

	switch kind {
	case "Gray":
		bob := myWin.primaryHDU.(fitsio.Image).Raw()
		for i := 0; i < len(bob); i += 1 {
			pixelSum += float64(bob[i])
		}
	case "Gray16":
		bob := myWin.primaryHDU.(fitsio.Image).Raw()
		for i := 0; i < len(bob)-2; i += 2 {
			valueUint16 := binary.LittleEndian.Uint16(bob[i : i+2])
			pixelSum += float64(valueUint16)
		}
	case "Gray32":
		bob := myWin.primaryHDU.(fitsio.Image).Raw()
		for i := 0; i < len(bob)-4; i += 4 {
			pixelSum += math.Float64frombits(binary.BigEndian.Uint64(bob[i : i+4]))
		}
	case "Gray64":
		bob := myWin.primaryHDU.(fitsio.Image).Raw()
		for i := 0; i < len(bob)-8; i += 8 {
			pixelSum += math.Float64frombits(binary.BigEndian.Uint64(bob[i : i+8]))
		}
	default:
		msg := fmt.Sprintf("Unexpected 'kind': %s", kind)
		dialog.ShowInformation("Oops", msg, myWin.parentWindow)
	}
	return pixelSum
}
