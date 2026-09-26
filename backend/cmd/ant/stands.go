package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	onecstand "ant/internal/infrastructure/integration/erp/onec/stand"
	"ant/internal/infrastructure/integration/ingest/stands"
	cncstand "ant/internal/infrastructure/integration/machinelogs/cnc/stand"
	weldstand "ant/internal/infrastructure/integration/machinelogs/welder/stand"
	ovstand "ant/internal/infrastructure/integration/vision/operatorvision/stand"
	vqcstand "ant/internal/infrastructure/integration/vision/visionqc/stand"
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
// Настройки — секция stands в ant.yaml (переопределение переменными):
//
//	stands.addr     (ANT_STANDS_ADDR)     — адрес HTTP роли (по умолчанию :8491);
//	stands.edge_url (ANT_STANDS_EDGE_URL) — локальный вход edge-агента для телеметрии (пусто — stand оборудования выключен);
//	stands.interval (ANT_STANDS_INTERVAL) — период телеметрии (по умолчанию 5s).
//
// Stand-ы Галактики, MES и VisionQC (эпики 31–33, 43) добавляются в реестр здесь.
func runStands(ctx context.Context, env *environment) error {
	sc := env.cfg.Stands
	addr := sc.Addr
	if addr == "" {
		addr = ":8491"
	}
	reg := stands.NewRegistry()
	// Эпик 30 (FR-91, AD-18): stand 1С — OData v3 и HTTP-сервис qc.v1 под
	// /stand/1c/erp/, страница «глазами 1С» /stand/1c/; состояние — схема
	// stand_onec (без БД — в памяти). Имя «onec» — то же для сценариев.
	st, err := onecStand(ctx, env)
	if err != nil {
		return err
	}
	reg.Add(st)
	reg.Add(onecstand.Alias(st))
	if edge := strings.TrimSpace(sc.EdgeURL); edge != "" {
		iv := sc.Interval
		if iv <= 0 {
			iv = 5 * time.Second
		}
		reg.Add(&stands.EquipmentStand{Name: "weld-is-1", EquipmentID: "IS-1", EdgeURL: edge, Interval: iv})
		// Эпик 23 (FR-149): станок ЧПУ и сварочный источник — нормальное
		// выполнение и выполнение с двумя отклонениями по очереди; заказ
		// выполнения — POST /stand/‹имя›/executions {"kind": …}.
		reg.Add(cncstand.New("cnc-1", "CNC-1", edge, iv))
		reg.Add(weldstand.New("weld-is-2", "IS-2", edge, iv))
		// Эпик 33 (FR-97, FR-126): VisionQC — камера КТ-3 со ступенями
		// анализатора (главная история: шов, поры, прожог, подрез, испорченный
		// кадр 0,3, блик), OperatorVision — гипотезы о действиях на сборке;
		// сцена по заказу — POST /stand/‹имя›/shots {"kind": …}.
		reg.Add(vqcstand.New(vqcstand.Options{Name: "visionqc-kt3", EdgeURL: edge, Interval: 3 * iv}))
		reg.Add(ovstand.New(ovstand.Options{Name: "operatorvision-asm", EdgeURL: edge, Interval: 6 * iv}))
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

// onecStand — stand 1С с состоянием в схеме stand_onec на пуле ядра (роль
// ant_app); ядро не открылось — состояние в памяти процесса роли stands.
func onecStand(ctx context.Context, env *environment) (*onecstand.Stand, error) {
	opt := onecstand.Options{Log: env.log.With("stand", onecstand.Name)}
	if c, err := env.core(ctx); err == nil {
		opt.Store = onecstand.Postgres{Pool: c.pool}
	} else {
		env.log.Warn("stand 1С: БД недоступна — состояние в памяти", "err", err)
	}
	return onecstand.New(opt)
}
