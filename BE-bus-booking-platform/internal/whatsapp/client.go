package whatsapp

import (
	"context"
	"sync"

	_ "github.com/lib/pq"
	qrcode "github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
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

func Connect(databaseURL string) error {
	dbLog := waLog.Stdout("Database", "INFO", true)

	container, err := sqlstore.New(context.Background(), "postgres", databaseURL, dbLog)
	if err != nil {
		return err
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return err
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	Client = whatsmeow.NewClient(deviceStore, clientLog)

	if Client.Store.ID == nil {
		// Pas encore connecté — il faut scanner un QR code
		qrChan, _ := Client.GetQRChannel(context.Background())
		err = Client.Connect()
		if err != nil {
			return err
		}
		RegisterHandlers()
		for evt := range qrChan {
			if evt.Event == "code" {
				println("Nouveau QR code WhatsApp généré — ouvre /api/whatsapp/qr")
				setCurrentQR(evt.Code)
			} else {
				println("Statut de connexion:", evt.Event)
			}
		}
	} else {
		// Déjà connecté précédemment
		err = Client.Connect()
		if err != nil {
			return err
		}
		RegisterHandlers()
	}

	return nil
}
