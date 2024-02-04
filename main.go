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
	"fyne.io/fyne/v2/widget"
	"github.com/astrogo/fitsio"
	"github.com/astrogo/fitsio/fltimg"
	"image"
	"math"
	"os"
	"reflect"
	"runtime/debug"
	"strings"
	"time"
)

type Config struct {
	parentWindow   fyne.Window
	App            fyne.App
	whiteSlider    *widget.Slider
	blackSlider    *widget.Slider
	fileSlider     *widget.Slider
	centerContent  *fyne.Container
	fitsFilePaths  []string
	fileLabel      *widget.Label
	timestampLabel *widget.Label
	//frameCountLabel      *widget.Label
	fileIndex            int
	autoPlayEnabled      bool
	playBackMilliseconds int64
	currentFilePath      string
	playDelay            time.Duration
	primaryHDU           fitsio.HDU
	timestamp            string
}

//go:embed help.txt
var helpText string

var myWin Config

func main() {
	// Increasing the frequency of garbage collection reduces the 'flicker' that sometimes occurs
	// as the contrast sliders are moved.
	debug.SetGCPercent(10)

	myApp := app.New()
	myWin.App = myApp

	myWin.autoPlayEnabled = false

	w := myApp.NewWindow("IOTA FITS video viewer")
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

	row1 := container.NewGridWithRows(1)
	myWin.fileLabel = widget.NewLabel("File name goes here")
	myWin.fileLabel.Alignment = fyne.TextAlignCenter
	row1.Add(myWin.fileLabel)

	myWin.timestampLabel = widget.NewLabel("timestamp goes here")
	//myWin.frameCountLabel = widget.NewLabel("Frame count goes here")
	row2 := container.NewHBox(layout.NewSpacer(), myWin.timestampLabel, layout.NewSpacer())
	//row2.Add(myWin.frameCountLabel)
	//row2.Add(myWin.timestampLabel)

	myWin.fileSlider = widget.NewSlider(0, 1000)
	myWin.fileSlider.OnChanged = func(value float64) { processFileSliderMove(value) }

	toolBar := container.NewHBox()
	toolBar.Add(layout.NewSpacer())
	toolBar.Add(widget.NewButton("-1", func() { processBackOneFrame() }))
	toolBar.Add(widget.NewButton("<", func() { go playBackward() }))
	toolBar.Add(widget.NewButton("||", func() { pauseAutoPlay() }))
	toolBar.Add(widget.NewButton(">", func() { go playForward() }))
	toolBar.Add(widget.NewButton("+1", func() { processForwardOneFrame() }))
	toolBar.Add(layout.NewSpacer())

	bottomItem := container.NewVBox(myWin.fileSlider, toolBar, row1, row2)

	fitsImage := getFitsImage()

	centerContent := container.NewBorder(
		nil,
		bottomItem,
		leftItem,
		rightItem,
		fitsImage)
	myWin.centerContent = centerContent
	w.SetContent(myWin.centerContent)
	w.CenterOnScreen()
	go showSplash()

	w.ShowAndRun()
}

func pauseAutoPlay() {
	myWin.autoPlayEnabled = false
}

func playForward() {
	if !checkForFrameRateSelected() {
		return
	}

	if myWin.autoPlayEnabled {
		return // autoPlay is already running
	}
	myWin.autoPlayEnabled = true // This can be set to false by clicking the pause button
	for {
		if !myWin.autoPlayEnabled {
			return
		}
		if myWin.fileIndex == len(myWin.fitsFilePaths)-1 {
			myWin.autoPlayEnabled = false
			return
		}
		processForwardOneFrame()
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

func playBackward() {
	if !checkForFrameRateSelected() {
		return
	}

	if myWin.autoPlayEnabled {
		return // autoPlay is already running
	}
	myWin.autoPlayEnabled = true // This can be set to false by clicking the pause button
	for {
		if !myWin.autoPlayEnabled {
			return
		}
		if myWin.fileIndex == 0 {
			myWin.autoPlayEnabled = false
			return
		}
		processBackOneFrame()
		time.Sleep(myWin.playDelay)
	}
}

func processBackOneFrame() {
	numFrames := len(myWin.fitsFilePaths)
	if numFrames == 0 {
		return
	}
	myWin.fileIndex -= 1
	if myWin.fileIndex < 0 {
		myWin.fileIndex += 1
		return
	} else {
		myWin.currentFilePath = myWin.fitsFilePaths[myWin.fileIndex]
		slideValue := float64(myWin.fileIndex) / float64(numFrames) * 1000.0
		myWin.fileSlider.SetValue(slideValue)
		getFitsImage()
	}
}

func processForwardOneFrame() {
	numFrames := len(myWin.fitsFilePaths)
	if numFrames == 0 {
		return
	}
	myWin.fileIndex += 1
	if myWin.fileIndex >= numFrames {
		myWin.fileIndex = numFrames - 1
		return
	} else {
		myWin.currentFilePath = myWin.fitsFilePaths[myWin.fileIndex]
		slideValue := float64(myWin.fileIndex) / float64(numFrames) * 1000.0
		myWin.fileSlider.SetValue(slideValue)
		getFitsImage()
	}
}

func processFileSliderMove(position float64) {
	numPaths := len(myWin.fitsFilePaths)
	if numPaths > 0 {
		// Compute the entry number corresponding to the slider position
		entryFloat := float64(numPaths) * position / 1000.0
		entryInt := int(math.Round(entryFloat))
		if entryInt >= numPaths {
			entryInt = numPaths - 1
		}
		pathRequested := myWin.fitsFilePaths[entryInt]
		myWin.currentFilePath = pathRequested
		myWin.fileIndex = entryInt
		getFitsImage()
	}
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
	showFolder.Show()
}

func processFitsFolderSelection(path fyne.ListableURI, err error) {
	if err != nil {
		fmt.Println(fmt.Errorf("%w\n", err))
		return
	}
	if path != nil {
		//fmt.Println(path.Path())
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
		fmt.Printf("%d fits files were found.\n", len(myWin.fitsFilePaths))
	}
	if len(myWin.fitsFilePaths) > 0 {
		getFitsImage()
	}
}

func showMetaData() {
	helpWin := myWin.App.NewWindow("FITS Meta-data")
	helpWin.Resize(fyne.Size{Height: 600, Width: 700})
	metaDataList := formatMetaData(myWin.primaryHDU)
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
	defaultImagePath := "enhanced-image-0.fit"
	fitsFilePath := ""

	// None of the following hacks reduced the 'flashing' during playback.
	//runtime.GC()
	//debug.SetGCPercent(-1)
	//defer debug.SetGCPercent(10)

	// If no fits folder has been selected yet, use the default image
	if len(myWin.fitsFilePaths) == 0 {
		fitsFilePath = defaultImagePath
	} else {
		fitsFilePath = myWin.currentFilePath
	}

	myWin.fileLabel.SetText(fitsFilePath)

	r, err1 := os.Open(fitsFilePath)
	if err1 != nil {
		panic(err1)
	}

	defer func(r *os.File) {
		err2 := r.Close()
		if err2 != nil {
			errMsg := fmt.Errorf("err2: %w", err2)
			fmt.Printf(errMsg.Error())
		}
	}(r)

	f, err3 := fitsio.Open(r)
	if err3 != nil {
		panic(err3)
	}

	defer func(f *fitsio.File) {
		err4 := f.Close()
		if err4 != nil {
			errMsg := fmt.Errorf("err4: %w", err4)
			fmt.Printf(errMsg.Error())
		}
	}(f)

	primaryHDU := f.HDU(0)
	myWin.primaryHDU = primaryHDU

	fyneImage := primaryHDU.(fitsio.Image).Image()
	kind := reflect.TypeOf(fyneImage).Elem().Name()
	if myWin.whiteSlider != nil {
		if kind == "Gray32" {
			fyneImage.(*fltimg.Gray32).Max = float32(myWin.whiteSlider.Value)
			fyneImage.(*fltimg.Gray32).Min = float32(myWin.blackSlider.Value)
		} else if kind == "Gray" {
			stretch(fyneImage.(*image.Gray).Pix, myWin.blackSlider.Value, myWin.whiteSlider.Value)
		} else {
			fmt.Printf("The image kind (%s) is unrecognized.\n", kind)
		}
	}
	fitsImage := canvas.NewImageFromImage(fyneImage)
	fitsImage.FillMode = canvas.ImageFillOriginal

	//size := len(primaryHDU.(fitsio.Image).Raw())
	//fmt.Printf("%d bytes in the image\n", size)
	//fmt.Printf("%d\n", primaryHDU.(fitsio.Image).Raw()[0])
	//fmt.Printf("HDU name: %s\n", primaryHDU.Name())
	//fmt.Printf("shape: %d x %d\n", primaryHDU.Header().Axes()[0], primaryHDU.Header().Axes()[1])
	//fmt.Println(primaryHDU.Header().Keys())
	//

	//fmt.Println(formatMetaData(primaryHDU))
	formatMetaData(primaryHDU) // We do this for the side effect of setting the timestamp

	if myWin.centerContent != nil {
		myWin.centerContent.Objects[0] = fitsImage
		//myWin.centerContent.Refresh()
	}

	return fitsImage
}

func formatMetaData(primaryHDU fitsio.HDU) []string {
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

	return metaDataText
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
	return fitsPaths
}

func stretch(old []byte, bot float64, top float64) {
	var floatVal float64
	var scale float64

	invert := bot > top
	if top > bot {
		scale = 255 / (top - bot)
	} else {
		scale = 255 / (bot - top)
		temp := bot
		bot = top
		top = temp
	}
	for i := 0; i < len(old); i++ {
		if float64(old[i]) <= bot {
			old[i] = 0
		} else if float64(old[i]) > top {
			old[i] = 255
		} else {
			floatVal = scale * (float64(old[i]) - bot)
			intVal := int(math.Round(floatVal))
			old[i] = byte(intVal)
		}
		if invert {
			old[i] = ^old[i]
		}
	}
}

func showSplash() {
	time.Sleep(1 * time.Second)
	helpWin := myWin.App.NewWindow("Hello user ...")
	helpWin.Resize(fyne.Size{Height: 400, Width: 700})
	scrollableText := container.NewVScroll(widget.NewRichTextWithText(helpText))
	helpWin.SetContent(scrollableText)
	helpWin.Show()
	helpWin.CenterOnScreen()
}
