package client

import (
	"context"
	"net/url"
)

// PeriodicTask is a record of one run of a scheduled background job.
//
// Read-only: SecObserve writes these itself as jobs run, so there is nothing
// to manage here beyond listing them.
type PeriodicTask struct {
	ID        int64  `json:"id"`
	Task      string `json:"task"`
	StartTime string `json:"start_time"`
	Duration  *int64 `json:"duration"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

const periodicTasksPath = "api/periodic_tasks/"

// PeriodicTasks lists periodic task runs, optionally filtered by task name
// (icontains) and status, most recent first.
func (c *Client) PeriodicTasks(ctx context.Context, task, status string) ([]PeriodicTask, error) {
	query := url.Values{}
	if task != "" {
		query.Set("task", task)
	}
	if status != "" {
		query.Set("status", status)
	}
	return List[PeriodicTask](ctx, c, periodicTasksPath, query)
}
