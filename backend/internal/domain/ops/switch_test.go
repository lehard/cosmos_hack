package ops

import (
	"testing"
	"time"
)

// FR-157, AD-47: состояние по умолчанию профиля, последнее решение, гард
// «стенд в prod», неустановленная система, приём от выключенной системы.
func TestSwitchStates(t *testing.T) {
	inst := []Installed{{System: "onec", Stand: true}, {System: "visionqc", Stand: true, Real: true}}
	st := States("demo", inst, nil)
	get := func(ss []SwitchState, sys string) SwitchState {
		for _, s := range ss {
			if s.System == sys {
				return s
			}
		}
		t.Fatalf("нет системы %s", sys)
		return SwitchState{}
	}
	if s := get(st, "onec"); s.State != SwitchStand || !s.Default || !s.Installed {
		t.Fatalf("1С в demo без решений — стенд по умолчанию: %+v", s)
	}
	if s := get(st, "galaktika"); s.Installed || s.State != SwitchDisabled {
		t.Fatalf("Галактика не установлена: %+v", s)
	}
	if len(st) != len(KnownSystems) {
		t.Fatalf("экран показывает все известные системы: %d", len(st))
	}
	recs := []StateRecord{{Seq: 5, System: "onec", State: SwitchDisabled, At: time.Unix(5, 0)}, {Seq: 3, System: "onec", State: SwitchStand}}
	if s := get(States("demo", inst, recs), "onec"); s.State != SwitchDisabled || s.Default || s.Last.Seq != 5 {
		t.Fatalf("последнее решение побеждает: %+v", s)
	}
	// Решение о неустановленной системе не действует.
	if s := get(States("demo", inst, []StateRecord{{Seq: 1, System: "mes", State: SwitchStand}}), "mes"); s.State != SwitchDisabled {
		t.Fatalf("неустановленная: %+v", s)
	}
	// prod: по умолчанию — реальная, если задана; стенд из журнала не действует.
	prod := States("prod", inst, []StateRecord{{Seq: 1, System: "visionqc", State: SwitchStand}})
	if s := get(prod, "visionqc"); s.State != SwitchEnabled {
		t.Fatalf("prod — стенд не действует: %+v", s)
	}
	if s := get(prod, "onec"); s.State != SwitchDisabled {
		t.Fatalf("prod, 1С только со стендом — выключена: %+v", s)
	}
}

func TestCheckSet(t *testing.T) {
	cur := SwitchState{System: "visionqc", Installed: true, Stand: true, Real: true, State: SwitchEnabled}
	if r := CheckSet("prod", cur, SwitchStand); r == nil || r.Code != CodeStandForbidden {
		t.Fatalf("prod: стенд — отказ ops.stand_forbidden, получено %+v", r)
	}
	if r := CheckSet("demo", cur, SwitchStand); r != nil {
		t.Fatalf("demo: стенд разрешён: %+v", r)
	}
	if r := CheckSet("prod", cur, SwitchDisabled); r != nil {
		t.Fatalf("выключить можно всегда: %+v", r)
	}
	if r := CheckSet("demo", cur, SwitchEnabled); r == nil || r.Code != CodeUnchanged {
		t.Fatalf("то же состояние: %+v", r)
	}
	onec := SwitchState{System: "onec", Installed: true, Stand: true, State: SwitchStand}
	if r := CheckSet("demo", onec, SwitchEnabled); r == nil || r.Code != CodeModeUnavailable {
		t.Fatalf("реальная 1С не задана адресом: %+v", r)
	}
	if r := CheckSet("demo", SwitchState{System: "mes", State: SwitchDisabled}, SwitchStand); r == nil || r.Code != CodeNotInstalled {
		t.Fatalf("неустановленная: %+v", r)
	}
}

func TestSourceBlocked(t *testing.T) {
	states := []SwitchState{{System: "onec", Installed: true, State: SwitchDisabled}, {System: "mes", Installed: true, State: SwitchStand}}
	if b := SourceBlocked("erp.onec", nil, states); !b.Blocked {
		t.Fatal("шлюз выключенной 1С отвергается")
	}
	if b := SourceBlocked("mes.b2mml", nil, states); b.Blocked {
		t.Fatal("включённая MES принимается")
	}
	if b := SourceBlocked("edge-weld-1", map[string]string{"edge-weld-1": "подмена"}, states); !b.Blocked {
		t.Fatal("источник, отключённый ops.source.disabled, отвергается")
	}
	if b := SourceBlocked("edge-kt3", nil, states); b.Blocked {
		t.Fatal("источник не внешней системы не затрагивается")
	}
	if SystemOfSource("cad.kompas") != "kompas" || SystemOfSource("partner:ENT02") != "partner" {
		t.Fatal("источники систем")
	}
}
