package documents_test

import (
	"encoding/json"
	"os"
	"testing"

	"ant/internal/domain/access"
	dom "ant/internal/domain/documents"
)

// Эталонные документы позвоночника (AD-12: «make check содержит тест
// „отпечаток сервера = отпечаток агента“ на эталонных документах»):
// contracts/crypto/test-vectors/documents.v1.json — шаблон, канонический
// content, каноническая отрисовка, rendering_hash, doc_digest и QR. Сервер
// (Compose при черновике) и агент токена (пересчёт из content и шаблона —
// Rebuild) обязаны получить эталон байт в байт; агент токена (эпик 38) и
// demo-signer собираются из того же коммита и проверяют себя по этому файлу.
// Изменились шаблон или правила отрисовки — эталон пересобирается
// (ANT_UPDATE_DOC_VECTORS=1) вместе с новой версией шаблона или doc_format_version.
const goldenPath = "../../../../contracts/crypto/test-vectors/documents.v1.json"

type golden struct {
	Comment   string        `json:"comment"`
	Documents []goldenEntry `json:"documents"`
}

type goldenEntry struct {
	DocumentID       string          `json:"document_id"`
	TemplateRef      string          `json:"template_ref"`
	DocFormatVersion int             `json:"doc_format_version"`
	Content          json.RawMessage `json:"content"`
	ContentJCS       string          `json:"content_canonical_utf8"`
	RenderingUTF8    string          `json:"rendering_utf8"`
	RenderingHash    string          `json:"rendering_hash"`
	DocDigest        string          `json:"doc_digest"`
	QR               string          `json:"qr"`
}

// goldenDocs — эталонные документы: содержимое — как собирает свёртка.
func goldenDocs(t *testing.T) []goldenEntry {
	env := repoEnv(t)
	type spec struct {
		tpl, id, subject string
		ctx              dom.DocContext
		body             map[string]any
	}
	specs := []spec{
		{tpl: dom.TemplateTraveler, id: "TRV-ENT01:FL-0007", subject: "item:ENT01:FL-0007", body: map[string]any{
			"item": map[string]any{"item_id": "ENT01:FL-0007", "item_type_id": "FL-100.01.001", "item_revision": "Б", "order_id": "ZK-1C-0042",
				"lots": []string{"LOT-AMG6-17"}, "process_version_hash": "streebog256:" + zeros},
			"rows": []any{
				map[string]any{"no": 1, "step_key": "machining.turn", "operation": "Токарная обработка", "operation_run_id": "RUN-1", "executor": "O17",
					"date": "2026-09-28T07:00:00.000Z — 2026-09-28T07:42:00.000Z", "signature": []string{"O17 (подпись personal, ур. 1)"},
					"params": []string{"Оборудование: CNC-3", "Программа: УП-017 ред. 4", "Ø 120: 120.02 mm (within)"},
					"otk": "годен — INS-01, 2026-09-28T08:00:00.000Z", "otk_by": "INS-01", "completion": "completed", "remarks": []string{}},
				map[string]any{"no": 2, "step_key": "welding.weld", "operation": "Сварка шва W-1", "operation_run_id": "RUN-2", "executor": "W21",
					"date": "2026-09-28T09:00:00.000Z — 2026-09-28T09:20:00.000Z", "signature": []string{"W21 (подпись personal, ур. 1)"},
					"params": []string{"Оборудование: WELD-1", "Длительность: 20 min"}, "otk": "признак дефекта — камера, 2026-09-28T09:30:00.000Z",
					"otk_by": "", "completion": "completed", "remarks": []string{"Заявление о несоответствии НС-0192F000"}},
			},
			"rows_count": 2, "otk_count": 1, "nc_count": 1,
			"nonconformities": []any{map[string]any{"nc_id": "0192f000-0000-7000-8000-000000000001", "number": "НС-0192F000", "status": "confirmed",
				"disposition": "как есть", "concession": "CON-1"}},
			"final": map[string]any{"status": "в работе", "at": ""},
		}},
		{tpl: dom.TemplateNCStatement, id: "NCS-0192f000-0000-7000-8000-000000000001", subject: "nonconformity:0192f000-0000-7000-8000-000000000001", body: map[string]any{
			"item": map[string]any{"item_id": "ENT01:FL-0007", "item_type_id": "FL-100.01.001"},
			"nc": map[string]any{"nc_id": "0192f000-0000-7000-8000-000000000001", "number": "НС-0192F000", "found_at": "2026-09-28T09:30:00.000Z",
				"found_by": "контроль «camera» (machine)", "stage": "welding.weld (выполнение RUN-2)", "requirement": "КД ФЛ-100.00.000 п. 3",
				"fact": "признак дефекта «burn_through» в зоне Z-1", "severity": "major", "defect_type": "burn_through",
				"evidence": []string{"наблюдение OBS-1"}, "confirmed_by": "INS-01", "confirmed_at": "2026-09-28T09:40:00.000Z", "reason": "Прожог подтверждён"},
		}},
		{tpl: dom.TemplateNCDisposition, id: "ncd-0192f000-0000-7000-8000-000000000001", subject: "nonconformity:0192f000-0000-7000-8000-000000000001",
			ctx: dom.DocContext{Decision: "use_as_is"}, body: map[string]any{
				"item": map[string]any{"item_id": "ENT01:FL-0007"},
				"nc":   map[string]any{"nc_id": "0192f000-0000-7000-8000-000000000001", "number": "НС-0192F000", "requirement": "КД ФЛ-100.00.000 п. 3", "severity": "major"},
				"decision": map[string]any{"disposition": "use_as_is", "label": "как есть", "scrap_kind": "", "concession": "CON-1", "claim": "",
					"reason": "в допуске по разрешению", "author": "INS-01", "at": "2026-09-28T10:00:00.000Z"},
			}},
	}
	var out []goldenEntry
	for _, s := range specs {
		tp, ok := env.Templates.ByRef(s.tpl)
		if !ok {
			t.Fatalf("нет шаблона %s", s.tpl)
		}
		d := &dom.Doc{ID: s.id, TemplateRef: tp.Ref(), DocType: tp.DocType, Title: tp.Title, Subject: s.subject, Context: s.ctx}
		stages := access.RequiredApprovals(tp.Route, access.ApprovalContext{Decision: s.ctx.Decision}, access.Policy{})
		b, err := dom.Compose(tp, d, 1, s.body, stages)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, goldenEntry{DocumentID: s.id, TemplateRef: tp.Ref(), DocFormatVersion: dom.DocFormatVersion, Content: b.Content, ContentJCS: string(b.Content),
			RenderingUTF8: b.HTML, RenderingHash: b.RenderingHash, DocDigest: b.Digest, QR: dom.QR(s.id, b.Digest)})
	}
	return out
}

func TestGoldenDocuments(t *testing.T) {
	docs := goldenDocs(t)
	if os.Getenv("ANT_UPDATE_DOC_VECTORS") == "1" {
		g := golden{Comment: "Эталонные документы AD-12: сервер (свёртка) и агент токена обязаны получить rendering_hash и doc_digest байт в байт. " +
			"Пересборка: ANT_UPDATE_DOC_VECTORS=1 go test ./internal/application/documents -run TestGoldenDocuments", Documents: docs}
		b, _ := json.MarshalIndent(g, "", "  ")
		if err := os.WriteFile(goldenPath, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	var g golden
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatal(err)
	}
	if len(g.Documents) != len(docs) {
		t.Fatalf("эталонов %d, собрано %d", len(g.Documents), len(docs))
	}
	env := repoEnv(t)
	for i, want := range g.Documents {
		// Сервер: содержимое, собранное свёрткой (Compose).
		srv := docs[i]
		if srv.DocDigest != want.DocDigest || srv.RenderingHash != want.RenderingHash || srv.RenderingUTF8 != want.RenderingUTF8 {
			t.Errorf("%s: сервер %s ≠ эталон %s", want.DocumentID, srv.DocDigest, want.DocDigest)
		}
		// Агент токена: только content из эталона и шаблон того же коммита.
		tp, _ := env.Templates.ByRef(want.TemplateRef)
		agent, err := dom.Rebuild(tp, &dom.Version{Content: want.Content})
		if err != nil || agent.Digest != want.DocDigest || agent.HTML != want.RenderingUTF8 {
			t.Errorf("%s: агент %s ≠ эталон %s (%v)", want.DocumentID, agent.Digest, want.DocDigest, err)
		}
		if c, _ := dom.Canonical(want.Content); string(c) != want.ContentJCS {
			t.Errorf("%s: канонический content", want.DocumentID)
		}
		if dom.QR(want.DocumentID, agent.Digest) != want.QR {
			t.Errorf("%s: QR", want.DocumentID)
		}
		// Независимая проверка правила: H(отрисовка) и H(JCS({content, rendering_hash, template_ref, doc_format_version})).
		if dom.RenderingHash(want.RenderingUTF8) != want.RenderingHash {
			t.Errorf("%s: rendering_hash", want.DocumentID)
		}
		if d, err := dom.DocDigest(want.Content, want.RenderingHash, want.TemplateRef, want.DocFormatVersion); err != nil || d != want.DocDigest {
			t.Errorf("%s: doc_digest по правилу AD-12", want.DocumentID)
		}
	}
}
