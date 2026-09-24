package entity

type DashboardSummary struct {
	TotalDataMigratedBytes    int64   `json:"total_data_migrated_bytes"`
	TotalFilesMigrated        int64   `json:"total_files_migrated"`
	TotalPendingFiles         int64   `json:"total_pending_files"`
	OverallProgressPercent    float64 `json:"overall_progress_percent"`
	LiveTransferRateMBps      float64 `json:"live_transfer_rate_mbps"`
	CurrentLatencyMs          float64 `json:"current_latency_ms"`
	DestinationFreeSpaceBytes uint64  `json:"destination_free_space_bytes"`

	StaleFilesOver1Year int64    `json:"stale_files_over_1_year"`
	MigrationEnabled    bool     `json:"migration_enabled"`
	VmTotalBytes        uint64   `json:"vm_total_bytes"`
	VmUsedBytes         uint64   `json:"vm_used_bytes"`
	VmFreeBytes         uint64   `json:"vm_free_bytes"`
	VmUsedPercent       float64  `json:"vm_used_percent"`
	RootPath            string   `json:"root_path"`
	LegacyPath          string   `json:"legacy_path"`
	ShardingType        string       `json:"sharding_type"`
	Clients             []string     `json:"clients"`
	LicenseTier         string       `json:"license_tier"`
	TotalLifetimeMigrations int64    `json:"total_lifetime_migrations"`
	MigrationHistory    []JobHistory `json:"migration_history"`
}

type MigrationStats struct {
	TotalFiles         int64
	MigratedFiles      int64
	TotalMigratedBytes int64
	StaleFiles         int64
}

type ExtensionStat struct {
	Extension string `json:"extension"`
	Count     int64  `json:"count"`
	SizeBytes int64  `json:"size_bytes"`
}

type LargeFile struct {
	BinID     int64  `json:"bin_id"`
	FileName  string `json:"file_name"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	Extension string `json:"extension"`
}

type StorageAnalytics struct {
	ExtensionStats []ExtensionStat `json:"extension_stats"`
	TopLargeFiles  []LargeFile     `json:"top_large_files"`
	ModuleStats    []ModuleStat    `json:"module_stats"`
	YearlyStats    []YearStat      `json:"yearly_stats"`
	MonthlyStats   []MonthlyStat   `json:"monthly_stats"`
}

type ModuleStat struct {
	Module    string `json:"module"`
	Count     int64  `json:"count"`
	SizeBytes int64  `json:"size_bytes"`
}

type YearStat struct {
	Year      int    `json:"year"`
	Count     int64  `json:"count"`
	SizeBytes int64  `json:"size_bytes"`
}

type MonthlyStat struct {
	Year      int   `json:"year"`
	Month     int   `json:"month"`
	Count     int64 `json:"count"`
	SizeBytes int64 `json:"size_bytes"`
}

type JobHistory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	VolumeBytes int64  `json:"volume_bytes"`
	Date        string `json:"date"`
	Duration    string `json:"duration"`
	Status      string `json:"status"`
}

type DuplicateGroup struct {
	Checksum     string `json:"checksum"`
	Count        int64  `json:"count"`
	SizeBytes    int64  `json:"size_bytes"`
	WastedBytes  int64  `json:"wasted_bytes"`
	ExampleName  string `json:"example_name"`
}

type DataQualityStats struct {
	TotalDuplicateFiles int64            `json:"total_duplicate_files"`
	TotalWastedBytes    int64            `json:"total_wasted_bytes"`
	DuplicateGroups     []DuplicateGroup `json:"duplicate_groups"`
}
