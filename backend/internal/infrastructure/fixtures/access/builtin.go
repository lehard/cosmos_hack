package access

// Встроенный мир заготовок: пакет world при инициализации регистрирует
// loader.Builtin — генератор строит в памяти те же сценарии, что лежат в
// scenarios/fixtures (AD-36). Импорт здесь, потому что адаптер access
// собирается в любом процессе api.
import _ "ant/internal/infrastructure/fixtures/world"
