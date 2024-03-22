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
	dKeyTyped            bool
	displayBuffer        []byte
	workingBuffer        []byte
	bytesPerPixel        int
	maxImg64             float64
	minImg64             float32
	maxImg32             float64
	minImg32             float32
	roiEntry             *dialog.FormDialog
	widthStr             binding.String
	heightStr            binding.String
	roiWidth             int
	roiHeight            int
	roiActive            bool
	roiChanged           bool
	x0                   int // ROI corners
	y0                   int
	x1                   int
	y1                   int
	xJogSize             int
	yJogSize             int
	upButton             *widget.Button
	downButton           *widget.Button
	leftButton           *widget.Button
	rightButton          *widget.Button
	centerButton         *widget.Button
	drawROIbutton        *widget.Button
	roiCheckbox          *widget.Check
	deletePathCheckbox   *widget.Check
	fileBrowserRequested bool
	setRoiButton         *widget.Button
	parentWindow         fyne.Window
	folderSelectWin      fyne.Window
	showFolder           *dialog.FileDialog
	folderSelect         *widget.Select
	selectionMade        bool
	folderSelected       string
	imageWidth           int
	imageHeight          int
	App                  fyne.App
	whiteSlider          *widget.Slider
	blackSlider          *widget.Slider
	autoContrastNeeded   bool
	fileSlider           *widget.Slider
	centerContent        *fyne.Container
	fitsFilePaths        []string
	fitsFolderHistory    []string
	numFiles             int
	waitingForFileRead   bool
	fitsImages           []*canvas.Image
	imageKind            string
	fileLabel            *widget.Label
	timestampLabel       *canvas.Text
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

const version = " 1.3.3"

//const browseText = "Open browser to select new folder"

// const clearText = "Clear all history (if you're sure!)"
//const clearText = "To remove a path, type D before opening this selection window, then click path to remove."

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

	if len(os.Args) > 1 {
		if os.Args[1] == "2" || os.Args[1] == "3" {
			myWin.fitsFolderHistory = []string{}
			saveFolderHistory()
		}
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

	w := myApp.NewWindow("IOTA FITS video player" + version)
	w.Resize(fyne.Size{Height: 800, Width: 1200})

	//w.Canvas().SetOnTypedKey(func(ke *fyne.KeyEvent) { recordKeyEvent(ke) })

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
		if os.Args[1] == "1" || os.Args[1] == "3" {
			myApp.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantLight})
		}
	}

	leftItem.Add(layout.NewSpacer())

	leftItem.Add(widget.NewButton("Build flash lightcurve", func() { buildFlashLightcurve() }))
	leftItem.Add(widget.NewButton("Timestamp FITS files", func() { addTimestampsToFitsFiles() }))

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

//func recordKeyEvent(ke *fyne.KeyEvent) {
//	//fmt.Println(ke.Name)
//	if ke.Name == fyne.KeyD {
//		// TODO remove print stmts
//		fmt.Println("Got a D typed")
//		// myWin.dKeyTyped = true
//	}
//}

func buildFlashLightcurve() {
	fmt.Println("Build flash lightcurve requested")
}

func addTimestampsToFitsFiles() {
	fmt.Println("Add timestamps to fits files")
}

func initializeConfig(running bool) {
	myWin.autoPlayEnabled = false
	myWin.loopStartIndex = -1
	myWin.loopEndIndex = -1
	myWin.roiActive = false
	myWin.roiChanged = false

	if running { // Must be running to have myWin.roiCheckbox built
		myWin.roiCheckbox.SetChecked(false)
	}

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

func showROI() {
	validateROIparameters()
	if myWin.roiActive {
		myWin.roiCheckbox.SetChecked(false)
		myWin.roiActive = false
		displayFitsImage()
	}

	x0 := myWin.x0
	x1 := myWin.x1
	y0 := myWin.y0
	y1 := myWin.y1

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

	if myWin.imageKind == "Gray32" {
		source := myWin.fitsImages[0].Image.(*fltimg.Gray32)

		//colorBytes := convUint32ToBytes([]byte{}, uint32(source.Min))
		colorBytes := []byte{255, 255, 255, 255}

		for i := x0; i < x1; i++ {
			//myWin.fitsImages[0].Image.(*fltimg.Gray32).Set(i, y0, color.White)

			j := pixOffset(i, y0, source.Rect, source.Stride, 4)
			for k := 0; k < 4; k++ {
				source.Pix[j+k] = colorBytes[k]
			}
		}
		for i := x0; i < x1+1; i++ {
			j := pixOffset(i, y1, source.Rect, source.Stride, 4)
			for k := 0; k < 4; k++ {
				source.Pix[j+k] = colorBytes[k]
			}
		}
		for i := y0; i < y1; i++ {
			j := pixOffset(x0, i, source.Rect, source.Stride, 4)
			for k := 0; k < 4; k++ {
				source.Pix[j+k] = colorBytes[k]
			}
		}
		for i := y0; i < y1; i++ {
			j := pixOffset(x1, i, source.Rect, source.Stride, 4)
			for k := 0; k < 4; k++ {
				source.Pix[j+k] = colorBytes[k]
			}
		}
	}

	if myWin.imageKind == "Gray64" {
		source := myWin.fitsImages[0].Image.(*fltimg.Gray64)
		colorBytes := []byte{255, 255, 255, 255, 255, 255, 255, 255}
		//colorBytes := []byte{63, 240, 0, 0, 0, 0, 0, 0}
		//colorBytes := []byte{0, 0, 0, 0, 0, 0, 0, 0}
		for i := x0; i < x1; i++ {
			//bobsColor := color.RGBA64{R: 65_535, G: 65_535, B: 65_535, A: 65_535}
			//source.Set(i, y0, bobsColor)

			j := pixOffset(i, y0, source.Rect, source.Stride, 8)
			for k := 0; k < 8; k++ {
				source.Pix[j+k] = colorBytes[k]
			}
		}
		for i := x0; i < x1+1; i++ {
			j := pixOffset(i, y1, source.Rect, source.Stride, 8)
			for k := 0; k < 8; k++ {
				source.Pix[j+k] = colorBytes[k]
			}
		}
		for i := y0; i < y1; i++ {
			j := pixOffset(x0, i, source.Rect, source.Stride, 8)
			for k := 0; k < 8; k++ {
				source.Pix[j+k] = colorBytes[k]
			}
		}
		for i := y0; i < y1; i++ {
			j := pixOffset(x1, i, source.Rect, source.Stride, 8)
			for k := 0; k < 8; k++ {
				source.Pix[j+k] = colorBytes[k]
			}
		}
	}

	myWin.centerContent.Refresh()
}

func moveRoiCenter() {
	myWin.x0 = myWin.imageWidth/2 - myWin.roiWidth/2
	myWin.y0 = myWin.imageHeight/2 - myWin.roiHeight/2
	myWin.x1 = myWin.x0 + myWin.roiWidth - 1
	myWin.y1 = myWin.y0 + myWin.roiHeight - 1

	myWin.roiChanged = true

	saveROIposToPreferences()

	displayFitsImage()
	showROI()
}

func saveROIposToPreferences() {
	myWin.App.Preferences().SetString("ROIx0", fmt.Sprintf("%d", myWin.x0))
	myWin.App.Preferences().SetString("ROIx1", fmt.Sprintf("%d", myWin.x1))
	myWin.App.Preferences().SetString("ROIy0", fmt.Sprintf("%d", myWin.y0))
	myWin.App.Preferences().SetString("ROIy1", fmt.Sprintf("%d", myWin.y1))
}

func moveRoiUp() {
	if myWin.y0 < myWin.yJogSize {
		dialog.ShowInformation("Information", "ROI too close to image boundary", myWin.parentWindow)
		return
	}

	myWin.y0 -= myWin.yJogSize
	myWin.y1 -= myWin.yJogSize
	saveROIposToPreferences()

	myWin.roiChanged = true
	displayFitsImage()
	showROI()
}

func moveRoiDown() {
	if myWin.y1+myWin.yJogSize > myWin.imageHeight {
		dialog.ShowInformation("Information", "ROI too close to image boundary", myWin.parentWindow)
		return
	}
	myWin.y0 += myWin.yJogSize
	myWin.y1 += myWin.yJogSize

	saveROIposToPreferences()

	myWin.roiChanged = true
	displayFitsImage()
	showROI()
}

func moveRoiLeft() {
	if myWin.x0 < myWin.xJogSize {
		dialog.ShowInformation("Information", "ROI too close to image boundary", myWin.parentWindow)
		return
	}

	myWin.x0 -= myWin.xJogSize
	myWin.x1 -= myWin.xJogSize

	saveROIposToPreferences()

	myWin.roiChanged = true
	displayFitsImage()
	showROI()
}

func moveRoiRight() {
	if myWin.x1+myWin.xJogSize > myWin.imageWidth {
		dialog.ShowInformation("Information", "ROI too close to image boundary", myWin.parentWindow)
		return
	}

	myWin.x0 += myWin.xJogSize
	myWin.x1 += myWin.xJogSize

	saveROIposToPreferences()

	myWin.roiChanged = true
	displayFitsImage()
	showROI()
}

func applyRoi(checked bool) {

	validateROIparameters()

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

func validateROIparameters() {
	// Validate ROI size and position - this is needed because the saved values from a previous
	// run with a different image may have resulted in the saving to preferences of values that
	// are wrong for the current image.
	var changeMade = false
	if myWin.roiWidth > myWin.imageWidth {
		changeMade = true
		myWin.roiWidth = myWin.imageWidth / 2
		myWin.x0 = myWin.imageWidth/2 - myWin.roiWidth/2
		myWin.x1 = myWin.x0 + myWin.roiWidth
		widthStr := fmt.Sprintf("%d", myWin.roiWidth)
		_ = myWin.widthStr.Set(widthStr)
		myWin.App.Preferences().SetString("ROIwidth", widthStr)
	}

	if myWin.roiHeight > myWin.imageHeight {
		changeMade = true
		myWin.roiHeight = myWin.imageHeight / 2
		myWin.y0 = myWin.imageHeight/2 - myWin.roiHeight/2
		myWin.y1 = myWin.y0 + myWin.roiHeight
		_ = myWin.heightStr.Set(fmt.Sprintf("%d", myWin.roiHeight))
		heightStr := fmt.Sprintf("%d", myWin.roiHeight)
		_ = myWin.heightStr.Set(heightStr)
		myWin.App.Preferences().SetString("ROIheight", heightStr)
	}

	if myWin.x1 >= myWin.imageWidth {
		changeMade = true
		myWin.x0 = myWin.imageWidth/2 - myWin.roiWidth/2
		myWin.x1 = myWin.x0 + myWin.roiWidth
	}

	if myWin.y1 >= myWin.imageHeight {
		changeMade = true
		myWin.y0 = myWin.imageHeight/2 - myWin.roiHeight/2
		myWin.y1 = myWin.y0 + myWin.roiHeight
	}

	if changeMade {
		saveROIposToPreferences()
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

func folderHistorySelect() {

	// Build a dialog for holding a history of recently opened FITS folders.
	// Provide a button to open a browser and a way to remove un-needed entries

	// myWin.fitsFolderHistory is []string holding recently opened folder paths
	// Note: processFolderSelection() is called on when a path (possibly blank) is selected from the dropdown list
	selector := widget.NewSelect(myWin.fitsFolderHistory, func(path string) { processFolderSelection(path) })
	myWin.folderSelect = selector // Save for use by openSelections()

	// Configure selector
	selector.PlaceHolder = "Make selection from folder history ..."
	folderSelectWin := myWin.App.NewWindow("FITS folder history (and options)")
	myWin.folderSelectWin = folderSelectWin
	folderSelectWin.Resize(fyne.Size{Height: 450, Width: 700})

	// Add control to allow user to specify that clicked-on paths be removed from the history
	deleteCheckbox := widget.NewCheck("Delete path clicked on", func(checked bool) {})
	myWin.deletePathCheckbox = deleteCheckbox

	topLine := container.NewHBox(
		deleteCheckbox,
		widget.NewButton("Open file browser", func() {
			openFileBrowser() // This does not open a browser, just sets a flag
		}),
		layout.NewSpacer())
	ctr := container.NewVBox(topLine, selector)
	ctr.Add(layout.NewSpacer())
	folderSelectWin.SetContent(ctr)
	folderSelectWin.CenterOnScreen()

	folderSelectWin.SetCloseIntercept(func() { processFolderSelectionClosed() })
	folderSelectWin.Show()
}

func openFileBrowser() {
	myWin.fileBrowserRequested = true
	myWin.folderSelectWin.Close()
	openNewFolderDialog("")
}

func removePath(paths []string, path string) []string {
	// This is used to remove FITS folder paths
	var newPaths []string
	for _, i := range paths {
		if i != path {
			newPaths = append(newPaths, i)
		}
	}
	return newPaths
}

func processFolderSelectionClosed() {
	myWin.folderSelectWin.Close()
	return
}

func processFolderSelection(path string) {

	if myWin.deletePathCheckbox.Checked {
		//fmt.Printf("Selection occurred while in Delete mode, so removing entry %s\n", path)
		myWin.fitsFolderHistory = removePath(myWin.fitsFolderHistory, path)
		saveFolderHistory()
		path = ""
	}
	myWin.folderSelected = path
	if path != "" {
		addPathToHistory(path) // ... only adds path if not already there
		saveFolderHistory()
	}
	myWin.selectionMade = true
	myWin.folderSelectWin.Close()
}

func saveFolderHistory() {
	myWin.App.Preferences().SetStringList("folderHistory", myWin.fitsFolderHistory)
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
			dialog.ShowInformation("Oops",
				"A integer > 0 is needed for ROI height.", myWin.parentWindow)
			_ = myWin.heightStr.Set(fmt.Sprintf("%d", myWin.roiHeight))
			return
		}

		if proposedRoiHeight > myWin.imageHeight {
			dialog.ShowInformation("Oops",
				fmt.Sprintf("ROI height cannot exceed %d", myWin.imageHeight),
				myWin.parentWindow)
			_ = myWin.heightStr.Set(fmt.Sprintf("%d", myWin.imageHeight/2))
			proposedRoiHeight = myWin.imageHeight / 2
		}

		if proposedRoiWidth > myWin.imageWidth {
			dialog.ShowInformation("Oops",
				fmt.Sprintf("ROI width cannot exceed %d", myWin.imageWidth),
				myWin.parentWindow)
			_ = myWin.widthStr.Set(fmt.Sprintf("%d", myWin.imageWidth/2))
			proposedRoiWidth = myWin.imageWidth / 2
		}

		myWin.App.Preferences().SetString("ROIwidth", widthStr)
		myWin.App.Preferences().SetString("ROIheight", heightStr)

		myWin.roiHeight = proposedRoiHeight
		myWin.roiWidth = proposedRoiWidth
		moveRoiCenter()

		// This causes the ROI change to be applied to the current image
		myWin.roiChanged = true
		//makeDisplayBuffer(myWin.roiWidth, myWin.roiHeight)
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

	//var lastFitsFolderStr string

	folderHistorySelect() // Build and open the selection dialog

	for { // Wait for a selection to be made in an infinite loop or Browser open button clicked
		time.Sleep(1 * time.Millisecond)
		if myWin.selectionMade || myWin.fileBrowserRequested {
			myWin.selectionMade = false
			break
		}
	}

	if myWin.selectionMade { // User clicked on an entry in the selection list
		// This Close() will invoke processFolderSelection()
		myWin.folderSelectWin.Close()
	}

	// If the user just closed the folder selection window or selected a blank line, "" is returned.
	if myWin.folderSelected == "" && !myWin.fileBrowserRequested {
		return
	}

	myWin.fileBrowserRequested = false

	if myWin.folderSelected != "" {
		processFitsFolderPath(myWin.folderSelected)
	}

}

func openNewFolderDialog(lastFitsFolderStr string) {
	lastFitsFolderStr = myWin.App.Preferences().StringWithFallback("lastFitsFolder", "")

	showFolder := dialog.NewFolderOpen(
		func(path fyne.ListableURI, err error) { processFitsFolderSelection(path, err) },
		myWin.parentWindow,
	)

	myWin.showFolder = showFolder

	showFolder.Resize(fyne.Size{
		Width:  800,
		Height: 600,
	})

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

func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func processFitsFolderSelection(path fyne.ListableURI, err error) {
	myWin.showFolder.Hide()
	if err != nil {
		fmt.Println(fmt.Errorf("%w\n", err))
		return
	}
	if path != nil {
		//fmt.Printf("folder selected: %s\n", path)
		initializeConfig(true)

		myWin.App.Preferences().SetString("lastFitsFolder", path.Path())

		myWin.fitsFilePaths = getFitsFilenames(path.Path())
		if len(myWin.fitsFilePaths) == 0 {
			dialog.ShowInformation("Oops",
				"No .fits files were found there!",
				myWin.parentWindow,
			)
			return
		}

		folderToLookFor := path.Path()
		addPathToHistory(folderToLookFor) // ... only adds path if not already there

		// A 'tidy' func that removes invalid entries: ones that don't exist or non-directory
		// This takes care of cases where the user moved or deleted a folder but the path
		// is still present in the history.
		var tidyFolderHistory []string
		for _, folderToCheck := range myWin.fitsFolderHistory {
			if pathExists(folderToCheck) {
				if isDirectory(folderToCheck) {
					tidyFolderHistory = append(tidyFolderHistory, folderToCheck)
				} else {
					continue
				}
			} else {
				continue
			}
		}
		myWin.fitsFolderHistory = tidyFolderHistory
		saveFolderHistory()

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

func addPathToHistory(path string) {
	// We only add the given path to the folder path history if it is not already there
	dupPath := false
	for _, folderName := range myWin.fitsFolderHistory {
		if folderName == path {
			dupPath = true
			break
		}
	}
	if !dupPath {
		myWin.fitsFolderHistory = append(myWin.fitsFolderHistory, path)
	}
}

func processFitsFolderPath(path string) {
	//fmt.Printf("folder selected: %s\n", path)
	initializeConfig(true)

	myWin.autoContrastNeeded = true

	myWin.fitsFilePaths = getFitsFilenames(path)
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

	if len(myWin.fitsFilePaths) > 0 {
		displayFitsImage()
	}
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

	// A side effect of this call is that myWin.displayBuffer is filled
	imageToUse, _, timestamp := getFitsImageFromFilePath(myWin.fitsFilePaths[myWin.fileIndex])

	if imageToUse == nil {
		return nil
	}

	myWin.timestampLabel.Text = timestamp

	if myWin.whiteSlider != nil {
		switch myWin.imageKind {
		case "Gray32":
			copy(myWin.displayBuffer, imageToUse.Image.(*fltimg.Gray32).Pix)
		case "Gray64":
			copy(myWin.displayBuffer, imageToUse.Image.(*fltimg.Gray64).Pix)
		case "Gray":
			applyContrastControls(imageToUse.Image.(*image.Gray).Pix, myWin.displayBuffer, "Gray")
			myWin.fitsImages[0].Image.(*image.Gray).Pix = myWin.displayBuffer
		case "Gray16":
			applyContrastControls(imageToUse.Image.(*image.Gray16).Pix, myWin.displayBuffer, "Gray16")
			myWin.fitsImages[0].Image.(*image.Gray16).Pix = myWin.displayBuffer
		default:
			dialog.ShowInformation("Oops",
				fmt.Sprintf("The image kind (%s) is unrecognized.", myWin.imageKind),
				myWin.parentWindow)
			return nil
		}
	}

	switch myWin.imageKind {
	case "Gray16":
		//printImageStats("@point 1")
		copy(myWin.fitsImages[0].Image.(*image.Gray16).Pix, myWin.displayBuffer)
		if !myWin.roiActive {
			restoreRect()
		} else {
			setROIrect()
		}
	case "Gray":
		//printImageStats("@point 2")
		myWin.fitsImages[0].Image.(*image.Gray).Pix = myWin.displayBuffer
		if !myWin.roiActive {
			restoreRect()
		} else {
			setROIrect()
		}
	case "Gray32":
		myWin.fitsImages[0].Image.(*fltimg.Gray32).Pix = myWin.displayBuffer
		myWin.fitsImages[0].Image.(*fltimg.Gray32).Min = float32(myWin.blackSlider.Value)
		myWin.fitsImages[0].Image.(*fltimg.Gray32).Max = float32(myWin.whiteSlider.Value)
		if !myWin.roiActive {
			restoreRect()
		} else {
			setROIrect()
		}
	case "Gray64":
		myWin.fitsImages[0].Image.(*fltimg.Gray64).Pix = myWin.displayBuffer
		myWin.fitsImages[0].Image.(*fltimg.Gray64).Min = myWin.blackSlider.Value
		myWin.fitsImages[0].Image.(*fltimg.Gray64).Max = myWin.whiteSlider.Value
		if !myWin.roiActive {
			restoreRect()
		} else {
			setROIrect()
		}
	default:
		dialog.ShowInformation("Oops",
			fmt.Sprintf("The image kind (%s) is unrecognized.", myWin.imageKind),
			myWin.parentWindow)
		return nil
	}

	myWin.centerContent.Objects[0] = myWin.fitsImages[0]

	myWin.centerContent.Refresh()
	myWin.waitingForFileRead = false // Signal to anyone waiting for file read completion

	return imageToUse
}

func restoreRect() {
	switch myWin.imageKind {
	case "Gray16":
		myWin.fitsImages[0].Image.(*image.Gray16).Rect.Max = image.Point{
			X: myWin.imageWidth,
			Y: myWin.imageHeight,
		}
		myWin.fitsImages[0].Image.(*image.Gray16).Rect.Min = image.Point{
			X: 0,
			Y: 0,
		}
	case "Gray":
		myWin.fitsImages[0].Image.(*image.Gray).Rect.Max = image.Point{
			X: myWin.imageWidth,
			Y: myWin.imageHeight,
		}
		myWin.fitsImages[0].Image.(*image.Gray).Rect.Min = image.Point{
			X: 0,
			Y: 0,
		}
	case "Gray32":
		myWin.fitsImages[0].Image.(*fltimg.Gray32).Rect.Max = image.Point{
			X: myWin.imageWidth,
			Y: myWin.imageHeight,
		}
		myWin.fitsImages[0].Image.(*fltimg.Gray32).Rect.Min = image.Point{
			X: 0,
			Y: 0,
		}
	case "Gray64":
		myWin.fitsImages[0].Image.(*fltimg.Gray64).Rect.Max = image.Point{
			X: myWin.imageWidth,
			Y: myWin.imageHeight,
		}
		myWin.fitsImages[0].Image.(*fltimg.Gray64).Rect.Min = image.Point{
			X: 0,
			Y: 0,
		}
	default:
		dialog.ShowInformation("Oops",
			fmt.Sprintf("The image kind (%s) is unrecognized.", myWin.imageKind),
			myWin.parentWindow)
	}
}

func setROIrect() {
	switch myWin.imageKind {
	case "Gray16":
		myWin.fitsImages[0].Image.(*image.Gray16).Rect.Max = image.Point{
			X: myWin.x1,
			Y: myWin.y1,
		}
		myWin.fitsImages[0].Image.(*image.Gray16).Rect.Min = image.Point{
			X: myWin.x0,
			Y: myWin.y0,
		}
	case "Gray":
		myWin.fitsImages[0].Image.(*image.Gray).Rect.Max = image.Point{
			X: myWin.x1,
			Y: myWin.x1,
		}
		myWin.fitsImages[0].Image.(*image.Gray).Rect.Min = image.Point{
			X: myWin.x0,
			Y: myWin.y0,
		}
	case "Gray32":
		myWin.fitsImages[0].Image.(*fltimg.Gray32).Rect.Max = image.Point{
			X: myWin.x1,
			Y: myWin.y1,
		}
		myWin.fitsImages[0].Image.(*fltimg.Gray32).Rect.Min = image.Point{
			X: myWin.x0,
			Y: myWin.y0,
		}
	case "Gray64":
		myWin.fitsImages[0].Image.(*fltimg.Gray64).Rect.Max = image.Point{
			X: myWin.x1,
			Y: myWin.y1,
		}
		myWin.fitsImages[0].Image.(*fltimg.Gray64).Rect.Min = image.Point{
			X: myWin.x0,
			Y: myWin.y0,
		}
	default:
		dialog.ShowInformation("Oops",
			fmt.Sprintf("The image kind (%s) is unrecognized.", myWin.imageKind),
			myWin.parentWindow)
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

	if fitsImage == nil {
		return
	}

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
	case "Gray64":
		myWin.bytesPerPixel = 8
	default:
		msg := fmt.Sprintf("%s is not an image kind that is supported.", kind)
		dialog.ShowInformation("Oops", msg, myWin.parentWindow)
		return
	}

	makeDisplayBuffer(myWin.imageWidth, myWin.imageHeight)

	myWin.fileSlider.SetValue(0)
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func pixOffset(x int, y int, r image.Rectangle, stride int, pixelByteCount int) int {
	ans := (y-r.Min.Y)*stride + (x-r.Min.X)*pixelByteCount
	return ans
}

//func conv8ByteSliceToFloat64(p []byte, offset int) (float64, uint64) {
//	valueUint64 := binary.LittleEndian.Uint64(p[offset : offset+8])
//	return float64(valueUint64), valueUint64
//}

//func conv4ByteSliceToFloat64(p []byte, offset int) (float64, uint32) {
//	valueUint32 := binary.LittleEndian.Uint32(p[offset : offset+4])
//	return float64(valueUint32), valueUint32
//}

//func conv2ByteSliceToFloat64(p []byte, offset int) (float64, uint16) {
//	valueUint16 := binary.LittleEndian.Uint16(p[offset : offset+2])
//	return float64(valueUint16), valueUint16
//}

//func conv1ByteSliceToFloat64(p []byte, offset int) (float64, uint8) {
//	return float64(p[offset]), p[offset]
//}

//func convByteSliceToFloat64(p []byte, offset int) (float64, uint64) {
//	valueUint64 := binary.LittleEndian.Uint64(p[offset : offset+8])
//	return float64(valueUint64), valueUint64
//}

//func convUint32ToBytes(b []byte, v uint32) []byte {
//	// LittleEndian order
//	return append(b,
//		byte(v),
//		byte(v>>8),
//		byte(v>>16),
//		byte(v>>24),
//	)
//}
//
//func convUint64ToBytes(b []byte, v uint64) []byte {
//	// LittleEndian order
//	return append(b,
//		byte(v),
//		byte(v>>8),
//		byte(v>>16),
//		byte(v>>24),
//		byte(v>>32),
//		byte(v>>40),
//		byte(v>>48),
//		byte(v>>56),
//	)
//}

func getFitsImageFromFilePath(filePath string) (*canvas.Image, []string, string) {
	// Ann important side effect of this function: it fills the myWin.displayBuffer []byte

	//TODO Remove this test code
	//bobFloat32, bobUint32 := convByteSliceToFloat32([]byte{1, 2, 3, 255}, 0)
	//bobBytes := convUint32ToBytes([]byte{}, bobUint32)
	//fmt.Println(bobBytes)
	//fmt.Printf("bobFloat32: %f\n", bobFloat32)
	//bobFloat64, bobUint64 := convByteSliceToFloat64([]byte{1, 2, 3, 4, 5, 6, 7, 255}, 0)
	//bobBytes = convUint64ToBytes([]byte{}, bobUint64)
	//fmt.Println(bobBytes)
	//fmt.Printf("bobFloat64: %f\n", bobFloat64)

	f := openFitsFile(filePath)
	myWin.primaryHDU = f.HDU(0)
	metaData, timestamp := formatMetaData(myWin.primaryHDU)

	closeErr := f.Close()
	if closeErr != nil {
		errMsg := fmt.Errorf("could not close %s: %w", filePath, closeErr)
		fmt.Printf(errMsg.Error())
	}

	goImage := myWin.primaryHDU.(fitsio.Image).Image()

	// TODO Test to show that Raw and Pix are the same thing (must be run on FITS with matching pixel type
	//bob := myWin.primaryHDU.(fitsio.Image).Raw()
	//minValue := 0.0
	//maxValue := 0.0
	//numPrinted := 0
	//for i := 0; i < len(bob)-8; i += 8 {
	//	ans := math.Float64frombits(binary.BigEndian.Uint64(bob[i : i+8]))
	//	if ans < minValue {
	//		minValue = ans
	//	}
	//	if ans > maxValue {
	//		maxValue = ans
	//	}
	//	if ans != 0.0 {
	//		numPrinted += 1
	//		if numPrinted < 1000 {
	//			fmt.Printf("%d %0.4f\n", i, ans)
	//		}
	//	}
	//}
	//fmt.Printf("%0.4f  %0.4f\n", minValue, maxValue)
	//fmt.Println(len(bob))
	//for i := 0; i < len(bob); i++ {
	//	if bob[i] != 0 {
	//		fmt.Printf("%d  %d\n", i, bob[i])
	//	}
	//}

	if goImage == nil {
		dialog.ShowInformation("Oops", "No images are present in the .fits file", myWin.parentWindow)
		return nil, []string{}, ""
	}
	kind := reflect.TypeOf(goImage).Elem().Name()
	myWin.imageKind = kind

	// It is desirable to 'normalize' the image so that only one image kind needs
	// to be dealt with by the rest of the code. We convert 16-bit integer, 32-bit float,
	// and 64-bit float to 8 bit byte.
	//var normed []byte
	// TODO Remove this code !!!
	//switch kind {
	//case "Gray":
	//	// Nothing to do - already in 8-bit form
	//case "Gray16":
	//	original := goImage.(*image.Gray16).Pix
	//	biggest := uint8(0)
	//	size := len(original)
	//	for i := 0; i < size; i += 2 {
	//		//normed = append(normed, original[i+1])
	//		original[i+1] = 224
	//		if original[i] > biggest {
	//			biggest = original[i]
	//		}
	//	}
	//	fmt.Printf("max: %d\n", biggest)
	//	//kind = "Gray"
	//	//goImage.(*image.Gray16).Pix = normed
	//case "Gray32":
	//case "Gray64":
	//default:
	//	msg := fmt.Sprintf("Unexpected 'kind': %s", kind)
	//	dialog.ShowInformation("Oops", msg, myWin.parentWindow)
	//	return nil, []string{}, ""
	//}

	fitsImage := canvas.NewImageFromImage(goImage) // This is a Fyne image

	validateROIsize(goImage)

	if myWin.roiActive {

		if kind == "Gray16" {
			// Test of homegrown SubImage  See kind == "Gray" below
			roi := goImage.(*image.Gray16)
			orgRect := roi.Rect
			orgPix := roi.Pix

			pixOffset := pixOffset(myWin.x0, myWin.y0, orgRect, roi.Stride, 2)
			roi.Pix = orgPix[pixOffset:]
			roi.Rect = image.Rect(myWin.x0, myWin.y0, myWin.x1, myWin.y1)

			fitsImage = canvas.NewImageFromImage(roi) // This is a Fyne image
			myWin.workingBuffer = make([]byte, len(fitsImage.Image.(*image.Gray16).Pix))
			copy(myWin.workingBuffer, fitsImage.Image.(*image.Gray16).Pix)
		}

		if kind == "Gray" {
			roi := goImage.(*image.Gray).SubImage(image.Rect(myWin.x0, myWin.y0, myWin.x1, myWin.y1))

			fitsImage = canvas.NewImageFromImage(roi) // This is a Fyne image
			myWin.workingBuffer = make([]byte, len(fitsImage.Image.(*image.Gray).Pix))
			copy(myWin.workingBuffer, fitsImage.Image.(*image.Gray).Pix)
		}

		if kind == "Gray64" {
			roi := goImage.(*fltimg.Gray64)
			orgRect := roi.Rect
			orgPix := roi.Pix
			pixoffset := pixOffset(myWin.x0, myWin.y0, orgRect, roi.Stride, 8)
			roi.Pix = orgPix[pixoffset:]
			roi.Rect = image.Rect(myWin.x0, myWin.y0, myWin.x1, myWin.y1)

			fitsImage = canvas.NewImageFromImage(roi) // This is a Fyne image
			myWin.workingBuffer = make([]byte, len(roi.Pix))
			copy(myWin.workingBuffer, roi.Pix)
		}

		if kind == "Gray32" {
			roi := goImage.(*fltimg.Gray32)
			orgRect := roi.Rect
			orgPix := roi.Pix
			pixoffset := pixOffset(myWin.x0, myWin.y0, orgRect, roi.Stride, 4)
			roi.Pix = orgPix[pixoffset:]
			roi.Rect = image.Rect(myWin.x0, myWin.y0, myWin.x1, myWin.y1)

			fitsImage = canvas.NewImageFromImage(roi) // This is a Fyne image
			myWin.workingBuffer = make([]byte, len(roi.Pix))
			copy(myWin.workingBuffer, roi.Pix)
		}

		if myWin.roiChanged {
			myWin.roiChanged = false
			myWin.fitsImages[0] = fitsImage
			myWin.centerContent.Objects[0] = fitsImage
		}
	}

	if !myWin.roiActive {
		if kind == "Gray" {
			copy(myWin.workingBuffer, fitsImage.Image.(*image.Gray).Pix)
		}
		if kind == "Gray16" {
			copy(myWin.workingBuffer, fitsImage.Image.(*image.Gray16).Pix)
		}
		if kind == "Gray32" {
			copy(myWin.workingBuffer, fitsImage.Image.(*fltimg.Gray32).Pix)
		}
		if kind == "Gray64" {
			copy(myWin.workingBuffer, fitsImage.Image.(*fltimg.Gray64).Pix)
		}
	}

	switch kind {
	case "Gray":
		fallthrough
	case "Gray16":
		myWin.whiteSlider.Max = 255.0
		myWin.whiteSlider.Min = 0.0
		myWin.blackSlider.Max = 255.0
		myWin.blackSlider.Min = 0.0
	case "Gray32":
		myWin.whiteSlider.Max = float64(fitsImage.Image.(*fltimg.Gray32).Max)
		myWin.whiteSlider.Min = float64(fitsImage.Image.(*fltimg.Gray32).Min)
		myWin.blackSlider.Max = float64(fitsImage.Image.(*fltimg.Gray32).Max)
		myWin.blackSlider.Min = float64(fitsImage.Image.(*fltimg.Gray32).Min)
	case "Gray64":
		myWin.whiteSlider.Max = fitsImage.Image.(*fltimg.Gray64).Max
		myWin.whiteSlider.Min = fitsImage.Image.(*fltimg.Gray64).Min
		myWin.blackSlider.Max = fitsImage.Image.(*fltimg.Gray64).Max
		myWin.blackSlider.Min = fitsImage.Image.(*fltimg.Gray64).Min
	}

	//fitsImage.FillMode = canvas.ImageFillOriginal
	fitsImage.FillMode = canvas.ImageFillContain
	return fitsImage, metaData, timestamp
}

func validateROIsize(goImage image.Image) {
	// Fix user setting ROI X size too large
	var changeMade = false
	width := goImage.Bounds().Max.X
	if myWin.roiWidth > width {
		changeMade = true
		myWin.roiWidth = width / 2
		_ = myWin.widthStr.Set(strconv.Itoa(width / 2))
	}

	// Fix user setting ROI Y size too large
	height := goImage.Bounds().Max.Y
	if myWin.roiHeight > height {
		changeMade = true
		myWin.roiHeight = height / 2
		_ = myWin.heightStr.Set(strconv.Itoa(height / 2))
	}

	if changeMade {
		myWin.x0 = width/2 - myWin.roiWidth/2
		myWin.x1 = myWin.x0 + myWin.roiWidth - 1
		myWin.y0 = height/2 - myWin.roiHeight/2
		myWin.y1 = myWin.y0 + myWin.roiHeight - 1
		saveROIposToPreferences()
	}
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
		}
		if kind == "Gray" {
			std, err = getStd(original, 1, 255)
		}
		//if kind == "Gray32" {
		//	std, err = getStd(original, 4, 255)
		//	fmt.Printf("std: %0.1f\n", std)
		//}
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
