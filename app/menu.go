package app

import (
	"strings"
)

// Role is a standard menu action the OS already knows how to perform.
type Role int

const (
	RoleNone Role = iota
	RoleQuit
	RoleCut
	RoleCopy
	RolePaste
	RoleSelectAll
	RoleSeparator
)

// A Menu is a native menu bar.
type Menu struct {
	Items []MenuItem
}

// A MenuItem is one entry, a separator, or a submenu.
//
// Do runs on the UI thread. It may call SaveFile, patch a live session, or
// Close the window. It is ignored when Role is not RoleNone.
type MenuItem struct {
	Title string
	Keys  string // "CmdOrCtrl+E", "CmdOrCtrl+Shift+N"
	Role  Role
	Do    func(w *Window)
	Items []MenuItem
}

// DefaultMenu is Quit under File, plus the Edit menu a text field needs.
func DefaultMenu() *Menu {
	return &Menu{Items: []MenuItem{
		{Title: "File", Items: []MenuItem{
			{Title: "Quit", Keys: "CmdOrCtrl+Q", Role: RoleQuit},
		}},
		editMenu(),
	}}
}

// EditMenu is Cut/Copy/Paste/Select All, as one top-level item.
func EditMenu() MenuItem {
	return editMenu()
}

func editMenu() MenuItem {
	return MenuItem{Title: "Edit", Items: []MenuItem{
		{Title: "Cut", Keys: "CmdOrCtrl+X", Role: RoleCut},
		{Title: "Copy", Keys: "CmdOrCtrl+C", Role: RoleCopy},
		{Title: "Paste", Keys: "CmdOrCtrl+V", Role: RolePaste},
		{Role: RoleSeparator},
		{Title: "Select All", Keys: "CmdOrCtrl+A", Role: RoleSelectAll},
	}}
}

type keyStroke struct {
	key       string
	shift     bool
	alt       bool
	ctrlOrCmd bool
}

func parseKeys(s string) keyStroke {
	var k keyStroke
	for _, part := range strings.Split(s, "+") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "cmdorctrl", "cmd", "command", "ctrl", "control":
			k.ctrlOrCmd = true
		case "shift":
			k.shift = true
		case "alt", "option":
			k.alt = true
		default:
			k.key = part
		}
	}
	return k
}

func orTitle(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func roleTitle(r Role) string {
	switch r {
	case RoleQuit:
		return "Quit"
	case RoleCut:
		return "Cut"
	case RoleCopy:
		return "Copy"
	case RolePaste:
		return "Paste"
	case RoleSelectAll:
		return "Select All"
	}
	return ""
}
