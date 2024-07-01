package main

import (
	"FITSreader/fitsio"
	_ "embed"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"slices"

	//"github.com/astrogo/fitsio"
	_ "github.com/qdm12/reprint"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	cmdLineFolder              string
	reportCount                int
	buildLightcurve            bool
	leftGoalpostStats          *EdgeStats
	rightGoalpostStats         *EdgeStats
	adjustSliders              bool
	blackSet                   bool
	whiteSet                   bool
	lightcurve                 []float64
	lcIndices                  []int
	lightCurveStartIndex       int
	lightCurveEndIndex         int
	displayBuffer              []byte
	workingBuffer              []byte
	bytesPerPixel              int
	maxImg64                   float64
	minImg64                   float32
	maxImg32                   float64
	minImg32                   float32
	roiEntry                   *dialog.FormDialog
	widthStr                   binding.String
	heightStr                  binding.String
	roiWidth                   int
	roiHeight                  int
	roiActive                  bool
	roiChanged                 bool
	x0                         int // ROI corners
	y0                         int
	x1                         int
	y1                         int
	xJogSize                   int
	yJogSize                   int
	upButton                   *widget.Button
	downButton                 *widget.Button
	leftButton                 *widget.Button
	rightButton                *widget.Button
	centerButton               *widget.Button
	drawROIbutton              *widget.Button
	roiCheckbox                *widget.Check
	deletePathCheckbox         *widget.Check
	addFlashTimestampsCheckbox *widget.Check
	fileBrowserRequested       bool
	setRoiButton               *widget.Button
	parentWindow               fyne.Window
	folderSelectWin            fyne.Window
	showFolder                 *dialog.FileDialog
	folderSelect               *widget.Select
	selectionMade              bool
	folderSelected             string
	imageWidth                 int
	imageHeight                int
	App                        fyne.App
	whiteSlider                *widget.Slider
	blackSlider                *widget.Slider
	autoContrastNeeded         bool
	fileSlider                 *widget.Slider
	centerContent              *fyne.Container
	fitsFilePaths              []string
	fitsFolderHistory          []string
	numFiles                   int
	waitingForFileRead         bool
	fitsImages                 []*canvas.Image
	leftGoalpostTimestamp      string
	rightGoalpostTimestamp     string
	fileLabel                  *widget.Label
	timestampLabel             *canvas.Text
	fileIndex                  int
	autoPlayEnabled            bool
	playBackMilliseconds       int64
	currentFilePath            string
	playDelay                  time.Duration
	primaryHDU                 fitsio.HDU
	timestamps                 []string
	metaData                   [][]string
	timestamp                  string
	loopStartIndex             int
	loopEndIndex               int
	hist                       []int
}

const version = " 1.3.9"

const edgeTimesFileName = "FLASH_EDGE_TIMES.txt"

const processedByIotaUtilities = "GPS: IotaGFT and Iota FITS reader"

//go:embed help.txt
var helpText string

var myWin Config

func main() {

	// We supply an ID (hopefully unique) because we need to use the preferences API
	myApp := app.NewWithID("com.gmail.ok.anderson.bob")
	myWin.App = myApp

	myWin.fileBrowserRequested = false

	// We start app using the dark theme. There are buttons to allow theme change
	myApp.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantDark})

	myWin.widthStr = binding.NewString()
	myWin.heightStr = binding.NewString()

	myWin.fitsFolderHistory = myWin.App.Preferences().StringListWithFallback("folderHistory",
		[]string{})

	//if len(os.Args) > 1 {
	//	if os.Args[1] == "2" || os.Args[1] == "3" {
	//		myWin.fitsFolderHistory = []string{}
	//		saveFolderHistory()
	//	}
	//}

	if len(os.Args) > 1 {
		myWin.cmdLineFolder = os.Args[1]
		fmt.Println("User gave folder to process on command line as:", myWin.cmdLineFolder)
	} else {
		myWin.cmdLineFolder = ""
	}

	widthStr := myWin.App.Preferences().StringWithFallback("ROIwidth", "100")
	heightStr := myWin.App.Preferences().StringWithFallback("ROIheight", "100")

	_ = myWin.widthStr.Set(widthStr)   // Ignore possibility of error
	_ = myWin.heightStr.Set(heightStr) // Ignore possibility of error

	myWin.roiWidth, _ = strconv.Atoi(widthStr)   // Ignore error
	myWin.roiHeight, _ = strconv.Atoi(heightStr) // Ignore error

	myWin.x0, _ = strconv.Atoi(myWin.App.Preferences().StringWithFallback("ROIx0", "0"))
	myWin.x1, _ = strconv.Atoi(myWin.App.Preferences().StringWithFallback("ROIx1", "0"))
	myWin.y0, _ = strconv.Atoi(myWin.App.Preferences().StringWithFallback("ROIy0", "0"))
	myWin.y1, _ = strconv.Atoi(myWin.App.Preferences().StringWithFallback("ROIy1", "0"))

	if myWin.y0 == myWin.y1 {
		myWin.y1 += myWin.roiHeight
	}

	if myWin.x0 == myWin.x1 {
		myWin.x1 += myWin.roiWidth
	}

	initializeConfig(false)

	w := myApp.NewWindow("IOTA FITS Utility" + version)
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
	selector.SetSelectedIndex(5)            // "max" default (from above selection list)
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
	//if len(os.Args) > 1 {
	//	if os.Args[1] == "1" || os.Args[1] == "3" {
	//		myApp.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantLight})
	//	}
	//}

	leftItem.Add(layout.NewSpacer())

	//leftItem.Add(widget.NewButton("Build flash lightcurve", func() { buildFlashLightcurve() }))
	leftItem.Add(widget.NewButton("Show flash lightcurve", func() { showFlashLightcurve() }))
	myWin.addFlashTimestampsCheckbox = widget.NewCheck("enable auto-timestamp-insertion", addFlashTimestamps)
	myWin.addFlashTimestampsCheckbox.SetChecked(true)
	leftItem.Add(myWin.addFlashTimestampsCheckbox)
	//leftItem.Add(widget.NewButton("Timestamp FITS files", func() { addTimestampsToFitsFiles() }))

	leftItem.Add(layout.NewSpacer())
	myWin.roiCheckbox = widget.NewCheck("Apply ROI", applyRoi)
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

	myWin.fileSlider = widget.NewSlider(0, 0) // Default maxInSlice - will be set by getFitsFileNames()
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
	go delayedExecution()
	w.ShowAndRun() // This blocks. Place no other code after this call.
}

func delayedExecution() {
	time.Sleep(time.Second * 1)
	if myWin.cmdLineFolder != "" {
		dialog.ShowInformation("New FITS file available for flash timestamp insertion:",
			fmt.Sprintf(
				"\n\n%s  is available.\n\nIf auto-timestamp-insertion is checked,\nopening the file will"+
					" trigger the timestamp insertion process.\n\n", myWin.cmdLineFolder), myWin.parentWindow)

	}
	//dialog.ShowInformation("Startup message:", "\nWe're awake now.\n", myWin.parentWindow)
	if myWin.cmdLineFolder != "" {
		//readEdgeTimeFile(myWin.cmdLineFolder)
		//processFitsFolderPickedFromHistory(myWin.cmdLineFolder)
	}
}

func doXaxis(img, grayImage image.Image, xmax, y int) {
	var oldColor color.Color
	var newColor color.Gray
	for x := 0; x < xmax; x++ {
		oldColor = img.At(x, y) // This could be Gray|Gray16|Gray32|Gray64
		newColor = color.GrayModel.Convert(oldColor).(color.Gray)
		grayImage.(*image.Gray).Set(x, y, newColor)
	}
}

func convertImageToGray(img image.Image) (grayImage image.Image) {
	//start := time.Now()
	//var oldColor color.Color
	//var newColor color.Gray
	bounds := img.Bounds()
	grayImage = image.NewGray(bounds)
	var wg sync.WaitGroup

	for y := 0; y < bounds.Max.Y; y++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			doXaxis(img, grayImage, bounds.Max.X, y)
		}()
		//for x := 0; x < bounds.Max.X; x++ {
		//	oldColor = img.At(x, y) // This could be Gray|Gray16|Gray32|Gray64
		//	newColor = color.GrayModel.Convert(oldColor).(color.Gray)
		//	grayImage.(*image.Gray).Set(x, y, newColor)
		//}
	}
	wg.Wait()
	//elapsed := time.Since(start)
	//fmt.Printf("Execution time: %s\n", elapsed)
	return grayImage
}

//func saveImageToFile(img image.Image, filename string) error {
//	f, err := os.Create(filename)
//	if err != nil {
//		return err
//	}
//	defer f.Close()
//
//	if err := png.Encode(f, img); err != nil {
//		return err
//	}
//	return nil
//}

func buildFlashLightcurve() {
	if myWin.numFiles == 0 {
		return // There are no frames to process
	}
	if myWin.loopStartIndex >= 0 && myWin.loopEndIndex >= 0 {
		askIfLoopPointsAreToBeUsed()
		return
	} else {
		myWin.lightCurveStartIndex = 0
		myWin.lightCurveEndIndex = myWin.numFiles - 1
	}
	runLightcurveAcquisition()
	addTimestampsToFitsFiles()
	myWin.leftGoalpostTimestamp = ""
	myWin.rightGoalpostTimestamp = ""
}

func runLightcurveAcquisition() {
	//fmt.Printf("\n\nframes indexed from %d to %d inclusive will be used to build flash lightcurve\n",
	//	myWin.lightCurveStartIndex, myWin.lightCurveEndIndex)

	myWin.lightcurve = []float64{} // Clear the lightcurve slice
	myWin.lcIndices = []int{}      // and the corresponding indices

	// This records the first frame as a side effect, but only if the slider changes value, so we force that.
	myWin.fileSlider.SetValue(float64(myWin.lightCurveEndIndex))

	// During the "play forward", a lightcurve will be calculated whenever the following flag is true
	myWin.buildLightcurve = true
	myWin.fileSlider.SetValue(float64(myWin.lightCurveStartIndex))

	// Normally, we invoke playForward as a go routine (go playForward) so that the pause button can be used.
	// Here we don't do this so that the generation of the lightcurve, once started, cannot be paused.
	playLightcurveForward()
	myWin.buildLightcurve = false

	//showFlashLightcurve()
	findFlashEdges()

	//fmt.Println("\nEnd of build lightcurve")
}

func alreadyHasIotaTimestamps(processedStr string) bool {
	f, err := os.OpenFile(myWin.fitsFilePaths[0], os.O_RDONLY, 0644)
	if err != nil {
		log.Fatalf("could not open file: %+v", err)
	}

	fits, err := fitsio.Open(f)
	if err != nil {
		log.Fatalf("\nCould not open FITS file: %+v\n", err)
	}

	hdu := fits.HDU(0)
	dateObsCard := hdu.Header().Get("DATE-OBS")
	f.Close()
	if dateObsCard == nil {
		return false
	}

	return dateObsCard.Comment == processedStr
}

func addFlashTimestamps(_ bool) {

}

func addTimestampsToFitsFiles() {
	//msg := fmt.Sprintf("Add timestamps to fits files entered.")
	//dialog.ShowInformation("Add timestamps report:", msg, myWin.parentWindow)
	if myWin.leftGoalpostTimestamp == "" || myWin.rightGoalpostTimestamp == "" {
		msg := fmt.Sprintf("There are no flash goalpost timestamps available.")
		dialog.ShowInformation("Add timestamps report:", msg, myWin.parentWindow)
	}
	fmt.Printf("\n left goalpost edge at %0.6f\n", myWin.leftGoalpostStats.edgeAt)
	fmt.Printf("right goalpost edge at %0.6f\n", myWin.rightGoalpostStats.edgeAt)

	leftFlashTime, err := time.Parse(time.RFC3339, myWin.leftGoalpostTimestamp)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(" left goalpost occurred @", leftFlashTime)
	}

	rightFlashTime, err := time.Parse(time.RFC3339, myWin.rightGoalpostTimestamp)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("right goalpost occurred @", rightFlashTime)
	}

	deltaFlashTime := rightFlashTime.Sub(leftFlashTime)
	//fmt.Println(deltaFlashTime.Nanoseconds())
	frameTime := float64(deltaFlashTime.Nanoseconds()) / 1_000_000_000 / (myWin.rightGoalpostStats.edgeAt - myWin.leftGoalpostStats.edgeAt)
	fmt.Printf("frame time: %0.6f\n", frameTime)

	pwmUncertainty := 0.000032 / 2 // This is correct only for the IOTA-GFT running the pwm at 31.36 kHz
	myWin.leftGoalpostStats.edgeSigma *= frameTime
	myWin.leftGoalpostStats.edgeSigma += pwmUncertainty
	myWin.rightGoalpostStats.edgeSigma *= frameTime
	myWin.rightGoalpostStats.edgeSigma += pwmUncertainty
	fmt.Printf(" left edge time uncertainty: %0.6f\n", myWin.leftGoalpostStats.edgeSigma)
	fmt.Printf("right edge time uncertainty: %0.6f\n", myWin.rightGoalpostStats.edgeSigma)
	t0Delta := time.Duration(myWin.leftGoalpostStats.edgeAt * frameTime * 1_000_000_000)
	t0 := leftFlashTime.Add(-t0Delta)
	myWin.timestamps = make([]string, 0)
	for i := range myWin.lightcurve {
		tn := t0.Add(time.Duration(float64(i) * frameTime * 1_000_000_000))
		tsStr := tn.Format("2006-01-02T15:04:05.000000")
		myWin.timestamps = append(myWin.timestamps, tsStr)
		//fmt.Println(i, tsStr)
	}

	i := 0
	for _, frameFile := range myWin.fitsFilePaths {
		f, err := os.OpenFile(frameFile, os.O_RDWR, 0644)
		if err != nil {
			log.Fatalf("could not open file: %+v", err)
		}

		outFile, err := fitsio.Create(f)
		if err != nil {
			fmt.Println(err)
		}

		fits, err := fitsio.Open(f)

		if err != nil {
			fmt.Printf("\nCould not open FITS file: %+v\n", err)
			return
		}

		hdu := fits.HDU(0)
		//var dateObsCard []fitsio.Card
		dateObsCard := hdu.Header().Get("DATE-OBS")
		cardList := hdu.(*fitsio.PrimaryHDU).Hdr.Cards

		if dateObsCard == nil {
			// Make a DATE-OBS card. Put it in a slice so that we can use slice.Concat()
			dateObsCard = new(fitsio.Card)
			dateObsCard.Name = "DATE-OBS"
			dateObsCard.Value = myWin.timestamps[i]
			dateObsCard.Comment = processedByIotaUtilities
			dateObsCardSlice := make([]fitsio.Card, 1)
			dateObsCardSlice[0] = *dateObsCard

			// We will form a complete new card list from the old one by inserting the new DATE-OBS
			// card immediately before the first COMMENT card, or the END card, whichever comes first.
			var newCardList []fitsio.Card
			for i, card := range cardList {
				if card.Name == "COMMENT" || card.Name == "END" {
					newCardList = cardList[0:i]
					newCardList = slices.Concat(newCardList, dateObsCardSlice)
					newCardList = slices.Concat(newCardList, cardList[i:])
					break
				}
			}

			// Replace the old cards with the new set, now augmented with a DATE-OBS card.
			hdu.(*fitsio.PrimaryHDU).Hdr.Cards = newCardList
		} else {
			hdu.Header().Set("DATE-OBS", myWin.timestamps[i], "GPS: IotaGFT and Iota FITS reader")
		}

		// It is essential to reset the 'write point' to the beginning of the file,
		// otherwise the outFile.Write(hdu) will simply append to the file (and be invisible to fits readers)
		_, err = f.Seek(0, 0)
		if err != nil {
			fmt.Println(err)
		}

		err = outFile.Write(hdu)
		if err != nil {
			fmt.Println(err)
		}

		err = outFile.Close()
		if err != nil {
			fmt.Println(err)
		}
		i += 1

		f.Close()

		_ = fits.Close()
	}
	msg := fmt.Sprintf("\nAll timestamps have been added to the file.\n\n"+
		"left edge time uncertainty estimate: %0.6f seconds\n\n"+
		"right edge uncertainty estimate: %0.6f seconds\n\n",
		myWin.leftGoalpostStats.edgeSigma, myWin.rightGoalpostStats.edgeSigma)
	dialog.ShowInformation("Add timestamps report:", msg, myWin.parentWindow)
}

func initializeConfig(running bool) {

	myWin.buildLightcurve = false
	myWin.autoPlayEnabled = false
	myWin.loopStartIndex = -1
	myWin.loopEndIndex = -1
	myWin.roiActive = false
	myWin.roiChanged = false

	if running { // Must be running to have myWin.roiCheckbox built
		myWin.roiCheckbox.SetChecked(false)
	}

	myWin.lightcurve = make([]float64, 0)
	myWin.displayBuffer = nil
	myWin.workingBuffer = nil
	myWin.bytesPerPixel = 0
	myWin.xJogSize = 20
	myWin.yJogSize = 20
	myWin.fitsFilePaths = nil
	myWin.waitingForFileRead = false
	myWin.selectionMade = false
	myWin.numFiles = 0
	myWin.fileIndex = 0
	myWin.autoPlayEnabled = false
	myWin.currentFilePath = ""
	myWin.timestamps = nil
}

type forcedVariant struct {
	fyne.Theme

	variant fyne.ThemeVariant
}

func (f *forcedVariant) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return f.Theme.Color(name, f.variant)
}

func processFileSliderMove(position float64) {
	myWin.fileIndex = int(position)
	myWin.fileLabel.SetText(myWin.fitsFilePaths[myWin.fileIndex])
	myWin.currentFilePath = myWin.fitsFilePaths[myWin.fileIndex]
	displayFitsImage()
}

func showMetaData() {
	helpWin := myWin.App.NewWindow("FITS Meta-data")
	helpWin.Resize(fyne.Size{Height: 600, Width: 700})
	_, metaDataList, _ := getFitsImageFromFilePath(myWin.fitsFilePaths[myWin.fileIndex])

	if metaDataList == nil {
		return
	}

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

	// A side effect of the next call is that myWin.displayBuffer is filled.
	imageToUse, _, timestamp := getFitsImageFromFilePath(myWin.fitsFilePaths[myWin.fileIndex])

	if imageToUse == nil {
		return nil
	}

	myWin.timestampLabel.Text = timestamp

	if myWin.whiteSlider != nil {
		if myWin.adjustSliders {
			if !myWin.whiteSet {
				setSlider(myWin.hist, 75, "white")
				myWin.whiteSet = true
			}
			if !myWin.blackSet {
				setSlider(myWin.hist, 75, "black")
				myWin.blackSet = true
				myWin.adjustSliders = false
			}
		}
		// set displayBuffer from imageToUse stretched according to contrast sliders
		applyContrastControls(imageToUse.Image.(*image.Gray).Pix, myWin.displayBuffer)
	}

	myWin.fitsImages[0].Image.(*image.Gray).Pix = myWin.displayBuffer

	if !myWin.roiActive {
		restoreRect()
	} else {
		setROIrect()
	}

	myWin.centerContent.Objects[0] = myWin.fitsImages[0]

	myWin.centerContent.Refresh()
	myWin.waitingForFileRead = false // Signal to anyone waiting for file read completion

	return imageToUse
}

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
		fmt.Println(err3)
	}

	return fitsHandle
}

func initializeImages() {

	// side effect: myWin.primaryHDU is set
	fitsImage, _, _ := getFitsImageFromFilePath(myWin.fitsFilePaths[0])

	if fitsImage == nil {
		return
	}

	myWin.adjustSliders = true
	myWin.blackSet = false
	myWin.whiteSet = false

	myWin.imageWidth = fitsImage.Image.Bounds().Max.X
	myWin.imageHeight = fitsImage.Image.Bounds().Max.Y

	myWin.fitsImages = append(myWin.fitsImages, fitsImage)

	goImage := myWin.primaryHDU.(fitsio.Image).Image()
	goImage = convertImageToGray(goImage)

	myWin.bytesPerPixel = 1

	makeDisplayBuffer(myWin.imageWidth, myWin.imageHeight)

	myWin.fileSlider.SetValue(0)
	myWin.lightcurve = make([]float64, 0)
}

func histogram(sample []byte, stride, cornerRow, cornerCol, size int) (hist []int) {
	hist = make([]int, 256)
	for row := cornerRow; row < cornerRow+size; row++ {
		for col := cornerCol; col < cornerCol+size; col++ {
			k := row*stride + col
			hist[sample[k]] += 1
		}
	}
	return hist
}

func reportROIsettings() {
	myWin.reportCount += 1
	fmt.Printf("roi report number: %d\n", myWin.reportCount)
	fmt.Printf("roiWidth: %d   roiHeight: %d\n", myWin.roiWidth, myWin.roiHeight)
	fmt.Printf("x0: %d  y0: %d  x1: %d  y1: %d\n\n", myWin.x0, myWin.y0, myWin.x1, myWin.y1)
}
func setSlider(hist []int, targetPercent int, sliderToSet string) {
	// Compute the pixel count (requiredPixelSum) we want the standard deviation bars to enclose
	totalPixelCount := 0
	for i := 0; i < len(hist); i++ {
		totalPixelCount += hist[i]
	}
	requiredPixelSum := totalPixelCount * targetPercent / 100

	// Find the index of the histogram peak value
	indexOfPeak := 0
	peakValue := 0
	for i := 0; i < len(hist); i++ {
		if hist[i] > peakValue {
			peakValue = hist[i]
			indexOfPeak = i
		}
	}
	//fmt.Printf("Peak value is %d at %d Need %d\n", peakValue, indexOfPeak, requiredPixelSum)

	// standard deviation calculation - sum of hist around peak must exceed requiredPixelSum
	stdLeft := indexOfPeak
	stdRight := indexOfPeak
	stdSum := peakValue
	keepGoing := true
	for keepGoing {
		if stdLeft > 0 {
			stdLeft -= 1
			stdSum += hist[stdLeft]
		}
		if stdRight < len(hist)-1 {
			stdRight += 1
			stdSum += hist[stdRight]
		}
		keepGoing = stdSum <= requiredPixelSum
	}
	//fmt.Printf("stdLeft: %d  stdRight: %d\n", stdLeft, stdRight)

	// Calculate slider position
	stdWidth := stdRight - stdLeft
	if stdWidth < 2 {
		stdWidth = 2
	}
	blackLevel := indexOfPeak - stdWidth
	if blackLevel < 0 {
		blackLevel = 0
	}
	whiteLevel := indexOfPeak + 6*stdWidth
	if whiteLevel > 255 {
		whiteLevel = 255
	}
	if sliderToSet == "black" {
		myWin.blackSlider.SetValue(float64(blackLevel))
		//fmt.Printf("Set black slider to %d\n", blackLevel)
	}
	if sliderToSet == "white" {
		myWin.whiteSlider.SetValue(float64(whiteLevel))
		//fmt.Printf("Set white slider to %d\n", whiteLevel)
	}
	//fmt.Printf("blackLevel: %d  whiteLevel: %d\n", blackLevel, whiteLevel)
}

func getFitsImageFromFilePath(filePath string) (*canvas.Image, []string, string) {
	// An important side effect of this function: it fills the myWin.displayBuffer []byte

	f := openFitsFile(filePath)
	myWin.primaryHDU = f.HDU(0)
	metaData, timestamp := formatMetaData(myWin.primaryHDU)

	closeErr := f.Close()
	if closeErr != nil {
		errMsg := fmt.Errorf("could not close %s: %w", filePath, closeErr)
		fmt.Printf(errMsg.Error())
	}

	goImage := myWin.primaryHDU.(fitsio.Image).Image()
	goImage = convertImageToGray(goImage)

	if myWin.adjustSliders {
		// Calculate coordinates of sampling aperture for histogram
		imageWidth := myWin.imageWidth
		imageHeight := myWin.imageHeight
		centerRow := imageHeight / 2
		centerCol := imageWidth / 2
		halfSize := 50
		cornerRow := centerRow - halfSize
		cornerCol := centerCol - halfSize

		myWin.hist = histogram(goImage.(*image.Gray).Pix, goImage.(*image.Gray).Stride, cornerRow, cornerCol, halfSize*2)
	}

	if goImage == nil {
		dialog.ShowInformation("Oops", "No images are present in the .fits file", myWin.parentWindow)
		return nil, []string{}, ""
	}

	fitsImage := canvas.NewImageFromImage(goImage) // This is a Fyne image

	if myWin.buildLightcurve {
		myWin.lightcurve = append(myWin.lightcurve, pixelSum())
		myWin.lcIndices = append(myWin.lcIndices, myWin.fileIndex)
		//fmt.Printf("fileIndex: %d\n", myWin.fileIndex)
	}

	validateROIsize(goImage)

	if myWin.roiActive {

		roi := goImage.(*image.Gray).SubImage(image.Rect(myWin.x0, myWin.y0, myWin.x1, myWin.y1))

		fitsImage = canvas.NewImageFromImage(roi)                                  // This is a Fyne image
		myWin.workingBuffer = make([]byte, len(fitsImage.Image.(*image.Gray).Pix)) // workingBuffer <- fitsImage
		copy(myWin.workingBuffer, fitsImage.Image.(*image.Gray).Pix)               // workingBuffer <- fitsImage
		copy(myWin.displayBuffer, fitsImage.Image.(*image.Gray).Pix)               // displayBuffer <- fitsImage

		if myWin.roiChanged {
			myWin.roiChanged = false
			myWin.fitsImages[0] = fitsImage
			myWin.centerContent.Objects[0] = fitsImage
		}
	}

	if !myWin.roiActive {
		copy(myWin.workingBuffer, fitsImage.Image.(*image.Gray).Pix)
	}

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

func applyContrastControls(original, stretched []byte) {
	// stretched is modified.    original is untouched.
	var floatVal float64
	var scale float64

	if len(original) > len(stretched) { // This should never happen - it's a coding error
		msg := fmt.Sprintf("input length: %d bytes  output length: %d bytes\n", len(original), len(stretched))
		dialog.ShowInformation("Oops - programming error", msg, myWin.parentWindow)
		return
	}

	bot := myWin.blackSlider.Value
	top := myWin.whiteSlider.Value

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
