package main

import (
	_ "embed"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/astrogo/fitsio"
	"github.com/astrogo/fitsio/fltimg"
	"github.com/montanaflynn/stats"
	_ "github.com/qdm12/reprint"
	"image"
	"image/color"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	displayBuffer        []byte
	workingBuffer        []byte
	bytesPerPixel        int
	roiEntry             *dialog.FormDialog
	widthStr             binding.String
	heightStr            binding.String
	roiWidth             int
	roiHeight            int
	roiActive            bool
	roiChanged           bool
	roiCenterXoffset     int // center has offset == 0
	roiCenterYoffset     int // center has offset == 0
	x0                   int // ROI corners
	y0                   int
	x1                   int
	y1                   int
	upButton             *widget.Button
	downButton           *widget.Button
	leftButton           *widget.Button
	rightButton          *widget.Button
	centerButton         *widget.Button
	drawROIbutton        *widget.Button
	roiCheckbox          *widget.Check
	setRoiButton         *widget.Button
	parentWindow         fyne.Window
	imageWidth           int
	imageHeight          int
	App                  fyne.App
	whiteSlider          *widget.Slider
	blackSlider          *widget.Slider
	autoContrastNeeded   bool
	fileSlider           *widget.Slider
	centerContent        *fyne.Container
	fitsFilePaths        []string
	numFiles             int
	waitingForFileRead   bool
	fitsImages           []*canvas.Image
	originalImage        *canvas.Image
	centerUnderlay       *canvas.Image
	imageKind            string
	fileLabel            *widget.Label
	timestampLabel       *canvas.Text
	busyLabel            *canvas.Text
	fileIndex            int
	autoPlayEnabled      bool
	playBackMilliseconds int64
	currentFilePath      string
	playDelay            time.Duration
	primaryHDU           fitsio.HDU
	timestamps           []string
	metaData             [][]string
	timestamp            string
	loopStartIndex       int
	loopEndIndex         int
}

const version = " 1.2.3"

//go:embed help.txt
var helpText string

var myWin Config

func main() {

	// We supply an ID (hopefully unique) because we need to use the preferences API
	myApp := app.NewWithID("com.gmail.ok.anderson.bob")
	myWin.App = myApp

	// We start app using the dark theme. There are buttons to allow theme change
	myApp.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantDark})

	myWin.autoPlayEnabled = false
	myWin.loopStartIndex = -1
	myWin.loopEndIndex = -1

	myWin.widthStr = binding.NewString()
	myWin.heightStr = binding.NewString()
	myWin.roiActive = false

	widthStr := myWin.App.Preferences().StringWithFallback("ROIwidth", "600")
	heightStr := myWin.App.Preferences().StringWithFallback("ROIheight", "400")
	roiCenterXstr := myWin.App.Preferences().StringWithFallback("ROIcenterX", "0")
	roiCenterYstr := myWin.App.Preferences().StringWithFallback("ROIcenterY", "0")

	_ = myWin.widthStr.Set(widthStr)   // Ignore possibility of error
	_ = myWin.heightStr.Set(heightStr) // Ignore possibility of error

	myWin.roiWidth, _ = strconv.Atoi(widthStr)              // Ignore error
	myWin.roiHeight, _ = strconv.Atoi(heightStr)            // Ignore error
	myWin.roiCenterXoffset, _ = strconv.Atoi(roiCenterXstr) // Ignore error
	myWin.roiCenterYoffset, _ = strconv.Atoi(roiCenterYstr) // Ignore error

	myWin.roiChanged = false

	w := myApp.NewWindow("IOTA FITS video player" + version)
	w.Resize(fyne.Size{Height: 800, Width: 1200})

	myWin.parentWindow = w

	sliderWhite := widget.NewSlider(0, 255)
	sliderWhite.OnChanged = func(value float64) { displayFitsImage() }
	sliderWhite.Orientation = 1
	sliderWhite.Value = 128
	myWin.whiteSlider = sliderWhite

	sliderBlack := widget.NewSlider(0, 255)
	sliderBlack.Orientation = 1
	sliderBlack.Value = 0
	sliderBlack.OnChanged = func(value float64) { displayFitsImage() }
	myWin.blackSlider = sliderBlack

	rightItem := container.NewHBox(sliderBlack, sliderWhite)

	leftItem := container.NewVBox()
	leftItem.Add(widget.NewButton("Select fits folder", func() { chooseFitsFolder() }))
	leftItem.Add(widget.NewButton("Show meta-data", func() { showMetaData() }))
	selector := widget.NewSelect([]string{"1 fps", "5 fps", "10 fps", "25 fps", "30 fps", "max"},
		func(opt string) { setPlayDelay(opt) })
	selector.PlaceHolder = "Set play fps"
	selector.SetSelectedIndex(2)
	myWin.playDelay = 97 * time.Millisecond // 100 - 3
	leftItem.Add(selector)

	leftItem.Add(widget.NewButton("Help", func() { showSplash() }))

	// These are left in if somebody requests a white theme option using buttons
	//leftItem.Add(widget.NewButton("Dark theme", func() {
	//	myApp.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantDark})
	//}))
	//leftItem.Add(widget.NewButton("Light theme", func() {
	//	myApp.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantLight})
	//}))

	// This lets the user pick the white theme by putting anything at all on the command line.
	if len(os.Args) > 1 {
		myApp.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantLight})
	}

	leftItem.Add(layout.NewSpacer())
	myWin.roiCheckbox = widget.NewCheck("Apply ROI", func(checked bool) { applyRoi(checked) })
	leftItem.Add(myWin.roiCheckbox)
	myWin.setRoiButton = widget.NewButton("Set ROI size", func() { roiEntry() })
	leftItem.Add(myWin.setRoiButton)

	up := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() { moveRoiUp() })
	down := widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() { moveRoiDown() })
	left := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() { moveRoiLeft() })
	right := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() { moveRoiRight() })
	center := widget.NewButtonWithIcon("", theme.MediaRecordIcon(), func() { moveRoiCenter() })

	myWin.upButton = up
	myWin.downButton = down
	myWin.leftButton = left
	myWin.rightButton = right
	myWin.centerButton = center

	toolBar1 := container.NewGridWithColumns(3)
	toolBar1.Add(widget.NewToolbar(widget.ToolbarItem(widget.NewToolbarSpacer())))
	toolBar1.Add(up)
	toolBar1.Add(widget.NewToolbar(widget.ToolbarItem(widget.NewToolbarSpacer())))

	toolBar2 := container.NewGridWithColumns(3)
	toolBar2.Add(left)
	toolBar2.Add(center)
	toolBar2.Add(right)

	toolBar3 := container.NewGridWithColumns(3)
	toolBar3.Add(widget.NewToolbar(widget.ToolbarItem(widget.NewToolbarSpacer())))
	toolBar3.Add(down)
	toolBar3.Add(widget.NewToolbar(widget.ToolbarItem(widget.NewToolbarSpacer())))

	leftItem.Add(toolBar1)
	leftItem.Add(toolBar2)
	leftItem.Add(toolBar3)

	myWin.drawROIbutton = widget.NewButton("Show ROI", func() { showROI() })
	leftItem.Add(myWin.drawROIbutton)

	disableRoiControls()

	leftItem.Add(layout.NewSpacer())
	leftItem.Add(widget.NewButton("Set loop start", func() { setLoopStart() }))
	leftItem.Add(widget.NewButton("Set loop end", func() { setLoopEnd() }))
	leftItem.Add(widget.NewButton("Run loop", func() { go runLoop() }))

	myWin.fileLabel = widget.NewLabel("File name goes here")

	myWin.timestampLabel = canvas.NewText("<timestamp goes here>", color.NRGBA{R: 255, A: 255})
	myWin.timestampLabel.TextSize = 25

	row1 := container.NewHBox(layout.NewSpacer(), myWin.timestampLabel, layout.NewSpacer())
	row2 := container.NewHBox(layout.NewSpacer(), myWin.fileLabel, layout.NewSpacer())

	myWin.fileSlider = widget.NewSlider(0, 0) // Default max - will be set by getFitsFileNames()
	myWin.fileSlider.OnChanged = func(value float64) { processFileSliderMove(value) }

	toolBar := container.NewHBox()
	toolBar.Add(layout.NewSpacer()) // To center the buttons (in conjunction with its "mate")
	toolBar.Add(widget.NewButton("-1", func() { processBackOneFrame() }))
	toolBar.Add(widget.NewButton("<", func() { go playBackward(false) }))
	toolBar.Add(widget.NewButton("||", func() { pauseAutoPlay() }))
	toolBar.Add(widget.NewButton(">", func() { go playForward(false) }))
	toolBar.Add(widget.NewButton("+1", func() { processForwardOneFrame() }))
	toolBar.Add(layout.NewSpacer()) // To center the buttons (in conjunction with its "mate")

	bottomItem := container.NewVBox(myWin.fileSlider, toolBar, row1, row2)

	centerItem := widget.NewLabel("") // Blank placeholder
	centerContent := container.NewBorder(
		nil,
		bottomItem,
		leftItem,
		rightItem,
		centerItem)

	myWin.centerContent = centerContent
	w.SetContent(myWin.centerContent)
	w.CenterOnScreen()

	w.ShowAndRun()
}

func showROI() {
	if myWin.roiActive {
		myWin.roiCheckbox.SetChecked(false)
		myWin.roiActive = false
		displayFitsImage()
	}

	x0 := myWin.roiCenterXoffset + myWin.imageWidth/2 - myWin.roiWidth/2
	y0 := myWin.roiCenterYoffset + myWin.imageHeight/2 - myWin.roiHeight/2
	x1 := x0 + myWin.roiWidth
	y1 := y0 + myWin.roiHeight

	//fmt.Printf("x0: %d  y0: %d   x1: %d  y1: %d\n", x0, y0, x1, y1)

	if myWin.imageKind == "Gray16" {
		for i := x0; i < x1; i++ {
			myWin.fitsImages[0].Image.(*image.Gray16).Set(i, y0, color.White)
		}
		for i := x0; i < x1+1; i++ {
			myWin.fitsImages[0].Image.(*image.Gray16).Set(i, y1, color.White)
		}
		for i := y0; i < y1; i++ {
			myWin.fitsImages[0].Image.(*image.Gray16).Set(x0, i, color.White)
		}
		for i := y0; i < y1; i++ {
			myWin.fitsImages[0].Image.(*image.Gray16).Set(x1, i, color.White)
		}
	}

	if myWin.imageKind == "Gray" {
		for i := x0; i < x1; i++ {
			myWin.fitsImages[0].Image.(*image.Gray).Set(i, y0, color.White)
		}
		for i := x0; i < x1+1; i++ {
			myWin.fitsImages[0].Image.(*image.Gray).Set(i, y1, color.White)
		}
		for i := y0; i < y1; i++ {
			myWin.fitsImages[0].Image.(*image.Gray).Set(x0, i, color.White)
		}
		for i := y0; i < y1; i++ {
			myWin.fitsImages[0].Image.(*image.Gray).Set(x1, i, color.White)
		}
	}
	myWin.centerContent.Refresh()
}

func moveRoiCenter() {
	myWin.roiCenterXoffset = 0
	myWin.roiCenterYoffset = 0
	myWin.roiChanged = true

	myWin.App.Preferences().SetString("ROIcenterX", fmt.Sprintf("%d", myWin.roiCenterXoffset))
	myWin.App.Preferences().SetString("ROIcenterY", fmt.Sprintf("%d", myWin.roiCenterYoffset))

	displayFitsImage()
	showROI()
}

func moveRoiUp() {
	myWin.roiCenterYoffset -= 20 // Move the image selection region down
	myWin.App.Preferences().SetString("ROIcenterY", fmt.Sprintf("%d", myWin.roiCenterYoffset))

	myWin.roiChanged = true
	displayFitsImage()
	showROI()
}

func moveRoiDown() {
	myWin.roiCenterYoffset += 20 // Move the image selection region up
	myWin.App.Preferences().SetString("ROIcenterY", fmt.Sprintf("%d", myWin.roiCenterYoffset))

	myWin.roiChanged = true
	displayFitsImage()
	showROI()
}

func moveRoiLeft() {
	myWin.roiCenterXoffset -= 20
	myWin.App.Preferences().SetString("ROIcenterX", fmt.Sprintf("%d", myWin.roiCenterXoffset))

	myWin.roiChanged = true
	displayFitsImage()
	showROI()
}

func moveRoiRight() {
	myWin.roiCenterXoffset += 20
	myWin.App.Preferences().SetString("ROIcenterX", fmt.Sprintf("%d", myWin.roiCenterXoffset))

	myWin.roiChanged = true
	displayFitsImage()
	showROI()
}

func applyRoi(checked bool) {

	myWin.roiActive = checked
	if checked {
		//printImageStats("apply ROI: ")
		myWin.roiChanged = true
		displayFitsImage()
	} else {
		//printImageStats("remove ROI: ")
		makeDisplayBuffer(myWin.imageWidth, myWin.imageHeight)
		restoreRect()
		displayFitsImage()
	}
}

func enableRoiControls() {
	myWin.roiCheckbox.Enable()
	myWin.setRoiButton.Enable()
	myWin.upButton.Enable()
	myWin.downButton.Enable()
	myWin.leftButton.Enable()
	myWin.rightButton.Enable()
	myWin.centerButton.Enable()
	myWin.drawROIbutton.Enable()
}

func disableRoiControls() {
	myWin.roiCheckbox.Disable()
	myWin.setRoiButton.Disable()
	myWin.upButton.Disable()
	myWin.downButton.Disable()
	myWin.leftButton.Disable()
	myWin.rightButton.Disable()
	myWin.centerButton.Disable()
	myWin.drawROIbutton.Disable()
}

func roiEntry() {
	widthWidget := widget.NewEntryWithData(myWin.widthStr)
	heightWidget := widget.NewEntryWithData(myWin.heightStr)
	item1 := widget.NewFormItem("width", widthWidget)
	item2 := widget.NewFormItem("height", heightWidget)
	items := []*widget.FormItem{item1, item2}
	myWin.roiEntry = dialog.NewForm("Enter ROI information", "OK", "Cancel", items,
		func(ok bool) { processRoiEntryInfo(ok) }, myWin.parentWindow)
	myWin.roiEntry.Show()
}

func processRoiEntryInfo(ok bool) {
	if ok {
		widthStr, err0 := myWin.widthStr.Get()
		if err0 != nil {
			dialog.ShowInformation("Oops", "format error", myWin.parentWindow)
		}
		heightStr, err1 := myWin.heightStr.Get()
		if err1 != nil {
			dialog.ShowInformation("Oops", "format error", myWin.parentWindow)
		}

		proposedRoiWidth, err2 := strconv.Atoi(widthStr)
		if err2 != nil {
			dialog.ShowInformation("Oops", "An integer is needed here.", myWin.parentWindow)
			_ = myWin.widthStr.Set(fmt.Sprintf("%d", myWin.roiWidth))
			return
		}

		proposedRoiHeight, err3 := strconv.Atoi(heightStr)
		if err3 != nil {
			dialog.ShowInformation("Oops", "An integer is needed here.", myWin.parentWindow)
			_ = myWin.heightStr.Set(fmt.Sprintf("%d", myWin.roiHeight))
			return
		}

		if proposedRoiWidth < 1 {
			dialog.ShowInformation("Oops", "An integer > 0 is needed for ROI width.", myWin.parentWindow)
			_ = myWin.widthStr.Set(fmt.Sprintf("%d", myWin.roiWidth))
			return
		}

		if proposedRoiHeight < 1 {
			dialog.ShowInformation("Oops", "A integer > 0 is needed for ROI height.", myWin.parentWindow)
			_ = myWin.heightStr.Set(fmt.Sprintf("%d", myWin.roiHeight))
			return
		}

		myWin.App.Preferences().SetString("ROIwidth", widthStr)
		myWin.App.Preferences().SetString("ROIheight", heightStr)

		myWin.roiHeight = proposedRoiHeight
		myWin.roiWidth = proposedRoiWidth

		// This causes the ROI change to be applied to the current image
		myWin.roiChanged = true
		makeDisplayBuffer(myWin.roiWidth, myWin.roiHeight)
		displayFitsImage()
	} else {
		// User cancelled - restore old values
		_ = myWin.widthStr.Set(fmt.Sprintf("%d", myWin.roiWidth))
		_ = myWin.heightStr.Set(fmt.Sprintf("%d", myWin.roiHeight))
		myWin.roiChanged = false
	}
}

func runLoop() {
	if myWin.loopStartIndex < 0 {
		dialog.ShowInformation("Oops", "You need to Set loop start", myWin.parentWindow)
		return
	}

	if myWin.loopEndIndex < 0 {
		dialog.ShowInformation("Oops", "You need to Set loop end", myWin.parentWindow)
		return
	}

	if myWin.loopStartIndex > myWin.loopEndIndex {
		playBackward(true)
	} else {
		playForward(true)
	}
}

func setLoopStart() {
	myWin.loopStartIndex = int(myWin.fileSlider.Value)
}

func setLoopEnd() {
	myWin.loopEndIndex = int(myWin.fileSlider.Value)
}

type forcedVariant struct {
	fyne.Theme

	variant fyne.ThemeVariant
}

func (f *forcedVariant) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return f.Theme.Color(name, f.variant)
}

func pauseAutoPlay() {
	myWin.autoPlayEnabled = false
}

func playForward(loop bool) {
	if !checkForFrameRateSelected() {
		return
	}

	var endPoint int
	if loop {
		endPoint = myWin.loopEndIndex
	} else {
		endPoint = myWin.numFiles - 1
	}

	if myWin.autoPlayEnabled { // This deals with the user re-clicking the play > button
		return // autoPlay is already running
	}

	myWin.autoPlayEnabled = true // This can/will be set to false by clicking the pause button

	for {
		if !myWin.autoPlayEnabled { // This is how we break out of the forever loop
			return
		}
		if myWin.fileIndex >= endPoint {
			// End point reached.
			if !loop {
				myWin.autoPlayEnabled = false
				return
			} else {
				// We go back to the loop start (the -1 is because processForwardOneFrame() increments myWin.fileIndex
				// before it displays the file at myWin.fileIndex
				myWin.fileIndex = myWin.loopStartIndex - 1
			}
		}
		// This flag will become true after file has been read and displayed by displayFitsImage()
		myWin.waitingForFileRead = true
		// This will increment myWin.fileIndex invoke getFItsImage() to display the image from that file
		processForwardOneFrame()
		for {
			if myWin.waitingForFileRead {
				time.Sleep(1 * time.Millisecond)
			} else {
				break
			}
		}
		time.Sleep(myWin.playDelay)
	}
}

func checkForFrameRateSelected() bool {
	if myWin.playDelay == time.Duration(0) {
		dialog.ShowInformation(
			"Something to do ...",
			"Select a playback frame rate.",
			myWin.parentWindow,
		)
		return false
	}
	return true
}

func setPlayDelay(opt string) {
	myWin.playDelay = 100 * time.Millisecond
	switch opt {
	case "1 fps":
		myWin.playDelay = 997 * time.Millisecond // 1000 - 3  (3 is fudge for display time)
	case "5 fps":
		myWin.playDelay = 197 * time.Millisecond // 200 - 3
	case "10 fps":
		myWin.playDelay = 97 * time.Millisecond // 100 - 3
	case "25 fps":
		myWin.playDelay = 37 * time.Millisecond // 40 - 3
	case "30 fps":
		myWin.playDelay = 30 * time.Millisecond // 33 - 3
	case "max":
		myWin.playDelay = 10 * time.Microsecond
	default:
		fmt.Printf("Unexpected frame rate of %s found in setPlayDelay()", opt)
	}
}

func playBackward(loop bool) {
	if !checkForFrameRateSelected() {
		return
	}

	if myWin.autoPlayEnabled {
		return // autoPlay is already running
	}

	var endPoint int
	if loop {
		endPoint = myWin.loopEndIndex
	} else {
		endPoint = 0
	}

	myWin.autoPlayEnabled = true // This can be set to false by clicking the pause button
	for {
		if !myWin.autoPlayEnabled {
			return
		}
		if myWin.fileIndex <= endPoint {
			// End point reached.
			if !loop {
				myWin.autoPlayEnabled = false
				return
			} else {
				// We go back to the loop start (the +1 is because processBackwardOneFrame() decrements myWin.fileIndex
				// before it displays the file at myWin.fileIndex
				myWin.fileIndex = myWin.loopStartIndex + 1
			}
		}
		// This flag will become true after file has been read and displayed by displayFitsImage()
		myWin.waitingForFileRead = true
		// This will decrement myWin.fileIndex and invoke getFItsImage() to display the image from that file
		processBackOneFrame()
		for {
			if myWin.waitingForFileRead {
				time.Sleep(1 * time.Millisecond)
			} else {
				break
			}
		}
		time.Sleep(myWin.playDelay)
	}
}

func processBackOneFrame() {
	numFrames := myWin.numFiles
	if numFrames == 0 {
		return
	}
	myWin.fileIndex -= 1
	if myWin.fileIndex < 0 {
		myWin.fileIndex += 1
		myWin.fileSlider.SetValue(0)
	} else {
		myWin.currentFilePath = myWin.fitsFilePaths[myWin.fileIndex]
	}
	myWin.fileSlider.SetValue(float64(myWin.fileIndex)) // This causes a call to displayFitsImage
	myWin.fileSlider.Refresh()
	return
}

func processForwardOneFrame() {
	numFrames := myWin.numFiles
	if numFrames == 0 {
		return
	}
	myWin.fileIndex += 1
	if myWin.fileIndex >= numFrames {
		myWin.fileIndex = numFrames - 1
	} else {
		myWin.currentFilePath = myWin.fitsFilePaths[myWin.fileIndex]
	}
	myWin.fileSlider.SetValue(float64(myWin.fileIndex)) // This causes a call to displayFitsImage()
	return
}

func processFileSliderMove(position float64) {
	myWin.fileIndex = int(position)
	myWin.fileLabel.SetText(myWin.fitsFilePaths[myWin.fileIndex])
	myWin.currentFilePath = myWin.fitsFilePaths[myWin.fileIndex]
	displayFitsImage()
}

func chooseFitsFolder() {
	showFolder := dialog.NewFolderOpen(
		func(path fyne.ListableURI, err error) { processFitsFolderSelection(path, err) },
		myWin.parentWindow,
	)
	showFolder.Resize(fyne.Size{
		Width:  800,
		Height: 600,
	})
	lastFitsFolderStr := myWin.App.Preferences().StringWithFallback("lastFitsFolder", "")

	if lastFitsFolderStr != "" {
		uriOfLastFitsFolder := storage.NewFileURI(lastFitsFolderStr)
		fitsDir, err := storage.ListerForURI(uriOfLastFitsFolder)
		if err != nil {
			myWin.App.Preferences().SetString("lastFitsFolder", "")
			lastFitsFolderStr = ""
		}

		showFolder.SetLocation(fitsDir)
		myWin.autoContrastNeeded = true
	}

	showFolder.Show()
}

func processFitsFolderSelection(path fyne.ListableURI, err error) {
	if err != nil {
		fmt.Println(fmt.Errorf("%w\n", err))
		return
	}
	if path != nil {
		//fmt.Printf("folder selected: %s\n", path)
		myWin.App.Preferences().SetString("lastFitsFolder", path.Path())
		myWin.fitsFilePaths = getFitsFilenames(path.Path())
		if len(myWin.fitsFilePaths) == 0 {
			dialog.ShowInformation("Oops",
				"No .fits files were found there!",
				myWin.parentWindow,
			)
			return
		}
		myWin.fileIndex = 0
		myWin.currentFilePath = myWin.fitsFilePaths[myWin.fileIndex]
		//fmt.Printf("%d fits files were found.\n", len(myWin.fitsFilePaths))
		myWin.fitsImages = []*canvas.Image{}
		myWin.timestamps = []string{}
		myWin.metaData = [][]string{}
		myWin.fileIndex = 0
		enableRoiControls()
		initializeImages()
		myWin.fileSlider.SetValue(0)
	}
	if len(myWin.fitsFilePaths) > 0 {
		displayFitsImage()
	}
}

func showMetaData() {
	helpWin := myWin.App.NewWindow("FITS Meta-data")
	helpWin.Resize(fyne.Size{Height: 600, Width: 700})
	_, metaDataList, _ := getFitsImageFromFilePath(myWin.fitsFilePaths[myWin.fileIndex])
	metaData := ""
	for _, line := range metaDataList {
		metaData += line + "\n"
	}
	scrollableText := container.NewVScroll(widget.NewRichTextWithText(metaData))
	helpWin.SetContent(scrollableText)
	helpWin.Show()
	helpWin.CenterOnScreen()
}

func displayFitsImage() fyne.CanvasObject {

	myWin.fileLabel.SetText(myWin.fitsFilePaths[myWin.fileIndex])

	// A side effect of this call is that myWin.displayBuffer is filled
	imageToUse, _, timestamp := getFitsImageFromFilePath(myWin.fitsFilePaths[myWin.fileIndex])
	myWin.timestampLabel.Text = timestamp

	if myWin.whiteSlider != nil {
		if myWin.imageKind == "Gray32" {
			imageToUse.Image.(*fltimg.Gray32).Max = float32(myWin.whiteSlider.Value)
			imageToUse.Image.(*fltimg.Gray32).Min = float32(myWin.blackSlider.Value)
		} else if myWin.imageKind == "Gray" {
			applyContrastControls(imageToUse.Image.(*image.Gray).Pix, myWin.displayBuffer, "Gray")
		} else if myWin.imageKind == "Gray16" {
			applyContrastControls(imageToUse.Image.(*image.Gray16).Pix, myWin.displayBuffer, "Gray16")
			myWin.fitsImages[0].Image.(*image.Gray16).Pix = myWin.displayBuffer
		} else {
			fmt.Printf("The image kind (%s) is unrecognized.\n", myWin.imageKind)
		}
	}

	if myWin.imageKind == "Gray16" {
		//printImageStats("@point 1")
		copy(myWin.fitsImages[0].Image.(*image.Gray16).Pix, myWin.displayBuffer)
		if !myWin.roiActive {
			restoreRect()
		} else {
			setROIrect()
		}
	}
	if myWin.imageKind == "Gray" {
		//printImageStats("@point 2")
		myWin.fitsImages[0].Image.(*image.Gray).Pix = myWin.displayBuffer
		if !myWin.roiActive {
			restoreRect()
		} else {
			setROIrect()
		}
	}
	myWin.centerContent.Objects[0] = myWin.fitsImages[0]

	myWin.centerContent.Refresh()
	myWin.waitingForFileRead = false // Signal to anyone waiting for file read completion

	return imageToUse
}

func restoreRect() {
	if myWin.imageKind == "Gray16" {
		myWin.fitsImages[0].Image.(*image.Gray16).Rect.Max = image.Point{
			X: myWin.imageWidth,
			Y: myWin.imageHeight,
		}
		myWin.fitsImages[0].Image.(*image.Gray16).Rect.Min = image.Point{
			X: 0,
			Y: 0,
		}
	}
	if myWin.imageKind == "Gray" {
		myWin.fitsImages[0].Image.(*image.Gray).Rect.Max = image.Point{
			X: myWin.imageWidth,
			Y: myWin.imageHeight,
		}
		myWin.fitsImages[0].Image.(*image.Gray).Rect.Min = image.Point{
			X: 0,
			Y: 0,
		}
	}
}

func setROIrect() {
	if myWin.imageKind == "Gray16" {
		myWin.fitsImages[0].Image.(*image.Gray16).Rect.Max = image.Point{
			X: myWin.x1,
			Y: myWin.y1,
		}
		myWin.fitsImages[0].Image.(*image.Gray16).Rect.Min = image.Point{
			X: myWin.x0,
			Y: myWin.y0,
		}
	}
	if myWin.imageKind == "Gray" {
		myWin.fitsImages[0].Image.(*image.Gray).Rect.Max = image.Point{
			X: myWin.x1,
			Y: myWin.x1,
		}
		myWin.fitsImages[0].Image.(*image.Gray).Rect.Min = image.Point{
			X: myWin.x0,
			Y: myWin.y0,
		}
	}
}

//func printImageStats(tag string) {
//	if myWin.imageKind == "Gray16" {
//		pixLength := len(myWin.fitsImages[0].Image.(*image.Gray16).Pix)
//		stride := myWin.fitsImages[0].Image.(*image.Gray16).Stride
//		rect := myWin.fitsImages[0].Image.(*image.Gray16).Rect
//		fmt.Printf("%s pixLength: %d  stride: %d  rect: %v\n", tag, pixLength, stride, rect)
//	}
//
//	if myWin.imageKind == "Gray" {
//		pixLength := len(myWin.fitsImages[0].Image.(*image.Gray).Pix)
//		stride := myWin.fitsImages[0].Image.(*image.Gray).Stride
//		rect := myWin.fitsImages[0].Image.(*image.Gray).Rect
//		fmt.Printf("%s pixLength: %d  stride: %d  rect: %v\n", tag, pixLength, stride, rect)
//	}
//}

func openFitsFile(fitsFilePath string) *fitsio.File {
	fileHandle, err1 := os.Open(fitsFilePath)
	if err1 != nil {
		errMsg := fmt.Errorf("os.Open() could not open %s: %w", fitsFilePath, err1)
		fmt.Printf(errMsg.Error())
		return nil
	}

	defer func(fileHandle *os.File) {
		err2 := fileHandle.Close()
		if err2 != nil {
			errMsg := fmt.Errorf("could not close %s: %w", fitsFilePath, err2)
			fmt.Printf(errMsg.Error())
		}
	}(fileHandle)

	fitsHandle, err3 := fitsio.Open(fileHandle)
	if err3 != nil {
		panic(err3)
	}

	return fitsHandle
}

func initializeImages() {

	// side effect: myWin.primaryHDU is set
	fitsImage, _, _ := getFitsImageFromFilePath(myWin.fitsFilePaths[0])

	myWin.imageWidth = fitsImage.Image.Bounds().Max.X
	myWin.imageHeight = fitsImage.Image.Bounds().Max.Y

	myWin.fitsImages = append(myWin.fitsImages, fitsImage)

	goImage := myWin.primaryHDU.(fitsio.Image).Image()
	kind := reflect.TypeOf(goImage).Elem().Name()

	switch kind {
	case "Gray":
		myWin.bytesPerPixel = 1
	case "Gray16":
		myWin.bytesPerPixel = 2
	case "Gray32":
		myWin.bytesPerPixel = 4
	default:
		msg := fmt.Sprintf("%s is not an image kind that is supported.", kind)
		dialog.ShowInformation("Oops", msg, myWin.parentWindow)
		return
	}

	makeDisplayBuffer(myWin.imageWidth, myWin.imageHeight)

	myWin.fileSlider.SetValue(0)
}

func getFitsImageFromFilePath(filePath string) (*canvas.Image, []string, string) {
	// This function has an important side effect: it fills the myWin.displayBuffer []byte
	f := openFitsFile(filePath)
	myWin.primaryHDU = f.HDU(0)
	metaData, timestamp := formatMetaData(myWin.primaryHDU)

	closeErr := f.Close()
	if closeErr != nil {
		errMsg := fmt.Errorf("could not close %s: %w", filePath, closeErr)
		fmt.Printf(errMsg.Error())
	}

	goImage := myWin.primaryHDU.(fitsio.Image).Image()
	kind := reflect.TypeOf(goImage).Elem().Name()
	myWin.imageKind = kind

	fitsImage := canvas.NewImageFromImage(goImage) // This is a Fyne image

	if myWin.roiActive {
		// Fix user setting ROI X size too large
		width := goImage.Bounds().Max.X
		if myWin.roiWidth > width {
			myWin.roiWidth = width
			_ = myWin.widthStr.Set(strconv.Itoa(width))
		}

		// Fix user setting ROI Y size too large
		height := goImage.Bounds().Max.Y
		if myWin.roiHeight > height {
			myWin.roiHeight = height
			_ = myWin.heightStr.Set(strconv.Itoa(height))
		}

		centerX := width / 2
		centerY := height / 2
		//fmt.Printf("width: %d  height: %d  centerX: %d  centerY: %d\n", width, height, centerX, centerY)
		x0 := centerX - myWin.roiWidth/2 + myWin.roiCenterXoffset
		y0 := centerY - myWin.roiHeight/2 + myWin.roiCenterYoffset
		x1 := x0 + myWin.roiWidth
		y1 := y0 + myWin.roiHeight

		myWin.x0 = x0
		myWin.y0 = y0
		myWin.x1 = x1
		myWin.y1 = y1

		//fmt.Println(x0, y0, x1, y1, image.Rect(x0, y0, x1, y1))

		if kind == "Gray16" {
			roi := goImage.(*image.Gray16).SubImage(image.Rect(x0, y0, x1, y1))
			fitsImage = canvas.NewImageFromImage(roi) // This is a Fyne image
			myWin.workingBuffer = make([]byte, len(fitsImage.Image.(*image.Gray16).Pix))
			copy(myWin.workingBuffer, fitsImage.Image.(*image.Gray16).Pix)
		}

		if kind == "Gray" {
			roi := goImage.(*image.Gray).SubImage(image.Rect(x0, y0, x1, y1))
			fitsImage = canvas.NewImageFromImage(roi) // This is a Fyne image
			myWin.workingBuffer = make([]byte, len(fitsImage.Image.(*image.Gray).Pix))
			copy(myWin.workingBuffer, fitsImage.Image.(*image.Gray).Pix)
		}

		if kind == "Gray32" {
			msg := "ROI selection not implemented for\nfloating point images."
			dialog.ShowInformation("Sorry", msg, myWin.parentWindow)
			myWin.roiCheckbox.SetChecked(false)
		} else {
			if myWin.roiChanged {
				myWin.roiChanged = false
				myWin.fitsImages[0] = fitsImage
				myWin.centerContent.Objects[0] = fitsImage
			}
		}
	}

	if !myWin.roiActive {
		if kind == "Gray" {
			copy(myWin.workingBuffer, fitsImage.Image.(*image.Gray).Pix)
		}
		if kind == "Gray16" {
			copy(myWin.workingBuffer, fitsImage.Image.(*image.Gray16).Pix)
		}
	}

	//fitsImage.FillMode = canvas.ImageFillOriginal
	fitsImage.FillMode = canvas.ImageFillContain
	return fitsImage, metaData, timestamp
}

func makeDisplayBuffer(width, height int) {
	myWin.displayBuffer = make([]byte, width*height*myWin.bytesPerPixel)
	myWin.workingBuffer = make([]byte, width*height*myWin.bytesPerPixel)
	// Diagnostic print ...
	//fmt.Printf("makeDisplayBuffer() made %d*%d*%d display buffer\n",
	//	width, height, myWin.bytesPerPixel)
}

func formatMetaData(primaryHDU fitsio.HDU) ([]string, string) {
	var cards []fitsio.Card
	var metaDataText []string
	var line string
	timestampFound := false

	for i := 0; i < len(primaryHDU.Header().Keys()); i += 1 {
		card := primaryHDU.(fitsio.Image).Header().Card(i)
		cards = append(cards, *card)
		if card.Comment == "" {
			line = fmt.Sprintf("%8s: %8v\n", card.Name, card.Value)
			metaDataText = append(metaDataText, line)
		} else {
			line = fmt.Sprintf("%8s: %8v (%s)\n", card.Name, card.Value, card.Comment)
			metaDataText = append(metaDataText, line)
		}
		if card.Name == "DATE-OBS" {
			timestampFound = true
			myWin.timestamp = fmt.Sprintf("%v", card.Value)
			myWin.timestamp = strings.Replace(myWin.timestamp, "T", " ", 1)
			myWin.timestampLabel.Text = myWin.timestamp
		}
	}

	if !timestampFound {
		myWin.timestamp = "<no timestamp found>"
		myWin.timestampLabel.Text = myWin.timestamp
	}

	return metaDataText, myWin.timestamp
}

func getFitsFilenames(folder string) []string {
	entries, err := os.ReadDir(folder)
	if err != nil {
		fmt.Println(fmt.Errorf("%w", err))
	}
	var fitsPaths []string
	for i := 0; i < len(entries); i += 1 {
		if !entries[i].IsDir() {
			name := entries[i].Name()
			if strings.HasSuffix(name, ".fits") {
				fitsPaths = append(fitsPaths, folder+"/"+name)
			}
		}
	}
	myWin.numFiles = len(fitsPaths)
	myWin.fileSlider.Max = float64(myWin.numFiles - 1)
	myWin.fileSlider.Min = 0.0
	return fitsPaths
}

func getStd(dataIn []byte, stride int, clip int) (float64, error) {
	var data []float64
	for i := 0; i < len(dataIn); i += stride {
		if int(dataIn[i]) < clip && int(dataIn[i]) > 0 {
			data = append(data, float64(dataIn[i]))
		}
	}
	return stats.StandardDeviation(data)
}

func applyContrastControls(original, stretched []byte, kind string) {
	// The slice stretched is modified. The slice original is untouched
	var floatVal float64
	var scale float64

	if len(original) > len(stretched) { // This should never happen - it's a coding error
		msg := fmt.Sprintf("input length: %d bytes  output length: %d bytes\n", len(original), len(stretched))
		dialog.ShowInformation("Oops - programming error", msg, myWin.parentWindow)
		return
	}

	bot := myWin.blackSlider.Value
	top := myWin.whiteSlider.Value

	var std float64
	var err error

	if myWin.autoContrastNeeded {
		myWin.autoContrastNeeded = false
		if kind == "Gray16" {
			std, err = getStd(original, 2, 255)
		} else {
			std, err = getStd(original, 1, 255)
		}
		if err != nil {
			fmt.Println(fmt.Errorf("getstd(): %w", err))
			return
		}
		//fmt.Printf("std: %0.1f\n", std)
		bot = -3 * std
		top = 5 * std
	}
	if bot < 0 {
		bot = 0
	}
	if top > 255 {
		top = 255
	}
	myWin.blackSlider.SetValue(bot)
	myWin.whiteSlider.SetValue(top)

	invert := bot > top
	if top > bot {
		scale = 255 / (top - bot)
	} else {
		scale = 255 / (bot - top)
		temp := bot
		bot = top
		top = temp
	}

	for i := 0; i < len(original); i++ {
		if float64(original[i]) <= bot {
			stretched[i] = 0
		} else if float64(original[i]) > top {
			stretched[i] = 255
		} else {
			floatVal = scale * (float64(original[i]) - bot)
			intVal := int(math.Round(floatVal))
			stretched[i] = byte(intVal)
		}
		if invert {
			stretched[i] = ^stretched[i]
		}
	}
	return
}

func showSplash() {
	//time.Sleep(500 * time.Millisecond)
	helpWin := myWin.App.NewWindow("Hello")
	helpWin.Resize(fyne.Size{Height: 450, Width: 700})
	scrollableText := container.NewVScroll(widget.NewRichTextWithText(helpText))
	helpWin.SetContent(scrollableText)
	helpWin.Show()
	helpWin.CenterOnScreen()
}
