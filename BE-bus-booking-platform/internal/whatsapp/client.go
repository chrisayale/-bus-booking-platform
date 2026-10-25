package whatsapp

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	_ "github.com/lib/pq"
	qrcode "github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var Client *whatsmeow.Client
var currentQR []byte
var qrMutex sync.RWMutex

func GetCurrentQR() []byte {
	qrMutex.RLock()
	defer qrMutex.RUnlock()
	return currentQR
}

func setCurrentQR(code string) {
	png, err := qrcode.Encode(code, qrcode.Medium, 400)
	if err != nil {
		return
	}
	qrMutex.Lock()
	currentQR = png
	qrMutex.Unlock()
}

// renderQRCodeTerminal renders a QR code as ASCII/ANSI in the terminal
func renderQRCodeTerminal(code string) string {
	// For terminal display, we'll output the raw code and a link to view it
	// since rendering ASCII QR from PNG in pure Go is complex
	return fmt.Sprintf(`
╔═══════════════════════════════════════════════════════════════╗
║                    WHATSAPP QR CODE                          ║
╠═══════════════════════════════════════════════════════════════╣
║  Scan this QR code with WhatsApp on your phone:             ║
║                                                              ║
║  1. Open WhatsApp → Settings → Linked Devices               ║
║  2. Tap "Link a Device"                                     ║
║  3. Scan the QR code displayed at:                          ║
║                                                              ║
║     👉 http://localhost:3000/api/whatsapp/qr  (local)       ║
║     👉 https://your-domain.com/api/whatsapp/qr  (production)║
║                                                              ║
║  Raw QR Code Data:                                           ║
║  %s
║                                                              ║
║  The code will also be available at /api/whatsapp/qr        ║
╚═══════════════════════════════════════════════════════════════╝
`, strings.ReplaceAll(code, "\n", "\n║  "))
}

func Connect(databaseURL string) error {
	log.Println("[WHATSAPP-INIT] Démarrage du client WhatsMeow...")

	dbLog := waLog.Stdout("Database", "INFO", true)

	log.Printf("[WHATSAPP-INIT] 🔗 Connecting to database: %s", maskDatabaseURL(databaseURL))
	container, err := sqlstore.New(context.Background(), "postgres", databaseURL, dbLog)
	if err != nil {
		log.Printf("[WHATSAPP-INIT] ❌ Failed to create SQL store: %v", err)
		return fmt.Errorf("sqlstore.New failed: %w", err)
	}
	log.Println("[WHATSAPP-INIT] ✓ SQL store created successfully")

	// FORCE FRESH PAIRING: Clear all existing devices to avoid stale/invalid sessions
	log.Println("[WHATSAPP-INIT] 🔄 Forcing fresh pairing - clearing existing device store...")
	if err := clearDeviceStore(container); err != nil {
		log.Printf("[WHATSAPP-INIT] ⚠️ Warning clearing device store: %v", err)
	}

	// Create brand new device store for fresh pairing
	deviceStore := container.NewDevice()
	if err := container.PutDevice(context.Background(), deviceStore); err != nil {
		log.Printf("[WHATSAPP-INIT] ❌ Failed to create new device store: %v", err)
		return fmt.Errorf("PutDevice failed: %w", err)
	}
	log.Println("[WHATSAPP-INIT] ✓ Fresh device store created")
	log.Printf("[WHATSAPP-INIT] ✓ New device JID: %s", deviceStore.ID.String())

	// INSTANTIATE CLIENT IMMEDIATELY - always available globally
	clientLog := waLog.Stdout("Client", "INFO", true)
	Client = whatsmeow.NewClient(deviceStore, clientLog)
	log.Printf("[WHATSAPP-INIT] ✓ WhatsMeow client created and stored in global variable")

	// CRITICAL: Register event handlers BEFORE connecting
	RegisterHandlers()
	log.Printf("[WHATSAPP] Event handlers registered")

	// Register connection state event handlers for diagnostics
	registerConnectionStateHandlers()

	// Start fresh pairing with QR code generation
	log.Println("[WHATSAPP-INIT] 🔄 Starting fresh WhatsApp pairing with QR code...")
	startNewPairing()
	log.Println("[WHATSAPP-INIT] ✓ startNewPairing initiated in background")
	return nil
}

// maskDatabaseURL masks the password in database URL for logging
func maskDatabaseURL(url string) string {
	// Simple masking: replace password part with ***
	if idx := strings.Index(url, "://"); idx != -1 {
		rest := url[idx+3:]
		if atIdx := strings.Index(rest, "@"); atIdx != -1 {
			userPass := rest[:atIdx]
			if colonIdx := strings.Index(userPass, ":"); colonIdx != -1 {
				return url[:idx+3] + userPass[:colonIdx+1] + "***" + rest[atIdx:]
			}
		}
	}
	return url
}

// startNewPairing initiates a fresh WhatsApp pairing flow with QR code display
// Runs in a goroutine so Connect() returns immediately while pairing continues in background
func startNewPairing() {
	log.Println("[WHATSAPP-INIT] 🔄 Starting fresh WhatsApp pairing...")
	log.Println("[WHATSAPP-INIT] Waiting for QR code scan...")

	qrChan, _ := Client.GetQRChannel(context.Background())
	err := Client.Connect()
	if err != nil {
		log.Printf("[WHATSAPP-INIT] ❌ Failed to connect: %v", err)
		// Don't return error - just log it, pairing will retry on reconnect
		return
	}

	log.Println("[WHATSAPP-INIT] ✓ Client connection initiated, waiting for QR channel...")

	// Process QR channel events in a goroutine so Connect() can return
	go func() {
		for evt := range qrChan {
			if evt.Event == "code" {
				log.Printf("[WHATSAPP-INIT] 📱 Nouveau QR code WhatsApp généré!")
				log.Printf("[WHATSAPP-INIT] 📱 Ouvrez /api/whatsapp/qr pour voir le QR code en image")

				// Display QR code in terminal as ASCII/ANSI
				qrDisplay := renderQRCodeTerminal(evt.Code)
				log.Printf("%s", qrDisplay)

				setCurrentQR(evt.Code)
				log.Printf("[WHATSAPP-INIT] ✓ QR code stored and ready for scanning")
			} else if evt.Event == "success" {
				log.Printf("[WHATSAPP-INIT] ✅ WhatsApp connected successfully! JID: %s", Client.Store.ID.User)
			} else {
				log.Printf("[WHATSAPP-INIT] Statut de connexion: %s", evt.Event)
			}
		}
	}()
}

// clearDeviceStore removes the existing device from the database to force new pairing
func clearDeviceStore(container *sqlstore.Container) error {
	ctx := context.Background()
	devices, err := container.GetAllDevices(ctx)
	if err != nil {
		return err
	}
	for _, device := range devices {
		if err := container.DeleteDevice(ctx, device); err != nil {
			log.Printf("[WHATSAPP-INIT] ⚠️ Failed to delete device %s: %v", device.ID, err)
		} else {
			log.Printf("[WHATSAPP-INIT] ✓ Deleted device from store: %s", device.ID)
		}
	}
	return nil
}

// registerConnectionStateHandlers registers handlers for connection state events
// to log Disconnected, ConnectFailure, and LoggedOut events for diagnostics
func registerConnectionStateHandlers() {
	Client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Disconnected:
			log.Printf("[WHATSAPP-CONNECTION] ❌ DISCONNECTED: Websocket closed by server")
		case *events.ConnectFailure:
			log.Printf("[WHATSAPP-CONNECTION] ❌ CONNECT FAILURE: Reason=%v, Message=%s", v.Reason, v.Message)
		case *events.LoggedOut:
			log.Printf("[WHATSAPP-CONNECTION] 🚪 LOGGED OUT: OnConnect=%v, Reason=%v", v.OnConnect, v.Reason)
		}
	})
}
