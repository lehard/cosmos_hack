package nonconformity_test

import "ant/internal/domain/machinelogs"

func mlRequest() machinelogs.NCRequest {
	return machinelogs.NCRequest{ItemID: item, OperationRunID: "RUN-1", StepKey: "welding.weld", EquipmentID: "WELD-1",
		WindowEventID: "00000000-0000-7000-8000-00000000abcd"}
}
