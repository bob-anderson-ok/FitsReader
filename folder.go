package main

import (
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"os"
	"strings"
	"time"
)

func processFitsFolderSelectedByFolderDialog(path fyne.ListableURI, err error) {
	myWin.showFolder.Hide()
	if err != nil {
		fmt.Println(fmt.Errorf("%w\n", err))
		return
	}
	processChosenListableURI(path)
}

func processChosenListableURI(path fyne.ListableURI) {
	if path != nil {
		processChosenFolderString(path.Path())
		////fmt.Printf("folder selected: %s\n", path)
		//myWin.leftGoalpostTimestamp = ""
		//myWin.rightGoalpostTimestamp = ""
		//initializeConfig(true)
		//
		//myWin.App.Preferences().SetString("lastFitsFolder", path.Path())
		//
		//myWin.fitsFilePaths = getFitsFilenames(path.Path())
		//if len(myWin.fitsFilePaths) == 0 {
		//	dialog.ShowInformation("Oops",
		//		"No .fits files were found there!",
		//		myWin.parentWindow,
		//	)
		//	return
		//}
		//
		//folderToLookFor := path.Path()
		//addPathToHistory(folderToLookFor) // ... only adds path if not already there
		//
		//// A 'tidy' func that removes invalid entries: ones that don't exist or non-directory
		//// This takes care of cases where the user moved or deleted a folder but the path
		//// is still present in the history.
		//var tidyFolderHistory []string
		//for _, folderToCheck := range myWin.fitsFolderHistory {
		//	if pathExists(folderToCheck) {
		//		if isDirectory(folderToCheck) {
		//			tidyFolderHistory = append(tidyFolderHistory, folderToCheck)
		//		} else {
		//			continue
		//		}
		//	} else {
		//		continue
		//	}
		//}
		//myWin.fitsFolderHistory = tidyFolderHistory
		//saveFolderHistory()
		//
		//myWin.fileIndex = 0
		//myWin.currentFilePath = myWin.fitsFilePaths[myWin.fileIndex]
		//myWin.fitsImages = []*canvas.Image{}
		//myWin.timestamps = []string{}
		//myWin.metaData = [][]string{}
		//myWin.fileIndex = 0
		//enableRoiControls()
		//initializeImages()
		//myWin.fileSlider.SetValue(0)
	}
	//if len(myWin.fitsFilePaths) > 0 {
	//	if !alreadyHasIotaTimestamps(processedByIotaUtilities) && myWin.addFlashTimestampsCheckbox.Checked {
	//		displayFitsImage()
	//		readEdgeTimeFile(path.Path())
	//		if myWin.leftGoalpostTimestamp != "" && myWin.rightGoalpostTimestamp != "" {
	//			buildFlashLightcurve()
	//		}
	//	}
	//	displayFitsImage()
	//}
}

func processChosenFolderString(path string) {
	if path != "" {
		//fmt.Printf("folder selected: %s\n", path)
		myWin.leftGoalpostTimestamp = ""
		myWin.rightGoalpostTimestamp = ""
		initializeConfig(true)

		myWin.App.Preferences().SetString("lastFitsFolder", path)

		myWin.fitsFilePaths = getFitsFilenames(path)
		if len(myWin.fitsFilePaths) == 0 {
			dialog.ShowInformation("Oops",
				"No .fits files were found there!",
				myWin.parentWindow,
			)
			return
		}

		folderToLookFor := path
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
		myWin.fitsImages = []*canvas.Image{}
		myWin.timestamps = []string{}
		myWin.metaData = [][]string{}
		myWin.fileIndex = 0
		enableRoiControls()
		initializeImages()
		myWin.fileSlider.SetValue(0)
	}
	if len(myWin.fitsFilePaths) > 0 {
		if !alreadyHasIotaTimestamps(processedByIotaUtilities) && myWin.addFlashTimestampsCheckbox.Checked {
			displayFitsImage()
			readEdgeTimeFile(path)
			if myWin.leftGoalpostTimestamp != "" && myWin.rightGoalpostTimestamp != "" {
				buildFlashLightcurve()
			}
		}
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

func openNewFolderDialog(lastFitsFolderStr string) {
	lastFitsFolderStr = myWin.App.Preferences().StringWithFallback("lastFitsFolder", "")

	if myWin.cmdLineFolder != "" {
		lastFitsFolderStr = myWin.cmdLineFolder
	}

	showFolder := dialog.NewFolderOpen(
		func(path fyne.ListableURI, err error) { processFitsFolderSelectedByFolderDialog(path, err) },
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

func processFitsFolderPickedFromHistory(path string) {
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
		//myWin.autoStretchCheckbox.SetChecked(true)
		displayFitsImage()
	}
	if myWin.leftGoalpostTimestamp != "" && myWin.rightGoalpostTimestamp != "" {
		buildFlashLightcurve()
	}
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

func readEdgeTimeFile(path string) {
	var onTimes []string

	myWin.leftGoalpostTimestamp = ""
	myWin.leftGoalpostTimestamp = ""

	filePath := path + "/" + edgeTimesFileName

	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		msg := fmt.Sprintf("Could not find edge time file @ %s\n", filePath)
		dialog.ShowInformation("Edge time file error:", msg, myWin.parentWindow)
	} else {
		//msg := fmt.Sprintf("edge time file found at: %s\n", filePath)
		//dialog.ShowInformation("Edge time file report:", msg, myWin.parentWindow)

		content, err := os.ReadFile(filePath) // Grab all the file in one gulp of []byte
		if err != nil {
			msg := fmt.Sprintf("Attempt to read edge time file gave error: %s\n", err)
			dialog.ShowInformation("Edge time file error:", msg, myWin.parentWindow)
		} else {
			lines := string(content) // Convert []byte to string
			bob := strings.Split(lines, "\n")

			for _, line := range bob {
				if !strings.Contains(line, "Z") {
					line += "Z" // Add a terminating Z if the timestamp did not already indicate that it was a UTC value
				}
				if strings.Contains(line, "on") {
					onLineParts := strings.Split(line, "on  ")
					onTimes = append(onTimes, onLineParts[1])
				}
			}
			if len(onTimes) < 2 {
				msg := fmt.Sprintln("Less than 2 flash-on times found in edge times file.")
				dialog.ShowInformation("Edge time file error:", msg, myWin.parentWindow)
			} else {
				myWin.leftGoalpostTimestamp = onTimes[0]
				myWin.rightGoalpostTimestamp = onTimes[len(onTimes)-1]
				//msg := fmt.Sprintf("left goalpost occurred @  %s\n\n", myWin.leftGoalpostTimestamp)
				//msg += fmt.Sprintf("right goalpost occurred @  %s\n\n", myWin.rightGoalpostTimestamp)
				//dialog.ShowInformation("Goalpost timestamp report:", msg, myWin.parentWindow)
			}
		}
	}
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

	if myWin.addFlashTimestampsCheckbox.Checked {
		myWin.leftGoalpostTimestamp = ""
		myWin.rightGoalpostTimestamp = ""
		readEdgeTimeFile(path)
		if myWin.leftGoalpostTimestamp != "" && myWin.rightGoalpostTimestamp != "" {
			buildFlashLightcurve()
		}
	}
}

func saveFolderHistory() {
	myWin.App.Preferences().SetStringList("folderHistory", myWin.fitsFolderHistory)
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

func chooseFitsFolder() {

	folderHistorySelect() // Build and open the selection dialog

	for { // Wait for a selection to be made in an infinite loop or Browser open button clicked
		time.Sleep(1 * time.Millisecond)
		if myWin.selectionMade || myWin.fileBrowserRequested {
			myWin.selectionMade = false
			break
		}
	}

	// If the user just closed the folder selection window or selected a blank line, "" is returned.
	if myWin.folderSelected == "" && !myWin.fileBrowserRequested {
		return
	}

	myWin.fileBrowserRequested = false

	if myWin.folderSelected != "" {
		myWin.lightcurve = make([]float64, 0)
		processFitsFolderPickedFromHistory(myWin.folderSelected)
	}

}
