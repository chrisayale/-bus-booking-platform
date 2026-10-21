# Complete Formatting Fixes Summary

## Issues Addressed

### Original Issues from User:
1. **Departure and Arrival Times Formatting**: Times were displaying as full timestamps instead of HH:MM format
2. **PDF Download Link Empty**: The PDF download link was appearing empty in ticket details

### Additional Requests from User:
1. **Departure and Arrival Dates Formatting**: Dates should show only the date part (not timestamp)
2. **Add Passenger Phone Number**: Include the passenger's phone number in the ticket details display

## Files Modified
- `internal/whatsapp/handler.go`

## Changes Made

### 1. Enhanced formatDate Function (Lines 185-217)
Enhanced to handle timestamp formats and extract just the date part before converting to DD/MM/YYYY format:
```go
func formatDate(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	// Handle potential timestamp formats and extract just the date part (YYYY-MM-DD)
	// Formats to handle:
	// - "2026-10-05T00:00:00Z" (ISO format)
	// - "2026-10-05 00:00:00" (space separated)
	// - "2026-10-05" (already just date)

	datePart := dateStr
	if strings.Contains(dateStr, "T") {
		// ISO format: "2006-01-02T15:04:05Z"
		parts := strings.Split(dateStr, "T")
		if len(parts) >= 2 {
			datePart = parts[0]
		}
	} else if strings.Contains(dateStr, " ") {
		// Space separated format: "2006-01-02 15:04:05"
		parts := strings.Split(dateStr, " ")
		if len(parts) >= 2 {
			datePart = parts[0]
		}
	}

	// Now split by '-' and rearrange to DD/MM/YYYY format
	parts := strings.Split(datePart, "-")
	if len(parts) != 3 {
		// If we still don't have 3 parts, return original
		return dateStr
	}
	return fmt.Sprintf("%s/%s/%s", parts[2], parts[1], parts[0])
}
```

### 2. Added formatTimeHHMM Function (Lines 219-260)
Formats time strings to HH:MM format, handling both regular times and potential timestamps:
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

### 3. Fixed DownloadURL Construction (Lines 1173-1176)
Added loop to construct DownloadURL for each ticket in the verification flow:
```go
// Construct DownloadURL for each ticket since it's not included in the SQL query
for i := range tickets {
	tickets[i].DownloadURL = fmt.Sprintf("%s/tickets/download/%s", AppURL, tickets[i].DownloadToken)
}
```

### 4. Updated Ticket Details Display (Lines 1246-1263)
Enhanced the ticket details display in StepMyTicketsList to:
- Add passenger phone number line
- Use formatDate for date fields
- Keep formatTimeHHMM for time fields
- Match the user's requested template format

```go
// Format detailed ticket information according to requested format
var b strings.Builder
b.WriteString("🎫 *Détails du billet* :\n")
b.WriteString(fmt.Sprintf("• *Passager* : %s\n", selectedTicket.PassengerName))
b.WriteString(fmt.Sprintf("• *Téléphone* : %s\n", selectedTicket.PassengerPhone))
b.WriteString(fmt.Sprintf("• *Agence* : %s\n", selectedTicket.AgencyName))
b.WriteString(fmt.Sprintf("• *Trajet* : %s ➔ %s\n", selectedTicket.DepartureCity, selectedTicket.ArrivalCity))
b.WriteString(fmt.Sprintf("• *Départ* : %s à %s\n", formatDate(selectedTicket.Date), formatTimeHHMM(selectedTicket.DepartureTime)))
b.WriteString(fmt.Sprintf("• *Arrivée* : %s à %s\n", formatDate(selectedTicket.ArrivalDate), formatTimeHHMM(selectedTicket.ArrivalTime)))
b.WriteString(fmt.Sprintf("• *Siège* : N° %s\n", selectedTicket.SeatNumber))
b.WriteString(fmt.Sprintf("• *Prix* : %.2f USD\n", selectedTicket.Price))
b.WriteString(fmt.Sprintf("• *Paiement* : %s\n", selectedTicket.PaymentMethod))
b.WriteString(fmt.Sprintf("• *Statut* : %s\n\n", selectedTicket.Status))
b.WriteString("📥 *Télécharger le billet PDF* :\n")
b.WriteString(fmt.Sprintf("%s\n\n", selectedTicket.DownloadURL))
b.WriteString("Tapez *menu* pour retourner au menu principal.")
return b.String()
```

## Result Format
The ticket details now display exactly as requested:
```
🎫 *Détails du billet* :
• *Passager* : {{PassengerName}}
• *Téléphone* : {{PassengerPhone}}
• *Agence* : {{AgencyName}}
• *Trajet* : {{DepartureCity}} ➔ {{ArrivalCity}}
• *Départ* : {{DepartureDate}} à {{DepartureTime}}
• *Arrivée* : {{ArrivalDate}} à {{ArrivalTime}}
• *Siège* : N° {{SeatNumber}}
• *Prix* : {{Price}} {{Currency}}
• *Paiement* : {{PaymentMethod}}
• *Statut* : {{Status}}

📥 *Télécharger le billet PDF* :
{{DownloadURL}}

Tapez *menu* pour retourner au menu principal.
```

Where:
- Dates are formatted as DD/MM/YYYY (e.g., "05/10/2026")
- Times are formatted as HH:MM (e.g., "18:30")
- Currency is displayed as "USD" (hardcoded as per existing implementation)
- DownloadURL is properly constructed as "%s/tickets/download/%s"

## Verification
- The project builds successfully with `go build ./cmd/api`
- All formatting fixes are implemented as requested
- Backward compatibility is maintained for existing functionality