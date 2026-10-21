# Formatting Fixes Summary

## Issues Fixed

### 1. Departure and Arrival Times Formatting
**Problem**: In the ticket details view, departure and arrival times were displaying as full timestamps (e.g., "0000-01-01T18:30:00Z") instead of just the time in HH:MM format.

**Solution**: 
- Added a new `formatTimeHHMM` function that converts time strings to "HH:MM" format
- This function handles both regular time formats and potential full timestamp formats
- Updated the ticket details display in `StepMyTicketsList` to use this function for both departure and arrival times

### 2. PDF Download Link Empty
**Problem**: In the ticket details view, the PDF download link was appearing empty.

**Solution**:
- In the verification flow (`StepVerificationMenu` case "1"), after querying the database for tickets, added a loop to construct the DownloadURL for each ticket
- The DownloadURL is constructed as "%s/tickets/download/%s" using AppURL and the ticket's DownloadToken
- This ensures that when displaying ticket details in `StepMyTicketsList`, the DownloadURL field is properly populated

## Files Modified
- `internal/whatsapp/handler.go`

## Changes Made

### Added Function
```go
// formatTimeHHMM formats a time string to HH:MM format
func formatTimeHHMM(timeStr string) string {
	if timeStr == "" {
		return ""
	}

	// Handle potential full timestamp format (e.g., "0000-01-01T18:30:00Z" or "2026-01-01 18:30:00")
	if strings.Contains(timeStr, "T") {
		// ISO format: "2006-01-02T15:04:05Z"
		parts := strings.Split(timeStr, "T")
		if len(parts) >= 2 {
			timeStr = parts[1]
			// Remove timezone part if present
			if strings.Contains(timeStr, "+") || strings.Contains(timeStr, "-") || strings.Contains(timeStr, "Z") {
				timePart := strings.Split(timeStr, ".")[0] // Remove milliseconds
				timePart = strings.Split(timePart, "+")[0]
				timePart = strings.Split(timePart, "-")[0]
				timePart = strings.Split(timePart, "Z")[0]
				timeStr = timePart
			}
		}
	} else if strings.Contains(timeStr, " ") {
		// Space separated format: "2006-01-02 15:04:05"
		parts := strings.Split(timeStr, " ")
		if len(parts) >= 2 {
			timeStr = parts[1]
		}
	}

	// Now extract HH:MM
	timeParts := strings.Split(timeStr, ":")
	if len(timeParts) >= 2 {
		hour := timeParts[0]
		minute := timeParts[1]
		// Pad with leading zeros if needed
		if len(hour) == 1 {
			hour = "0" + hour
		}
		if len(minute) == 1 {
			minute = "0" + minute
		}
		return fmt.Sprintf("%s:%s", hour, minute)
	}

	// If we can't parse it, return original
	return timeStr
}
```

### Fixed DownloadURL Construction
```go
// Construct DownloadURL for each ticket since it's not included in the SQL query
for i := range tickets {
	tickets[i].DownloadURL = fmt.Sprintf("%s/tickets/download/%s", AppURL, tickets[i].DownloadToken)
}
```

### Fixed Time Display in Ticket Details
```go
b.WriteString(fmt.Sprintf("• *Départ* : %s à %s\n", selectedTicket.Date, formatTimeHHMM(selectedTicket.DepartureTime)))
b.WriteString(fmt.Sprintf("• *Arrivée* : %s à %s\n", selectedTicket.ArrivalDate, formatTimeHHMM(selectedTicket.ArrivalTime)))
b.WriteString(fmt.Sprintf("%s\n\n", selectedTicket.DownloadURL))
```

## Testing
The fixes have been applied and should resolve both formatting issues:
1. Departure and arrival times now show in HH:MM format (e.g., "18:30")
2. PDF download links now properly display the full URL for downloading tickets