package audit

type AuditEventType string

const (
	UNIVERSE_ELIGIBILITY   AuditEventType = "UNIVERSE_ELIGIBILITY"
	RANK_TOPN              AuditEventType = "RANK_TOPN"
	DEEP_SCAN              AuditEventType = "DEEP_SCAN"
	TOPK_SELECTION         AuditEventType = "TOPK_SELECTION"
	STAGE_CHANGED          AuditEventType = "STAGE_CHANGED"
	AIGATE_CALL            AuditEventType = "AIGATE_CALL"
	WEBUI_REQUEST          AuditEventType = "WEBUI_REQUEST"
	ALERT_RAISED           AuditEventType = "ALERT_RAISED"
	DISK_HEALTH_SAMPLE     AuditEventType = "DISK_HEALTH_SAMPLE"
	DB_WRITER_BACKPRESSURE AuditEventType = "DB_WRITER_BACKPRESSURE"
	FILTERS_REFRESHED      AuditEventType = "FILTERS_REFRESHED"
	INTENT_STATE_CHANGED   AuditEventType = "INTENT_STATE_CHANGED"
	RECONCILE_DIFF         AuditEventType = "RECONCILE_DIFF"
	ORDER_SUBMIT           AuditEventType = "ORDER_SUBMIT"
	ORDER_CANCEL           AuditEventType = "ORDER_CANCEL"
	ORDER_CANCEL_REPLACE   AuditEventType = "ORDER_CANCEL_REPLACE"
)

var eventTypes = map[AuditEventType]struct{}{
	UNIVERSE_ELIGIBILITY:   {},
	RANK_TOPN:              {},
	DEEP_SCAN:              {},
	TOPK_SELECTION:         {},
	STAGE_CHANGED:          {},
	AIGATE_CALL:            {},
	WEBUI_REQUEST:          {},
	ALERT_RAISED:           {},
	DISK_HEALTH_SAMPLE:     {},
	DB_WRITER_BACKPRESSURE: {},
	FILTERS_REFRESHED:      {},
	INTENT_STATE_CHANGED:   {},
	RECONCILE_DIFF:         {},
	ORDER_SUBMIT:           {},
	ORDER_CANCEL:           {},
	ORDER_CANCEL_REPLACE:   {},
}

func IsValidEventType(eventType AuditEventType) bool {
	_, ok := eventTypes[eventType]
	return ok
}
