package tmuxscan

import (
	"fmt"
	"sort"
	"strings"
)

func Parse(output string) ([]Session, error) {
	sessionsByID := map[string]*Session{}
	windowsByID := map[string]*Window{}
	panesByPID := map[string]*Pane{}
	sessionOrder := make([]string, 0)
	windowOrder := map[string][]string{}
	windowPanes := map[string][]*Pane{}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := splitRecord(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "S":
			if len(fields) < 5 {
				return nil, fmt.Errorf("malformed tmux session record: %q", line)
			}

			session := &Session{
				ID:       fields[1],
				Name:     fields[2],
				Attached: parseInt(fields[4]),
			}
			sessionsByID[session.ID] = session
			sessionOrder = append(sessionOrder, session.ID)

		case "W":
			if len(fields) < 6 {
				return nil, fmt.Errorf("malformed tmux window record: %q", line)
			}

			sessionID := fields[1]
			window := &Window{
				ID:     fields[2],
				Index:  fields[3],
				Name:   fields[4],
				Active: parseBool(fields[5]),
			}
			windowsByID[window.ID] = window
			windowOrder[sessionID] = append(windowOrder[sessionID], window.ID)

		case "P":
			if len(fields) < 8 {
				return nil, fmt.Errorf("malformed tmux pane record: %q", line)
			}

			windowID := fields[2]
			if _, ok := windowsByID[windowID]; !ok {
				continue
			}

			pane := parsePane(fields)
			panesByPID[pane.PID] = pane
			windowPanes[windowID] = append(windowPanes[windowID], pane)

		case "R":
			if len(fields) < 5 {
				return nil, fmt.Errorf("malformed process record: %q", line)
			}

			panePID := fields[1]
			pane, ok := panesByPID[panePID]
			if !ok {
				continue
			}

			pane.Processes = append(pane.Processes, Process{
				PID:     fields[2],
				Command: fields[3],
				Args:    fields[4],
			})
		}
	}

	for windowID, panes := range windowPanes {
		sort.SliceStable(panes, func(i, j int) bool {
			return numericLess(panes[i].Index, panes[j].Index)
		})

		window, ok := windowsByID[windowID]
		if !ok {
			continue
		}

		for _, pane := range panes {
			pane.Agent = DetectAgent(*pane)
			window.Panes = append(window.Panes, *pane)
		}
	}

	for sessionID, ids := range windowOrder {
		session, ok := sessionsByID[sessionID]
		if !ok {
			continue
		}

		sort.SliceStable(ids, func(i, j int) bool {
			left := windowsByID[ids[i]]
			right := windowsByID[ids[j]]
			return numericLess(left.Index, right.Index)
		})

		for _, windowID := range ids {
			if window, ok := windowsByID[windowID]; ok {
				session.Windows = append(session.Windows, *window)
			}
		}
	}

	sessions := make([]Session, 0, len(sessionOrder))
	for _, sessionID := range sessionOrder {
		session, ok := sessionsByID[sessionID]
		if ok {
			sessions = append(sessions, *session)
		}
	}

	return sessions, nil
}

func parsePane(fields []string) *Pane {
	if len(fields) >= 9 {
		return &Pane{
			ID:          fields[3],
			Index:       fields[4],
			PID:         fields[5],
			Command:     fields[6],
			CurrentPath: fields[7],
			Active:      parseBool(fields[8]),
		}
	}

	return &Pane{
		ID:          fields[3],
		Index:       fields[4],
		Command:     fields[5],
		CurrentPath: fields[6],
		Active:      parseBool(fields[7]),
	}
}
