package main

import (
	_ "embed"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/astrogo/fitsio"
	"github.com/astrogo/fitsio/fltimg"
	"image"
	"image/color"
	"math"
	"os"
	"reflect"
	"strings"
	"time"
)

type Config struct {
	parentWindow         fyne.Window
	App                  fyne.App
	whiteSlider          *widget.Slider
	blackSlider          *widget.Slider
	fileSlider           *widget.Slider
	centerContent        *fyne.Container
	zeroPix              []byte
	fitsFilePaths        []string
	numFiles             int
	waitingForFileRead   bool
	fitsImages           []*canvas.Image
	imageKind            string
	fileLabel            *widget.Label
	timestampLabel       *widget.Label
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

const DefaultImageName = "FITS-player-default-image.fits"

const version = " 1.0.3"

//go:embed help.txt
var helpText string

//go:embed FITS-player-default-image.fits
var defaultImageFile []byte

var myWin Config

func main() {

	// Copy our embedded default image file to the current working directory
	var permissions os.FileMode
	permissions = 0666
	err := os.WriteFile(DefaultImageName, defaultImageFile, permissions)
	if err != nil {
		panic(err)
	}

	// We supply an ID (hopefully unique) because we need to use the preferences API
	myApp := app.NewWithID("com.gmail.ok.anderson.bob")
	myWin.App = myApp

	// We start app using the dark theme. There are buttons to allow theme change
	myApp.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantDark})

	myWin.autoPlayEnabled = false
	myWin.loopStartIndex = -1
	myWin.loopEndIndex = -1

	w := myApp.NewWindow("IOTA FITS video player" + version)
	w.Resize(fyne.Size{Height: 800, Width: 1200})

	myWin.parentWindow = w

	sliderWhite := widget.NewSlider(0, 255)
	sliderWhite.OnChanged = func(value float64) { getFitsImage() }
	sliderWhite.Orientation = 1
	sliderWhite.Value = 255
	myWin.whiteSlider = sliderWhite

	sliderBlack := widget.NewSlider(0, 255)
	sliderBlack.Orientation = 1
	sliderBlack.Value = 0
	sliderBlack.OnChanged = func(value float64) { getFitsImage() }
	myWin.blackSlider = sliderBlack

	rightItem := container.NewHBox(sliderBlack, sliderWhite)

	leftItem := container.NewVBox()
	leftItem.Add(widget.NewButton("Select fits folder", func() { chooseFitsFolder() }))
	leftItem.Add(widget.NewButton("Show meta-data", func() { showMetaData() }))
	selector := widget.NewSelect([]string{"1 fps", "5 fps", "10 fps", "25 fps", "30 fps", "max"},
		func(opt string) { setPlayDelay(opt) })
	selector.PlaceHolder = "Set play fps"
	leftItem.Add(selector)

	leftItem.Add(widget.NewButton("Dark theme", func() {
		myApp.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantDark})
	}))
	leftItem.Add(widget.NewButton("Light theme", func() {
		myApp.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantLight})
	}))

	//busyText := canvas.NewText("READING FILES", color.NRGBA{R: 255, A: 255})
	//myWin.busyLabel = busyText
	//myWin.busyLabel.Hidden = true
	//leftItem.Add(myWin.busyLabel)

	leftItem.Add(layout.NewSpacer())
	leftItem.Add(widget.NewButton("Set loop start", func() { setLoopStart() }))
	leftItem.Add(widget.NewButton("Set loop end", func() { setLoopEnd() }))
	leftItem.Add(widget.NewButton("Run loop", func() { go runLoop() }))

	row1 := container.NewGridWithRows(1)
	myWin.fileLabel = widget.NewLabel("File name goes here")
	myWin.fileLabel.Alignment = fyne.TextAlignCenter
	row1.Add(myWin.fileLabel)

	myWin.timestampLabel = widget.NewLabel("timestamp goes here")
	row2 := container.NewHBox(layout.NewSpacer(), myWin.timestampLabel, layout.NewSpacer())

	myWin.fileSlider = widget.NewSlider(0, 0) // Default max - will be set by getFitsFileNames()
	myWin.fileSlider.OnChanged = func(value float64) { processFileSliderMove(value) }

	toolBar := container.NewHBox()
	toolBar.Add(layout.NewSpacer())
	toolBar.Add(widget.NewButton("-1", func() { processBackOneFrame() }))
	toolBar.Add(widget.NewButton("<", func() { go playBackward(false) }))
	toolBar.Add(widget.NewButton("||", func() { pauseAutoPlay() }))
	toolBar.Add(widget.NewButton(">", func() { go playForward(false) }))
	toolBar.Add(widget.NewButton("+1", func() { processForwardOneFrame() }))
	toolBar.Add(layout.NewSpacer())

	bottomItem := container.NewVBox(myWin.fileSlider, toolBar, row1, row2)

	initialImage := getFitsImage() // Get the initial image

	centerContent := container.NewBorder(
		nil,
		bottomItem,
		leftItem,
		rightItem,
		initialImage)

	myWin.centerContent = centerContent
	w.SetContent(myWin.centerContent)
	w.CenterOnScreen()
	go showSplash()

	w.ShowAndRun()
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
		//fmt.Println("Loop will be run in reverse")
		playBackward(true)
	} else {
		//fmt.Println("Loop will run forward")
		playForward(true)
	}
}

func setLoopStart() {
	myWin.loopStartIndex = int(myWin.fileSlider.Value)
	//fmt.Printf("Loop start index: %d\n", myWin.loopStartIndex)
}

func setLoopEnd() {
	myWin.loopEndIndex = int(myWin.fileSlider.Value)
	//fmt.Printf("Loop end index: %d\n", myWin.loopEndIndex)
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
		// This flag will become true after file has been read and displayed by getFitsImage()
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
	//fmt.Printf("|%s| was selected\n", opt)
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
		// This flag will become true after file has been read and displayed by getFitsImage()
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
		//getFitsImage()
	}
	myWin.fileSlider.SetValue(float64(myWin.fileIndex)) // This causes a call to getFitsImage
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
		//getFitsImage()
	}
	myWin.fileSlider.SetValue(float64(myWin.fileIndex)) // This causes a call to getFitsImage()
	return
}

func processFileSliderMove(position float64) {
	myWin.fileIndex = int(position)
	myWin.fileLabel.SetText(myWin.fitsFilePaths[myWin.fileIndex])
	myWin.currentFilePath = myWin.fitsFilePaths[myWin.fileIndex]
	getFitsImage()
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
			fmt.Println(fmt.Errorf("ListerForURI(%s) failed: %w", lastFitsFolderStr, err))
			return
		} else {
			//fmt.Println("lastFitsFolder:", fitsDir.Path())
		}
		if fitsDir != nil {
			//fmt.Printf("\npath: %s  name: %s  scheme: %s, authority: %s\n\n",
			//	fitsDir.Path(), fitsDir.Name(), fitsDir.Scheme(), fitsDir.Authority())
		}
		showFolder.SetLocation(fitsDir)
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
		initializeImages()
		myWin.fileSlider.SetValue(0)
	}
	if len(myWin.fitsFilePaths) > 0 {
		getFitsImage()
	}
}

func showMetaData() {
	helpWin := myWin.App.NewWindow("FITS Meta-data")
	helpWin.Resize(fyne.Size{Height: 600, Width: 700})
	_, metaDataList, _ := getFitsImageFromFilePath(myWin.fitsFilePaths[myWin.fileIndex])
	//metaDataList := myWin.metaData[myWin.fileIndex]
	metaData := ""
	for _, line := range metaDataList {
		metaData += line + "\n"
	}
	scrollableText := container.NewVScroll(widget.NewRichTextWithText(metaData))
	helpWin.SetContent(scrollableText)
	helpWin.Show()
	helpWin.CenterOnScreen()
}

func getFitsImage() fyne.CanvasObject {
	fitsFilePath := ""

	// If no fits folder has been selected yet, use the default image
	if myWin.numFiles == 0 {
		myWin.numFiles = 1
		fitsFilePath = DefaultImageName
		myWin.fitsFilePaths = append(myWin.fitsFilePaths, fitsFilePath)
		initializeImages()
		myWin.fileIndex = 0
		myWin.fileSlider.SetValue(0)
	} else {
		fitsFilePath = myWin.currentFilePath
	}

	myWin.fileLabel.SetText(myWin.fitsFilePaths[myWin.fileIndex])

	imageToUse, _, timestamp := getFitsImageFromFilePath(myWin.fitsFilePaths[myWin.fileIndex])
	myWin.timestampLabel.SetText(timestamp)

	if myWin.whiteSlider != nil {
		if myWin.imageKind == "Gray32" {
			imageToUse.Image.(*fltimg.Gray32).Max = float32(myWin.whiteSlider.Value)
			imageToUse.Image.(*fltimg.Gray32).Min = float32(myWin.blackSlider.Value)
		} else if myWin.imageKind == "Gray" {
			if myWin.fileIndex == 0 { // Save the (currently unmodified) index 0 image pixels
				stretch(myWin.zeroPix, myWin.fitsImages[0].Image.(*image.Gray).Pix)
			} else {
				stretch(imageToUse.Image.(*image.Gray).Pix, myWin.fitsImages[0].Image.(*image.Gray).Pix)
			}
		} else if myWin.imageKind == "Gray16" {
			if myWin.fileIndex == 0 { // Use the saved image 0 pixels as source
				stretch(myWin.zeroPix, myWin.fitsImages[0].Image.(*image.Gray16).Pix)
			} else {
				stretch(imageToUse.Image.(*image.Gray16).Pix, myWin.fitsImages[0].Image.(*image.Gray16).Pix)
			}
		} else {
			fmt.Printf("The image kind (%s) is unrecognized.\n", myWin.imageKind)
		}
	}

	if myWin.centerContent != nil {
		if myWin.fileIndex == 0 { // Initialize the target where pixels will be displayed from.
			myWin.centerContent.Objects[0] = myWin.fitsImages[0]
		}
		myWin.centerContent.Refresh()
	}
	myWin.waitingForFileRead = false // Signal to anyone waiting for file read completion
	//fmt.Println(myWin.currentFilePath)

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
		panic(err3)
	}

	return fitsHandle
}

func initializeImages() {

	fitsImage, _, _ := getFitsImageFromFilePath(myWin.fitsFilePaths[0]) // side effect: myWin.primaryHDU is set

	myWin.fitsImages = append(myWin.fitsImages, fitsImage)

	getZeroPix(myWin.fitsFilePaths[0])

	myWin.fileSlider.SetValue(0)

}

func getZeroPix(pathToFrameZero string) {
	fitsImage, _, _ := getFitsImageFromFilePath(pathToFrameZero)
	// Save the pixels from the first image because we modify in place those pixels during image display
	if myWin.imageKind == "Gray16" {
		myWin.zeroPix = make([]byte, len(fitsImage.Image.(*image.Gray16).Pix))
		copy(myWin.zeroPix, fitsImage.Image.(*image.Gray16).Pix)
	}

	// Save the pixel from the first image because we modify in place those pixels during image display
	if myWin.imageKind == "Gray" {
		myWin.zeroPix = make([]byte, len(fitsImage.Image.(*image.Gray).Pix))
		copy(myWin.zeroPix, fitsImage.Image.(*image.Gray).Pix)
	}
}

func getFitsImageFromFilePath(filePath string) (*canvas.Image, []string, string) {
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
	//fitsImage.FillMode = canvas.ImageFillOriginal
	fitsImage.FillMode = canvas.ImageFillContain
	return fitsImage, metaData, timestamp
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
			myWin.timestampLabel.SetText(myWin.timestamp)
		}
	}

	if !timestampFound {
		myWin.timestamp = "<no timestamp found>"
		myWin.timestampLabel.SetText(myWin.timestamp)
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

func stretch(source []byte, old []byte) {
	var floatVal float64
	var scale float64

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
	for i := 0; i < len(source); i++ {
		if float64(source[i]) <= bot {
			old[i] = 0
		} else if float64(source[i]) > top {
			old[i] = 255
		} else {
			floatVal = scale * (float64(source[i]) - bot)
			intVal := int(math.Round(floatVal))
			old[i] = byte(intVal)
		}
		if invert {
			old[i] = ^old[i]
		}
	}
	return
}

func showSplash() {
	time.Sleep(500 * time.Millisecond)
	helpWin := myWin.App.NewWindow("Hello")
	helpWin.Resize(fyne.Size{Height: 400, Width: 700})
	scrollableText := container.NewVScroll(widget.NewRichTextWithText(helpText))
	helpWin.SetContent(scrollableText)
	helpWin.Show()
	helpWin.CenterOnScreen()
}
