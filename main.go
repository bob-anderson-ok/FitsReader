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
	"runtime"
	"runtime/debug"
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
	fitsFilePaths        []string
	fileLabel            *widget.Label
	fileIndex            int
	autoPlayEnabled      bool
	playBackMilliseconds int64
	currentFilePath      string
	playDelay            time.Duration
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

	row2 := container.NewGridWithRows(1)
	row2.Add(widget.NewLabel("Other stuff goes here..."))
	row2.Add(widget.NewLabel("... and here..."))

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
	checkForFrameRateSelected()
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

func checkForFrameRateSelected() {
	if myWin.playDelay == time.Duration(0) {
		dialog.ShowInformation(
			"Something to do ...",
			"Select a playback frame rate.",
			myWin.parentWindow,
		)
	}
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
	checkForFrameRateSelected()

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
		myWin.fileIndex = 0
		myWin.currentFilePath = myWin.fitsFilePaths[myWin.fileIndex]
		fmt.Printf("%d fits files were found.\n", len(myWin.fitsFilePaths))
	}
	if len(myWin.fitsFilePaths) > 0 {
		getFitsImage()
	}
}

func showMetaData() {
	fmt.Println("Time to show meta-data")
}

func getFitsImage() fyne.CanvasObject {
	defaultImagePath := "enhanced-image-0.fit"
	fitsFilePath := ""
	runtime.GC()

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

	if myWin.centerContent != nil {
		myWin.centerContent.Objects[0] = fitsImage
		myWin.centerContent.Refresh()
		//fitsNames := getFitsFilenames()
		//fmt.Println(fitsNames)
	}

	return fitsImage
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
