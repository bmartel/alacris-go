package model

import (
	"reflect"
	"testing"
)

func TestCleanLabelsAllowsCustom(t *testing.T) {
	got := cleanLabels([]string{"Launch", "Design Review", "design-review", "???", ""})
	want := []string{"launch", "design-review"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cleanLabels = %v, want %v", got, want)
	}
}

func TestSlugLabel(t *testing.T) {
	if got := slugLabel("  Design Review!! "); got != "design-review" {
		t.Fatalf("slugLabel = %q", got)
	}
	if got := slugLabel("abcdefghijklmnopqrstuvwxyz"); got != "abcdefghijklmnopqrstuvwx" {
		t.Fatalf("slugLabel truncated = %q", got)
	}
}
