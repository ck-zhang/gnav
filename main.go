package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Names []string `yaml:"workspace_names"`
}

var (
	configFile = filepath.Join(os.Getenv("HOME"), ".config", "gnav", "workspaces.yaml")
	cfg        = &Config{}
)

func loadConfig() error {
	b, err := ioutil.ReadFile(configFile)
	if os.IsNotExist(err) {
		cfg.Names = []string{"Workspace 1", "Workspace 2"}
		return saveConfig()
	}
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, cfg)
}

func saveConfig() error {
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configFile, data, 0644)
}

// -----------------------------------------------------------------------------
// Basic commands: dynamic, rename, create, switch
// -----------------------------------------------------------------------------

func getSystemWorkspaceCount() (int, error) {
	out, err := exec.Command("wmctrl", "-d").Output()
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return len(lines), nil
}

func getDynamic() (bool, error) {
	out, err := exec.Command("gsettings", "get",
		"org.gnome.mutter", "dynamic-workspaces").Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

func setDynamic(on bool) error {
	val := "false"
	if on {
		val = "true"
	}
	return exec.Command("gsettings", "set",
		"org.gnome.mutter", "dynamic-workspaces", val).Run()
}

func switchWorkspace(idx int) error {
	if idx < 1 {
		return errors.New("invalid workspace index")
	}
	cmd := exec.Command("wmctrl", "-s", strconv.Itoa(idx-1))
	return cmd.Run()
}

func renameLocal(index int, newName string) error {
	if index < 1 {
		return fmt.Errorf("invalid index: %d", index)
	}
	for len(cfg.Names) < index {
		cfg.Names = append(cfg.Names, fmt.Sprintf("Workspace %d", len(cfg.Names)+1))
	}
	cfg.Names[index-1] = newName
	return saveConfig()
}

func createWorkspaces(num int) error {
	if num < 1 {
		return errors.New("workspaces must be >= 1")
	}
	sc, err := getSystemWorkspaceCount()
	if err != nil {
		return err
	}
	if num > sc {
		_ = exec.Command("gsettings", "set",
			"org.gnome.desktop.wm.preferences", "num-workspaces",
			strconv.Itoa(num)).Run()
		_ = exec.Command("gsettings", "set",
			"org.gnome.mutter", "dynamic-workspaces", "false").Run()
	}
	for len(cfg.Names) < num {
		cfg.Names = append(cfg.Names, fmt.Sprintf("Workspace %d", len(cfg.Names)+1))
	}
	return saveConfig()
}

// -----------------------------------------------------------------------------
// Wofi integration
// -----------------------------------------------------------------------------

func wofiIntegration() error {
	sc, err := getSystemWorkspaceCount()
	if err != nil {
		return err
	}
	for i := 0; i < sc; i++ {
		var name string
		if i < len(cfg.Names) {
			name = cfg.Names[i]
		} else {
			name = fmt.Sprintf("Workspace %d", i+1)
		}
		fmt.Printf("%d: %s\n", i+1, name)
	}
	return nil
}

func parseWofiSelection() error {
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return errors.New("no input")
	}
	line := strings.TrimSpace(scanner.Text())
	if line == "" {
		return errors.New("empty input")
	}
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return errors.New("invalid format: 'idx: name'")
	}
	idx, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return err
	}
	return switchWorkspace(idx)
}

func wofiRun() error {
	sc, err := getSystemWorkspaceCount()
	if err != nil {
		return err
	}
	if sc < 1 {
		return errors.New("no system workspaces")
	}
	var buf bytes.Buffer
	for i := 0; i < sc; i++ {
		var nm string
		if i < len(cfg.Names) {
			nm = cfg.Names[i]
		} else {
			nm = fmt.Sprintf("Workspace %d", i+1)
		}
		fmt.Fprintf(&buf, "%d: %s\n", i+1, nm)
	}
	cmd := exec.Command("wofi", "--show", "dmenu", "-i", "--allow-images")
	cmd.Stdin = &buf
	out, err2 := cmd.Output()
	if err2 != nil {
		return fmt.Errorf("wofi canceled or error: %v", err2)
	}
	sel := strings.TrimSpace(string(out))
	if sel == "" {
		return errors.New("no selection from wofi")
	}
	parts := strings.SplitN(sel, ":", 2)
	if len(parts) < 2 {
		return errors.New("invalid selection format from wofi")
	}
	idx, e := strconv.Atoi(strings.TrimSpace(parts[0]))
	if e != nil {
		return e
	}
	return switchWorkspace(idx)
}

// -----------------------------------------------------------------------------
// TUI
// -----------------------------------------------------------------------------

func setTUIViewTheme() {
	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor("#1E1E2E")
	tview.Styles.ContrastBackgroundColor = tcell.GetColor("#313244")
	tview.Styles.MoreContrastBackgroundColor = tcell.GetColor("#45475A")
	tview.Styles.BorderColor = tcell.GetColor("#F5E0DC")
	tview.Styles.TitleColor = tcell.GetColor("#F5E0DC")
	tview.Styles.GraphicsColor = tcell.GetColor("#F5E0DC")
	tview.Styles.PrimaryTextColor = tcell.GetColor("#D9E0EE")
	tview.Styles.SecondaryTextColor = tcell.GetColor("#D9E0EE")
	tview.Styles.TertiaryTextColor = tcell.GetColor("#D9E0EE")
	tview.Styles.InverseTextColor = tcell.GetColor("#1E1E2E")
	tview.Styles.ContrastSecondaryTextColor = tcell.GetColor("#F5E0DC")
}

type TUI struct {
	app    *tview.Application
	layout *tview.Flex
	list   *tview.List
}

func runTUI() error {
	setTUIViewTheme()
	sc, _ := getSystemWorkspaceCount()
	app := tview.NewApplication()

	head := tview.NewTextView()
	head.SetText("GNAV TUI").SetTextAlign(tview.AlignCenter)

	footText := `[↑/↓ or j/k] Move  [Enter] Switch  [R]ename  [N]ew  [Z]Dynamic  [Q/Esc]Quit`
	foot := tview.NewTextView()
	foot.SetText(footText)

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Workspaces ")
	list.ShowSecondaryText(false)

	for i := 0; i < sc; i++ {
		var nm string
		if i < len(cfg.Names) {
			nm = cfg.Names[i]
		} else {
			nm = fmt.Sprintf("Workspace %d", i+1)
		}
		list.AddItem(fmt.Sprintf("(%d) %s", i+1, nm), "", 0, nil)
	}

	tui := &TUI{
		app:    app,
		layout: nil,
		list:   list,
	}

	reload := func() {
		_ = loadConfig()
		s, _ := getSystemWorkspaceCount()
		list.Clear()
		for i := 0; i < s; i++ {
			var nm string
			if i < len(cfg.Names) {
				nm = cfg.Names[i]
			} else {
				nm = fmt.Sprintf("Workspace %d", i+1)
			}
			list.AddItem(fmt.Sprintf("(%d) %s", i+1, nm), "", 0, nil)
		}
	}

	list.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		sCount, _ := getSystemWorkspaceCount()
		if index < sCount {
			switchWorkspace(index + 1)
		} else if index == sCount {
			switchWorkspace(index + 1)
		}
	})

	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Key() {
		case tcell.KeyEsc:
			app.Stop()
			return nil
		case tcell.KeyUp, tcell.KeyDown:
			return ev
		}
		switch ev.Rune() {
		case 'q', 'Q':
			app.Stop()
			return nil
		case 'j':
			n := (list.GetCurrentItem() + 1) % list.GetItemCount()
			list.SetCurrentItem(n)
			return nil
		case 'k':
			n := (list.GetCurrentItem() - 1 + list.GetItemCount()) % list.GetItemCount()
			list.SetCurrentItem(n)
			return nil
		case 'r', 'R':
			i := list.GetCurrentItem() + 1
			renameDialog(i, reload, tui)
			return nil
		case 'n', 'N':
			createDialog(reload, tui)
			return nil
		case 'z', 'Z':
			toggleDynamic(tui, reload)
			return nil
		}
		return ev
	})

	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	flex.AddItem(head, 1, 1, false)
	flex.AddItem(list, 0, 6, true)
	flex.AddItem(foot, 1, 1, false)

	tui.layout = flex
	app.SetRoot(flex, true).SetFocus(list)
	return app.Run()
}

func renameDialog(idx int, refresh func(), tui *TUI) {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(fmt.Sprintf("Rename Local #%d", idx))

	var cur string
	if idx-1 < len(cfg.Names) {
		cur = cfg.Names[idx-1]
	} else {
		cur = fmt.Sprintf("Workspace %d", idx)
	}

	form.AddInputField("Name", cur, 20, nil, nil)
	form.AddButton("OK", func() {
		newN := form.GetFormItemByLabel("Name").(*tview.InputField).GetText()
		if newN != "" {
			_ = renameLocal(idx, newN)
			refresh()
		}
		tui.app.SetRoot(tui.layout, true).SetFocus(tui.list)
	})
	form.AddButton("Cancel", func() {
		tui.app.SetRoot(tui.layout, true).SetFocus(tui.list)
	})
	tui.app.SetRoot(form, true).SetFocus(form)
}

func createDialog(refresh func(), tui *TUI) {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle("Create # of Workspaces")

	form.AddInputField("Count", "", 3, nil, nil)
	form.AddButton("OK", func() {
		c := form.GetFormItemByLabel("Count").(*tview.InputField).GetText()
		n, err := strconv.Atoi(c)
		if err == nil && n > 0 {
			_ = createWorkspaces(n)
			refresh()
		}
		tui.app.SetRoot(tui.layout, true).SetFocus(tui.list)
	})
	form.AddButton("Cancel", func() {
		tui.app.SetRoot(tui.layout, true).SetFocus(tui.list)
	})
	tui.app.SetRoot(form, true).SetFocus(form)
}

func toggleDynamic(tui *TUI, refresh func()) {
	cur, err := getDynamic()
	if err != nil {
		showModal(tui, fmt.Sprintf("Error: %v", err), "OK", nil)
		return
	}
	nv := !cur
	if e := setDynamic(nv); e != nil {
		showModal(tui, fmt.Sprintf("Error setting dynamic: %v", e), "OK", nil)
		return
	}
	refresh()

	msg := "Dynamic Workspaces = OFF"
	if nv {
		msg = "Dynamic Workspaces = ON"
	}
	showModal(tui, msg, "OK", nil)
}

func showModal(tui *TUI, msg, label string, done func()) {
	m := tview.NewModal()
	m.SetText(msg).AddButtons([]string{label})
	m.SetDoneFunc(func(_ int, _ string) {
		if done != nil {
			done()
		} else {
			tui.app.SetRoot(tui.layout, true).SetFocus(tui.list)
		}
	})
	tui.app.SetRoot(m, false).SetFocus(m)
}

// -----------------------------------------------------------------------------
// Main + cobra
// -----------------------------------------------------------------------------

func main() {
	_ = loadConfig()

	root := &cobra.Command{
		Use: "gnav",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTUI()
		},
	}

	root.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "Display workspace names",
		RunE: func(_ *cobra.Command, _ []string) error {
			sc, _ := getSystemWorkspaceCount()
			for i := 0; i < sc; i++ {
				var n string
				if i < len(cfg.Names) {
					n = cfg.Names[i]
				} else {
					n = fmt.Sprintf("Workspace %d", i+1)
				}
				fmt.Printf("[%d] %s\n", i+1, n)
			}
			return nil
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "rename <index> <newName>",
		Short: "Rename a workspace",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			i, e := strconv.Atoi(args[0])
			if e != nil {
				return e
			}
			newN := strings.Join(args[1:], " ")
			return renameLocal(i, newN)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "create <num>",
		Short: "Add or expand static workspaces",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			x, e := strconv.Atoi(args[0])
			if e != nil {
				return e
			}
			return createWorkspaces(x)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "switch <index>",
		Short: "Switch to workspace by index",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			i, e := strconv.Atoi(args[0])
			if e != nil {
				return e
			}
			return switchWorkspace(i)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "dynamic <on|off>",
		Short: "Enable/disable GNOME dynamic workspaces",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			switch strings.ToLower(args[0]) {
			case "on":
				return setDynamic(true)
			case "off":
				return setDynamic(false)
			default:
				return errors.New("usage: gnav dynamic on|off")
			}
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "wofi",
		Short: "Output workspace list for wofi",
		RunE: func(_ *cobra.Command, _ []string) error {
			return wofiIntegration()
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "wofi-switch",
		Short: "Switch workspace from wofi input",
		RunE: func(_ *cobra.Command, _ []string) error {
			return parseWofiSelection()
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "wofi-run",
		Short: "Interactive workspace selection with wofi",
		RunE: func(_ *cobra.Command, _ []string) error {
			return wofiRun()
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "interactive",
		Short: "Launch text-based UI",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTUI()
		},
	})

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
