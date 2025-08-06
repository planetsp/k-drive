package ui

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	log "github.com/planetsp/k-drive/pkg/logging"

	c "github.com/planetsp/k-drive/pkg/config"
	s "github.com/planetsp/k-drive/pkg/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/cmd/fyne_settings/settings"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var tableData = [][]string{
	{"Filename", "Date Modified", "Location", "Status"}}
var cloudProvider string
var workingDirectory string
var configReady = make(chan bool, 1)

func init() {
	// Initialize variables safely
	if c.IsConfigLoaded() {
		config := c.GetConfig()
		cloudProvider = config.CloudProvider
		workingDirectory = config.WorkingDirectory
	} else {
		cloudProvider = "aws s3"
		workingDirectory = "Not configured"
	}
}

func RunUI() {
	log.Info("Starting ui")
	myApp := app.New()

	myWindow := myApp.NewWindow(c.GetConfig().AppName)
	myWindow.Resize(fyne.NewSize(900, 400))
	myWindow.SetMainMenu(makeMenu(myApp, myWindow))

	// Check if configuration is loaded
	if !c.IsConfigLoaded() {
		showConfigDialog(myApp, myWindow)
	} else {
		configReady <- true
	}

	grid := container.New(layout.NewGridLayout(1), makeHeader(), makeFileList(), container.NewMax())
	myWindow.SetContent(grid)
	myWindow.ShowAndRun()
}

func GetConfigReadyChannel() <-chan bool {
	return configReady
}

func showConfigDialog(app fyne.App, parentWindow fyne.Window) {
	configWindow := app.NewWindow("Configuration Required")
	configWindow.Resize(fyne.NewSize(500, 400))
	configWindow.SetFixedSize(true)

	config := c.CreateDefaultConfig()

	// Create form entries
	workingDirEntry := widget.NewEntry()
	workingDirEntry.SetText(config.WorkingDirectory)
	workingDirEntry.SetPlaceHolder("e.g., /home/user/sync-folder/")

	bucketEntry := widget.NewEntry()
	bucketEntry.SetText(config.BucketName)
	bucketEntry.SetPlaceHolder("e.g., my-sync-bucket")

	pollingEntry := widget.NewEntry()
	pollingEntry.SetText(strconv.Itoa(int(config.LocalDirectoryPollingFrequency)))
	pollingEntry.SetPlaceHolder("3")

	// Browse button for working directory
	browseBtn := widget.NewButton("Browse", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				workingDirEntry.SetText(uri.Path())
			}
		}, parentWindow)
	})

	// Create and validate directory button
	createDirBtn := widget.NewButton("Create Directory", func() {
		path := workingDirEntry.Text
		if path != "" {
			err := os.MkdirAll(path, 0755)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Failed to create directory: %v", err), configWindow)
			} else {
				dialog.ShowInformation("Success", "Directory created successfully!", configWindow)
			}
		}
	})

	// Form content
	content := container.NewVBox(
		widget.NewLabel("K-Drive Configuration"),
		widget.NewSeparator(),

		widget.NewLabel("Working Directory:"),
		container.NewBorder(nil, nil, nil, container.NewHBox(browseBtn, createDirBtn), workingDirEntry),
		widget.NewLabel("This is the local folder that will be synchronized with the cloud."),

		widget.NewLabel(""),
		widget.NewLabel("S3 Bucket Name:"),
		bucketEntry,
		widget.NewLabel("The AWS S3 bucket name for cloud storage."),

		widget.NewLabel(""),
		widget.NewLabel("Polling Frequency (seconds):"),
		pollingEntry,
		widget.NewLabel("How often to check for changes in the cloud."),

		widget.NewLabel(""),
		widget.NewSeparator(),
	)

	saveBtn := widget.NewButton("Save & Start", func() {
		// Validate inputs
		workingDir := workingDirEntry.Text
		bucketName := bucketEntry.Text
		pollingStr := pollingEntry.Text

		if workingDir == "" {
			dialog.ShowError(fmt.Errorf("Working directory is required"), configWindow)
			return
		}

		if bucketName == "" {
			dialog.ShowError(fmt.Errorf("S3 bucket name is required"), configWindow)
			return
		}

		pollingFreq, err := strconv.Atoi(pollingStr)
		if err != nil || pollingFreq <= 0 {
			dialog.ShowError(fmt.Errorf("Polling frequency must be a positive number"), configWindow)
			return
		}

		// Ensure working directory exists
		if _, err := os.Stat(workingDir); os.IsNotExist(err) {
			dialog.ShowConfirm("Directory doesn't exist",
				fmt.Sprintf("The directory '%s' doesn't exist. Create it?", workingDir),
				func(create bool) {
					if create {
						err := os.MkdirAll(workingDir, 0755)
						if err != nil {
							dialog.ShowError(fmt.Errorf("Failed to create directory: %v", err), configWindow)
							return
						}
						saveConfiguration(workingDir, bucketName, pollingFreq, configWindow)
					}
				}, configWindow)
		} else {
			saveConfiguration(workingDir, bucketName, pollingFreq, configWindow)
		}
	})

	cancelBtn := widget.NewButton("Cancel", func() {
		app.Quit()
	})

	buttonContainer := container.NewHBox(saveBtn, cancelBtn)

	fullContent := container.NewVBox(content, buttonContainer)
	scrollContainer := container.NewScroll(fullContent)

	configWindow.SetContent(scrollContainer)
	configWindow.Show()
}

func saveConfiguration(workingDir, bucketName string, pollingFreq int, configWindow fyne.Window) {
	// Ensure working directory ends with separator
	if !filepath.IsAbs(workingDir) {
		dialog.ShowError(fmt.Errorf("Working directory must be an absolute path"), configWindow)
		return
	}

	if workingDir[len(workingDir)-1] != filepath.Separator {
		workingDir += string(filepath.Separator)
	}

	config := &c.Configuration{
		AppName:                        "K-Drive",
		WorkingDirectory:               workingDir,
		CloudProvider:                  "aws s3",
		BucketName:                     bucketName,
		LocalDirectoryPollingFrequency: time.Duration(pollingFreq),
	}

	err := c.SaveConfig(config)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to save configuration: %v", err), configWindow)
		return
	}

	// Update global variables
	cloudProvider = config.CloudProvider
	workingDirectory = config.WorkingDirectory

	dialog.ShowInformation("Success", "Configuration saved successfully! The sync client will now start.", configWindow)
	configWindow.Close()

	// Signal that config is ready
	configReady <- true
}
func AddSyncInfoToFyneTable(syncInfo *s.SyncInfo) {
	log.Info("Adding %s to Fyne Table", syncInfo.Filename, syncInfo.Location)
	slice := []string{syncInfo.Filename,
		syncInfo.DateModified.Format("Mon Jan _2 15:04:05 2006"),
		syncInfo.Location.String(),
		syncInfo.SyncStatus.String()}
	tableData = append(tableData, slice)
}
func SetWorkingDirectory(workingDir string) {
	workingDirectory = workingDir
}
func makeHeader() *widget.Label {
	headerString := fmt.Sprintf(
		`This is a mini Sync Client written in Golang by @planetsp on GitHub.
I used Fyne to write the GUI, and the official %s SDKs to write the storage logic.
The local monitoring logic is taken from the watchdog library.
The local working directory is "%s".`, cloudProvider, workingDirectory)
	appInfo := widget.NewLabel(headerString)
	return appInfo
}

func makeFileList() *widget.Table {
	list := widget.NewTable(
		func() (int, int) {
			return len(tableData), len(tableData[0])
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Super duper wide string")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(GenerateLabelText(tableData[i.Row][i.Col]))
			// bold the first row
			if i.Row == 0 {
				o.(*widget.Label).TextStyle.Bold = true
			}
		})
	return list
}

func makeMenu(a fyne.App, w fyne.Window) *fyne.MainMenu {
	newItem := fyne.NewMenuItem("New", nil)
	checkedItem := fyne.NewMenuItem("Checked", nil)
	checkedItem.Checked = true
	disabledItem := fyne.NewMenuItem("Disabled", nil)
	disabledItem.Disabled = true
	otherItem := fyne.NewMenuItem("Other", nil)
	otherItem.ChildMenu = fyne.NewMenu("",
		fyne.NewMenuItem("Project", func() { fmt.Println("Menu New->Other->Project") }),
		fyne.NewMenuItem("Mail", func() { fmt.Println("Menu New->Other->Mail") }),
	)
	newItem.ChildMenu = fyne.NewMenu("",
		fyne.NewMenuItem("File", func() { fmt.Println("Menu New->File") }),
		fyne.NewMenuItem("Directory", func() { fmt.Println("Menu New->Directory") }),
		otherItem,
	)
	settingsItem := fyne.NewMenuItem("Settings", func() {
		w := a.NewWindow("Fyne Settings")
		w.SetContent(settings.NewSettings().LoadAppearanceScreen(w))
		w.Resize(fyne.NewSize(480, 480))
		w.Show()
	})

	configItem := fyne.NewMenuItem("Configuration", func() {
		showConfigDialog(a, w)
	})

	cutItem := fyne.NewMenuItem("Cut", func() {
		shortcutFocused(&fyne.ShortcutCut{
			Clipboard: w.Clipboard(),
		}, w)
	})
	copyItem := fyne.NewMenuItem("Copy", func() {
		shortcutFocused(&fyne.ShortcutCopy{
			Clipboard: w.Clipboard(),
		}, w)
	})
	pasteItem := fyne.NewMenuItem("Paste", func() {
		shortcutFocused(&fyne.ShortcutPaste{
			Clipboard: w.Clipboard(),
		}, w)
	})
	findItem := fyne.NewMenuItem("Find", func() { fmt.Println("Menu Find") })

	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("Documentation", func() {
			u, _ := url.Parse("https://developer.fyne.io")
			_ = a.OpenURL(u)
		}),
		fyne.NewMenuItem("Support", func() {
			u, _ := url.Parse("https://fyne.io/support/")
			_ = a.OpenURL(u)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Sponsor", func() {
			u, _ := url.Parse("https://fyne.io/sponsor/")
			_ = a.OpenURL(u)
		}))

	// a quit item will be appended to our first (File) menu
	file := fyne.NewMenu("File", newItem, checkedItem, disabledItem)
	if !fyne.CurrentDevice().IsMobile() {
		file.Items = append(file.Items, fyne.NewMenuItemSeparator(), configItem, settingsItem)
	}
	return fyne.NewMainMenu(
		file,
		fyne.NewMenu("Edit", cutItem, copyItem, pasteItem, fyne.NewMenuItemSeparator(), findItem),
		helpMenu,
	)
}
func shortcutFocused(s fyne.Shortcut, w fyne.Window) {
	if focused, ok := w.Canvas().Focused().(fyne.Shortcutable); ok {
		focused.TypedShortcut(s)
	}
}

func GenerateLabelText(originalStr string) string {
	label := originalStr
	if len(label) > 20 {
		label = fmt.Sprintf("%s...", originalStr[0:20])
	}
	return label
}
