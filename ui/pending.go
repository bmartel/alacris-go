package ui

import alacris "github.com/bmartel/alacris-go"

// Pending is the stylesheet that hides every Alacris UI element until it is
// defined. Put it in <head> next to alacris.Scripts.
func Pending() alacris.Pending {
	return alacris.Pending{Tags: Tags}
}
