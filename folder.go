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
	"log"
	"os"
	"strings"
	"time"
)

func changeFolderSeparatorToBackslash(path string) string {
	//trace(path)
	windowsPath := strings.Replace(path, "/", "\\", -1)
	return windowsPath
}

func processFitsFolderSelectedByFolderDialog(path fyne.ListableURI, err error) {
	trace(path.Path())
	log.Println("")
	log.Println("Note: Fyne error - uri is not listable - is normal and not a problem")
	myWin.showFolder.Hide()
	if err != nil {
		log.Println(fmt.Errorf("%w\n", err))
		return
	}
	processChosenListableURI(path)
}

func processChosenListableURI(path fyne.ListableURI) {
	trace(path.Path())
	myWin.folderSelected = changeFolderSeparatorToBackslash(path.Path())
	myWin.cmdLineFolder = myWin.folderSelected
	log.Println("")
	log.Printf("Folder selected: %s", myWin.folderSelected)
	processChosenFolderString(myWin.folderSelected)
}

func processChosenFolderString(path string) {
	trace(path)
	if path != "" {
		myWin.leftGoalpostTimestamp = ""
		myWin.rightGoalpostTimestamp = ""
		initializeConfig(true)

		myWin.App.Preferences().SetString("lastFitsFolder", path)

		myWin.numDroppedFrames = 0
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

		tidyHistoryList()

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
		processNewFolder()
		displayFitsImage()
	}
}

func tidyHistoryList() {
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
}

func addPathToHistory(path string) {
	trace(path)
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
	trace(lastFitsFolderStr)
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
	//trace(path)
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}

func pathExists(path string) bool {
	//trace(path)
	_, err := os.Stat(path)
	return err == nil
}

func processFitsFolderPickedFromHistory(path string) {
	trace(path)

	log.Println("")
	log.Printf("folder selected: %s\n", path)
	initializeConfig(true)

	myWin.autoContrastNeeded = true

	myWin.numDroppedFrames = 0
	myWin.fitsFilePaths = getFitsFilenames(path)
	if len(myWin.fitsFilePaths) == 0 {
		dialog.ShowInformation("Oops",
			"No .fits files were found there!",
			myWin.parentWindow,
		)
		return
	}
	myWin.folderSelected = path
	myWin.cmdLineFolder = myWin.folderSelected
	myWin.fileIndex = 0
	myWin.currentFilePath = myWin.fitsFilePaths[myWin.fileIndex]
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

func openFileBrowser() {
	trace("")
	myWin.fileBrowserRequested = true
	myWin.folderSelectWin.Close()
	openNewFolderDialog("")
}

func removePath(paths []string, path string) []string {
	trace(path)
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
	trace("")
	myWin.folderSelectWin.Close()
	myWin.selectionMade = true
	return
}

func readEdgeTimeFile(path string) {
	trace(path)
	var onTimes []string
	var filePath string

	myWin.leftGoalpostTimestamp = ""
	myWin.leftGoalpostTimestamp = ""

	if strings.HasSuffix(path, "\\") {
		filePath = path + edgeTimesFileName
	} else {
		filePath = path + "\\" + edgeTimesFileName
	}

	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		msg := fmt.Sprintf("Could not find edge time file @ %s\n", filePath)
		dialog.ShowInformation("Edge time file error:", msg, myWin.parentWindow)
	} else {
		content, err := os.ReadFile(filePath) // Grab all the file in one gulp of []byte
		if err != nil {
			msg := fmt.Sprintf("Attempt to read edge time file gave error: %s\n", err)
			dialog.ShowInformation("Edge time file error:", msg, myWin.parentWindow)
		} else {
			lines := string(content) // Convert []byte to string
			bob := strings.Split(lines, "\n")

			for _, line := range bob {
				if strings.Contains(line, "|") { // Valid IotaGFT edge time format
					if strings.Contains(line, "on") {
						parts := strings.Split(line, "|")
						myWin.gpsUtcOffsetString = parts[1]
						onLineParts := strings.Split(parts[0], "on  ")
						onTimes = append(onTimes, onLineParts[1])
					}
				} else { // Legacy format - just in case
					if !strings.Contains(line, "Z") {
						line += "Z" // Add a terminating Z if the timestamp did not already indicate that it was a UTC value
					}
					if strings.Contains(line, "on") {
						onLineParts := strings.Split(line, "on  ")
						onTimes = append(onTimes, onLineParts[1])
					}
				}
			}
			if len(onTimes) < 2 {
				msg := fmt.Sprintln("Less than 2 flash-on times found in edge times file.")
				dialog.ShowInformation("Edge time file error:", msg, myWin.parentWindow)
			} else {
				myWin.leftGoalpostTimestamp = onTimes[0]
				myWin.rightGoalpostTimestamp = onTimes[len(onTimes)-1]
			}
		}
	}
}

func processFolderSelection(path string) {
	trace(path)
	if myWin.deletePathCheckbox.Checked {
		myWin.fitsFolderHistory = removePath(myWin.fitsFolderHistory, path)
		saveFolderHistory()
		path = ""
	}
	myWin.folderSelected = path
	myWin.cmdLineFolder = myWin.folderSelected

	if path != "" {
		addPathToHistory(path) // ... only adds path if not already there
		saveFolderHistory()
	}
	myWin.selectionMade = true
	myWin.folderSelectWin.Close()

	myWin.numDroppedFrames = 0
	myWin.fitsFilePaths = getFitsFilenames(path)
	processNewFolder()

	//if myWin.addFlashTimestampsCheckbox.Checked {
	//	myWin.leftGoalpostTimestamp = ""
	//	myWin.rightGoalpostTimestamp = ""
	//	readEdgeTimeFile(path)
	//	if myWin.leftGoalpostTimestamp != "" && myWin.rightGoalpostTimestamp != "" {
	//		//buildFlashLightcurve()
	//		addTimestampsToFitsFiles()
	//
	//	}
	//}
}

func saveFolderHistory() {
	trace("")
	myWin.App.Preferences().SetStringList("folderHistory", myWin.fitsFolderHistory)
}

func folderHistorySelect() {
	trace("")
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
			openFileBrowser()
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
	trace("")
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
		processFitsFolderPickedFromHistory(myWin.folderSelected)
	}
}
