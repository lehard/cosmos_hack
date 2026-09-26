package agent

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"time"

	"ant/internal/contracts/procs"
	dom "ant/internal/domain/signing"
)

// API — операции ядра над JSON для оболочек (WASM в расширении и страница;
// локальная программа зовёт функции пакета напрямую). Каждый ответ —
// {"ok":true,…} или {"ok":false,"code":…,"message":…}: код — из каталога
// ошибок, его показывает интерфейс.

// Version — версия ядра; Build — хеш сборки (ставит компоновщик).
var (
	Version = "0.1.0"
	Build   = "dev"
)

// Reply — ответ операции.
type Reply map[string]any

func fail(err error) Reply {
	return Reply{"ok": false, "code": CodeOf(err), "message": err.Error()}
}

func okReply(kv ...any) Reply {
	r := Reply{"ok": true}
	for i := 0; i+1 < len(kv); i += 2 {
		r[kv[i].(string)] = kv[i+1]
	}
	return r
}

// JSON — ответ строкой.
func (r Reply) JSON() string {
	b, _ := json.Marshal(r)
	return string(b)
}

// CallContext — обстоятельства вызова от оболочки.
type CallContext struct {
	Now            string `json:"now"`
	WorkplaceID    string `json:"workplace_id"`
	SeenCheckpoint int64  `json:"seen_checkpoint"`
	Profile        string `json:"profile"`
}

func (c CallContext) context() Context {
	t, err := time.Parse(TimeLayout, c.Now)
	if err != nil {
		t = time.Now().UTC()
	}
	return Context{Now: t, WorkplaceID: c.WorkplaceID, SeenCheckpoint: c.SeenCheckpoint, Profile: c.Profile}
}

// APIVersion — версия ядра и известные форматы документа.
func APIVersion() Reply {
	return okReply("agent_version", Version, "build", Build, "doc_format_versions", DocFormatVersions,
		"level1_actions", dom.Level1Actions())
}

// APIInspectKey — открытые сведения файла ключа (без хранения).
func APIInspectKey(file string) Reply {
	f, _, err := ParseKeyFile([]byte(file))
	if err != nil {
		return fail(err)
	}
	return okReply("key_ref", f.KeyRef, "profile", f.Profile, "fingerprint", f.Fingerprint, "person_id", PersonOf(f.KeyRef))
}

// SealRequest — что зашифровать.
type SealRequest struct {
	Files          []string `json:"files"`
	PersonID       string   `json:"person_id"`
	KeyStorage     string   `json:"key_storage"`
	StorageVariant string   `json:"storage_variant"`
	Now            string   `json:"now"`
}

// APISeal — набор ключей из файлов под PIN.
func APISeal(reqJSON, pin string) Reply {
	var rq SealRequest
	if err := json.Unmarshal([]byte(reqJSON), &rq); err != nil {
		return fail(refuse(CodeInvalid, "%v", err))
	}
	files := make([][]byte, 0, len(rq.Files))
	for _, f := range rq.Files {
		files = append(files, []byte(f))
	}
	b, err := NewBundle(files, rq.PersonID)
	if err != nil {
		return fail(err)
	}
	if rq.KeyStorage == "" {
		rq.KeyStorage = dom.StorageSoftwareBrowser
	}
	s, err := Seal(b, pin, SealOptions{KeyStorage: rq.KeyStorage, StorageVariant: rq.StorageVariant, Now: CallContext{Now: rq.Now}.context().Now}, rand.Reader)
	if err != nil {
		return fail(refuse(CodeInvalid, "%v", err))
	}
	return okReply("sealed", s)
}

// APIUnlock — проверить PIN; ответ — ключ хранилища для памяти сеанса.
func APIUnlock(sealedJSON, pin string) Reply {
	var s Sealed
	if err := json.Unmarshal([]byte(sealedJSON), &s); err != nil {
		return fail(refuse(CodeInvalid, "%v", err))
	}
	dk := DeriveKey(s, pin)
	if _, err := Open(s, dk); err != nil {
		return fail(err)
	}
	return okReply("dk_b64", base64.StdEncoding.EncodeToString(dk))
}

// APIPrepare — что будет подписано: отпечаток и сводка (для окна подтверждения).
func APIPrepare(blockJSON, sealedJSON, ctxJSON string) Reply {
	b, s, c, err := parseCall(blockJSON, sealedJSON, ctxJSON)
	if err != nil {
		return fail(err)
	}
	p, err := Prepare(b, s.PersonID, s.Keys, c.context())
	if err != nil {
		return fail(err)
	}
	return okReply("prepared", p, "key_storage", s.KeyStorage, "storage_variant", s.StorageVariant)
}

// SignCall — подпись: запрос, хранилище, ключ сеанса, подтверждённый
// отпечаток (что видел человек в окне), журнал.
type SignCall struct {
	Block           json.RawMessage `json:"block"`
	Sealed          json.RawMessage `json:"sealed"`
	DKB64           string          `json:"dk_b64"`
	Context         CallContext     `json:"context"`
	ConfirmedDigest string          `json:"confirmed_digest"`
	Journal         []Entry         `json:"journal"`
}

// APISign — подписать: заново собрать подписываемое, сверить с тем, что
// подтвердил человек, проверить частоту уровня 1, подписать и дописать журнал.
func APISign(callJSON string) Reply {
	var sc SignCall
	if err := json.Unmarshal([]byte(callJSON), &sc); err != nil {
		return fail(refuse(CodeInvalid, "%v", err))
	}
	b, s, _, err := parseCall(string(sc.Block), string(sc.Sealed), "")
	if err != nil {
		return fail(err)
	}
	ctx := sc.Context.context()
	p, err := Prepare(b, s.PersonID, s.Keys, ctx)
	if err != nil {
		return fail(err)
	}
	if p.Level == dom.Level2 && p.DocDigest != sc.ConfirmedDigest {
		return fail(refuse(CodeChanged, "подписываемое изменилось после окна подтверждения — подпишите заново"))
	}
	if p.Level == dom.Level1 {
		if wait, ok := RateCheck(sc.Journal, ctx.Now, DefaultRate); !ok {
			return fail(refuse(CodeRate, "не больше %d подписей уровня 1 в минуту — подождите %d с", DefaultRate.Max, int(wait.Seconds())+1))
		}
	}
	dk, err := base64.StdEncoding.DecodeString(sc.DKB64)
	if err != nil || len(dk) == 0 {
		return fail(refuse(CodePIN, "ключ заблокирован — введите PIN"))
	}
	bundle, err := Open(s, dk)
	if err != nil {
		return fail(err)
	}
	env, err := Sign(p, bundle)
	if err != nil {
		return fail(refuse(CodeTampered, "%v", err))
	}
	j, e := Append(sc.Journal, p, env)
	return okReply("envelope", env, "prepared", p, "entry", e, "journal", Trim(j), "key_storage", s.KeyStorage,
		"storage_variant", s.StorageVariant)
}

// APIShiftReport — сменный рапорт по локальному журналу (без подписи —
// подпись уровня 3 делает APISign над содержимым рапорта).
func APIShiftReport(journalJSON, baseJSON string) Reply {
	var j []Entry
	var base dom.ShiftReport
	if err := json.Unmarshal([]byte(journalJSON), &j); err != nil {
		return fail(refuse(CodeInvalid, "%v", err))
	}
	if err := json.Unmarshal([]byte(baseJSON), &base); err != nil {
		return fail(refuse(CodeInvalid, "%v", err))
	}
	r, err := ShiftReport(j, base)
	if err != nil {
		return fail(refuse(CodeInvalid, "%v", err))
	}
	return okReply("report", r)
}

func parseCall(blockJSON, sealedJSON, ctxJSON string) (procs.SignBlock, Sealed, CallContext, error) {
	var b procs.SignBlock
	var s Sealed
	var c CallContext
	if err := json.Unmarshal([]byte(blockJSON), &b); err != nil {
		return b, s, c, refuse(CodeInvalid, "запрос подписи: %v", err)
	}
	if err := json.Unmarshal([]byte(sealedJSON), &s); err != nil || len(s.Keys) == 0 {
		return b, s, c, refuse(CodeTokenMissing, "ключ не загружен")
	}
	if ctxJSON != "" {
		if err := json.Unmarshal([]byte(ctxJSON), &c); err != nil {
			return b, s, c, refuse(CodeInvalid, "контекст: %v", err)
		}
	}
	return b, s, c, nil
}
