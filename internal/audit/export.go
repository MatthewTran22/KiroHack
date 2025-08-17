package audit

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// exportJSON exports an audit report as JSON
func (s *serviceImpl) exportJSON(report *AuditReport) ([]byte, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal report to JSON: %w", err)
	}
	return data, nil
}

// exportCSV exports an audit report as CSV
func (s *serviceImpl) exportCSV(report *AuditReport) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"ID",
		"Timestamp",
		"Event Type",
		"Level",
		"User ID",
		"Session ID",
		"IP Address",
		"User Agent",
		"Resource",
		"Action",
		"Result",
		"Request ID",
		"Duration (ms)",
		"Error Code",
		"Error Message",
		"Details",
	}

	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, entry := range report.Entries {
		row := []string{
			entry.ID,
			entry.Timestamp.Format(time.RFC3339),
			string(entry.EventType),
			string(entry.Level),
			stringPtrToString(entry.UserID),
			stringPtrToString(entry.SessionID),
			entry.IPAddress,
			entry.UserAgent,
			entry.Resource,
			entry.Action,
			entry.Result,
			entry.RequestID,
			durationPtrToString(entry.Duration),
			stringPtrToString(entry.ErrorCode),
			stringPtrToString(entry.ErrorMessage),
			mapToString(entry.Details),
		}

		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("failed to flush CSV writer: %w", err)
	}

	return buf.Bytes(), nil
}

// Helper functions for CSV export

func stringPtrToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func durationPtrToString(ptr *time.Duration) string {
	if ptr == nil {
		return ""
	}
	return strconv.FormatInt(ptr.Nanoseconds()/1000000, 10) // Convert to milliseconds
}

func mapToString(m map[string]interface{}) string {
	if m == nil {
		return ""
	}

	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	return string(data)
}
