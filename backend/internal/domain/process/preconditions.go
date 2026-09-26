package process

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"ant/internal/contracts/errcodes"
)

// Предусловия операции (FR-17) и связанные ограничения: лимит доработок по
// зоне (FR-18), скрытые работы (FR-20), вмешательство (FR-21), статусы со
// сроком и окна между шагами (FR-16). Одна функция проверки для гарда
// команды (AD-39) и для свёртки внешнего факта: команду гард отклоняет, факт
// принимается с реакцией operation.precondition.failed (AD-30).

// Reference — срез справочников на occurred_at (AD-31), нужный
// предусловиям: квалификации исполнителей и поверка оборудования. Пусто —
// справочник не подключён (эпики 19, 26): условие «не проверено», а не
// «выполнено» и не «нарушено».
type Reference struct {
	Qualifications map[string][]Validity `json:"qualifications,omitempty"`
	Equipment      map[string][]Validity `json:"equipment,omitempty"`
}

// Validity — запись справочника с интервалом действия [From, To).
type Validity struct {
	Ref  string     `json:"ref"`
	From time.Time  `json:"from"`
	To   *time.Time `json:"to,omitempty"`
}

func (v Validity) at(t time.Time) bool {
	return !t.Before(v.From) && (v.To == nil || t.Before(*v.To))
}

// failure — невыполненное предусловие: вид (перечисление
// operation.precondition.failed), режим, объект и код отказа гарда.
type failure struct {
	Kind    string
	Mode    string
	Subject string
	Detail  string
	Code    errcodes.Code
	Params  []string
}

// ParseWindowRef — ref окна «‹шаг›<=‹длительность ISO 8601›» (например,
// edge_prep<=PT8H): окно от завершения шага до начала этой операции или срок
// статуса, установленного шагом (FR-16, FR-17).
func ParseWindowRef(ref string) (string, time.Duration, error) {
	step, dur, ok := strings.Cut(ref, "<=")
	if !ok || strings.TrimSpace(step) == "" {
		return "", 0, fmt.Errorf("ожидалось ‹шаг›<=‹длительность ISO 8601›")
	}
	d, err := ParseISODuration(strings.TrimSpace(dur))
	return strings.TrimSpace(step), d, err
}

// stepMatches — ключ шага совпадает с ref целиком или последним сегментом(ами).
func stepMatches(stepKey, ref string) bool {
	return stepKey == ref || strings.HasSuffix(stepKey, "."+ref)
}

// lastCompleted — когда последний раз завершён шаг ref (ok=false — не завершался).
func (s State) lastCompleted(ref string) (time.Time, bool) {
	for i := len(s.History) - 1; i >= 0; i-- {
		h := s.History[i]
		if stepMatches(h.StepKey, ref) && h.Left != nil && h.Via == "completed" {
			return *h.Left, true
		}
	}
	return time.Time{}, false
}

// zonesFor — зоны доработки шага: зоны последнего найденного дефекта среди
// зон шага; неизвестны — все зоны шага (ограничивать безопаснее, AD-27);
// у шага нет зон — изделие целиком («*»).
func (s State) zonesFor(n *Node) []string {
	var out []string
	for _, z := range s.DefectZones {
		if contains(n.Zones, z) {
			out = append(out, z)
		}
	}
	if len(out) == 0 {
		out = append(out, n.Zones...)
	}
	if len(out) == 0 {
		out = []string{"*"}
	}
	return out
}

// runsAt — выполнения шага (в маршруте); zone ≠ "" — только охватившие зону.
func (s State) runsAt(node, zone string) int {
	c := 0
	for _, k := range sortedKeys(s.Runs) {
		r := s.Runs[k]
		if r.Node != node || r.Detached {
			continue
		}
		if zone != "" && !contains(r.Zones, zone) && !contains(r.Zones, "*") {
			continue
		}
		c++
	}
	return c
}

// ReworkUse — расход лимита доработок: зона, сколько доработок будет с
// этим выполнением, лимит с учётом разрешений.
type ReworkUse struct {
	Zone  string `json:"zone"`
	Used  int    `json:"used"`
	Limit int    `json:"limit"`
}

// reworkUse — доработки шага n для выполнения, которое включает runs
// выполнений (своё — последнее). extraZones — зоны этого выполнения.
func (s State) reworkUse(n *Node, zones []string, prospective bool) []ReworkUse {
	limit, ok := n.ReworkLimit()
	if !ok {
		return nil
	}
	add := 0
	if prospective {
		add = 1
	}
	var out []ReworkUse
	if n.Props.ReworkLimitScope == reworkScopeZone {
		for _, z := range zones {
			runs := s.runsAt(n.ID, z) + add
			out = append(out, ReworkUse{Zone: z, Used: runs - 1, Limit: limit + s.Waivers[z] + s.Waivers[n.StepKey()]})
		}
		return out
	}
	runs := s.runsAt(n.ID, "") + add
	return []ReworkUse{{Zone: n.StepKey(), Used: runs - 1, Limit: limit + s.Waivers[n.StepKey()] + s.Waivers["*"]}}
}

// checks — невыполненные предусловия операции на шаге n в момент at
// исполнителем operator на оборудовании equipment (FR-17, FR-18, FR-20, FR-21).
// prospective — проверка до начала (гард); иначе выполнение уже учтено.
func (s State) checks(env Env, n *Node, operator, equipment string, zones []string, at time.Time, prospective bool) []failure {
	var out []failure
	for _, pc := range n.Preconditions {
		mode := pc.Mode
		switch pc.Kind {
		case "qualification":
			if env.Reference.Qualifications == nil || operator == "" {
				continue
			}
			if !anyValid(env.Reference.Qualifications[operator], pc.Ref, at) {
				out = append(out, failure{Kind: pc.Kind, Mode: mode, Subject: operator, Code: errcodes.ProcessQualificationExpired,
					Detail: "квалификация " + pc.Ref + " исполнителя " + operator + " не действует на " + at.UTC().Format("2006-01-02"),
					Params: []string{"performer", operator, "date", at.UTC().Format("2006-01-02")}})
			}
		case "equipment_verification":
			if env.Reference.Equipment == nil || equipment == "" {
				continue
			}
			if !anyValid(env.Reference.Equipment[equipment], "", at) {
				out = append(out, failure{Kind: pc.Kind, Mode: mode, Subject: equipment, Code: errcodes.ProcessPreconditionFailed,
					Detail: "поверка оборудования " + equipment + " не действует", Params: []string{"condition", "поверка оборудования " + equipment}})
			}
		case "time_window", "status_expired":
			ref, dur, err := ParseWindowRef(pc.Ref)
			if err != nil {
				continue
			}
			done, ok := s.lastCompleted(ref)
			if !ok || at.Sub(done) <= dur {
				continue
			}
			code, params := errcodes.ProcessStatusExpired, []string{"status", ref}
			if pc.Kind == "time_window" {
				code, params = errcodes.ProcessPreconditionFailed, []string{"condition", "окно " + pc.Ref}
			}
			out = append(out, failure{Kind: pc.Kind, Mode: mode, Subject: "", Code: code, Params: params,
				Detail: fmt.Sprintf("от завершения %s прошло %s — больше %s", ref, at.Sub(done), dur)})
		case "zone_check":
			if f, bad := s.zoneCheck(pc.Ref, mode); bad {
				out = append(out, f)
			}
		case "item_blocked":
			if s.Containment == "block" {
				out = append(out, failure{Kind: pc.Kind, Mode: mode, Code: errcodes.NonconformityItemBlocked, Detail: "изделие заблокировано"})
			}
		case "open_intervention":
			if iv := s.OpenInterventions(); len(iv) > 0 {
				out = append(out, failure{Kind: pc.Kind, Mode: mode, Subject: iv[0].ID, Code: errcodes.NonconformityInterventionOpen,
					Detail: "открыта запись вмешательства " + iv[0].ID})
			}
		case "lot_accepted":
			if s.lotRejected() {
				out = append(out, failure{Kind: "item_blocked", Mode: mode, Code: errcodes.ProcessPreconditionFailed,
					Detail: "партия не принята на входном контроле", Params: []string{"condition", "партия принята"}})
			}
		}
	}
	// FR-20: скрытые работы — до операции, закрывающей доступ к зоне, нужна
	// завершённая проверка зоны.
	for _, z := range n.ClosesZones() {
		if f, bad := s.zoneCheck(z, preconditionBlock); bad && !hasZoneFailure(out, z) {
			out = append(out, f)
		}
	}
	// FR-18: лимит доработок по зоне; сверх — только по разрешению (decision.rework_limit.waived).
	for _, u := range s.reworkUse(n, zones, prospective) {
		if u.Used > u.Limit {
			out = append(out, failure{Kind: "rework_limit", Mode: preconditionBlock, Subject: objectRef(u.Zone), Code: errcodes.ProcessReworkLimitExceeded,
				Detail: fmt.Sprintf("доработка %d при лимите %d (зона %s) — нужно разрешение уполномоченного", u.Used, u.Limit, u.Zone),
				Params: []string{"zone", u.Zone, "used", strconv.Itoa(u.Used), "limit", strconv.Itoa(u.Limit)}})
		}
	}
	return out
}

func objectRef(s string) string {
	if s == "*" {
		return ""
	}
	return s
}

func hasZoneFailure(fs []failure, zone string) bool {
	for _, f := range fs {
		if f.Kind == "zone_check" && f.Subject == zone {
			return true
		}
	}
	return false
}

func (s State) zoneCheck(zone, mode string) (failure, bool) {
	z, ok := s.Zones[zone]
	if ok && z.CheckedAt != nil && !z.Outdated && !z.Defect {
		return failure{}, false
	}
	why := "нет завершённой проверки"
	switch {
	case ok && z.Outdated:
		why = "результаты устарели после вмешательства"
	case ok && z.Defect:
		why = "последняя проверка нашла признаки дефекта"
	}
	return failure{Kind: "zone_check", Mode: mode, Subject: zone, Code: errcodes.ProcessZoneCheckRequired,
		Detail: "зона " + zone + ": " + why, Params: []string{"zone", zone}}, true
}

// lotRejected — последнее решение входного контроля — не принять.
func (s State) lotRejected() bool {
	for _, k := range sortedKeys(s.Gates) {
		g := s.Gates[k]
		if strings.HasPrefix(k, "incoming.") && (g.Last == ResolutionReject || g.Last == ResolutionInsufficientData) {
			return true
		}
	}
	return false
}

func anyValid(list []Validity, ref string, at time.Time) bool {
	for _, v := range list {
		if (ref == "" || v.Ref == ref) && v.at(at) {
			return true
		}
	}
	return false
}
