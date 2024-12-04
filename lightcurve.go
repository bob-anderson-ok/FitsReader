package main

import (
	"FITSreader/fitsio"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"time"
)

func showFlashLightcurve() {

	buildPlot() // Writes flashLightcurve.png in current working directory

	pngWin := myWin.App.NewWindow("'flash' lightcurve")
	pngWin.Resize(fyne.Size{Height: 450, Width: 1500})

	testImage := canvas.NewImageFromFile("flashLightcurve.png")
	pngWin.SetContent(testImage)
	pngWin.CenterOnScreen()
	pngWin.Show()
}

func showSysTimePlots() {

	buildStartTimePlot() // Writes timestampPlot.png in current working directory
	//buildDeadtimePlot()      // Writes deadtimePlot.png
	buildFrameDeltasPlot() // Writes frameDeltasPlot.png
	//buildExposureTimesPlot() // Writes exposureTimesPlot.png

	timestampWin := myWin.App.NewWindow("system timestamp plot")
	timestampWin.Resize(fyne.Size{Height: 500, Width: 1500})
	timestampImage := canvas.NewImageFromFile("timestampPlot.png")
	timestampWin.SetContent(timestampImage)
	timestampWin.CenterOnScreen()
	timestampWin.Show()

	//deadTimeWin := myWin.App.NewWindow("dead time plot")
	//deadTimeWin.Resize(fyne.Size{Height: 500, Width: 1500})
	//deadTimeImage := canvas.NewImageFromFile("deadtimePlot.png")
	//deadTimeWin.SetContent(deadTimeImage)
	//deadTimeWin.CenterOnScreen()
	//deadTimeWin.Show()

	frameDeltasWin := myWin.App.NewWindow("frame to frame time deltas plot")
	frameDeltasWin.Resize(fyne.Size{Height: 500, Width: 1500})
	frameDeltasImage := canvas.NewImageFromFile("frameDeltasPlot.png")
	frameDeltasWin.SetContent(frameDeltasImage)
	frameDeltasWin.CenterOnScreen()
	frameDeltasWin.Show()

	//exposureTimesWin := myWin.App.NewWindow("exposure times plot")
	//exposureTimesWin.Resize(fyne.Size{Height: 500, Width: 1500})
	//exposureTimesImage := canvas.NewImageFromFile("exposureTimesPlot.png")
	//exposureTimesWin.SetContent(exposureTimesImage)
	//exposureTimesWin.CenterOnScreen()
	//exposureTimesWin.Show()
}

func buildPlot() {

	n := len(myWin.lightcurve)
	if n == 0 {
		return
	}

	myPts := make(plotter.XYs, n)
	for i := range myPts {
		myPts[i].X = float64(i + myWin.lightCurveStartIndex)
		myPts[i].Y = myWin.lightcurve[i]
	}

	plot.DefaultFont = font.Font{Typeface: "Liberation", Variant: "Sans", Style: 0, Weight: 3, Size: font.Points(20)}

	plt := plot.New()
	plt.X.Min = 0
	plt.X.Max = float64(myWin.numFiles)
	plt.Title.Text = "'flash' lightcurve"
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

func sysTimeToSeconds(t time.Time) float64 {
	seconds := t.Unix()
	nanoseconds := t.Nanosecond()
	return float64(seconds) + float64(nanoseconds)/1_000_000_000.0
}

func buildStartTimePlot() {

	n := len(myWin.sysStartTimes)
	if n == 0 {
		return
	}

	myPts := make(plotter.XYs, n)
	for i := range myPts {
		myPts[i].Y = float64(i)
		myPts[i].X = sysTimeToSeconds(myWin.sysStartTimes[i])
	}

	plot.DefaultFont = font.Font{Typeface: "Liberation", Variant: "Sans", Style: 0, Weight: 3, Size: font.Points(20)}

	plt := plot.New()
	plt.X.Min = sysTimeToSeconds(myWin.sysStartTimes[0])
	plt.X.Max = sysTimeToSeconds(myWin.sysStartTimes[len(myWin.sysStartTimes)-1])
	plt.Title.Text = "system timestamp plot"
	plt.X.Label.Text = "time"
	plt.Y.Label.Text = "reading number"

	plotutil.DefaultGlyphShapes[0] = plotutil.Shape(5) // set point shape to filled circle

	err := plotutil.AddScatters(plt, myPts)
	if err != nil {
		panic(err)
	}

	err = plt.Save(21*vg.Inch, 6*vg.Inch, "timestampPlot.png")
	if err != nil {
		panic(err)
	}
}

func buildDeadtimePlot() {
	n := len(myWin.sysDeadtimeSeconds)
	if n == 0 {
		return
	}

	myPts := make(plotter.XYs, n)
	for i := range myPts {
		myPts[i].X = float64(i)
		myPts[i].Y = myWin.sysDeadtimeSeconds[i]
	}

	plot.DefaultFont = font.Font{Typeface: "Liberation", Variant: "Sans", Style: 0, Weight: 3, Size: font.Points(20)}

	plt := plot.New()
	plt.X.Min = 0
	plt.X.Max = float64(n)
	plt.Title.Text = "frame dead time plot"
	plt.X.Label.Text = "reading number"
	plt.Y.Label.Text = "dead time (seconds)"

	plotutil.DefaultGlyphShapes[0] = plotutil.Shape(5) // set point shape to filled circle

	err := plotutil.AddScatters(plt, myPts)
	if err != nil {
		panic(err)
	}

	err = plt.Save(21*vg.Inch, 6*vg.Inch, "deadtimePlot.png")
	if err != nil {
		panic(err)
	}
}

func buildFrameDeltasPlot() {
	n := len(myWin.sysTimeDeltaSeconds)
	if n == 0 {
		return
	}

	myPts := make(plotter.XYs, n)
	for i := range myPts {
		myPts[i].X = float64(i)
		myPts[i].Y = myWin.sysTimeDeltaSeconds[i]
	}

	plot.DefaultFont = font.Font{Typeface: "Liberation", Variant: "Sans", Style: 0, Weight: 3, Size: font.Points(20)}

	plt := plot.New()
	plt.X.Min = 0
	plt.X.Max = float64(n)
	plt.Title.Text = "frame to frame time delta plot"
	plt.X.Label.Text = "reading number"
	plt.Y.Label.Text = "delta frame (seconds)"

	plotutil.DefaultGlyphShapes[0] = plotutil.Shape(5) // set point shape to filled circle

	err := plotutil.AddScatters(plt, myPts)
	if err != nil {
		panic(err)
	}

	err = plt.Save(21*vg.Inch, 6*vg.Inch, "frameDeltasPlot.png")
	if err != nil {
		panic(err)
	}
}

func buildExposureTimesPlot() {
	n := len(myWin.sysExposureSeconds)
	if n == 0 {
		return
	}

	myPts := make(plotter.XYs, n)
	for i := range myPts {
		myPts[i].X = float64(i)
		myPts[i].Y = myWin.sysExposureSeconds[i]
	}

	plot.DefaultFont = font.Font{Typeface: "Liberation", Variant: "Sans", Style: 0, Weight: 3, Size: font.Points(20)}

	plt := plot.New()
	plt.X.Min = 0
	plt.X.Max = float64(n)
	plt.Title.Text = "exposure times"
	plt.X.Label.Text = "reading number"
	plt.Y.Label.Text = "exposure time (seconds)"

	plotutil.DefaultGlyphShapes[0] = plotutil.Shape(5) // set point shape to filled circle

	err := plotutil.AddScatters(plt, myPts)
	if err != nil {
		panic(err)
	}

	err = plt.Save(21*vg.Inch, 6*vg.Inch, "exposureTimesPlot.png")
	if err != nil {
		panic(err)
	}
}

//func askIfLoopPointsAreToBeUsed() {
//	startFrameWidget := widget.NewEntry()
//	endFrameWidget := widget.NewEntry()
//	startFrameWidget.Text = fmt.Sprintf("%d", myWin.loopStartIndex)
//	endFrameWidget.Text = fmt.Sprintf("%d", myWin.loopEndIndex)
//	item1 := widget.NewFormItem("start index", startFrameWidget)
//	item2 := widget.NewFormItem("end index", endFrameWidget)
//	items := []*widget.FormItem{item1, item2}
//	loopPointQuery := dialog.NewForm("Should loop start and end indices be used\n"+
//		"to bracket lightcurve?", "Use", "Don't use", items,
//		func(useLoopPoints bool) { processLoopPointUsageAnswer(useLoopPoints) }, myWin.parentWindow)
//	loopPointQuery.Show()
//}

//func processLoopPointUsageAnswer(useLoopPoints bool) {
//	if useLoopPoints {
//		myWin.lightCurveStartIndex = myWin.loopStartIndex
//		myWin.lightCurveEndIndex = myWin.loopEndIndex
//	} else {
//		myWin.lightCurveStartIndex = 0
//		myWin.lightCurveEndIndex = myWin.numFiles - 1
//	}
//	runLightcurveAcquisition()
//}

func pixelSum() float64 {
	var pixelSum float64

	imagePixels := myWin.primaryHDU.(fitsio.Image).Raw()
	for i := 0; i < len(imagePixels); i += 1 {
		pixelSum += float64(imagePixels[i])
	}
	return pixelSum
}
