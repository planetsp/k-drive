package ui

import (
	"fmt"
	"net/url"

	log "github.com/planetsp/k-drive/pkg/logging"

	c "github.com/planetsp/k-drive/pkg/config"
	s "github.com/planetsp/k-drive/pkg/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/cmd/fyne_settings/settings"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var tableData = [][]string{
	{"Filename", "Date Modified", "Location", "Status"}}
var cloudProvider = c.GetConfig().CloudProvider
var workingDirectory = c.GetConfig().WorkingDirectory

func RunUI() {
	log.Info("Starting ui")
	myApp := app.New()

	myWindow := myApp.NewWindow(c.GetConfig().AppName)
	myWindow.Resize(fyne.NewSize(900, 400))
	myWindow.SetMainMenu(makeMenu(myApp, myWindow))

	grid := container.New(layout.NewGridLayout(1), makeHeader(), makeFileList(), container.NewMax())
	myWindow.SetContent(grid)
	myWindow.ShowAndRun()
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
		file.Items = append(file.Items, fyne.NewMenuItemSeparator(), settingsItem)
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
