package materials

import (
	"context"
	"net/http"
	"time"

	app "ant/internal/application/materials"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "materials"

// Register объявляет операции модуля materials: загрузка материала по адресу
// содержимого, метаданные и байты (AD-23; FR-102, FR-139).
//
// Загрузка — сырое тело (RawBody) с Content-Type файла, метаданные — в
// параметрах запроса: base64 в JSON увеличил бы сканы на треть. Метаданных
// команды человека (basis_seq) у загрузки нет — это служебная запись
// material.object.stored; решение, ссылающееся на материал (заверение бумажной
// подписи, решение по несоответствию), — отдельная команда со своим CommandHeader.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	r := httpapi.Post("/materials", "Загрузить материал",
		"AD-23: скан бумажной подписи, кадр, вложение сохраняется в хранилище материалов по адресу H(байты) и шифруется конвертной схемой; "+
			"в журнале — только адрес и метаданные. Повтор тех же байтов — тот же адрес (идемпотентно).")
	r.NoCommandMeta = true
	r.Status = http.StatusCreated
	httpapi.Register(api, r,
		platform.Action{ID: "materials.material.upload", Class: platform.ClassRecord, Owner: owner, Emits: []catalog.Type{catalog.MaterialObjectStored}},
		func(ctx context.Context, in *struct {
			ContentType    string `header:"Content-Type" doc:"Тип содержимого файла."`
			Kind           string `query:"kind" required:"true" enum:"photo,video,illustration,protocol,log_excerpt,scan,other" doc:"Вид материала."`
			ItemID         string `query:"item_id" maxLength:"128" doc:"Изделие."`
			CapturedAt     string `query:"captured_at" format:"date-time" doc:"Когда снято."`
			IsIllustration bool   `query:"is_illustration" doc:"Иллюстрация, а не доказательство."`
			ProvenanceNote string `query:"provenance_note" maxLength:"512" doc:"Происхождение."`
			RawBody        []byte `contentType:"application/octet-stream"`
		}) (*httpapi.Out[app.MaterialInfo], error) {
			up := app.MaterialUpload{MediaType: in.ContentType, Kind: in.Kind, ItemID: in.ItemID, IsIllustration: in.IsIllustration, ProvenanceNote: in.ProvenanceNote, Bytes: in.RawBody}
			if in.CapturedAt != "" {
				t, err := time.Parse(time.RFC3339Nano, in.CapturedAt)
				if err != nil {
					return nil, platform.Fail(errcodes.ApiValidationFailed, "field", "captured_at", "reason", err.Error())
				}
				up.CapturedAt = &t
			}
			v, err := c.Upload(ctx, up)
			if err != nil {
				return nil, err
			}
			return httpapi.OK(v), nil
		})

	type addrIn struct {
		Address string `path:"address" maxLength:"160" doc:"Адрес содержимого streebog256:…."`
	}
	httpapi.Read(api, httpapi.Get("/materials/{address}", "Метаданные материала",
		"AD-23: вид, тип и размер, изделие, время съёмки, «иллюстрация, а не доказательство»."),
		platform.Action{ID: "materials.material.read", Owner: owner},
		func(ctx context.Context, in *addrIn, _ platform.Moment) (app.MaterialInfo, error) {
			return q.Material(ctx, in.Address)
		})

	type contentOut struct {
		httpapi.Meta
		ContentType string `header:"Content-Type" doc:"Тип содержимого."`
		Body        []byte
	}
	content := httpapi.Get("/materials/{address}/content", "Содержимое материала",
		"AD-23: байты материала (кадр, иллюстрация, скан, протокол) как есть, тип — заголовок Content-Type; адрес проверяется после расшифрования и чтения.")
	content.Binary = []string{"image/*", "application/pdf", "*/*"}
	httpapi.Register(api, content,
		platform.Action{ID: "materials.material.content", Class: platform.ClassRead, Owner: owner},
		func(ctx context.Context, in *addrIn) (*contentOut, error) {
			v, err := q.Content(ctx, in.Address)
			if err != nil {
				return nil, err
			}
			return &contentOut{ContentType: v.MediaType, Body: v.Bytes}, nil
		})
}
