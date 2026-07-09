package entity

// ValidTaskType reports whether s is a known task type string.
func ValidTaskType(s string) bool {
	switch TaskType(s) {
	case TaskTypeFeature, TaskTypeBug, TaskTypeReview,
		TaskTypeTriage, TaskTypeTest, TaskTypeResearch, TaskTypeChore:
		return true
	default:
		return false
	}
}

// ValidTaskStatus reports whether s is a known task status string.
func ValidTaskStatus(s string) bool {
	switch TaskStatus(s) {
	case TaskStatusPending, TaskStatusInProgress, TaskStatusAwaitingConfirmation,
		TaskStatusBlocked, TaskStatusDoneSuccess, TaskStatusDoneFailed, TaskStatusCancelled:
		return true
	default:
		return false
	}
}
