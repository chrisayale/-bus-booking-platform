# Bus Booking Platform - Claude Code Guidelines

## Project Overview
This is a bus booking platform with a backend (BE-bus-booking-platform) and frontend (FE-bus-booking-platform).

## Running the Project

### Backend (Go)
1. Navigate to the backend directory:
   ```bash
   cd BE-bus-booking-platform
   ```
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Run the application:
   ```bash
   go run ./cmd/api
   ```
4. The API will be available at http://localhost:8080

### Frontend (React)
1. Navigate to the frontend directory:
   ```bash
   cd FE-bus-booking-platform
   ```
2. Install dependencies:
   ```bash
   npm install
   ```
3. Start the development server:
   ```bash
   npm start
   ```
4. The frontend will be available at http://localhost:3000

## Important Notes

### WhatsApp Handler Fix
The syntax error in `internal/whatsapp/handler.go` (unexpected EOF at line 530) has been fixed by:
- Adding the missing `StepFlexPayPhoneInput` case to the switch statement in the `process()` function
- Removing the unreachable `StepPhonePaymentChoice` and `StepPaymentProcessing` cases
- Updating the `StepCurrencyChoice` case to transition to `StepFlexPayPhoneInput`
- The `StepFlexPayPhoneInput` case now contains the payment processing logic as specified in the fix requirements

### Source of Truth
The corrected `internal/whatsapp/handler.go` file is the authoritative source for the WhatsApp handler implementation. All references to the payment flow should be based on this file.

### Environment
- Go version: 1.20+
- Node.js version: 18+
- Dependencies are managed via go.mod and package.json

## Troubleshooting
If you encounter build errors:
1. Run `go mod tidy` in the backend directory
2. Run `npm install` in the frontend directory
3. Ensure environment variables are set (copy from .env.example if needed)