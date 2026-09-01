package ui_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/bmartel/alacris-go/ui"
)

func TestButtonRenders(t *testing.T) {
	got := render(t, ui.Button(ui.ButtonProps{Variant: "tonal"}).Text("Save"))
	for _, want := range []string{`<ui-button`, `variant="tonal"`, `Save`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestPendingListsEveryTag(t *testing.T) {
	p := ui.Pending()
	if len(p.Tags) != 69 {
		t.Errorf("Pending has %d tags, want 69", len(p.Tags))
	}
	got := render(t, p)
	if !strings.Contains(got, "ui-button:not(:defined)") {
		t.Errorf("pending stylesheet missing ui-button: %s", got)
	}
}

func TestButtonVarsRefuseUnknownKeys(t *testing.T) {
	el := ui.Button(ui.ButtonProps{}).Apply(ui.ButtonVars, map[string]string{
		"--ui-button-colour": "red",
	})
	var b bytes.Buffer
	err := el.Render(templ.InitializeContext(context.Background()), &b)
	if err == nil {
		t.Fatalf("unknown var was accepted: %s", b.String())
	}
	if !strings.Contains(err.Error(), "not part of this theming contract") {
		t.Errorf("error = %v", err)
	}
}

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var b bytes.Buffer
	if err := c.Render(templ.InitializeContext(context.Background()), &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}
