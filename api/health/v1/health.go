package v1

type HealthResponse struct {
	UptimeSeconds uint64 `json:"uptime_seconds"`
}
