package simulation

import (
	"os"

	app "ant/internal/application/simulation"
)

// Files — определения сценариев из каталога scenarios/ на диске (том только
// для чтения в контейнере): разбор — app.FSDefinitions над os.DirFS.
type Files struct {
	app.FSDefinitions
	// Root — каталог scenarios (в нём definitions/ и expected/).
	Root string
}

// NewFiles — определения из каталога root.
func NewFiles(root string) *Files {
	return &Files{FSDefinitions: app.FSDefinitions{FS: os.DirFS(root)}, Root: root}
}

// ErrNotFound — определения нет.
var ErrNotFound = app.ErrNoDefinition
