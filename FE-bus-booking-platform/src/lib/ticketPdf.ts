import jsPDF from "jspdf"
import type { Booking, Ticket } from "../types"

export interface TicketDownloadData {
  ticket: Ticket
  booking: Booking
  qrDataUrl?: string
  journey?: {
    origin: string
    destination: string
    date: string
    departure: string
    arrival: string
  }
}

export async function downloadTicketPdf(data: TicketDownloadData): Promise<void> {
  const { ticket, booking, journey } = data

  // Colors
  const blue = [37, 99, 235] as const
  const slate900 = [15, 23, 42] as const
  const slate700 = [51, 65, 85] as const
  const slate500 = [100, 116, 139] as const
  const slate400 = [148, 163, 184] as const
  const slate200 = [226, 232, 240] as const
  const slate100 = [241, 245, 249] as const
  const emerald600 = [5, 150, 105] as const

  const statusColors: Record<string, [number, number, number]> = {
    active: [5, 150, 105],
    checked_in: [37, 99, 235],
    missed: [234, 88, 12],
    cancelled: [220, 38, 38],
  }

  const ticketStatusColor = statusColors[ticket.status] || slate500
  const ticketStatusLabel = ticket.status.charAt(0).toUpperCase() + ticket.status.slice(1)

  // Page dimensions (A6: 105 x 148 mm)
  const pageW = 105
  const pageH = 148
  const margin = 8
  const contentW = pageW - margin * 2 // 89mm

  const doc = new jsPDF({
    orientation: "portrait",
    unit: "mm",
    format: "a6",
  })

  // ---- Helper: draw rounded rect ----
  function roundedRect(x: number, y: number, w: number, h: number, r: number, style: "F" | "S" | "FD") {
    doc.roundedRect(x, y, w, h, r, r, style)
  }

  // =========================================================
  // HEADER — blue gradient bar
  // =========================================================
  doc.setFillColor(blue[0], blue[1], blue[2])
  roundedRect(margin, margin, contentW, 10, 4, "F")

  // "BUS" badge
  doc.setFillColor(255, 255, 255)
  doc.setFontSize(9)
  doc.setTextColor(255, 255, 255)
  doc.setFont("helvetica", "bold")
  doc.text("BUS", margin + 4, margin + 6.5)

  // Brand name
  doc.setFontSize(11)
  doc.text("BusBook", margin + 13, margin + 6.5)

  // Booking reference
  doc.setFont("helvetica", "normal")
  doc.setFontSize(7)
  doc.text("Booking Ref", pageW - margin - 2, margin + 4.5, { align: "right" })
  doc.setFont("helvetica", "bold")
  doc.setFontSize(8)
  doc.text(booking.booking_reference, pageW - margin - 2, margin + 8, { align: "right" })

  // =========================================================
  // ROUTE SECTION
  // =========================================================
  const routeY = margin + 16

  // Route background
  doc.setFillColor(slate100[0], slate100[1], slate100[2])
  roundedRect(margin, routeY, contentW, 18, 4, "F")

  // Route border
  doc.setDrawColor(slate200[0], slate200[1], slate200[2])
  doc.setLineWidth(0.5)
  roundedRect(margin, routeY, contentW, 18, 4, "S")

  const routeMidX = margin + contentW / 2
  const routeCenterY = routeY + 9

  // Origin
  doc.setFont("helvetica", "bold")
  doc.setFontSize(11)
  doc.setTextColor(slate900[0], slate900[1], slate900[2])
  doc.text(journey?.origin || "—", routeMidX - 20, routeCenterY - 1.5, { align: "center" })

  doc.setFont("helvetica", "normal")
  doc.setFontSize(7)
  doc.setTextColor(slate500[0], slate500[1], slate500[2])
  doc.text(journey?.departure || "—", routeMidX - 20, routeCenterY + 4.5, { align: "center" })

  // Arrow with date
  const arrowY = routeCenterY - 2
  doc.setDrawColor(blue[0], blue[1], blue[2])
  doc.setLineWidth(1)
  doc.line(routeMidX - 14, arrowY, routeMidX + 14, arrowY)

  // Arrow head
  doc.setFillColor(blue[0], blue[1], blue[2])
  doc.triangle(routeMidX + 14, arrowY, routeMidX + 10, arrowY - 2.5, routeMidX + 10, arrowY + 2.5, "F")

  // Date label above arrow
  doc.setFont("helvetica", "normal")
  doc.setFontSize(7)
  doc.setTextColor(slate500[0], slate500[1], slate500[2])
  doc.text(journey?.date || "—", routeMidX, arrowY - 4, { align: "center" })

  // Destination
  doc.setFont("helvetica", "bold")
  doc.setFontSize(11)
  doc.setTextColor(slate900[0], slate900[1], slate900[2])
  doc.text(journey?.destination || "—", routeMidX + 20, routeCenterY - 1.5, { align: "center" })

  doc.setFont("helvetica", "normal")
  doc.setFontSize(7)
  doc.setTextColor(slate500[0], slate500[1], slate500[2])
  doc.text(journey?.arrival || "—", routeMidX + 20, routeCenterY + 4.5, { align: "center" })

  // =========================================================
  // PASSENGER & SEAT
  // =========================================================
  const passY = routeY + 24

  // Passenger label
  doc.setFont("helvetica", "normal")
  doc.setFontSize(7)
  doc.setTextColor(slate500[0], slate500[1], slate500[2])
  doc.text("PASSENGER", margin, passY)

  // Passenger name
  doc.setFont("helvetica", "bold")
  doc.setFontSize(11)
  doc.setTextColor(slate900[0], slate900[1], slate900[2])
  doc.text(ticket.passenger_name, margin, passY + 5)

  // Passenger phone
  if (ticket.passenger_phone) {
    doc.setFont("helvetica", "normal")
    doc.setFontSize(8)
    doc.setTextColor(slate500[0], slate500[1], slate500[2])
    doc.text(ticket.passenger_phone, margin, passY + 10)
  }

  // Seat box (right side)
  const seatBoxW = 28
  const seatBoxX = pageW - margin - seatBoxW
  doc.setFillColor(239, 246, 255) // blue-50
  roundedRect(seatBoxX, passY - 3, seatBoxW, 18, 4, "F")

  doc.setFont("helvetica", "normal")
  doc.setFontSize(7)
  doc.setTextColor(blue[0], blue[1], blue[2])
  doc.text("SEAT", seatBoxX + seatBoxW / 2, passY + 2, { align: "center" })

  doc.setFont("helvetica", "bold")
  doc.setFontSize(16)
  doc.setTextColor(blue[0], blue[1], blue[2])
  doc.text(ticket.seat_number, seatBoxX + seatBoxW / 2, passY + 13, { align: "center" })

  // =========================================================
  // QR CODE
  // =========================================================
  const qrY = passY + 20
  const qrSize = 36 // mm

  // Use pre-rendered QR canvas from the page — no generation needed
  const qrCenterX = pageW / 2 - qrSize / 2
  doc.setFillColor(255, 255, 255)
  roundedRect(qrCenterX - 2, qrY - 2, qrSize + 4, qrSize + 4, 4, "F")
  doc.setDrawColor(slate200[0], slate200[1], slate200[2])
  doc.setLineWidth(0.5)
  roundedRect(qrCenterX - 2, qrY - 2, qrSize + 4, qrSize + 4, 4, "S")

  if (data.qrDataUrl) {
    doc.addImage(data.qrDataUrl, "PNG", qrCenterX, qrY, qrSize, qrSize)
  }

  // QR label
  doc.setFont("helvetica", "normal")
  doc.setFontSize(7)
  doc.setTextColor(slate400[0], slate400[1], slate400[2])
  doc.text("Show this QR code at boarding", pageW / 2, qrY + qrSize + 6, { align: "center" })

  // =========================================================
  // DASHED SEPARATOR
  // =========================================================
  const sepY = qrY + qrSize + 11
  doc.setDrawColor(slate200[0], slate200[1], slate200[2])
  doc.setLineWidth(0.5)
  doc.setLineDashPattern([2, 2], 0)
  doc.line(margin, sepY, pageW - margin, sepY)
  doc.setLineDashPattern([], 0) // reset

  // =========================================================
  // PRICE & STATUS
  // =========================================================
  const infoY = sepY + 8

  // Price label
  doc.setFont("helvetica", "normal")
  doc.setFontSize(7)
  doc.setTextColor(slate500[0], slate500[1], slate500[2])
  doc.text("PRICE", margin, infoY)

  // Price amount
  doc.setFont("helvetica", "bold")
  doc.setFontSize(16)
  doc.setTextColor(emerald600[0], emerald600[1], emerald600[2])
  doc.text(`${ticket.price} MAD`, margin, infoY + 7)

  // Status badge (right side)
  const statusText = ticketStatusLabel
  const statusBadgeW = doc.getTextWidth(statusText) + 6
  const statusBadgeX = pageW - margin - statusBadgeW - 6

  // Status dot
  doc.setFillColor(ticketStatusColor[0], ticketStatusColor[1], ticketStatusColor[2])
  doc.circle(statusBadgeX, infoY + 3.5, 1.5, "F")

  // Status label
  doc.setFont("helvetica", "bold")
  doc.setFontSize(8)
  doc.setTextColor(slate700[0], slate700[1], slate700[2])
  doc.text(statusText, statusBadgeX + 4, infoY + 4.5)

  // =========================================================
  // FOOTER
  // =========================================================
  const footerY = pageH - margin - 1

  doc.setFillColor(slate100[0], slate100[1], slate100[2])
  roundedRect(margin, pageH - margin - 8, contentW, 8, 4, "F")

  doc.setFont("helvetica", "normal")
  doc.setFontSize(6)
  doc.setTextColor(slate400[0], slate400[1], slate400[2])
  doc.text("Present this ticket at boarding. Valid for the scheduled journey only.", pageW / 2, footerY, { align: "center" })

  // =========================================================
  // SAVE
  // =========================================================
  doc.save(`ticket-${booking.booking_reference}-seat-${ticket.seat_number}.pdf`)
}
