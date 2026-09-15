package utils

import (
	"github.com/shirou/gopsutil/v3/disk"
)

type DiskSpace struct {
	TotalBytes  uint64  `json:"total_bytes"`
	FreeBytes   uint64  `json:"free_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

func GetStorageDiskSpace(path string) (DiskSpace, error) {
	if path == "" {
		path = "."
	}
	usage, err := disk.Usage(path)
	if err != nil {
		return DiskSpace{}, err
	}
	return DiskSpace{
		TotalBytes:  usage.Total,
		FreeBytes:   usage.Free,
		UsedBytes:   usage.Used,
		UsedPercent: usage.UsedPercent,
	}, nil
}
