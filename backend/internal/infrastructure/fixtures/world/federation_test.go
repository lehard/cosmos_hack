package world

import (
	"errors"
	"testing"

	dom "ant/internal/domain/federation"
	"ant/internal/infrastructure/security/profiles"
)

// TestFederationExtracts — UJ-9, FR-132: выписка металлургического завода
// проверяется получателем по корням из акта регистрации (подтверждено);
// у поставщика колец только подпись шлюза (подтверждено сервером);
// изменённая после подписи — отклоняется; без регистрации партнёра — «не подтверждено».
func TestFederationExtracts(t *testing.T) {
	f, err := LoadFederation(repoFS())
	if err != nil || f == nil {
		t.Fatalf("федерация: %v", err)
	}
	m := f.Manifest
	mz := f.Files["mz01-heat-5512.extract.json"]
	v, err := dom.VerifyExtract(mz, "MZ01", f.partnerRoots("MZ01"), profiles.Verify)
	if err != nil || v.Origin != dom.OriginVerified {
		t.Fatalf("МЗ-1: %v %s (%s)", err, v.Origin, v.Reason)
	}
	if v.Extract.Origin.HeatNo != "5512" || v.Extract.Subject.RecipientRef != "LOT-ZF-201" {
		t.Fatalf("МЗ-1: плавка/партия %+v", v.Extract.Subject)
	}
	humans := 0
	for _, s := range v.Signatures {
		if s.Verification != dom.SigValid {
			t.Fatalf("подпись %s: %s", s.KeyRef, s.Verification)
		}
		if s.Human {
			humans++
		}
	}
	if humans != 2 {
		t.Fatalf("подписей людей по актам: %d", humans)
	}
	pk, err := dom.VerifyExtract(f.Files["pk02-lot-r116.extract.json"], "PK02", f.partnerRoots("PK02"), profiles.Verify)
	if err != nil || pk.Origin != dom.OriginServerOnly {
		t.Fatalf("ПК-2: %v %s", err, pk.Origin)
	}
	if _, err := dom.VerifyExtract(f.Files[m.Tampered], "MZ01", f.partnerRoots("MZ01"), profiles.Verify); !errors.Is(err, dom.ErrTampered) {
		t.Fatalf("изменённая выписка принята: %v", err)
	}
	un, err := dom.VerifyExtract(mz, "", nil, profiles.Verify)
	if err != nil || un.Origin != dom.OriginUnverified {
		t.Fatalf("без регистрации: %v %s", err, un.Origin)
	}
	// Чужие корни (другой партнёр) — ключи не проверяемы, но и не «изменена».
	other, err := dom.VerifyExtract(mz, "MZ01", f.partnerRoots("PK02"), profiles.Verify)
	if err != nil || other.Origin != dom.OriginUnverified {
		t.Fatalf("чужие корни: %v %s", err, other.Origin)
	}
	// Исходящая выписка ENT01 → СБ-1 проверяется у сборщика нашими корнями (self).
	out, err := dom.VerifyExtract(f.Files[m.Outgoing[0].File], "ENT01", m.Self.Roots, profiles.Verify)
	if err != nil || out.Origin != dom.OriginVerified {
		t.Fatalf("исходящая у получателя: %v %s", err, out.Origin)
	}
}
