package reference

import (
	"context"

	"ant/internal/application/platform"
	app "ant/internal/application/reference"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля reference: справочники как версионируемые
// данные журнала с датой действия (AD-31, FR-17, FR-80, FR-81, FR-95).
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	const owner = "reference"
	emits := func(t ...catalog.Type) []catalog.Type { return t }
	type momentIn struct{ httpapi.MomentQuery }

	httpapi.Read(api, httpapi.Get("/reference/item-types", "Номенклатура", "Изделия и номенклатура: обозначение, ревизия, зоны, состав, маркировка — срез на момент (AD-31)."),
		platform.Action{ID: "reference.item_type.list", Owner: owner},
		func(ctx context.Context, in *momentIn, m platform.Moment) (app.RefItemTypeList, error) {
			return q.ItemTypes(ctx, m)
		})
	httpapi.Read(api, httpapi.Get("/reference/locations", "Места", "Здания, цеха, линии, участки, рабочие места, склады, изоляторы — иерархия областей прав (AD-15)."),
		platform.Action{ID: "reference.location.list", Owner: owner},
		func(ctx context.Context, in *momentIn, m platform.Moment) (app.RefLocationList, error) {
			return q.Locations(ctx, m)
		})
	httpapi.Read(api, httpapi.Get("/reference/equipment", "Оборудование и поверка", "FR-17: оборудование с поверкой и калибровкой на дату."),
		platform.Action{ID: "reference.equipment.list", Owner: owner, Subject: "equipment"},
		func(ctx context.Context, in *momentIn, m platform.Moment) (app.RefEquipmentList, error) {
			return q.Equipment(ctx, m)
		})
	httpapi.Read(api, httpapi.Get("/reference/calendar", "Производственный календарь", "AD-4: сроки считаются по производственному календарю и графику смен."),
		platform.Action{ID: "reference.calendar.read", Owner: owner},
		func(ctx context.Context, in *struct {
			Year int `query:"year" minimum:"0" maximum:"9999" doc:"Год; 0 — текущий."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.RefCalendar, error) {
			return q.Calendar(ctx, in.Year, m)
		})
	httpapi.Read(api, httpapi.Get("/reference/shifts", "Смены", "FR-81: график смен по местам."),
		platform.Action{ID: "reference.shift.list", Owner: owner},
		func(ctx context.Context, in *struct {
			LocationID string `query:"location_id" maxLength:"128"`
			httpapi.MomentQuery
		}, m platform.Moment) (app.RefShiftList, error) {
			return q.Shifts(ctx, in.LocationID, m)
		})
	httpapi.Read(api, httpapi.Get("/reference/external-ids", "Соответствия внешних ID", "AD-18, FR-95: соответствия ID 1С, Галактики, MES, КОМПАС; конфликт — сигнал, не перезапись."),
		platform.Action{ID: "reference.external_id.list", Owner: owner},
		func(ctx context.Context, in *struct {
			System string `query:"system" maxLength:"32"`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.RefExternalIDList, error) {
			return q.ExternalIDs(ctx, in.System, m, in.Page())
		})

	httpapi.Do(api, httpapi.Post("/reference/item-types", "Определить позицию номенклатуры", "AD-31: новая версия с датой действия; изменение задним числом доходит до изделий адресованными записями стадии (AD-42)."),
		platform.Action{ID: "reference.item_type.define", Class: platform.ClassRecord, Owner: owner, Emits: emits(catalog.ReferenceItemTypeDefined), SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.DefineItemType }) (platform.Receipt, error) {
			return c.DefineItemType(ctx, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/reference/locations", "Определить место", "AD-31."),
		platform.Action{ID: "reference.location.define", Class: platform.ClassRecord, Owner: owner, Emits: emits(catalog.ReferenceLocationDefined), SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.DefineLocation }) (platform.Receipt, error) {
			return c.DefineLocation(ctx, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/reference/equipment", "Определить оборудование", "AD-31."),
		platform.Action{ID: "reference.equipment.define", Class: platform.ClassRecord, Owner: owner, Subject: "equipment", Emits: emits(catalog.ReferenceEquipmentDefined), SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.DefineEquipment }) (platform.Receipt, error) {
			return c.DefineEquipment(ctx, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/reference/equipment/{equipment_id}/verification", "Записать поверку", "FR-17: поверка или калибровка — предусловие операции на дату; метролог."),
		platform.Action{ID: "reference.equipment.verify", Class: platform.ClassRecord, Owner: owner, Subject: "equipment", Emits: emits(catalog.ReferenceEquipmentVerified), SignatureLevel: 2},
		func(ctx context.Context, in *struct {
			EquipmentID string `path:"equipment_id" maxLength:"128"`
			Body        app.VerifyEquipment
		}) (platform.Receipt, error) {
			return c.VerifyEquipment(ctx, in.EquipmentID, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/reference/calendar", "Определить производственный календарь", "AD-31."),
		platform.Action{ID: "reference.calendar.define", Class: platform.ClassRecord, Owner: owner, Emits: emits(catalog.ReferenceCalendarDefined), SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.DefineCalendar }) (platform.Receipt, error) {
			return c.DefineCalendar(ctx, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/reference/shifts", "Запланировать смену", "FR-81."),
		platform.Action{ID: "reference.shift.schedule", Class: platform.ClassRecord, Owner: owner, Emits: emits(catalog.ReferenceShiftScheduled), SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.ScheduleShift }) (platform.Receipt, error) {
			return c.ScheduleShift(ctx, in.Body)
		})
}
