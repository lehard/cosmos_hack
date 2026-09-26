package platform

import (
	"fmt"
	"regexp"
	"strings"

	"ant/internal/contracts/catalog"
)

// Class — класс операции для x-ant-action (AD-27, AD-40): read — чтение,
// record — команда без последствий на осях и без разрешения, остальные — как в
// каталоге типов записей. make check падает, если у операции нет класса.
type Class string

const (
	ClassRead         Class = "read"
	ClassRecord       Class = "record"
	ClassProtective   Class = "protective"
	ClassPermissive   Class = "permissive"
	ClassIrreversible Class = "irreversible"
)

// Classes — допустимые классы операций.
var Classes = []Class{ClassRead, ClassRecord, ClassProtective, ClassPermissive, ClassIrreversible}

// Action — описание операции API (x-ant-action, AD-40): id — он же OperationID
// Huma, ключ Casbin и действие @casl; класс, критичность и группа CA (AD-28),
// модуль-владелец (одна операция — одна реализация команды, AD-39), обязательные
// гарды, типы записей, которые операция эмитит, и минимальный уровень подписи.
type Action struct {
	ID       string
	Class    Class
	Critical bool
	// CAGroup — группа критического действия (AD-28); только у Critical.
	CAGroup string
	// Owner — модуль-владелец операции; равен первому сегменту ID.
	Owner string
	// Subject — вид объекта (EntityKind) для прав @casl и допустимых действий; пусто — «all».
	Subject string
	// Guards — имена обязательных доменных гардов модуля-владельца (AD-39).
	Guards []string
	// Emits — типы записей журнала, которые пишет операция; их эмитент по каталогу — Owner.
	Emits []catalog.Type
	// SignatureLevel — минимальный уровень подписи 0–3 (AD-13); 0 — без подписи.
	SignatureLevel int
	// Anonymous — операция доступна без сеанса (только вход: демо-персоны, открыть сеанс).
	Anonymous bool
}

var actionID = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$`)

// Validate проверяет описание операции: формат id, класс, владельца, эмитируемые
// типы (AD-40: модуль не эмитит чужой тип; AD-27: у операции есть класс).
func (a Action) Validate() error {
	if !actionID.MatchString(a.ID) {
		return fmt.Errorf("x-ant-action %q: id не вида ‹модуль›.‹объект›.‹действие›", a.ID)
	}
	if a.Class == "" {
		return fmt.Errorf("x-ant-action %s: у операции нет класса (AD-27)", a.ID)
	}
	valid := false
	for _, c := range Classes {
		valid = valid || c == a.Class
	}
	if !valid {
		return fmt.Errorf("x-ant-action %s: неизвестный класс %q", a.ID, a.Class)
	}
	if a.Owner == "" || !strings.HasPrefix(a.ID, a.Owner+".") {
		return fmt.Errorf("x-ant-action %s: модуль-владелец %q не совпадает с первым сегментом id", a.ID, a.Owner)
	}
	if a.Class == ClassRead && (len(a.Emits) > 0 || a.Critical) {
		return fmt.Errorf("x-ant-action %s: чтение не эмитит записей и не критично", a.ID)
	}
	if a.Critical && a.CAGroup == "" {
		return fmt.Errorf("x-ant-action %s: у критической операции нет группы CA (AD-28)", a.ID)
	}
	for _, t := range a.Emits {
		info, ok := catalog.Lookup(t)
		if !ok {
			return fmt.Errorf("x-ant-action %s: тип %q не из каталога", a.ID, t)
		}
		if info.Emitter != a.Owner {
			return fmt.Errorf("x-ant-action %s: модуль %s эмитит чужой тип %s (эмитент — %s, AD-40)", a.ID, a.Owner, t, info.Emitter)
		}
	}
	if a.SignatureLevel < 0 || a.SignatureLevel > 3 {
		return fmt.Errorf("x-ant-action %s: уровень подписи %d вне 0–3", a.ID, a.SignatureLevel)
	}
	return nil
}

// IsCommand — операция меняет состояние (всё, кроме чтения).
func (a Action) IsCommand() bool { return a.Class != ClassRead }

// ObjectRef — объект операции для прав и допустимых действий (AD-15): вид
// (EntityKind) и id; пустой — операция над «всем» (списки, столы).
type ObjectRef struct {
	Kind string
	ID   string
}
