package whatsapp

import "fmt"

func GetTicketsByPhone(phone string) ([]CreatedTicket, error) {
	var tickets []CreatedTicket
	err := DB.Select(&tickets, `
		SELECT
			t.id,
			COALESCE(t.passenger_name, 'Non spécifié') AS passenger_name,
			COALESCE(t.seat_number::text, 'N/A') AS seat_number,
			COALESCE(r.base_price, 0.00) AS price,
			COALESCE(t.download_token, '') AS download_token
		FROM tickets t
		JOIN bookings b ON t.booking_id = b.id
		JOIN journeys j ON t.journey_id = j.id
		JOIN routes r ON j.route_id = r.id
		WHERE t.passenger_phone LIKE '%' || $1 || '%'
		ORDER BY t.created_at DESC
		LIMIT 10
	`, phone)
	if err != nil {
		return nil, err
	}
	return tickets, nil
}

// GetTicketDetailsByID retrieves complete details for a specific ticket by its ID
func GetTicketDetailsByID(ticketID string) (*CreatedTicket, error) {
	var ticket CreatedTicket
	// Query to get complete ticket details including journey, route, and booking information
	// Using column aliases to match CreatedTicket struct field names
	// Note: Casting DATE and TIME fields to text to ensure proper string conversion
	err := DB.Get(&ticket, `
		SELECT
			t.seat_number,
			t.passenger_name,
			t.passenger_phone,
			t.price,
			t.status,
			t.qr_token,
			t.download_token,
			r.origin AS departure_city,
			r.destination AS arrival_city,
			j.departure_time::text,
			j.arrival_time::text,
			j.departure_date::text AS date,
			j.arrival_date::text AS arrival_date,
			o.name AS agency_name,
			b.booking_reference
		FROM tickets t
		JOIN journeys j ON t.journey_id = j.id
		JOIN routes r ON j.route_id = r.id
		JOIN organizations o ON j.organization_id = o.id
		JOIN bookings b ON t.booking_id = b.id
		WHERE t.id = $1
	`, ticketID)
	if err != nil {
		return nil, err
	}

	// Construct DownloadURL using AppURL
	if ticket.DownloadToken != "" {
		ticket.DownloadURL = fmt.Sprintf("%s/tickets/download/%s", AppURL, ticket.DownloadToken)
	}

	return &ticket, nil
}
