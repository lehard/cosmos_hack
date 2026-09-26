package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"ant/internal/infrastructure/integration/ingest/stands"
	cncstand "ant/internal/infrastructure/integration/machinelogs/cnc/stand"
	weldstand "ant/internal/infrastructure/integration/machinelogs/welder/stand"
)

// Роль stands (AD-18, AD-6: одна копия) — каркас эпика 06: stand-ы внешних
// систем и оборудования и служебный порт сбоев. Тело роли подменяется здесь,
// реестр ролей в main.go не меняется.
func init() {
	roles["stands"] = role{run: runStands}
}

// runStands поднимает HTTP роли stands: протоколы stand-ов /stand/‹имя›/ и
// служебный порт сбоев /stand/_control/. Stand оборудования (сварочный
// источник IS-1) включается, если задан адрес edge-агента.
//
// Переменные окружения (до секции stands в ant.yaml, эпик 02):
//
//	ANT_STANDS_ADDR       — адрес HTTP роли (по умолчанию :8491);
//	ANT_STANDS_EDGE_URL   — локальный вход edge-агента для телеметрии (пусто — stand оборудования выключен);
//	ANT_STANDS_INTERVAL   — период телеметрии (по умолчанию 5s).
//
// Stand-ы 1С, Галактики, MES и VisionQC (эпики 30–33, 43) добавляются в реестр здесь.
func runStands(ctx context.Context, env *environment) error {
	addr := envOr("ANT_STANDS_ADDR", ":8491")
	reg := stands.NewRegistry()
	if edge := strings.TrimSpace(os.Getenv("ANT_STANDS_EDGE_URL")); edge != "" {
		iv, err := time.ParseDuration(envOr("ANT_STANDS_INTERVAL", "5s"))
		if err != nil {
			return err
		}
		reg.Add(&stands.EquipmentStand{Name: "weld-is-1", EquipmentID: "IS-1", EdgeURL: edge, Interval: iv})
		// Эпик 23 (FR-149): станок ЧПУ и сварочный источник — нормальное
		// выполнение и выполнение с двумя отклонениями по очереди; заказ
		// выполнения — POST /stand/‹имя›/executions {"kind": …}.
		reg.Add(cncstand.New("cnc-1", "CNC-1", edge, iv))
		reg.Add(weldstand.New("weld-is-2", "IS-2", edge, iv))
	}
	srv := &http.Server{Addr: addr, Handler: reg.Handler(), ReadHeaderTimeout: 10 * time.Second,
		BaseContext: func(net.Listener) context.Context { return ctx }}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	env.log.Info("stand-ы слушают", "addr", ln.Addr().String())
	errc := make(chan error, 2)
	go func() { errc <- srv.Serve(ln) }()
	go func() { errc <- reg.Run(ctx) }()
	select {
	case err := <-errc:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(sctx)
}
