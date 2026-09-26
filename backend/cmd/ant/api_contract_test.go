package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"ant/internal/application/platform"
)

// Волна 1 (эпик 02, «Что ожидаем в итоге»): все операции контракта объявлены,
// у каждой x-ant-action, и любой корректный по схеме вызов отвечает 501
// problem+json с кодом api.not_implemented — кроме операций, которые общий
// декоратор (Gate) вычисляет сам. Запросы строятся по схемам спецификации.
func TestEveryOperationAnswers501(t *testing.T) {
	mux := http.NewServeMux()
	a := buildAPI(mux, apiOptions{mode: platform.ModeFixtures})
	oapi := a.Huma().OpenAPI()
	reg := oapi.Components.Schemas
	gateOps := map[string]bool{"access.permission.list": true, "access.permission.explain": true}

	n := 0
	for path, item := range oapi.Paths {
		for method, op := range map[string]*huma.Operation{"GET": item.Get, "POST": item.Post, "PUT": item.Put, "PATCH": item.Patch, "DELETE": item.Delete} {
			if op == nil {
				continue
			}
			n++
			if _, ok := op.Extensions["x-ant-action"]; !ok {
				t.Errorf("%s: нет x-ant-action", op.OperationID)
			}
			url := path
			q := []string{}
			hdr := http.Header{}
			for _, p := range op.Parameters {
				v := fmt.Sprint(sample(reg, p.Schema, 0))
				switch p.In {
				case "path":
					url = strings.ReplaceAll(url, "{"+p.Name+"}", v)
				case "query":
					if p.Required {
						q = append(q, p.Name+"="+v)
					}
				case "header":
					if p.Required {
						hdr.Set(p.Name, v)
					}
				}
			}
			if len(q) > 0 {
				url += "?" + strings.Join(q, "&")
			}
			var body []byte
			if op.RequestBody != nil {
				for ct, mt := range op.RequestBody.Content {
					if strings.Contains(ct, "json") {
						body, _ = json.Marshal(sample(reg, mt.Schema, 0))
						hdr.Set("Content-Type", ct)
					} else {
						body = []byte("x")
						hdr.Set("Content-Type", ct)
					}
					break
				}
			}
			req := httptest.NewRequest(method, url, bytes.NewReader(body))
			for k, v := range hdr {
				req.Header[k] = v
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if gateOps[op.OperationID] {
				continue
			}
			var p struct {
				Code string `json:"code"`
			}
			_ = json.Unmarshal(rec.Body.Bytes(), &p)
			if rec.Code != http.StatusNotImplemented || p.Code != "api.not_implemented" {
				t.Errorf("%s %s (%s): %d %s", method, url, op.OperationID, rec.Code, strings.TrimSpace(rec.Body.String()))
			}
		}
	}
	if n < 100 {
		t.Fatalf("операций в спецификации %d — ждали все экраны §3a", n)
	}
}

var samples = []string{"x", "CA-1", "ENT01:x1", "a", "a1", "x@1", "ok", "2026-09-26", "1", "main", "item:ENT01:x1", "urn:ant:problem:api.not_found", "streebog256:" + strings.Repeat("0", 64)}

// sample — минимальное значение, проходящее схему (обязательные поля, первые
// значения перечислений, минимумы).
func sample(reg huma.Registry, s *huma.Schema, depth int) any {
	if s == nil || depth > 12 {
		return nil
	}
	if s.Ref != "" {
		return sample(reg, reg.SchemaFromRef(s.Ref), depth+1)
	}
	if s.Const != nil {
		return s.Const
	}
	if len(s.Enum) > 0 {
		return s.Enum[0]
	}
	switch s.Type {
	case huma.TypeString:
		switch s.Format {
		case "uuid":
			return "018f2b3c-0000-7000-8000-000000000001"
		case "date-time":
			return "2026-09-26T08:00:00.000Z"
		case "date":
			return "2026-09-26"
		}
		for _, c := range samples {
			if s.MinLength != nil && len(c) < *s.MinLength {
				c = c + strings.Repeat("x", *s.MinLength-len(c))
			}
			if s.Pattern == "" || regexp.MustCompile(s.Pattern).MatchString(c) {
				return c
			}
		}
		return "x"
	case huma.TypeInteger, huma.TypeNumber:
		v := 0.0
		if s.Minimum != nil {
			v = *s.Minimum
		}
		return int64(v)
	case huma.TypeBoolean:
		return false
	case huma.TypeArray:
		out := []any{}
		k := 0
		if s.MinItems != nil {
			k = *s.MinItems
		}
		for range k {
			out = append(out, sample(reg, s.Items, depth+1))
		}
		return out
	case huma.TypeObject, "":
		out := map[string]any{}
		for _, r := range s.Required {
			if slices.Contains(s.Required, r) {
				out[r] = sample(reg, s.Properties[r], depth+1)
			}
		}
		return out
	}
	return nil
}
