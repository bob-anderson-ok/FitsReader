package main

import (
	"encoding/binary"
	"fmt"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/astrogo/fitsio"
	"math"
)

func askIfLoopPointsAreToBeUsed() {
	startFrameWidget := widget.NewEntry()
	endFrameWidget := widget.NewEntry()
	startFrameWidget.Text = fmt.Sprintf("%d", myWin.loopStartIndex)
	endFrameWidget.Text = fmt.Sprintf("%d", myWin.loopEndIndex)
	item1 := widget.NewFormItem("start frame", startFrameWidget)
	item2 := widget.NewFormItem("end frame", endFrameWidget)
	items := []*widget.FormItem{item1, item2}
	loopPointQuery := dialog.NewForm("Should loop start and end frames be used\n"+
		"to bracket lightcurve?", "Use", "Don't use", items,
		func(useLoopPoints bool) { processLoopPointUsageAnswer(useLoopPoints) }, myWin.parentWindow)
	loopPointQuery.Show()
}

func processLoopPointUsageAnswer(useLoopPoints bool) {
	if useLoopPoints {
		myWin.lightCurveStartFrame = myWin.loopStartIndex
		myWin.lightCurveEndFrame = myWin.loopEndIndex
	} else {
		myWin.lightCurveStartFrame = 0
		myWin.lightCurveEndFrame = myWin.numFiles - 1
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
