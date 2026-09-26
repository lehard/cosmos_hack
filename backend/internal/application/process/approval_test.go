package process_test

import (
	"context"
	"strings"
	"testing"

	"ant/internal/application/platform"
	app "ant/internal/application/process"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	dp "ant/internal/domain/process"
)

// fakeApprovals — лист утверждения в памяти вместо documents (эпик 28).
type fakeApprovals struct {
	requested map[string]string // документ → разница (comment)
	route     app.ApprovalRoute
}

func (f *fakeApprovals) Request(_ context.Context, v app.VersionRecord, decision, comment string, _ platform.CommandMeta) (string, error) {
	id := "DOC-" + v.ID
	f.requested[id] = decision + "\n" + comment
	return id, nil
}

func (f *fakeApprovals) Route(context.Context, string) (app.ApprovalRoute, error) { return f.route, nil }

// UJ-4 (FR-22…FR-24): технолог добавляет точку предъявления после сварки,
// отправляет на кворум (лист с читаемой разницей), кворум подписывает,
// версия вступает в силу; изделие, запущенное раньше, остаётся на своей.
func TestApprovalPathUJ4(t *testing.T) {
	w := newLiveWorld(t)
	ctx := context.Background()
	fa := &fakeApprovals{requested: map[string]string{}, route: app.ApprovalRoute{Need: 3}}
	w.svc.Approvals = fa
	old := "ent01:FL-0001"
	w.register(old, 0)
	w.sync()

	xml := strings.Replace(string(flangeXML(t)), `<ant:properties stepKey="welding.kt3_camera" stepKind="automated_inspection" inspectionPoint="KT-3" />`,
		`<ant:properties stepKey="welding.kt3_camera" stepKind="automated_inspection" inspectionPoint="KT-3" />
        <ant:presentationPoint authority="qc_acceptance" role="quality_inspector" waitLimitMinutes="60" />`, 1)
	rc, err := w.svc.DraftVersion(ctx, app.DraftVersion{Label: "v2", BaseVersionID: app.SeedVersionID, BpmnXML: xml})
	if err != nil {
		t.Fatal(err)
	}
	draftID := rc.EventIDs[0]
	if strings.HasPrefix(draftID, "draft-") == false {
		vl, _ := w.svc.Versions(ctx, "", platform.Moment{})
		for _, v := range vl.Items {
			if v.Status == dp.StatusDraft {
				draftID = v.VersionID
			}
		}
	}

	// Ввести черновик в действие без утверждения нельзя.
	if _, err := w.svc.ActivateVersion(ctx, draftID, app.ActivateVersion{}); err == nil {
		t.Fatal("черновик введён в действие без утверждения")
	}
	if _, err := w.svc.SubmitVersion(ctx, draftID, app.SubmitVersion{Note: "Точка предъявления после камеры шва"}); err != nil {
		t.Fatal(err)
	}
	doc := fa.requested["DOC-"+draftID]
	if !strings.Contains(doc, "Добавлена точка предъявления после «КТ-3 Камера на шов") || !strings.Contains(doc, "вместо v1") {
		t.Fatalf("лист утверждения без читаемой разницы: %s", doc)
	}
	v, _ := w.svc.Version(ctx, draftID, platform.Moment{})
	if v.Status != dp.StatusOnApproval || v.ApprovalDocumentID != "DOC-"+draftID || v.BaseVersionID != app.SeedVersionID {
		t.Fatalf("после отправки: %+v", v)
	}
	if w.count(catalog.NormativeVersionSubmitted) != 1 {
		t.Fatal("normative.version.submitted не записан")
	}

	// Маршрут не закрыт, подписей две — отказ «не хватает подписей кворума».
	fa.route = app.ApprovalRoute{Have: 2, Need: 3, Signatures: []dp.Signature{
		{Role: "technologist", Person: "TEC-01", Authority: dp.QuorumAuthority, Valid: true},
		{Role: "quality_inspector", Person: "INS-01", Authority: dp.QuorumAuthority, Valid: true}}}
	_, err = w.svc.ActivateVersion(ctx, draftID, app.ActivateVersion{})
	if pe, ok := platform.AsError(err); !ok || pe.Code != errcodes.ProcessQuorumIncomplete || !strings.Contains(pe.Detail, "production_manager") {
		t.Fatalf("неполный кворум: %v", err)
	}
	vl, _ := w.svc.Versions(ctx, "", platform.Moment{})
	for _, x := range vl.Items {
		if x.VersionID == draftID && (x.Quorum == nil || x.Quorum.Have != 2 || x.Quorum.Need != 3) {
			t.Fatalf("прогресс кворума на утверждении: %+v", x.Quorum)
		}
	}

	// Кворум подписал, маршрут закрыт — версия вступает в силу, прежняя выведена.
	closed := "0190a000-0000-7000-8000-00000000c105"
	fa.route.Signatures = append(fa.route.Signatures, dp.Signature{Role: "production_manager", Person: "PM-01", Authority: dp.QuorumAuthority, Valid: true})
	fa.route.Have, fa.route.Closed, fa.route.RouteClosedEventID = 3, true, closed
	if _, err := w.svc.ActivateVersion(ctx, draftID, app.ActivateVersion{RouteClosedEventID: closed}); err != nil {
		t.Fatal(err)
	}
	vl, _ = w.svc.Versions(ctx, "", platform.Moment{})
	st := map[string]string{}
	for _, x := range vl.Items {
		st[x.VersionID] = x.Status
	}
	if st[draftID] != dp.StatusActive || st[app.SeedVersionID] != dp.StatusRetired {
		t.Fatalf("статусы после ввода в действие: %v", st)
	}
	if w.count(catalog.NormativeVersionActivated) != 1 {
		t.Fatal("normative.version.activated не записан")
	}

	// Карта показывает новую действующую версию; изделие, запущенное раньше, — на своей.
	lm, err := w.svc.LiveMap(ctx, app.LiveMapQuery{}, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if lm.ProcessVersion.ProcessVersionID != draftID || len(lm.Items) != 1 || lm.Items[0].ProcessVersionID != app.SeedVersionID {
		t.Fatalf("карта после ввода: %+v, изделия %+v", lm.ProcessVersion, lm.Items)
	}
	// Изделие старой версии продолжает исполняться: версия выведена, но утверждена (генезис).
	w.fact(old, catalog.DecisionPresentationResolved, 1, map[string]any{"step_key": "incoming.zt1_lot_acceptance", "closing_point": "ZT-1",
		"resolution": "accept", "presentation_no": 1, "method_event_ids": []string{}})
	w.sync()
	lm, _ = w.svc.LiveMap(ctx, app.LiveMapQuery{ProcessVersionID: app.SeedVersionID}, platform.Moment{})
	if len(lm.Items) != 1 || lm.Items[0].StepKey == "incoming.lot_registration" {
		t.Fatalf("изделие прежней версии не продвинулось: %+v", lm.Items)
	}

	// Вывести действующую — только действующую.
	if _, err := w.svc.RetireVersion(ctx, app.SeedVersionID, app.RetireVersion{ReasonText: "повтор"}); err == nil {
		t.Fatal("выведенная версия выведена повторно")
	}
	if _, err := w.svc.RetireVersion(ctx, draftID, app.RetireVersion{ReasonText: "проверка"}); err != nil {
		t.Fatal(err)
	}
	if w.count(catalog.NormativeVersionRetired) != 1 {
		t.Fatal("normative.version.retired не записан")
	}
}

func TestDiffText(t *testing.T) {
	got := app.DiffText([]app.ProcessDiffEntry{{Kind: "presentationPointAdded", Step: "Сварка"}, {Kind: "propertyChanged", Element: "Сварка", Property: "reworkLimit", From: 2, To: nil}})
	if got != "Добавлена точка предъявления после «Сварка»; «Сварка»: reworkLimit 2 → —" {
		t.Fatal(got)
	}
	if app.DiffText(nil) == "" {
		t.Fatal("пустая разница без текста")
	}
}

// blankXML — пустой процесс «старт → контроль → конец», как у «Создать процесс» в
// интерфейсе (frontend/src/widgets/process-registry/model/bpmn-file.ts).
const blankXML = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" xmlns:ant="urn:ant:bpmn-ext:1" id="Definitions_Process_New" targetNamespace="urn:ant:process:Process_New" expressionLanguage="urn:ant:expr:1">
  <bpmn:process id="Process_New" name="Корпус датчика" isExecutable="true">
    <bpmn:startEvent id="Start" name="Начало">
      <bpmn:extensionElements>
        <ant:properties stepKey="start" />
      </bpmn:extensionElements>
      <bpmn:outgoing>Flow_Start_Check</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="Check" name="Контроль ОТК">
      <bpmn:documentation>Контроль человеком перед выпуском: без него путь к конечному событию не проходит проверку (FR-13).</bpmn:documentation>
      <bpmn:extensionElements>
        <ant:properties stepKey="inspection.final" stepKind="human_inspection" />
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_Start_Check</bpmn:incoming>
      <bpmn:outgoing>Flow_Check_End</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="End" name="Конец">
      <bpmn:extensionElements>
        <ant:properties stepKey="end" />
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_Check_End</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_Start_Check" sourceRef="Start" targetRef="Check" />
    <bpmn:sequenceFlow id="Flow_Check_End" sourceRef="Check" targetRef="End" />
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_Process_New">
    <bpmndi:BPMNPlane id="BPMNPlane_Process_New" bpmnElement="Process_New">
      <bpmndi:BPMNShape id="Start_di" bpmnElement="Start">
        <dc:Bounds x="180" y="160" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Check_di" bpmnElement="Check">
        <dc:Bounds x="280" y="138" width="120" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="End_di" bpmnElement="End">
        <dc:Bounds x="480" y="160" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNEdge id="Flow_Start_Check_di" bpmnElement="Flow_Start_Check">
        <di:waypoint x="216" y="178" />
        <di:waypoint x="280" y="178" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_Check_End_di" bpmnElement="Flow_Check_End">
        <di:waypoint x="400" y="178" />
        <di:waypoint x="480" y="178" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>
`

// Новый процесс (эпик 39): черновик с новым главным bpmn:process — новый
// процесс в списке; первая версия проходит тот же кворум.
func TestNewProcessFromBlank(t *testing.T) {
	w := newLiveWorld(t)
	ctx := context.Background()
	fa := &fakeApprovals{requested: map[string]string{}}
	w.svc.Approvals = fa
	if _, err := w.svc.DraftVersion(ctx, app.DraftVersion{Label: "v1", BpmnXML: blankXML}); err != nil {
		t.Fatal(err)
	}
	pl, _ := w.svc.Processes(ctx, platform.Moment{})
	if len(pl.Items) != 2 || pl.Items[1].ProcessID != "Process_New" || pl.Items[1].Name != "Корпус датчика" || pl.Items[1].Status != "draft" || pl.Items[1].ActiveVersion != nil {
		t.Fatalf("новый процесс в списке: %+v", pl.Items)
	}
	vl, err := w.svc.Versions(ctx, "Process_New", platform.Moment{})
	if err != nil || len(vl.Items) != 1 {
		t.Fatalf("версии нового процесса: %+v %v", vl, err)
	}
	id := vl.Items[0].VersionID
	if _, err := w.svc.SubmitVersion(ctx, id, app.SubmitVersion{}); err != nil {
		t.Fatal(err)
	}
	if doc := fa.requested["DOC-"+id]; !strings.Contains(doc, "первую версию v1 нового процесса «Корпус датчика»") {
		t.Fatalf("лист нового процесса: %s", doc)
	}
	fa.route = app.ApprovalRoute{Closed: true, RouteClosedEventID: "0190a000-0000-7000-8000-00000000c106", Have: 3, Need: 3, Signatures: []dp.Signature{
		{Role: "technologist", Authority: dp.QuorumAuthority, Valid: true}, {Role: "quality_inspector", Authority: dp.QuorumAuthority, Valid: true},
		{Role: "production_manager", Authority: dp.QuorumAuthority, Valid: true}}}
	if _, err := w.svc.ActivateVersion(ctx, id, app.ActivateVersion{}); err != nil {
		t.Fatal(err)
	}
	pl, _ = w.svc.Processes(ctx, platform.Moment{})
	if pl.Items[1].Status != "active" || pl.Items[1].ActiveVersion == nil || pl.Items[1].ActiveVersion.VersionID != id || !pl.Items[0].IsDefault {
		t.Fatalf("после ввода: %+v", pl.Items)
	}
	// Основной процесс не сменился: карта без параметра — фланец.
	lm, _ := w.svc.LiveMap(ctx, app.LiveMapQuery{}, platform.Moment{})
	if lm.ProcessID != "Process_Flange" {
		t.Fatalf("основной процесс сменился: %s", lm.ProcessID)
	}
	// Файл без наших расширений — отказ с кодом и id элемента (FR-13).
	bad := strings.Replace(blankXML, `<ant:properties stepKey="end" />`, ``, 1)
	_, err = w.svc.DraftVersion(ctx, app.DraftVersion{Label: "v2", BpmnXML: bad})
	if pe, ok := platform.AsError(err); !ok || pe.Code != errcodes.ProcessStepKeyMissing || pe.Params["element"] != "End" {
		t.Fatalf("файл без step_key: %v", err)
	}
}
