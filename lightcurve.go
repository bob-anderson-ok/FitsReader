package main

import (
	"fmt"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
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
	//fmt.Printf("frames %d to %d inclusive will be used to build flash lightcurve\n",
	//	myWin.lightCurveStartFrame, myWin.lightCurveEndFrame)
}
