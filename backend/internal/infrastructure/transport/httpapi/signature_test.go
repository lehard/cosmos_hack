package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
)

// Декоратор подписи (Д-59): команде с уровнем подписи порт проверки получает
// тело ровно как пришло и параметры пути; отказ порта — ответ с его кодом;
// принятая подпись — в контексте обработчика.

type fakeChecker struct {
	got    platform.SignedRequest
	refuse bool
}

func (f *fakeChecker) CheckRequest(_ context.Context, _ platform.Action, rq platform.SignedRequest) (platform.Signature, bool, error) {
	f.got = rq
	if f.refuse {
		return platform.Signature{}, false, platform.Fail(errcodes.SigningNoSignaturePath)
	}
	return platform.Signature{Status: "valid", KeyStorage: "software_browser", Envelope: []byte(`{}`)}, false, nil
}

type confirmIn struct {
	NCID string `path:"nc_id" maxLength:"64"`
	Body struct {
		platform.CommandHeader
		Reason string `json:"reason,omitempty"`
	}
}

type confirmOut struct {
	Body struct {
		KeyStorage string `json:"key_storage"`
	}
}

func TestSignatureDecorator(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		mux := http.NewServeMux()
		chk := &fakeChecker{refuse: refuse}
		api := New(mux, Config{Signatures: chk})
		act := platform.Action{ID: "nonconformity.nonconformity.confirm", Class: platform.ClassProtective, Owner: "nonconformity",
			Emits: []catalog.Type{catalog.DecisionNonconformityConfirmed}, SignatureLevel: 2}
		Register(api, Post("/nonconformities/{nc_id}/confirm", "t", "t"), act, func(ctx context.Context, _ *confirmIn) (*confirmOut, error) {
			out := &confirmOut{}
			sig, _ := platform.SignatureFrom(ctx)
			out.Body.KeyStorage = sig.KeyStorage
			return out, nil
		})
		body := `{"command_id":"0192f1a0-0000-7000-8000-000000000001","basis_seq":1,"policy_seq":2,  "reason":"x"}`
		rq := httptest.NewRequest(http.MethodPost, Prefix+"/nonconformities/NC-1/confirm", strings.NewReader(body))
		rq.Header.Set("Content-Type", "application/json")
		rw := httptest.NewRecorder()
		mux.ServeHTTP(rw, rq)
		if string(chk.got.Body) != body || chk.got.Params["nc_id"] != "NC-1" || chk.got.Meta.PolicySeq != 2 {
			t.Fatalf("порт получил: %+v", chk.got)
		}
		switch {
		case refuse && (rw.Code != 422 && rw.Code/100 != 4 || !strings.Contains(rw.Body.String(), "signing.no_signature_path")):
			t.Fatalf("отказ: %d %s", rw.Code, rw.Body.String())
		case !refuse && (rw.Code != 200 || !strings.Contains(rw.Body.String(), "software_browser")):
			t.Fatalf("принято: %d %s", rw.Code, rw.Body.String())
		}
	}
}
