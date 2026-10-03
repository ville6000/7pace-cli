package api

// WorkLog is the request/response body for the 7pace Timetracker REST
// worklog endpoint. Length is in seconds. A worklog must have either a
// comment or an associated work item, and Length must be greater than 0.
type WorkLog struct {
	Timestamp    string       `json:"timestamp"`
	Length       int          `json:"length"`
	WorkItemID   *int         `json:"workItemID,omitempty"`
	Comment      string       `json:"comment,omitempty"`
	ActivityType *ActivityRef `json:"activityType,omitempty"`
}

// ActivityRef refers to a 7pace activity type by ID.
type ActivityRef struct {
	ID string `json:"id"`
}
