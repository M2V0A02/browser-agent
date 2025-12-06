// ports/browser_core.go
package ports

import (
	"browser-agent/internal/infrastructure/browser/rodwrapper"
)

type BrowserCore interface {
	Page() (*rodwrapper.Page, error)

	Close()
}
