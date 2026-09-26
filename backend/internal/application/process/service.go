package process

// Service — реализация live ведущих портов модуля process (AD-36):
// LiveService. Без зависимостей (NewService: выгрузка OpenAPI, тесты API)
// операции отвечают 501 api.not_implemented.
type Service = LiveService

// NewService создаёт реализацию live без зависимостей (операции — 501).
func NewService() *Service { return &LiveService{} }
