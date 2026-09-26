package b2mml

// Типы подмножества B2MML-JSON (contracts/integrations/mes/b2mml): имена —
// как в XSD B2MML V7. Рукописные; соответствие схемам проверяет контрактный
// тест на эталонах и проверка каждого исходящего сообщения.

// Sender — отправитель BOD.
type Sender struct {
	LogicalID        string `json:"LogicalID"`
	ComponentID      string `json:"ComponentID,omitempty"`
	ConfirmationCode string `json:"ConfirmationCode,omitempty"`
}

// Receiver — получатель BOD.
type Receiver struct {
	LogicalID string `json:"LogicalID"`
}

// ApplicationArea — заголовок сообщения.
type ApplicationArea struct {
	Sender           Sender    `json:"Sender"`
	Receiver         *Receiver `json:"Receiver,omitempty"`
	CreationDateTime string    `json:"CreationDateTime"`
	BODID            string    `json:"BODID"`
}

// Quantity — количество.
type Quantity struct {
	QuantityString string `json:"QuantityString"`
	UnitOfMeasure  string `json:"UnitOfMeasure,omitempty"`
}

// Empty — глагол без параметров.
type Empty struct{}

// ActionCriteria — критерий глагола Sync.
type ActionCriteria struct {
	ActionExpression struct {
		ActionCode string `json:"actionCode"`
	} `json:"ActionExpression"`
}

// SyncVerb — глагол Sync.
type SyncVerb struct {
	ActionCriteria ActionCriteria `json:"ActionCriteria"`
}

// Lot — MaterialLot / MaterialSubLot подмножества.
type Lot struct {
	ID                   string `json:"ID"`
	MaterialLotID        string `json:"MaterialLotID,omitempty"`
	Description          string `json:"Description,omitempty"`
	MaterialDefinitionID string `json:"MaterialDefinitionID,omitempty"`
	Disposition          string `json:"Disposition"`
	Status               string `json:"Status,omitempty"`
	StorageLocation      string `json:"StorageLocation,omitempty"`
}

// SyncBody — тело SyncMaterialLot / SyncMaterialSubLot.
type SyncBody struct {
	ReleaseID       string          `json:"releaseID"`
	VersionID       string          `json:"versionID,omitempty"`
	ApplicationArea ApplicationArea `json:"ApplicationArea"`
	DataArea        struct {
		Sync           SyncVerb `json:"Sync"`
		MaterialLot    []Lot    `json:"MaterialLot,omitempty"`
		MaterialSubLot []Lot    `json:"MaterialSubLot,omitempty"`
	} `json:"DataArea"`
}

// SyncMaterialLot — блок партии.
type SyncMaterialLot struct {
	SyncMaterialLot SyncBody `json:"SyncMaterialLot"`
}

// SyncMaterialSubLot — блок экземпляра.
type SyncMaterialSubLot struct {
	SyncMaterialSubLot SyncBody `json:"SyncMaterialSubLot"`
}

// ErrorMessage — ошибка BOD.
type ErrorMessage struct {
	ErrorCode        string `json:"ErrorCode"`
	ErrorType        string `json:"ErrorType,omitempty"`
	ErrorDescription string `json:"ErrorDescription,omitempty"`
}

// BOD — подтверждение одного сообщения.
type BOD struct {
	OriginalApplicationArea struct {
		BODID            string `json:"BODID"`
		CreationDateTime string `json:"CreationDateTime,omitempty"`
	} `json:"OriginalApplicationArea"`
	BODSuccessMessage *struct {
		Duplicate bool `json:"Duplicate,omitempty"`
	} `json:"BODSuccessMessage,omitempty"`
	BODFailureMessage *struct {
		ErrorMessage []ErrorMessage `json:"ErrorMessage"`
	} `json:"BODFailureMessage,omitempty"`
}

// ConfirmBOD — подтверждение MES.
type ConfirmBOD struct {
	ConfirmBOD struct {
		ReleaseID       string          `json:"releaseID"`
		ApplicationArea ApplicationArea `json:"ApplicationArea"`
		DataArea        struct {
			Confirm Empty `json:"Confirm"`
			BOD     []BOD `json:"BOD"`
		} `json:"DataArea"`
	} `json:"ConfirmBOD"`
}

// MaterialRequirement — требование к материалу.
type MaterialRequirement struct {
	MaterialDefinitionID string    `json:"MaterialDefinitionID"`
	MaterialLotID        string    `json:"MaterialLotID,omitempty"`
	MaterialSubLotID     string    `json:"MaterialSubLotID,omitempty"`
	MaterialUse          string    `json:"MaterialUse,omitempty"`
	Quantity             *Quantity `json:"Quantity,omitempty"`
}

// SegmentRequirement — требование сегмента.
type SegmentRequirement struct {
	ID                   string `json:"ID"`
	Description          string `json:"Description,omitempty"`
	ProcessSegmentID     string `json:"ProcessSegmentID"`
	EarliestStartTime    string `json:"EarliestStartTime,omitempty"`
	LatestEndTime        string `json:"LatestEndTime,omitempty"`
	EquipmentRequirement []struct {
		EquipmentID string `json:"EquipmentID"`
	} `json:"EquipmentRequirement,omitempty"`
	MaterialRequirement []MaterialRequirement `json:"MaterialRequirement,omitempty"`
}

// OperationsRequest — производственный запрос.
type OperationsRequest struct {
	ID                 string               `json:"ID"`
	Description        string               `json:"Description,omitempty"`
	Priority           *int                 `json:"Priority,omitempty"`
	SegmentRequirement []SegmentRequirement `json:"SegmentRequirement"`
}

// ProcessOperationsSchedule — задание MES.
type ProcessOperationsSchedule struct {
	ProcessOperationsSchedule struct {
		ReleaseID       string          `json:"releaseID"`
		VersionID       string          `json:"versionID,omitempty"`
		ApplicationArea ApplicationArea `json:"ApplicationArea"`
		DataArea        struct {
			Process            Empty `json:"Process"`
			OperationsSchedule []struct {
				ID                string              `json:"ID"`
				Description       string              `json:"Description,omitempty"`
				OperationsRequest []OperationsRequest `json:"OperationsRequest"`
			} `json:"OperationsSchedule"`
		} `json:"DataArea"`
	} `json:"ProcessOperationsSchedule"`
}

// OperationsEvent — событие операции.
type OperationsEvent struct {
	ID                  string `json:"ID"`
	Description         string `json:"Description,omitempty"`
	Category            string `json:"Category"`
	EffectiveTimestamp  string `json:"EffectiveTimestamp"`
	RecordTimestamp     string `json:"RecordTimestamp,omitempty"`
	OperationsRequestID string `json:"OperationsRequestID,omitempty"`
	SegmentResponseID   string `json:"SegmentResponseID"`
	ProcessSegmentID    string `json:"ProcessSegmentID"`
	MaterialSubLotID    string `json:"MaterialSubLotID"`
	EquipmentID         string `json:"EquipmentID,omitempty"`
	PersonnelID         string `json:"PersonnelID,omitempty"`
}

// NotifyOperationsEvent — события операций MES.
type NotifyOperationsEvent struct {
	NotifyOperationsEvent struct {
		ReleaseID       string          `json:"releaseID"`
		VersionID       string          `json:"versionID,omitempty"`
		ApplicationArea ApplicationArea `json:"ApplicationArea"`
		DataArea        struct {
			Notify          Empty             `json:"Notify"`
			OperationsEvent []OperationsEvent `json:"OperationsEvent"`
		} `json:"DataArea"`
	} `json:"NotifyOperationsEvent"`
}

// About — описание канала (HTTP-привязка).
type About struct {
	LogicalID string   `json:"logical_id"`
	ReleaseID string   `json:"release_id"`
	Supported []string `json:"supported"`
	Product   string   `json:"product,omitempty"`
}
