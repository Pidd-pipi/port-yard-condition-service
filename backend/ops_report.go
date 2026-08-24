package main

import (
	"context"
	"sort"
)

// OpsReport aggregates the operations records for a daily shift summary.
type OpsReport struct {
	Domain        string              `json:"domain"`
	GeneratedAt   string              `json:"generated_at"`
	Total         int                 `json:"total"`
	ByStatus      map[OpsStatus]int   `json:"by_status"`
	ByPriority    map[OpsPriority]int `json:"by_priority"`
	ActiveOwner   string              `json:"busiest_owner"`
	OldestActive  string              `json:"oldest_active_record"`
	UpdatedRecent int                 `json:"updated_last_hour"`
	TopOwners     []string            `json:"top_owners"`
}

// buildOpsReport computes a summary from the current store contents.
func buildOpsReport(service *OpsService) OpsReport {
	items, err := service.store.List(context.Background())
	if err != nil {
		return OpsReport{Domain: opsDomainName, GeneratedAt: service.clock.Stamp(), ByStatus: map[OpsStatus]int{}, ByPriority: map[OpsPriority]int{}}
	}
	report := OpsReport{
		Domain:      opsDomainName,
		GeneratedAt: service.clock.Stamp(),
		ByStatus:    map[OpsStatus]int{},
		ByPriority:  map[OpsPriority]int{},
	}
	oldest := ""
	for _, item := range items {
		report.Total++
		report.ByStatus[item.Status]++
		report.ByPriority[item.Priority]++
		if item.Status == OpsStatusActive {
			if oldest == "" || item.UpdatedAt < oldest {
				oldest = item.UpdatedAt
			}
		}
		if opsAge(service.clock.Now(), item.UpdatedAt) <= 3600 {
			report.UpdatedRecent++
		}
	}
	report.OldestActive = oldest
	report.TopOwners = topOwners(items, 5)
	if len(report.TopOwners) > 0 {
		report.ActiveOwner = report.TopOwners[0]
	}
	return report
}

// topOwners returns the owner names sorted by descending active record count.
func topOwners(items []OpsRecord, limit int) []string {
	counts := map[string]int{}
	for _, item := range items {
		if item.Status == OpsStatusActive {
			counts[item.Owner]++
		}
	}
	owners := make([]string, 0, len(counts))
	for owner := range counts {
		owners = append(owners, owner)
	}
	sort.Slice(owners, func(i, j int) bool {
		if counts[owners[i]] != counts[owners[j]] {
			return counts[owners[i]] > counts[owners[j]]
		}
		return owners[i] < owners[j]
	})
	if limit < 0 || limit > len(owners) {
		limit = len(owners)
	}
	return owners[:limit]
}
