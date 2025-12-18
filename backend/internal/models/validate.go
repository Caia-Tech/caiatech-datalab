package models

import "strings"

func NormalizeSplit(s string) (Split, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	split := Split(s)
	switch split {
	case SplitTrain, SplitValid, SplitTest:
		return split, true
	default:
		return "", false
	}
}

func NormalizeConversationStatus(s string) (ConversationStatus, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	st := ConversationStatus(s)
	switch st {
	case ConversationStatusDraft, ConversationStatusPending, ConversationStatusApproved, ConversationStatusRejected, ConversationStatusArchived:
		return st, true
	default:
		return "", false
	}
}
