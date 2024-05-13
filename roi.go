package main

import (
	"fmt"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/astrogo/fitsio/fltimg"
	"image"
	"image/color"
	"strconv"
)

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

func applyAutoStretch(checked bool) {
	if checked {
		//fmt.Println("AutoStretch turned on")
		myWin.blackSlider.Hide()
		myWin.whiteSlider.Hide()
		displayFitsImage()
	} else {
		//fmt.Println("AutoStretch turned off")
		myWin.blackSlider.Show()
		myWin.whiteSlider.Show()
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

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func pixOffset(x int, y int, r image.Rectangle, stride int, pixelByteCount int) int {
	ans := (y-r.Min.Y)*stride + (x-r.Min.X)*pixelByteCount
	return ans
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
