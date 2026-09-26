package notifications

// NotificationSummary — сводка уведомлений пользователя для шапки (FR-57).
type NotificationSummary struct {
	Unread int                        `json:"unread" minimum:"0" doc:"Непрочитанных всего."`
	ByKind *NotificationSummaryByKind `json:"by_kind,omitempty" doc:"По видам: информация, тревога, задача, запрос решения."`
}

// NotificationSummaryByKind — непрочитанные по видам (FR-57).
type NotificationSummaryByKind struct {
	Info            int `json:"info" minimum:"0"`
	Alarm           int `json:"alarm" minimum:"0"`
	Task            int `json:"task" minimum:"0"`
	DecisionRequest int `json:"decision_request" minimum:"0"`
}
