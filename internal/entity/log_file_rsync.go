package entity

type LogFileRsync struct {
	ID          int64   `json:"id"`
	Server      string  `json:"server"`
	LastBinID   int64   `json:"last_bin_id"`
	BinExecuted []int64 `json:"bin_executed"`
	ErrorLogs   string  `json:"error_logs"`
}
