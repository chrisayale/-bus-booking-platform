---
name: fix_created_ticket
description: Fixed the CreatedTicket struct to include missing db tags for SQL query mapping
metadata:
  type: project
---

Added db tags to the CreatedTicket struct in internal/whatsapp/booking.go to map struct fields to database columns returned by the query in ticket_utils.go. Specifically:

- Added `db:"destination"` to Destination field
- Added `db:"seat_number"` to SeatNumber field
- Added `db:"passenger_name"` to PassengerName field
- Added `db:"passenger_phone"` to PassengerPhone field
- Added `db:"price"` to Price field
- Added `db:"download_token"` to DownloadToken field
- Added `db:"qr_token"` to QRToken field
- Added `db:"-"` to DownloadURL field (not selected in query)

This resolves the error: "missing destination name passenger_name in *[]whatsapp.CreatedTicket"

Verified that all columns in the SQL query have corresponding struct fields with appropriate db tags.