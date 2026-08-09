package live

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

// templText is a templ.Component that writes s verbatim.
func templText(s string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}
