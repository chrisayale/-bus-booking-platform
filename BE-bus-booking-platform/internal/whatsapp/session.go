package whatsapp

import "sync"

type Step string

const (
	StepMenu                   Step = "menu"
	StepSearchOrigin           Step = "search_origin"
	StepSearchDestination      Step = "search_destination"
	StepSearchDateChoice       Step = "search_date_choice"
	StepSearchDate             Step = "search_date"
	StepSearchResults          Step = "search_results"
	StepSeatsCount             Step = "seats_count"
	StepPassengerName          Step = "passenger_name"
	StepPassengerPhone         Step = "passenger_phone"
	StepConfirm                Step = "confirm"
	StepSearchAgency           Step = "search_agency"
	StepPhonePaymentChoice     Step = "payment_phone_choice"
	StepPaymentOtherPhoneInput Step = "payment_other_phone_input"
	StepPaymentProcessing      Step = "payment_processing"
	StepPaymentProcessingWait  Step = "payment_processing_wait"
	StepPaymentCompleted       Step = "payment_completed"
	StepCurrencyChoice         Step = "currency_choice"
	StepPaymentMethodChoice    Step = "payment_method_choice"
	StepMyTicketsPhone         Step = "my_tickets_phone"
	StepMyTicketsList          Step = "my_tickets_list"
	StepMyTicketDetails        Step = "my_ticket_details"
	StepAgencyChoice           Step = "agency_choice"
	StepAgencyMenu             Step = "agency_menu"
	StepAgencyDateChoice       Step = "agency_date_choice"
	StepAgencyDateInput        Step = "agency_date_input"
	StepAgencyVoyages          Step = "agency_voyages"
	// Agency creation flow
	StepAgencyName         Step = "agency_name"
	StepAgencyManagerName  Step = "agency_manager_name"
	StepAgencyAddress      Step = "agency_address"
	StepAgencyContactPhone Step = "agency_contact_phone"
	StepAgencyConfirm      Step = "agency_confirm"
	// Password change menu
	StepPasswordChangeMenu Step = "password_change_menu"
	// FlexPay phone number input
	StepFlexPayPhoneInput    Step = "flexpay_phone_input"
	// Payment processing in progress (to prevent reentrancy)
	StepPaymentProcessingInProgress Step = "payment_processing_in_progress"
	// Payment failed/expired states
	StepPaymentFailed        Step = "payment_failed"
	StepPaymentExpired       Step = "payment_expired"
	// Verification flow
	StepVerificationMenu     Step = "verification_menu"
	StepViewTickets          Step = "view_tickets"
	StepSelectTicketDetails  Step = "select_ticket_details"
	StepCheckTicketInput     Step = "check_ticket_input"
	StepAgentPinInput        Step = "agent_pin_input"
	StepAgentCashPayment     Step = "agent_cash_payment"
	StepAgentTicketEmission  Step = "agent_ticket_emission"
	StepPassengerCheckType   Step = "passenger_check_type"
	StepCheckCodeInput       Step = "check_code_input"
	StepCheckQRInput         Step = "check_qr_input"
	// Controller flow
	StepControllerInput               Step = "controller_input"
	StepControllerPin                 Step = "controller_pin"
	StepControllerMenu                Step = "controller_menu"
	StepControllerActions             Step = "controller_actions"
	StepControllerPaymentMethod       Step = "controller_payment_method"
	StepControllerInputBookingRef     Step = "controller_input_booking_ref"
	StepControllerInputPhone          Step = "controller_input_phone"
	StepControllerInputQRCode         Step = "controller_input_qr_code"
	StepControllerPaymentReferenceInput Step = "controller_payment_reference_input"
	StepControllerCheckInConfirm      Step = "controller_checkin_confirm"
	StepControllerCheckInProcess      Step = "controller_checkin_process"
	// Controller PIN authentication flow (new)
	StepControllerEnterPin            Step = "controller_enter_pin"
	StepControllerSelectAction        Step = "controller_select_action"
	StepControllerValidatePayment     Step = "controller_validate_payment"
	StepControllerDoCheckin           Step = "controller_do_checkin"
	StepControllerCashCheckinConfirm  Step = "controller_cash_checkin_confirm"
	StepControllerPayAllConfirm     Step = "controller_pay_all_confirm"
	StepControllerActionChoice        Step = "controller_action_choice"
	// Payment validation search sub-steps
	StepControllerSearchByRef         Step = "controller_search_by_ref"
	StepControllerSearchByPhone       Step = "controller_search_by_phone"
	StepControllerSearchByName        Step = "controller_search_by_name"
	StepControllerSearchByQR          Step = "controller_search_by_qr"
)

type Agency struct {
	ID   string
	Name string
}

type PassengerDraft struct {
	SeatNumber string
	Name       string
	Phone      string
}

type Session struct {
	Step                  Step
	PreviousStep          Step // for back navigation
	Origin                string
	Destination           string
	Date                  string
	Results               []JourneyResult
	SelectedJourneyID     string
	SelectedJourney       JourneyResult
	SelectedAgencyID      string
	SelectedAgency        Agency
	FilterAgencyID        string // agency ID filter for search (when searching within a specific agency)
	SeatsNeeded           int
	Passengers            []PassengerDraft
	CurrentPassengerIndex int
	PaymentPhone          string // phone number to use for Mobile Money (if empty, use WhatsApp number)
	PaymentMethod         string // "cash", "mobile", "card"
	Currency              string // "USD" or "CDF"
	OrderNumber           string // FlexPay order number for tracking payment status
	Agencies              []Agency
	AgencyPage            int    // current page for agency pagination (0-based)
	VoyagePage            int    // current page for voyage pagination (0-based)
	// Agency creation flow
	AgencyName   string
	ManagerName  string
	Address      string
	ContactPhone string
	// User's WhatsApp phone number (verified from session)
	UserPhone    string
	AgentID      string
	IsControllerAuthenticated bool
	// PaymentError stores the last payment error message for retry functionality
	PaymentError string
	// NotificationSent indicates whether a final notification (success or error) has been sent to avoid duplicates.
	NotificationSent bool
	// HasRetried indicates whether the user has already attempted a payment retry
	HasRetried     bool
	// Tickets stores the list of tickets for the current user when in my_tickets_list step
	Tickets        []CreatedTicket
	// Data stores additional session data as key-value pairs
	Data           map[string]interface{}
	AgentName                 string
	AssignedAgencies          []Agency
}

type JourneyResult struct {
	ID              string
	Origin          string
	Destination     string
	Date            string
	Time            string
	ArrivalDate     string
	ArrivalTime     string
	Price           float64
	AgencyName      string
	AgencyPhone     string
	BusRegistration string
}

var sessions = map[string]*Session{}
var sessionsMu sync.Mutex

func getSession(phone string) *Session {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	s, ok := sessions[phone]
	if !ok {
		s = &Session{Step: StepMenu, UserPhone: phone, NotificationSent: false, PaymentError: "", HasRetried: false, IsControllerAuthenticated: false, AgentID: "", Data: make(map[string]interface{})}
		sessions[phone] = s
	}
	return s
}

func saveSession(phone string, s *Session) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	sessions[phone] = s
}

func resetSession(phone string) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	sessions[phone] = &Session{Step: StepMenu, UserPhone: phone, NotificationSent: false, PaymentError: "", HasRetried: false, IsControllerAuthenticated: false, AgentID: "", Data: make(map[string]interface{})}
}

// SetSessionStep sets the session step for a given phone number.
func SetSessionStep(phone string, step Step) {
	s := getSession(phone)
	s.Step = step
	saveSession(phone, s)
}

// GoBack sets the session to the previous step.
func GoBack(phone string) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	s, ok := sessions[phone]
	if ok && s.PreviousStep != "" {
		s.Step = s.PreviousStep
		// Clear previous step when going back? We'll keep it as is for now.
		saveSession(phone, s)
	}
}