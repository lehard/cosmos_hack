package main

import (
	"net/http"
	"os"
	"path/filepath"

	"ant/internal/application/platform"
)

// dumpOpenAPI выгружает спецификацию HTTP API из операций Huma в файл (AD-20:
// Go-операции Huma → contracts/openapi.yaml → клиент orval). Спецификация не
// зависит от режима портов и конфигурации: выгрузка детерминирована.
func dumpOpenAPI(path string) error {
	a := buildAPI(http.NewServeMux(), apiOptions{mode: platform.ModeLive})
	b, err := a.OpenAPIYAML()
	if err != nil {
		return err
	}
	head := []byte("# СГЕНЕРИРОВАНО из операций Huma (backend/internal/infrastructure/transport) командой\n" +
		"# `ant -openapi` в make generate — руками не править (AD-20). Контракт фронтенда: клиент orval + Vue Query.\n")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(head, b...), 0o644)
}
