package quality

import (
	"testing"

	"ant/internal/contracts/statuses"
)

// Порт quality → nonconformity (эпик 21) и notifications: заготовка
// потребителя. Поздний модуль читает Upstream.Quality.Requests и строит свои
// записи; здесь — что он получит и что ключи запросов стабильны при
// пересвёртке (слот его реакции — по Request.Key, AD-3).
func TestRequestsPortForLateModules(t *testing.T) {
	var b builder
	b.run("RUN-W1", "welding.weld")
	b.xray(OutcomeDefect, defect("W-LOF", "W-1.U3", "", "critical", 0))
	b.xray(OutcomeDefect, defect("X-ODD", "W-1.U7", "", "minor", 0))
	s, _ := fold(testEnv(), b.out)

	type want struct{ kind, task, role string }
	got := map[want]int{}
	for _, r := range s.Requests {
		got[want{r.Kind, r.TaskKind, r.RoleID}]++
		if r.Key == "" || len(r.Causes) == 0 {
			t.Fatalf("запрос без ключа или оснований: %+v", r)
		}
		if r.Kind == RequestContain && r.SignalID == s.Signals[0].SignalID && r.Containment != statuses.ContainmentItemHold {
			t.Fatalf("несплавление — блок: %+v", r)
		}
	}
	for _, w := range []want{
		{RequestDraftNC, "", ""},                           // черновик несоответствия по несплавлению
		{RequestContain, "", ""},                           // блок изделия
		{RequestTask, "isolate_move", "site_foreman"},      // переместить в изолятор — мастер
		{RequestTask, "decision_required", "technologist"}, // вопрос технологу: вид без требования КД
	} {
		if got[w] == 0 {
			t.Fatalf("нет запроса %+v среди %+v", w, s.Requests)
		}
	}
	if got[want{RequestDraftNC, "", ""}] != 1 {
		t.Fatalf("черновик только по несплавлению (вопрос технологу — не брак): %+v", s.Requests)
	}
	again, _ := fold(testEnv(), b.out)
	for i := range s.Requests {
		if s.Requests[i].Key != again.Requests[i].Key {
			t.Fatalf("ключ запроса не стабилен: %s ≠ %s", s.Requests[i].Key, again.Requests[i].Key)
		}
	}
}
