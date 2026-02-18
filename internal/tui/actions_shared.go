package tui

import "strings"

// --- Shared action helpers ---

func firstSelectableCursor(items []chezmoiActionItem) int {
	for i, item := range items {
		if isChezmoiActionSelectable(item) {
			return i
		}
	}
	return 0
}

func nextSelectableCursor(items []chezmoiActionItem, current, delta int) int {
	if len(items) == 0 {
		return 0
	}
	idx := current
	for {
		idx += delta
		if idx < 0 || idx >= len(items) {
			return current
		}
		if isChezmoiActionSelectable(items[idx]) {
			return idx
		}
	}
}

func appendActionItem(items []chezmoiActionItem, label string, action chezmoiAction, desc string, enabled bool, reason string) []chezmoiActionItem {
	item := chezmoiActionItem{
		label:       label,
		description: desc,
		action:      action,
	}
	if !enabled {
		item.label = actionLabelWithReason(label, reason)
		item.disabled = true
		item.unavailableReason = reason
	}
	return append(items, item)
}

func appendActionItemWithCapability(items []chezmoiActionItem, label string, action chezmoiAction, avail capability) []chezmoiActionItem { //nolint:unparam // label will vary once more actions use capabilities
	item := chezmoiActionItem{
		label:  label,
		action: action,
	}
	if !avail.Available {
		item.label = actionLabelWithReason(label, avail.Reason)
		item.disabled = true
		item.unavailableReason = avail.Reason
	}
	return append(items, item)
}

func isChezmoiActionSelectable(item chezmoiActionItem) bool {
	return item.action != chezmoiActionNone && !item.disabled
}

// --- Action label/message formatting ---

func actionLabelWithReason(label, reason string) string {
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return label + " (unavailable)"
	}
	return label + " (unavailable: " + trimmed + ")"
}

func actionUnavailableMessage(reason string) string {
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return "Unavailable in this environment"
	}
	return "Unavailable: " + trimmed
}
