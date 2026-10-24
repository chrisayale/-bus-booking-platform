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

	container, err := sqlstore.New(context.Background(), "postgres", databaseURL, dbLog)
	if err != nil {
		log.Printf("[WHATSAPP-INIT] ❌ Failed to create SQL store: %v", err)
		return err
	}
	log.Println("[WHATSAPP-INIT] ✓ SQL store created successfully")

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		log.Printf("[WHATSAPP-INIT] ❌ Failed to get device store: %v", err)
		return err
	}
	log.Println("[WHATSAPP-INIT] ✓ Device store retrieved")

	clientLog := waLog.Stdout("Client", "INFO", true)
	Client = whatsmeow.NewClient(deviceStore, clientLog)
	log.Printf("[WHATSAPP-INIT] ✓ WhatsMeow client created")

	// CRITICAL: Register event handlers BEFORE connecting
	// This ensures we receive all events including historical messages on reconnect
	RegisterHandlers()
	log.Printf("[WHATSAPP] Event handlers registered")

	// Register connection state event handlers for diagnostics
	registerConnectionStateHandlers()

	if Client.Store.ID == nil {
		// Pas encore connecté — il faut scanner un QR code
		log.Println("[WHATSAPP-INIT] ⚠️ No paired device found (Store.ID is nil)")
		log.Println("[WHATSAPP-INIT] Waiting for QR code scan...")
		
		qrChan, _ := Client.GetQRChannel(context.Background())
		err = Client.Connect()
		if err != nil {
			log.Printf("[WHATSAPP-INIT] ❌ Failed to connect: %v", err)
			return err
		}
		
		log.Println("[WHATSAPP-INIT] ✓ Client connection initiated, waiting for QR channel...")
		
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
	} else {
		// Déjà connecté précédemment
		log.Printf("[WHATSAPP-INIT] 🔄 Existing session found for JID: %s", Client.Store.ID.User)
		err = Client.Connect()
		if err != nil {
			log.Printf("[WHATSAPP-INIT] ❌ Failed to reconnect: %v", err)
			return err
		}
		log.Printf("[WHATSAPP-INIT] ✅ WhatsMeow reconnecté avec succès avec le JID: %s", Client.Store.ID.User)
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
