package meilisearch

import (
	"context"
	"fmt"
	"time"

	ms "github.com/meilisearch/meilisearch-go"

	"shared/pkg/search"
)

func (c *client) awaitTask(ctx context.Context, taskUID int64) *search.SearchError {
	ticker := time.NewTicker(c.cfg.TaskPollInterval)
	defer ticker.Stop()

	deadline := time.Now().Add(c.cfg.TaskPollTimeout)

	for {
		select {
		case <-ctx.Done():
			return search.ErrInternal("context cancelled while awaiting task", ctx.Err())

		case t := <-ticker.C:
			if t.After(deadline) {
				return search.NewError(search.ErrCodeTimeout, fmt.Sprintf("task %d did not complete within timeout", taskUID), nil)
			}

			task, err := c.ms.GetTaskWithContext(ctx, taskUID)
			if err != nil {
				return mapError(err)
			}

			switch task.Status {
			case ms.TaskStatusSucceeded:
				return nil
			case ms.TaskStatusFailed:
				return search.ErrInternal(task.Error.Message, fmt.Errorf("task %d failed: %s", taskUID, task.Error.Code))
			case ms.TaskStatusCanceled:
				return search.ErrInternal(fmt.Sprintf("task %d was cancelled", taskUID), nil)
			}
		}
	}
}
