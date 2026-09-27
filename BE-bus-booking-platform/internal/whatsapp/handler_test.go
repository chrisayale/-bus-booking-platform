package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// mockClient implements a subset of whatsmeow.Client needed for testing.
type mockClient struct {
	SentMessages []struct {
		To      types.JID
		Message string
	}
	UploadedImages []struct {
		Data       []byte
		Format     whatsmeow.MediaType
		Caption    string
		URL        string
		DirectPath string
		MediaKey   []byte
		SHA256     []byte
		FileSHA256 []byte
		FileLength uint64
	}
}

// SendMessage records the message and returns nil error.
func (m *mockClient) SendMessage(ctx context.Context, to types.JID, msg *waE2E.Message, _ ...whatsmeow.SendRequestExtra) (whatsmeow.SendResponse, error) {
	if msg.GetConversation() != "" {
		m.SentMessages = append(m.SentMessages, struct {
			To      types.JID
			Message string
		}{To: to, Message: msg.GetConversation()})
	}
	// Return a dummy SendResponse.
	return whatsmeow.SendResponse{}, nil
}

// Upload records the upload and returns a dummy upload result.
func (m *mockClient) Upload(ctx context.Context, data []byte, _ whatsmeow.MediaType) (*whatsmeow.UploadedFile, error) {
	m.UploadedImages = append(m.UploadedImages, struct {
		Data       []byte
		Format     whatsmeow.MediaType
		Caption    string
		URL        string
		DirectPath string
		MediaKey   []byte
		SHA256     []byte
		FileSHA256 []byte
		FileLength uint64
	}{Data: data})
	// Return a dummy uploaded file.
	return &whatsmeow.UploadedFile{
		URL:           "https://dummy.url/image.png",
		DirectPath:    "direct/path",
		MediaKey:      []byte{0x01, 0x02, 0x03},
		FileEncSHA256: []byte{0x04, 0x05, 0x06},
		FileSHA256:    []byte{0x07, 0x08, 0x09},
		FileLength:    uint64(len(data)),
	}, nil
}

// mockDB implements a subset of sqlx.DB needed for testing.
type mockDB struct {
	// We'll define methods that return predefined results.
	SearchJourneysResult         []JourneyResult
	SearchJourneysError          error
	SearchUpcomingJourneysResult []JourneyResult
	SearchUpcomingJourneysError  error
	SearchJourneysOnDateResult   []JourneyResult
	SearchJourneysOnDateError    error
	GetBookedSeatsResult         []string
	GetBookedSeatsError          error
	// For CreateGuestBooking, we need to mock multiple queries.
	// We'll mock the Get for journey, the Get for count, the Get for base_price, and the Exec.
	// We'll use a flag to track which call we are on.
	createGuestBookingStep int
	// We'll store the journey for the Get journey call.
	createGuestBookingJourney models.Journey
	// We'll store the base price for the Get base_price call.
	createGuestBookingBasePrice float64
}

// We need to implement the methods that the handler uses.
// Note: The handler uses DB.Select and DB.Get for some queries.
// We'll implement Select and Get to call the appropriate mock functions based on the query.
// For simplicity, we will assume that the test only uses the mocked functions above and not the raw SQL.
// However, the handler uses DB.Select in getActiveAgencies, getVoyagesByAgencyID, etc.
// Since we are testing the direct journey search, we don't need those.
// We'll implement Select and Get to return an error if called, so we can see if they are unexpectedly used.
func (m *mockDB) Select(dest interface{}, query string, args ...interface{}) error {
	// For simplicity, we will handle only the queries we expect in the test.
	// We'll check the query string to determine what to return.
	if strings.Contains(query, "SELECT j.id, j.organization_id, j.route_id, j.bus_id") && strings.Contains(query, "FROM journeys j") && strings.Contains(query, "JOIN routes r ON j.route_id = r.id") && strings.Contains(query, "JOIN organizations o ON j.organization_id = o.id") && strings.Contains(query, "WHERE r.origin ILIKE") && strings.Contains(query, "AND r.destination ILIKE") && strings.Contains(query, "AND j.departure_date = ") {
		// This is the SearchJourneys query.
		if m.SearchJourneysError != nil {
			return m.SearchJourneysError
		}
		// We need to set the dest to a slice of JourneyResult.
		// We assume dest is a pointer to a slice.
		slicePtr := dest.(*[]JourneyResult)
		*slicePtr = m.SearchJourneysResult
		return nil
	}
	if strings.Contains(query, "SELECT j.id, j.organization_id, j.route_id, j.bus_id") && strings.Contains(query, "WHERE r.origin ILIKE") && strings.Contains(query, "AND r.destination ILIKE") && strings.Contains(query, "AND j.departure_date >= CURRENT_DATE") {
		// This is the SearchUpcomingJourneys query.
		if m.SearchUpcomingJourneysError != nil {
			return m.SearchUpcomingJourneysError
		}
		slicePtr := dest.(*[]JourneyResult)
		*slicePtr = m.SearchUpcomingJourneysResult
		return nil
	}
	if strings.Contains(query, "SELECT j.id, j.organization_id, j.route_id, j.bus_id") && strings.Contains(query, "WHERE j.departure_date = $1") {
		// This is the SearchJourneysOnDate query.
		if m.SearchJourneysOnDateError != nil {
			return m.SearchJourneysOnDateError
		}
		slicePtr := dest.(*[]JourneyResult)
		*slicePtr = m.SearchJourneysOnDateResult
		return nil
	}
	if strings.Contains(query, "SELECT seat_number FROM tickets WHERE journey_id=$1 AND status IN ('active', 'checked_in')") {
		// This is the GetBookedSeats query.
		if m.GetBookedSeatsError != nil {
			return m.GetBookedSeatsError
		}
		slicePtr := dest.(*[]string)
		*slicePtr = m.GetBookedSeatsResult
		return nil
	}
	// If we don't recognize the query, return an error.
	return fmt.Errorf("unexpected Select query: %s", query)
}

func (m *mockDB) Get(dest interface{}, query string, args ...interface{}) error {
	// We'll handle the Get queries for CreateGuestBooking.
	if strings.Contains(query, "SELECT * FROM journeys WHERE id=$1 FOR UPDATE") {
		// This is the Get journey call in CreateGuestBooking.
		if m.createGuestBookingStep != 0 {
			return fmt.Errorf("unexpected Get journey call at step %d", m.createGuestBookingStep)
		}
		m.createGuestBookingStep = 1
		// We need to set dest to a pointer to a models.Journey.
		journeyPtr := dest.(*models.Journey)
		*journeyPtr = m.createGuestBookingJourney
		return nil
	}
	if strings.Contains(query, "SELECT COUNT(*) FROM tickets WHERE journey_id=$1 AND seat_number=$2 AND status IN ('active', 'checked_in')") {
		// This is the Get count call in CreateGuestBooking.
		if m.createGuestBookingStep != 1 {
			return fmt.Errorf("unexpected Get count call at step %d", m.createGuestBookingStep)
		}
		m.createGuestBookingStep = 2
		// We need to set dest to a pointer to an int.
		countPtr := dest.(*int)
		*countPtr = 0 // Assume no seats are taken.
		return nil
	}
	if strings.Contains(query, "SELECT base_price FROM routes WHERE id=$1") {
		// This is the Get base_price call in CreateGuestBooking.
		if m.createGuestBookingStep != 2 {
			return fmt.Errorf("unexpected Get base_price call at step %d", m.createGuestBookingStep)
		}
		m.createGuestBookingStep = 3
		// We need to set dest to a pointer to a float64.
		pricePtr := dest.(*float64)
		*pricePtr = m.createGuestBookingBasePrice
		return nil
	}
	// If we don't recognize the query, return an error.
	return fmt.Errorf("unexpected Get query: %s", query)
}

// We also need to mock the Exec method for the inserts in CreateGuestBooking.
// However, the handler does not check the result of Exec, so we can ignore it.
// We'll implement Exec to do nothing and return nil.
func (m *mockDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	// Return a dummy result.
	return sql.ResultFunc(func() (int64, error) {
		return 0, nil
	}), nil
}

// We also need to mock the Beginx method for the transaction in CreateGuestBooking.
func (m *mockDB) Beginx() (*sqlx.Tx, error) {
	// Return a dummy transaction.
	return &sqlx.Tx{}, nil
}

// We also need to mock the QueryRowx method for the insert booking in CreateGuestBooking.
func (m *mockDB) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	// Return a dummy row.
	return sqlx.NewRow(nil, nil)
}

// We also need to mock the Scan method on the row.
func (m *mockDB) Scan(dest ...interface{}) error {
	// We'll assume this is for the booking ID.
// We'll set the first argument (bookingID) to a dummy value.
	if len(dest) > 0 {
		*dest.(*string) = "dummybookingid"
	}
	return nil
}

// We also need to mock the Commit method.
func (m *mockDB) Commit() error {
	return nil
}

// We also need to mock the Rollback method.
func (m *mockDB) Rollback() error {
	return nil
}

// We also need to mock the GetCurrentQR and setCurrentQR functions? Not used in the flow.

// We also need to mock the RegisterHandlers function? It is called in the Connect function, but we are not calling Connect in the test.

// We also need to mock the utils.GenerateReference and utils.GenerateQRToken and utils.GenerateDownloadToken?
// We'll leave them as is and hope they don't break the test. Alternatively, we can mock them, but for simplicity we'll assume they work.

// We also need to mock the models.NullString? We'll leave it.

// We also need to mock the whatsmeow.SendResponse type? We already used it.

// We also need to mock the whatsmeow.UploadedFile type? We already used it.

// We also need to mock the whatsmeow.SendRequestExtra type? We used it in the SendMessage signature.

// Now, we need to mock the global variables.
// We'll save the originals and restore them after the test.
var (
	originalClient WhatsAppClient
	originalDB     *sqlx.DB
	originalAppURL string
)

func TestDirectJourneySearchFlow(t *testing.T) {
	// Setup mocks
	mockCli := &mockClient{}
	mockDB := &mockDB{}

	// Save originals
	originalClient = Client
	originalDB = DB
	originalAppURL = AppURL

	// Replace with mocks
	Client = mockCli
	DB = mockDB
	AppURL = "http://localhost.test"

	// Restore after test
	defer func() {
		Client = originalClient
		DB = originalDB
		AppURL = originalAppURL
	}()

	// Define test data
	testPhone := "+243999888777"
	testSender := types.JID{User: "243999888777"} // Assuming the phone number is the user part

	// Mock the database functions for the direct journey search.
	// We expect SearchJourneys to be called with origin, destination, date.
	mockDB.SearchJourneysResult = []JourneyResult{
		{
			ID:              "journey1",
			Origin:          "Kinshasa",
			Destination:     "Goma",
			AgencyName:      "AyaleTrans",
			Date:            time.Now().Format("2006-01-02"),
			Time:            "08:00:00",
			ArrivalTime:     "14:00:00",
			Price:           10.0,
			BusRegistration: "ABC123",
			AgencyPhone:     "+243999000111",
		},
		{
			ID:              "journey2",
			Origin:          "Kinshasa",
			Destination:     "Goma",
			AgencyName:      "Voyages du Congo",
			Date:            time.Now().Format("2006-01-02"),
			Time:            "13:00:00",
			ArrivalTime:     "19:00:00",
			Price:           12.0,
			BusRegistration: "XYZ789",
			AgencyPhone:     "+243999000222",
		},
	}
	mockDB.SearchJourneysError = nil

	// We also need to mock GetBookedSeats to return an empty slice (no seats booked yet).
	mockDB.GetBookedSeatsResult = []string{}
	mockDB.GetBookedSeatsError = nil

	// We also need to mock the CreateGuestBooking function.
	// We'll set up the mockDB to return a dummy journey and base price.
	mockDB.createGuestBookingJourney = models.Journey{
		RouteID: "route1",
	}
	mockDB.createGuestBookingBasePrice = 10.0

	// We also need to mock the session management.
	// The getSession and resetSession functions use a global session store.
	// We can either mock that or rely on the existing implementation.
	// For simplicity, we will use the existing in-memory session store (which is a map).
	// We'll clear the session for the test phone before starting.
	resetSession(testPhone)

	// Now, simulate the flow step by step.

	// Helper function to send a message and get the response.
	send := func(text string) string {
		// Get the current session for the phone.
		sess := getSession(testPhone)
		// Process the message.
		resp := process(testPhone, sess, text, testSender)
		return resp
	}

	// Step 1: User sends "1" (menu option 1)
	resp := send("1")
	if !strings.Contains(resp, "Ville de départ ?") {
		t.Errorf("Expected response to contain 'Ville de départ ?', got: %q", resp)
		t.FailNow()
	}

	// Step 2: User sends origin
	resp = send("Kinshasa")
	if !strings.Contains(resp, "Ville de destination ?") {
		t.Errorf("Expected response to contain 'Ville de destination ?', got: %q", resp)
		t.FailNow()
	}

	// Step 3: User sends destination
	resp = send("Goma")
	if !strings.Contains(resp, "Quelle date ?") {
		t.Errorf("Expected response to contain 'Quelle date ?', got: %q", resp)
		t.FailNow()
	}
	if !strings.Contains(resp, "Aujourd'hui") {
		t.Errorf("Expected response to contain 'Aujourd'hui', got: %q", resp)
		t.FailNow()
	}
	if !strings.Contains(resp, "Saisir une date") {
		t.Errorf("Expected response to contain 'Saisir une date', got: %q", resp)
		t.FailNow()
	}

	// Step 4: User chooses today (sends "1")
	resp = send("1")
	if !strings.Contains(resp, "Liste des voyages disponibles") {
		t.Errorf("Expected response to contain 'Liste des voyages disponibles', got: %q", resp)
		t.FailNow()
	}
	if !strings.Contains(resp, "Kinshasa → Goma") {
		t.Errorf("Expected response to contain 'Kinshasa → Goma', got: %q", resp)
		t.FailNow()
	}
	if !strings.Contains(resp, "AyaleTrans") {
		t.Errorf("Expected response to contain 'AyaleTrans', got: %q", resp)
		t.FailNow()
	}
	if !strings.Contains(resp, "Tapez le numéro du voyage choisi") {
		t.Errorf("Expected response to contain 'Tapez le numéro du voyage choisi', got: %q", resp)
		t.FailNow()
	}

	// Step 5: User selects the first journey (sends "1")
	resp = send("1")
	if !strings.Contains(resp, "Combien de sièges voulez-vous réserver ?") {
		t.Errorf("Expected response to contain 'Combien de sièges voulez-vous réserver ?', got: %q", resp)
		t.FailNow()
	}

	// Step 6: User enters number of seats (2)
	resp = send("2")
	if !strings.Contains(resp, "Passager 1/2 — Nom complet ?") {
		t.Errorf("Expected response to contain 'Passager 1/2 — Nom complet ?', got: %q", resp)
		t.FailNow()
	}

	// Step 7: User enters name for passenger 1
	resp = send("John Doe")
	if !strings.Contains(resp, "Numéro de téléphone de ce passager") {
		t.Errorf("Expected response to contain 'Numéro de téléphone de ce passager', got: %q", resp)
		t.FailNow()
	}

	// Step 8: User enters phone for passenger 1 (valid)
	resp = send("243111222333")
	if !strings.Contains(resp, "Passager 2/2 — Nom complet ?") {
		t.Errorf("Expected response to contain 'Passager 2/2 — Nom complet ?', got: %q", resp)
		t.FailNow()
	}

	// Step 9: User enters name for passenger 2
	resp = send("Jane Doe")
	if !strings.Contains(resp, "Numéro de téléphone de ce passager") {
		t.Errorf("Expected response to contain 'Numéro de téléphone de ce passager', got: %q", resp)
		t.FailNow()
	}

	// Step 10: User enters phone for passenger 2 (valid)
	resp = send("243444555666")
	if !strings.Contains(resp, "Récapitulatif :") {
		t.Errorf("Expected response to contain 'Récapitulatif :', got: %q", resp)
		t.FailNow()
	}
	if !strings.Contains(resp, "John Doe") {
		t.Errorf("Expected response to contain 'John Doe', got: %q", resp)
		t.FailNow()
	}
	if !strings.Contains(resp, "Jane Doe") {
		t.Errorf("Expected response to contain 'Jane Doe', got: %q", resp)
		t.FailNow()
	}
	if !strings.Contains(resp, "Confirmez-vous la réservation ?") {
		t.Errorf("Expected response to contain 'Confirmez-vous la réservation ?', got: %q", resp)
		t.FailNow()
	}

	// Step 11: User confirms with "oui"
	resp = send("oui")
	if !strings.Contains(resp, "Comment voulez-vous payer ?") {
		t.Errorf("Expected response to contain 'Comment voulez-vous payer ?', got: %q", resp)
		t.FailNow()
	}
	if !strings.Contains(resp, "Cash (à bord ou au guichet)") {
		t.Errorf("Expected response to contain 'Cash (à bord ou au guichet)', got: %q", resp)
		t.FailNow()
	}
	if !strings.Contains(resp, "Paiement en ligne (carte bancaire)") {
		t.Errorf("Expected response to contain 'Paiement en ligne (carte bancaire)', got: %q", resp)
		t.FailNow()
	}

	// Step 12: User chooses cash (sends "1")
	resp = send("1")
	// We expect two messages: one for cash confirmation and one for final message.
	// However, the process function returns only the last message to be sent.
	// Actually, the cash branch sends two messages but returns the final message.
	// We can check the sent messages in the mock client.
	// We expect at least two messages sent: the cash confirmation and the final message.
	if len(mockCli.SentMessages) != 2 {
		t.Errorf("Expected 2 messages sent, got %d", len(mockCli.SentMessages))
		t.FailNow()
	}
	// Check the first message (cash confirmation)
	if !strings.Contains(mockCli.SentMessages[0].Message, "Votre réservation a bien été enregistrée") {
		t.Errorf("Expected first message to contain cash confirmation, got: %q", mockCli.SentMessages[0].Message)
		t.FailNow()
	}
	// Check the second message (final message)
	if !strings.Contains(mockCli.SentMessages[1].Message, "Toute l'équipe de AyaleTrans vous souhaite un très bon voyage") {
		t.Errorf("Expected second message to contain final message, got: %q", mockCli.SentMessages[1].Message)
		t.FailNow()
	}

	// Additionally, we can check that the session was reset (by checking that getSession returns a nil or zero session).
	// But note: the resetSession function is called in the cash branch after sending the cash message and before returning the final message.
	// However, the session is reset before we return, so the session for the phone should be cleared.
	// We can check that the session step is StepMenu (the default after reset).
	sess := getSession(testPhone)
	if sess.Step != StepMenu {
		t.Errorf("Expected session to be reset to menu step, got: %d", sess.Step)
		t.FailNow()
	}

	// Test passed.
	t.Log("Direct journey search flow with cash payment passed.")
}