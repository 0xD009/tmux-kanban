package core

type ReviewItem struct {
	SessionKey  string
	HostName    string
	SessionName string
	Agent       string
	Target      string
}

type ReviewQueue struct {
	Cursor    int
	CursorKey string
	Skipped   map[string]bool
}

func (q ReviewQueue) CursorIndex(keys []string) int {
	if len(keys) == 0 {
		return 0
	}
	if q.CursorKey != "" {
		for i, key := range keys {
			if key == q.CursorKey {
				return i
			}
		}
	}
	if q.Cursor < 0 {
		return 0
	}
	if q.Cursor >= len(keys) {
		return len(keys) - 1
	}
	return q.Cursor
}

func (q ReviewQueue) Clamp(keys []string) ReviewQueue {
	if len(keys) == 0 {
		q.Cursor = 0
		q.CursorKey = ""
		return q
	}
	index := q.CursorIndex(keys)
	q.Cursor = index
	q.CursorKey = keys[index]
	return q
}

func (q ReviewQueue) Move(keys []string, delta int) (ReviewQueue, bool) {
	if len(keys) == 0 {
		q.Cursor = 0
		q.CursorKey = ""
		return q, false
	}
	current := q.CursorIndex(keys)
	next := current + delta
	if next < 0 {
		next = 0
	}
	if next >= len(keys) {
		next = len(keys) - 1
	}
	q.Cursor = next
	q.CursorKey = keys[next]
	return q, next != current
}

func (q ReviewQueue) AdvanceAfter(keys []string, currentKey string) ReviewQueue {
	if len(keys) == 0 {
		q.Cursor = 0
		q.CursorKey = ""
		return q
	}

	next := q.Cursor
	if currentKey != "" {
		for i, key := range keys {
			if key == currentKey {
				next = i + 1
				break
			}
		}
	}
	if next >= len(keys) {
		next = 0
	}
	if next < 0 {
		next = 0
	}
	q.Cursor = next
	q.CursorKey = keys[next]
	return q
}

func (q ReviewQueue) Skip(key string) ReviewQueue {
	if q.Skipped == nil {
		q.Skipped = map[string]bool{}
	}
	q.Skipped[key] = true
	return q
}

func (q ReviewQueue) UnskipAll() ReviewQueue {
	q.Skipped = map[string]bool{}
	return q
}

func FilterSkipped(items []ReviewItem, skipped map[string]bool) []ReviewItem {
	if len(skipped) == 0 {
		return items
	}
	queue := make([]ReviewItem, 0, len(items))
	for _, item := range items {
		if !skipped[item.SessionKey] {
			queue = append(queue, item)
		}
	}
	return queue
}
