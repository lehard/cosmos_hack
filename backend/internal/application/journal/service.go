package journal

// Service — реализация live ведущих портов модуля journal (AD-36): сценарии
// приложения над доменом, журналом и проекциями. В волне 1 — заглушка:
// все операции отвечают 501 (Unimplemented).
type Service struct {
	Unimplemented
}

// NewService создаёт реализацию live.
func NewService() *Service { return &Service{} }

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)
