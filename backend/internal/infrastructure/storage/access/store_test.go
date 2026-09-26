package access_test

import (
	"context"
	"os"
	"testing"
	"time"

	app "ant/internal/application/access"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	accessdom "ant/internal/domain/access"
	dj "ant/internal/domain/journal"
	"ant/internal/infrastructure/security/identity"
	accessstore "ant/internal/infrastructure/storage/access"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/journaltest"
	"ant/internal/infrastructure/storage/journal/migrator"
)

// Схема access, проекция политики из журнала, шина безопасности и сеансы scs
// на своей БД (make dev-db). Без ANT_DB_HOST пропускается.
func TestAccessOnPostgres(t *testing.T) {
	if os.Getenv("ANT_DB_HOST") == "" {
		t.Skip("нет своей БД (make dev-db)")
	}
	ctx := context.Background()
	db := journaltest.NewDB(t)
	set := migrator.Set{Module: "access", FS: accessstore.Migrations, Dir: accessstore.MigrationsDir}
	if a, err := migrator.Up(ctx, db.Admin.ConnConfig, nil, set); err != nil || len(a) != 1 {
		t.Fatalf("миграция access: %v %v", a, err)
	}
	pool := db.AppPool(t)
	j := journalstore.NewStore(pool, journaltest.SysClock{})
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: dj.ZeroLink.String(), Partitions: 4}

	// Учётные данные: заявка, занятый логин, неудачи и блокировка, активация.
	creds := accessstore.NewCredentials(pool)
	hash, _ := identity.DefaultArgon2id.Hash("пароль-проверки")
	if err := creds.Create(ctx, app.Credential{Login: "ivanov", PersonID: "U-ivanov", Hash: hash, Status: app.AccountPending, DisplayName: "Иванов"}); err != nil {
		t.Fatal(err)
	}
	if err := creds.Create(ctx, app.Credential{Login: "ivanov", PersonID: "X", Hash: hash, Status: app.AccountPending}); err != app.ErrLoginTaken {
		t.Fatalf("логин занят: %v", err)
	}
	now := time.Now().UTC()
	var c app.Credential
	for i := 0; i < 3; i++ {
		var err error
		if c, err = creds.Failed(ctx, "ivanov", now, 3, time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	if !c.LockedUntil.After(now) || c.Failures != 0 {
		t.Fatalf("блокировка после 3 неудач: %+v", c)
	}
	if err := creds.Activate(ctx, "ivanov", "U-ivanov"); err != nil {
		t.Fatal(err)
	}
	if c, ok, err := creds.Get(ctx, "ivanov"); err != nil || !ok || c.Status != app.AccountActive || !c.LockedUntil.IsZero() || c.DisplayName != "Иванов" {
		t.Fatalf("активация: %+v %v %v", c, ok, err)
	}
	if list, err := creds.List(ctx); err != nil || len(list) != 1 {
		t.Fatalf("список: %v %v", list, err)
	}

	// Решения access в журнале → проекция политики (policy_seq растёт).
	w := app.JournalDecisions{Journal: j, DomainBuild: dj.ZeroLink.String()}
	at := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	rc, err := w.Write(ctx, app.Batch{Actor: "ADM-01", OccurredAt: at, Meta: platform.CommandMeta{}, Records: []app.Record{
		{Type: catalog.AccessPersonRegistered, Stream: "person:U-ivanov", Data: ev.AccessPersonRegisteredV1{PersonID: "U-ivanov", DisplayName: "Иванов"}},
		{Type: catalog.AccessAccountActivated, Stream: "person:U-ivanov", Data: ev.AccessAccountActivatedV1{PersonID: "U-ivanov", Login: "ivanov"}},
		{Type: catalog.PolicyRoleAssigned, Stream: "policy:ent01", Data: ev.PolicyRoleAssignedV1{PersonID: "U-ivanov", RoleID: "technologist", Scope: "ent01", ValidFrom: ev.Timestamp(at)}},
	}})
	if err != nil || rc.Seq == 0 || len(rc.EventIDs) != 3 {
		t.Fatalf("решение: %+v %v", rc, err)
	}
	seed, err := identity.LoadSeed(os.DirFS("../../../../.."))
	if err != nil {
		t.Fatal(err)
	}
	proj := app.NewProjection(accessdom.FromSeed(seed), accessstore.PolicyLog{Journal: j, Codec: codec})
	pol, err := proj.Policy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if pol.Seq != rc.Seq {
		t.Fatalf("policy_seq %d, запись %d", pol.Seq, rc.Seq)
	}
	p, ok := app.PrincipalOf(pol, "U-ivanov", "", at.Add(time.Minute))
	if !ok || p.Role != "technologist" || p.Name != "Иванов" || !p.HasRole("staff") {
		t.Fatalf("субъект из журнала: %+v", p)
	}
	if x, ok := pol.PersonByLogin("ivanov"); !ok || !x.Active {
		t.Fatalf("учётная запись в политике: %+v", x)
	}

	// Шина безопасности: неудачный вход и отказ — записи семейства security.
	bus := accessstore.NewSecurityBus(j, codec)
	if err := bus.AuthFailed(ctx, app.AuthFailure{Login: "ivanov", ClientIP: "10.0.0.1", Reason: app.AuthBadCredentials, At: at}); err != nil {
		t.Fatal(err)
	}
	if err := bus.AccessDenied(ctx, app.Denial{PersonID: "U-ivanov", ActionID: "access.policy.grant", Object: platform.ObjectRef{Kind: "workplace", ID: "WP-CNC-1"}, Code: "access.forbidden", At: at}); err != nil {
		t.Fatal(err)
	}
	// Повтор того же отказа в пределах минуты не пишется.
	if err := bus.AccessDenied(ctx, app.Denial{PersonID: "U-ivanov", ActionID: "access.policy.grant", Object: platform.ObjectRef{Kind: "workplace", ID: "WP-CNC-1"}, Code: "access.forbidden", At: at}); err != nil {
		t.Fatal(err)
	}
	var n int
	for _, e := range journaltest.ReadAll(t, j, "main") {
		if e.EventType == string(catalog.SecurityAuthFailed) || e.EventType == string(catalog.SecurityAccessDenied) {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("события security: %d", n)
	}

	// Сеансы scs в access.sessions: вход по паролю, выход.
	dir, err := identity.LoadDirectory(os.DirFS("../../../../.."))
	if err != nil {
		t.Fatal(err)
	}
	idp, err := identity.NewLocal(dir, proj, creds, identity.DefaultArgon2id, bus, identity.SessionStore(pool), identity.LocalOptions{
		At: func(context.Context) time.Time { return at.Add(time.Minute) }, LockFor: time.Nanosecond})
	if err != nil {
		t.Fatal(err)
	}
	pp, token, err := idp.Open(ctx, app.SessionCreate{Login: "ivanov", Password: "пароль-проверки"})
	if err != nil || pp.Role != "technologist" {
		t.Fatalf("вход: %+v %v", pp, err)
	}
	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM access.sessions`).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("сеанс в Postgres: %d %v", rows, err)
	}
	if got, _ := idp.Identify(ctx, app.Credentials{SessionToken: token}); got.PersonID != "U-ivanov" {
		t.Fatalf("сеанс: %+v", got)
	}
	if err := idp.Close(ctx, token); err != nil {
		t.Fatal(err)
	}
	if got, _ := idp.Identify(ctx, app.Credentials{SessionToken: token}); !got.Anonymous() {
		t.Fatal("после выхода")
	}
}
