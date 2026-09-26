package main

import (
	"io"
	"net/http"

	appvision "ant/internal/application/vision"
	"ant/internal/infrastructure/integration/vision/operatorvision"
	"ant/internal/infrastructure/integration/vision/visionqc"
	storevision "ant/internal/infrastructure/storage/vision"
)

// VisionRoute — вход внешней системы видеофиксации на краю (эпик 33; FR-97,
// FR-126, AD-18): POST /v1/vision/‹система› — сообщение по протоколу системы
// (contracts/integrations/vision) → адаптер (порт SignalAdapter) → события
// контракта с вектором версий → буфер агента. Заменить stand реальной системой
// — направить её сюда; заменить систему другой — другой адаптер в этом
// списке; ядро не меняется.
type VisionRoute struct {
	Adapter appvision.SignalAdapter
	Relay   appvision.Relay
}

// VisionRoutes — адаптеры VisionQC и OperatorVision демо-установки;
// illustrations — каталог распакованного набора иллюстраций (пусто — офлайн
// файлов нет: иллюстрации не прикладываются, в ограничениях — ссылка).
func VisionRoutes(coreURL, illustrations string, client *http.Client) ([]VisionRoute, error) {
	cat, err := storevision.Illustrations()
	if err != nil {
		return nil, err
	}
	relay := appvision.Relay{Catalog: cat}
	if illustrations != "" {
		relay.Files = visionqc.DirIllustrations{Dir: illustrations}
		relay.Materials = visionqc.HTTPMaterials{CoreURL: coreURL, Client: client}
	}
	return []VisionRoute{
		{Adapter: visionqc.New(nil), Relay: relay},
		{Adapter: operatorvision.New(), Relay: relay},
	}, nil
}

// registerVision — маршруты /v1/vision/‹система›: отказ адаптера (сообщение не
// по протоколу) — 400 системе, в ant ничего не уходит.
func registerVision(mux *http.ServeMux, a *Agent, routes []VisionRoute) {
	for _, rt := range routes {
		mux.HandleFunc("POST /v1/vision/"+rt.Adapter.System(), func(w http.ResponseWriter, r *http.Request) {
			b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4<<20))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			sig, err := rt.Adapter.Translate(b)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			evs, err := rt.Relay.Events(r.Context(), sig)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			enqueue(w, a, evs)
		})
	}
}
