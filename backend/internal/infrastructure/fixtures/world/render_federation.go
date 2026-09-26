package world

import (
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"sync"
	"time"

	fedapp "ant/internal/application/federation"
	itemapp "ant/internal/application/item"
	dom "ant/internal/domain/federation"
	"ant/internal/infrastructure/fixtures/loader"
	"ant/internal/infrastructure/security/profiles"
)

// Федерация предприятий в мире заготовок (эпик 41; FR-131, FR-132, AD-19):
// партнёры с корнями доверия и подписанные выписки паспорта из
// scenarios/federation (манифест и пакеты DSSE — ANT_UPDATE_FEDERATION=1,
// federation_gen_test.go). Выписки проверяются при генерации тем же кодом,
// что у live (domain/federation.VerifyExtract + ГОСТ Р 34.10-2012), —
// статус происхождения и подписи в ответах не выдуманы.

// FederationDir — каталог партнёров и выписок от корня репозитория.
const FederationDir = "scenarios/federation"

// FederationPartner — партнёр в манифесте.
type FederationPartner struct {
	Code       string   `json:"code"`
	Name       string   `json:"name"`
	Role       string   `json:"role,omitempty"`
	Endpoint   string   `json:"endpoint,omitempty"`
	Roots      []string `json:"roots"`
	DocumentID string   `json:"document_id,omitempty"`
}

// FederationSample — выписка сценария.
type FederationSample struct {
	File       string `json:"file"`
	Partner    string `json:"partner"`
	Item       string `json:"item,omitempty"`
	DocumentID string `json:"document_id,omitempty"`
	MessageID  string `json:"message_id,omitempty"`
}

// FederationManifest — manifest.json каталога федерации.
type FederationManifest struct {
	Self     FederationPartner   `json:"self"`
	Partners []FederationPartner `json:"partners"`
	Incoming []FederationSample  `json:"incoming"`
	Outgoing []FederationSample  `json:"outgoing"`
	Tampered string              `json:"tampered,omitempty"`
}

// Federation — манифест и байты выписок.
type Federation struct {
	Manifest FederationManifest
	Files    map[string][]byte

	// verified — итог проверки выписки (файл и партнёр → Verification): байты
	// выписок неизменны, а проверка подписей ГОСТ Р 34.10-2012 дорогая
	// (~десятки мс на выписку); без памяти federationViews проверял бы каждую
	// выписку заново на каждом изделии каждого шага — холодный старт api
	// строит мир заготовок в памяти и тратил на это ~3 мин CPU.
	mu       sync.Mutex
	verified map[string]verifyResult
}

type verifyResult struct {
	v   dom.Verification
	err error
}

// verify — dom.VerifyExtract выписки file (канал партнёра partner; "" — без
// корней, для исходящих), с памятью: одна проверка на файл за процесс.
// Verification только читается (fedapp.ViewOf строит свежий вид), поэтому
// общий экземпляр безопасен.
func (f *Federation) verify(file, partner string) (dom.Verification, error) {
	key := file + "\x00" + partner
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.verified[key]; ok {
		return r.v, r.err
	}
	var roots []string
	if partner != "" {
		roots = f.partnerRoots(partner)
	}
	v, err := dom.VerifyExtract(f.Files[file], partner, roots, profiles.Verify)
	if f.verified == nil {
		f.verified = map[string]verifyResult{}
	}
	f.verified[key] = verifyResult{v: v, err: err}
	return v, err
}

// LoadFederation читает scenarios/federation (нет каталога — федерации нет).
func LoadFederation(fsys fs.FS) (*Federation, error) {
	b, err := fs.ReadFile(fsys, path.Join(FederationDir, "manifest.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f := &Federation{Files: map[string][]byte{}}
	if err := json.Unmarshal(b, &f.Manifest); err != nil {
		return nil, err
	}
	names, err := fs.Glob(fsys, FederationDir+"/*.json")
	if err != nil {
		return nil, err
	}
	for _, n := range names {
		if f.Files[path.Base(n)], err = fs.ReadFile(fsys, n); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// FederationPartnerRoots — отпечатки корней партнёра code.
func (f *Federation) partnerRoots(code string) []string {
	for _, p := range f.Manifest.Partners {
		if p.Code == code {
			return p.Roots
		}
	}
	return nil
}

// federationViews — выписки на шаге: входящие — с приёмки партии (или с
// начала мира), исходящая — после выпуска изделия.
func (c *Ctx) federationViews() []fedapp.PassportExtractView {
	f := c.M.federation
	var out []fedapp.PassportExtractView
	for _, s := range f.Manifest.Incoming {
		raw := f.Files[s.File]
		v, err := f.verify(s.File, s.Partner)
		if err != nil {
			panic("федерация: выписка " + s.File + " не проходит проверку: " + err.Error())
		}
		at := c.M.Steps[0]
		out = append(out, fedapp.ViewOf(v, raw, "incoming", s.Partner, v.Digest, at, nil))
	}
	for _, s := range f.Manifest.Outgoing {
		it := c.M.itemByID[s.Item]
		if it == nil || it.Spec.Rel.Time().IsZero() || it.Spec.Rel.Time().After(c.T) {
			continue
		}
		raw := f.Files[s.File]
		v, err := f.verify(s.File, "")
		if err != nil {
			panic("федерация: выписка " + s.File + ": " + err.Error())
		}
		at := it.Spec.Rel.Time().Add(30 * time.Minute)
		if at.After(c.T) {
			at = c.T
		}
		view := fedapp.ViewOf(v, raw, "outgoing", s.Partner, v.Digest, at, nil)
		view.Extract.DocumentID, view.Extract.MessageID = s.DocumentID, s.MessageID
		ack := true
		view.Extract.Acknowledged = &ack
		view.OriginReason = "исходящая: подписи проверяет получатель по нашим корням из его акта регистрации"
		out = append(out, view)
	}
	return out
}

// renderFederation — federation.partner.list, federation.extract.list (все,
// по направлению и по партнёру), federation.extract.read по отпечатку.
func renderFederation(c *Ctx) []loader.Response {
	f := c.M.federation
	if f == nil {
		return nil
	}
	var partners []fedapp.Partner
	for _, p := range f.Manifest.Partners {
		partners = append(partners, fedapp.Partner{PartnerCode: p.Code, Name: p.Name, RootFingerprints: p.Roots, Endpoint: p.Endpoint,
			DocumentID: p.DocumentID, RegisteredAt: c.M.Steps[0].UTC(), Channel: "ok"})
	}
	out := []loader.Response{resp("federation.partner.list", fedapp.PartnerList{Items: partners})}
	views := c.federationViews()
	all := make([]fedapp.PassportExtract, 0, len(views))
	for _, v := range views {
		all = append(all, v.Extract)
		out = append(out, resp("federation.extract.read", v, "extract_digest", v.Extract.ExtractDigest))
	}
	out = append(out, resp("federation.extract.list", fedapp.PassportExtractList{Items: fedapp.Filter(all, "", "")}))
	for _, d := range []string{"incoming", "outgoing"} {
		out = append(out, resp("federation.extract.list", fedapp.PassportExtractList{Items: fedapp.Filter(all, d, "")}, "direction", d))
	}
	for _, p := range f.Manifest.Partners {
		out = append(out, resp("federation.extract.list", fedapp.PassportExtractList{Items: fedapp.Filter(all, "", p.Code)}, "partner_code", p.Code))
	}
	return out
}

// federationGenealogy — узел «выписка паспорта поставщика» под партией lot
// (FR-45, FR-132: выписка — корень генеалогии) для паспорта изделия.
func (c *Ctx) federationGenealogy(lot string) []itemapp.GenealogyNode {
	f := c.M.federation
	if f == nil {
		return nil
	}
	var out []itemapp.GenealogyNode
	for _, v := range c.federationViews() {
		if v.Extract.Direction != "incoming" || v.Extract.Subject == nil || v.Extract.Subject.ID != lot {
			continue
		}
		prov := map[string]string{dom.OriginVerified: "происхождение подтверждено: подписи " + v.Extract.PartnerCode + " проверены",
			dom.OriginServerOnly: "происхождение подтверждено сервером отправителя", dom.OriginUnverified: "происхождение не подтверждено"}[v.Extract.OriginStatus]
		label := "Выписка " + v.Extract.PartnerCode
		if v.Extract.HeatNo != "" {
			label += ": плавка " + v.Extract.HeatNo
		}
		out = append(out, itemapp.GenealogyNode{Ref: v.Extract.ExtractDigest, Kind: "partner_extract", Label: label, ParentRef: lot, Provenance: prov})
	}
	return out
}
