package dto

import "github.com/quiqxiq/roslib-mikhmon/domain"

type LogEntryResponse struct {
	ID      string `json:"id"`
	Time    string `json:"time,omitempty"`
	Topics  string `json:"topics,omitempty"`
	Message string `json:"message"`
}

func FromDomainLogEntry(l domain.LogEntry) LogEntryResponse {
	return LogEntryResponse{
		ID: l.ID, Time: l.Time, Topics: l.Topics, Message: l.Message,
	}
}

func FromDomainLogEntries(ls []domain.LogEntry) []LogEntryResponse {
	out := make([]LogEntryResponse, len(ls))
	for i, l := range ls {
		out[i] = FromDomainLogEntry(l)
	}
	return out
}
