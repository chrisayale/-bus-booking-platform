

import { useEffect, useState } from "react"
import { api } from "../../api/client"
import { useTranslate } from "../../hooks/useTranslate"
import type { Booking, Ticket } from "../../types"
import { Ticket as TicketIcon, XCircle, Trash2, Download, Bus, User, Armchair, Clock, Printer } from "lucide-react"
import { QRCodeCanvas } from "qrcode.react"
import { Link } from "react-router-dom"
import { downloadTicketPdf } from "../../lib/ticketPdf"

export default function MyBookings() {
  const { t } = useTranslate()
  const [bookings, setBookings] = useState<Booking[]>([])
  const [downloadingTickets, setDownloadingTickets] = useState<Set<string>>(new Set())

  useEffect(() => {
    api.get<Booking[]>("/bookings").then((data) => setBookings(Array.isArray(data) ? data : [])).catch(console.error)
  }, [])

  const cancelTicket = async (bookingId: string, ticketId: string, ticketStatus: string) => {
    const message = ticketStatus === "checked_in"
      ? t("myBookings.checkedInCancelWarning")
      : t("myBookings.cancelTicketConfirm")
    if (!confirm(message)) return
    try {
      await api.delete(`/ticket/${ticketId}`)
      setBookings(bookings.map((b) => {
        if (b.id !== bookingId) return b
        const updatedTickets = b.tickets?.map((t) =>
          t.id === ticketId ? { ...t, status: "cancelled" as const } : t
        )
        const allCancelled = updatedTickets?.every((t) => t.status === "cancelled")
        return {
          ...b,
          tickets: updatedTickets,
          status: allCancelled ? "cancelled" as const : b.status,
        }
      }))
    } catch (err) {
      alert((err as Error).message)
    }
  }

  const deleteBooking = async (id: string) => {
    if (!confirm(t("myBookings.deleteWarning"))) return
    try {
      await api.delete(`/bookings/${id}`)
      setBookings(bookings.filter((b) => b.id !== id))
    } catch (err) {
      alert((err as Error).message)
    }
  }

  const statusLabel = (status: string) => {
    const key = `myBookings.status.${status}`
    const translated = t(key)
    return translated !== key ? translated : status
  }

  const statusColors: Record<string, string> = {
    confirmed: "bg-green-100 text-green-700",
    pending: "bg-yellow-100 text-yellow-700",
    cancelled: "bg-red-100 text-red-700",
    refunded: "bg-gray-100 text-gray-500",
  }

  const ticketStatusColors: Record<string, string> = {
    active: "bg-green-100 text-green-700 border-green-200",
    checked_in: "bg-blue-100 text-blue-700 border-blue-200",
    missed: "bg-orange-100 text-orange-700 border-orange-200",
    cancelled: "bg-red-100 text-red-700 border-red-200",
    refunded: "bg-gray-100 text-gray-500 border-gray-200",
  }

  return (
    <div className="max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-6 flex items-center gap-2">
        <TicketIcon className="h-6 w-6 text-blue-600" />
        {t("myBookings.title")}
      </h1>
      <div className="space-y-6">
        {bookings.map((booking) => (
          <div key={booking.id} className="bg-white rounded-2xl border border-gray-200 shadow-sm overflow-hidden">
            {/* Booking Header */}
            <div className="bg-gradient-to-r from-blue-600 to-blue-700 px-5 py-3 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Bus className="h-5 w-5 text-white/80" />
                <span className="text-white font-semibold tracking-wide">{booking.booking_reference}</span>
                <span className={`text-xs px-2.5 py-1 rounded-full font-medium ${statusColors[booking.status] || "bg-gray-100"}`}>
                  {statusLabel(booking.status)}
                </span>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-white font-bold text-lg">{booking.total_amount} USD</span>
                {booking.status === "cancelled" && (
                  <button
                    onClick={() => deleteBooking(booking.id)}
                    className="p-1.5 text-white/60 hover:text-red-300 hover:bg-white/10 rounded-lg transition"
                    aria-label={t("myBookings.delete")}
                    title={t("myBookings.delete")}
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                )}
              </div>
            </div>

            {/* Ticket Cards */}
            {booking.tickets && booking.tickets.length > 0 && (
              <div className="p-4 space-y-4">
                {booking.tickets.map((ticket) => (
                  <TicketCard
                    key={ticket.id}
                    ticket={ticket}
                    booking={booking}
                    downloading={downloadingTickets.has(ticket.id)}
                                        onDownload={() => handleDownload(ticket, booking, setDownloadingTickets)}
                    onPrint={() => handleDownload(ticket, booking, setDownloadingTickets, "print")}
                    onCancel={() => cancelTicket(booking.id, ticket.id, ticket.status)}
                    statusLabel={statusLabel}
                    ticketStatusColors={ticketStatusColors}
                  />
                ))}
              </div>
            )}

            {/* Booking Date */}
            <div className="px-5 py-2 bg-gray-50 border-t border-gray-100">
              <p className="text-xs text-gray-400">
                {t("myBookings.booked")} {new Date(booking.created_at).toLocaleDateString()} &middot; {booking.tickets?.length || 0} {t("myBookings.tickets")}
              </p>
            </div>
          </div>
        ))}
        {bookings.length === 0 && (
          <div className="text-center py-16 bg-white rounded-2xl border border-gray-200">
            <TicketIcon className="h-12 w-12 text-gray-300 mx-auto mb-4" />
            <p className="text-gray-500 mb-2">
              {t("myBookings.noBookings")}
            </p>
            <Link to="/search" className="text-blue-600 hover:underline font-medium">
              {t("myBookings.searchTrips")}
            </Link>
          </div>
        )}
      </div>
    </div>
  )
}

async function handleDownload(
  ticket: Ticket,
  booking: Booking,
  setDownloadingTickets: React.Dispatch<React.SetStateAction<Set<string>>>,
  mode: "download" | "print" = "download"
) {
  setDownloadingTickets((prev) => new Set(prev).add(ticket.id))
  try {
    // Grab the already-rendered QR canvas from the DOM
    const qrWrapper = document.querySelector(`[data-qr-for="${ticket.id}"]`)
    const qrCanvas = qrWrapper?.querySelector<HTMLCanvasElement>("canvas")
    if (!qrCanvas) console.warn("[PDF] QR canvas not found for ticket", ticket.id)
    const qrDataUrl = qrCanvas?.toDataURL("image/png")
        await downloadTicketPdf({
      ticket,
      booking,
      qrDataUrl,
      journey: ticket.journey
        ? {
            origin: ticket.journey.route?.origin || "",
            destination: ticket.journey.route?.destination || "",
            date: ticket.journey.departure_date,
            departure: ticket.journey.departure_time?.slice(0, 5),
            arrival: ticket.journey.arrival_time?.slice(0, 5),
          }
        : undefined,
      agencyName: ticket.journey?.organization?.name,
      agencyPhone: ticket.journey?.organization?.phone,
      busRegistration: ticket.journey?.bus?.registration_number,
    }, mode)
  } catch (err) {
    alert("Failed to download ticket: " + (err as Error).message)
  } finally {
    setDownloadingTickets((prev) => {
      const next = new Set(prev)
      next.delete(ticket.id)
      return next
    })
  }
}

function TicketCard({
  ticket,
  booking,
  downloading,
  onDownload,
  onPrint,
  onCancel,
  statusLabel,
  ticketStatusColors,
}: {
  ticket: Ticket
  booking: Booking
  downloading: boolean
  onDownload: () => void
  onPrint: () => void
  onCancel: () => void
  statusLabel: (status: string) => string
  ticketStatusColors: Record<string, string>
}) {
  const { t } = useTranslate()
  const isActive = ticket.status === "active" || ticket.status === "checked_in"
  const canCancel = (ticket.status === "active" || ticket.status === "checked_in" || ticket.status === "missed") && booking.status === "confirmed"

  const journey = ticket.journey
  const route = journey?.route

  return (
    <div className="relative bg-gradient-to-br from-white to-blue-50 border border-blue-100 rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-shadow">
      {/* Ticket cut-out effect (left side) */}
      <div className="absolute left-0 top-0 bottom-0 w-3 bg-gradient-to-b from-blue-600 to-blue-700 rounded-l-xl" />

      {/* Dashed separation line for ticket look */}
      <div className="relative ml-3 pl-4 py-3 pr-4">
        {/* Route section */}
        {route && journey && (
          <div className="mb-3">
            <div className="flex items-center justify-between gap-4">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 text-base font-bold text-gray-900">
                  <span className="truncate">{route.origin}</span>
                  <span className="text-blue-500 shrink-0">→</span>
                  <span className="truncate">{route.destination}</span>
                </div>
                <div className="flex items-center gap-3 mt-1 text-xs text-gray-500">
                  <span className="flex items-center gap-1">
                    <Clock className="h-3 w-3" />
                    {journey.departure_date} at {journey.departure_time}
                  </span>
                </div>
              </div>
              <div className="text-right shrink-0">
                <div className="text-lg font-bold text-blue-700">
                  {ticket.price.toFixed(2)} USD
                </div>
              </div>
            </div>
            <div className="h-px bg-gradient-to-r from-blue-200 via-blue-100 to-transparent my-2" />
          </div>
        )}

        {/* Top section - Passenger */}
        <div className="flex items-start justify-between mb-3">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <User className="h-4 w-4 text-blue-500" />
              <span className="font-semibold text-gray-900 text-sm truncate">{ticket.passenger_name}</span>
              {ticket.passenger_phone && (
                <span className="text-xs text-gray-400">{ticket.passenger_phone}</span>
              )}
            </div>
            <div className="flex items-center gap-2 text-xs text-gray-500">
              <Armchair className="h-3.5 w-3.5" />
              <span>Seat <strong className="text-gray-700">{ticket.seat_number}</strong></span>
              <span className={`text-xs px-2 py-0.5 rounded-full border font-medium ${ticketStatusColors[ticket.status] || "bg-gray-100 text-gray-500"}`}>
                {statusLabel(ticket.status)}
              </span>
            </div>
          </div>
        </div>

        {/* QR Code Section - prominent and centered */}
        {isActive && ticket.qr_token && (
          <div className="flex flex-col items-center mb-3">
            <div className="bg-white rounded-xl p-2 shadow-sm border border-blue-100">
              <div data-qr-for={ticket.id}>
                <QRCodeCanvas value={ticket.qr_token} size={200} level="M" includeMargin />
              </div>
            </div>
            <p className="text-[10px] text-gray-400 mt-1.5 tracking-wider uppercase">{t("myBookings.showToBoard")}</p>
          </div>
        )}

        {/* Dashed separator */}
        <div className="border-t border-dashed border-gray-200 my-2" />

        {/* Actions */}
        <div className="flex items-center justify-end gap-2 pt-1">
                 {isActive && ticket.qr_token && (
            <>
              <button
                onClick={onDownload}
                disabled={downloading}
                className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-50 text-emerald-700 hover:bg-emerald-100 border border-emerald-200 rounded-lg transition text-xs font-medium disabled:opacity-50"
              >
                <Download className="h-3.5 w-3.5" />
                {downloading ? "..." : t("myBookings.download")}
              </button>
              <button
                onClick={onPrint}
                disabled={downloading}
                className="flex items-center gap-1.5 px-3 py-1.5 bg-gray-50 text-gray-700 hover:bg-gray-100 border border-gray-200 rounded-lg transition text-xs font-medium disabled:opacity-50"
              >
                <Printer className="h-3.5 w-3.5" />
                Imprimer
              </button>
            </>
          )}
          {canCancel && (
            <button
              onClick={onCancel}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-red-50 text-red-600 hover:bg-red-100 border border-red-200 rounded-lg transition text-xs font-medium"
            >
              <XCircle className="h-3.5 w-3.5" />
              {t("myBookings.cancel")}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

