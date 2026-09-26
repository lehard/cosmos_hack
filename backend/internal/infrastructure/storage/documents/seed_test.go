package documents_test

import (
	"bytes"
	"io/fs"
	"os"
	"testing"

	app "ant/internal/application/documents"
	dom "ant/internal/domain/documents"
	storage "ant/internal/infrastructure/storage/documents"
)

// Встроенная копия нормативного слоя совпадает с репозиторием, шаблоны
// позвоночника описаны полностью, политика читается.
func TestSeedMatchesRepo(t *testing.T) {
	repo := os.DirFS("../../../../..")
	for _, p := range []string{app.PathTemplates, app.PathPolicy} {
		want, err := fs.ReadFile(repo, p)
		if err != nil {
			t.Fatal(err)
		}
		got, err := fs.ReadFile(storage.Seed(), p)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: встроенная копия расходится с репозиторием — скопируйте файл в seed/", p)
		}
	}
	env, err := storage.SeedEnv(dom.VerificationDemo)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{dom.TemplateTraveler, dom.TemplateNCStatement, dom.TemplateNCDisposition, dom.TemplateProcessApproval, dom.TemplateDecisionRequest} {
		if tp, ok := env.Templates.ByRef(id); !ok || !tp.Complete() || len(tp.Route) == 0 {
			t.Errorf("шаблон %s не описан полностью", id)
		}
	}
	if p, ok := env.People.Find("HQC-01"); !ok || !p.HasRole("quality_inspector") || !p.HasAuthority("nc_disposition") {
		t.Errorf("политика: %+v", p)
	}
}
