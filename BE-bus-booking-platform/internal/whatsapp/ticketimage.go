package whatsapp

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	qrcode "github.com/skip2/go-qrcode"
	xfont "golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

func loadFace(ttfBytes []byte, size float64) xfont.Face {
	f, err := truetype.Parse(ttfBytes)
	if err != nil {
		return nil
	}
	return truetype.NewFace(f, &truetype.Options{Size: size, DPI: 72})
}

func renderTicketCard(agencyName, bookingRef, origin, destination, date, depTime, arrTime, passengerName, passengerPhone, seatNumber string, price float64, status, busReg, agencyPhone, qrToken string) ([]byte, error) {
	const W, H = 700, 1080
	dc := gg.NewContext(W, H)
	dc.SetColor(color.White)
	dc.Clear()

	margin := 30.0
	blue := color.RGBA{37, 99, 235, 255}

	// Header
	dc.SetColor(blue)
	dc.DrawRoundedRectangle(margin, margin, W-2*margin, 90, 16)
	dc.Fill()

	dc.SetColor(color.White)
	dc.SetFontFace(loadFace(gobold.TTF, 28))
	dc.DrawStringAnchored("BUS  "+agencyName, margin+24, margin+45, 0, 0.5)

	dc.SetFontFace(loadFace(goregular.TTF, 15))
	dc.DrawStringAnchored("Booking Ref", W-margin-24, margin+28, 1, 0.5)
	dc.SetFontFace(loadFace(gobold.TTF, 18))
	dc.DrawStringAnchored(bookingRef, W-margin-24, margin+58, 1, 0.5)

	// Route box
	routeY := margin + 90 + 20
	routeH := 130.0
	dc.SetColor(color.RGBA{241, 245, 249, 255})
	dc.DrawRoundedRectangle(margin, routeY, W-2*margin, routeH, 16)
	dc.Fill()

	midX := float64(W) / 2
	dc.SetColor(color.RGBA{15, 23, 42, 255})
	dc.SetFontFace(loadFace(gobold.TTF, 24))
	dc.DrawStringAnchored(origin, margin+100, routeY+50, 0.5, 0.5)
	dc.DrawStringAnchored(destination, W-margin-100, routeY+50, 0.5, 0.5)

	dc.SetColor(blue)
	dc.SetFontFace(loadFace(goregular.TTF, 28))
	dc.DrawStringAnchored("→", midX, routeY+50, 0.5, 0.5)

	dc.SetColor(color.RGBA{100, 116, 139, 255})
	dc.SetFontFace(loadFace(goregular.TTF, 14))
	dc.DrawStringAnchored(date, midX, routeY+20, 0.5, 0.5)
	dc.DrawStringAnchored(depTime, margin+100, routeY+82, 0.5, 0.5)
	dc.DrawStringAnchored(arrTime, W-margin-100, routeY+82, 0.5, 0.5)

	// Passenger
	passY := routeY + routeH + 40
	dc.SetColor(color.RGBA{100, 116, 139, 255})
	dc.SetFontFace(loadFace(goregular.TTF, 13))
	dc.DrawStringAnchored("PASSENGER", margin, passY, 0, 0.5)

	dc.SetColor(color.RGBA{15, 23, 42, 255})
	dc.SetFontFace(loadFace(gobold.TTF, 22))
	dc.DrawStringAnchored(passengerName, margin, passY+28, 0, 0.5)

	if passengerPhone != "" {
		dc.SetColor(color.RGBA{100, 116, 139, 255})
		dc.SetFontFace(loadFace(goregular.TTF, 15))
		dc.DrawStringAnchored(passengerPhone, margin, passY+54, 0, 0.5)
	}

	// Seat box
	seatBoxW := 140.0
	seatBoxX := float64(W) - margin - seatBoxW
	dc.SetColor(color.RGBA{239, 246, 255, 255})
	dc.DrawRoundedRectangle(seatBoxX, passY-10, seatBoxW, 90, 14)
	dc.Fill()
	dc.SetColor(blue)
	dc.SetFontFace(loadFace(goregular.TTF, 13))
	dc.DrawStringAnchored("SEAT", seatBoxX+seatBoxW/2, passY+15, 0.5, 0.5)
	dc.SetFontFace(loadFace(gobold.TTF, 32))
	dc.DrawStringAnchored(seatNumber, seatBoxX+seatBoxW/2, passY+55, 0.5, 0.5)

	// QR code
	qrSize := 300
	qrY := int(passY) + 110
	qrPNG, err := qrcode.Encode(qrToken, qrcode.Medium, qrSize)
	if err == nil {
		qrImg, err2 := png.Decode(bytes.NewReader(qrPNG))
		if err2 == nil {
			qrX := (W - qrSize) / 2
			dc.SetColor(color.White)
			dc.DrawRoundedRectangle(float64(qrX-10), float64(qrY-10), float64(qrSize+20), float64(qrSize+20), 12)
			dc.Fill()
			dc.DrawImage(qrImg, qrX, qrY)
		}
	}

	dc.SetColor(color.RGBA{148, 163, 184, 255})
	dc.SetFontFace(loadFace(goregular.TTF, 14))
	dc.DrawStringAnchored("Show this QR code at boarding", midX, float64(qrY+qrSize+30), 0.5, 0.5)

	// Separator
	sepY := float64(qrY+qrSize) + 60
	dc.SetColor(color.RGBA{226, 232, 240, 255})
	dc.SetLineWidth(1)
	dc.DrawLine(margin, sepY, W-margin, sepY)
	dc.Stroke()

	// Price & status
	infoY := sepY + 40
	dc.SetColor(color.RGBA{100, 116, 139, 255})
	dc.SetFontFace(loadFace(goregular.TTF, 13))
	dc.DrawStringAnchored("PRICE", margin, infoY, 0, 0.5)

	dc.SetColor(color.RGBA{5, 150, 105, 255})
	dc.SetFontFace(loadFace(gobold.TTF, 34))
	dc.DrawStringAnchored(fmt.Sprintf("%.0f USD", price), margin, infoY+38, 0, 0.5)

	dc.SetColor(color.RGBA{5, 150, 105, 255})
	dc.DrawCircle(W-margin-120, infoY, 6)
	dc.Fill()
	dc.SetColor(color.RGBA{51, 65, 85, 255})
	dc.SetFontFace(loadFace(gobold.TTF, 16))
	dc.DrawStringAnchored(status, W-margin-105, infoY, 0, 0.5)

	// Bus / Tel
	busY := infoY + 62
	dc.SetColor(color.RGBA{100, 116, 139, 255})
	dc.SetFontFace(loadFace(goregular.TTF, 14))
	line := "Bus: " + busReg
	if agencyPhone != "" {
		line += "   |   Tel: " + agencyPhone
	}
	dc.DrawStringAnchored(line, margin, busY, 0, 0.5)

	// Footer
	footerY := busY + 40
	dc.SetColor(color.RGBA{241, 245, 249, 255})
	dc.DrawRoundedRectangle(margin, footerY-22, W-2*margin, 44, 10)
	dc.Fill()
	dc.SetColor(color.RGBA{148, 163, 184, 255})
	dc.SetFontFace(loadFace(goregular.TTF, 12))
	dc.DrawStringAnchored("Present this ticket at boarding. Valid for the scheduled journey only.", midX, footerY, 0.5, 0.5)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
