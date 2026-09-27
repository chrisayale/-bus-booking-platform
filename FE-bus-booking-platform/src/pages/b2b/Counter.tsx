import { useEffect, useState } from "react"
import { api } from "../../api/client"
import { downloadTicketPdf } from "../../lib/ticketPdf"
import { QRCodeCanvas } from "qrcode.react"
import type { Journey, Booking } from "../../types"
import { Plus, Trash2, Ticket as TicketIcon, Download, Printer, Users, Phone, MapPin, Calendar } from "lucide-react"

interface SeatDraft {
  seat_number: string
  passenger_name: string
  passenger_phone: string
}

interface RecentTicket {
  ticket_id: string
  passenger_name: string
  passenger_phone?: string
  seat_number: string
  price: number
  ticket_status: string
  origin: string
  destination: string
  registration_number: string
  departure_date: string
  departure_time: string
  booking_reference: string
  booked_at: string
}

export default function Counter() {
  const [journeys, setJourneys] = useState<Journey[]>([])
  const [journeyId, setJourneyId] = useState("")
  const [bookedSeats, setBookedSeats] = useState<string[]>([])
  const [seats, setSeats] = useState<SeatDraft[]>([])
  const [seatNumber, setSeatNumber] = useState("")
  const [passengerName, setPassengerName] = useState("")
  const [passengerPhone, setPassengerPhone] = useState("")
  const [guestName, setGuestName] = useState("")
  const [guestEmail, setGuestEmail] = useState("")
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
    const [confirmedBooking, setConfirmedBooking] = useState<Booking | null>(null)
  const [downloadingId, setDownloadingId] = useState<string | null>(null)
  const [recentTickets, setRecentTickets] = useState<RecentTicket[]>([])

  const selectedJourney = journeys.find((j) => j.id === journeyId)

  const fetchRecentTickets = () => {
    api.get<RecentTicket[]>("/tickets").then((data) => setRecentTickets((Array.isArray(data) ? data : []).slice(0, 15))).catch(console.error)
  }

  useEffect(() => {
    api.get<Journey[]>("/journeys").then((data) => setJourneys(Array.isArray(data) ? data : [])).catch(console.error)
    fetchRecentTickets()
  }, [])

  useEffect(() => {
    if (!journeyId) {
      setBookedSeats([])
      return
    }
    api.get<{ booked_seats: string[] }>(`/journeys/${journeyId}/seats`)
      .then((data) => setBookedSeats(data.booked_seats || []))
      .catch(() => setBookedSeats([]))
  }, [journeyId])

  const takenSeats = new Set([...bookedSeats, ...seats.map((s) => s.seat_number)])

  const handleAddSeat = () => {
    if (!seatNumber || !passengerName) return
    if (takenSeats.has(seatNumber)) {
      setError(`Le siège ${seatNumber} est déjà pris`)
      return
    }
    setSeats([...seats, { seat_number: seatNumber, passenger_name: passengerName, passenger_phone: passengerPhone }])
    setSeatNumber("")
    setPassengerName("")
    setPassengerPhone("")
    setError(null)
  }

  const handleRemoveSeat = (idx: number) => {
    setSeats(seats.filter((_, i) => i !== idx))
  }

  const handleSubmit = async () => {
    if (!journeyId || seats.length === 0 || !guestName) {
      setError("Sélectionnez un voyage, ajoutez au moins un passager, et renseignez le nom du client")
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      const email = guestEmail || `guichet+${Date.now()}@ayaletrans.local`
      const data = await api.post<{ booking: Booking }>("/guest/checkout", {
        journey_id: journeyId,
        guest_email: email,
        guest_name: guestName,
        seats: seats.map((s) => ({
          seat_number: s.seat_number,
          passenger_name: s.passenger_name,
          passenger_phone: s.passenger_phone || undefined,
        })),
      })
            setConfirmedBooking(data.booking)
      fetchRecentTickets()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setSubmitting(false)
    }
  }

  const handleNewSale = () => {
    setConfirmedBooking(null)
    setJourneyId("")
    setSeats([])
    setGuestName("")
    setGuestEmail("")
  }

  const buildTicketData = (ticket: Booking["tickets"][number]) => ({
    ticket,
    booking: confirmedBooking!,
    journey: selectedJourney
      ? {
          origin: selectedJourney.route?.origin || "",
          destination: selectedJourney.route?.destination || "",
          date: selectedJourney.departure_date,
          departure: selectedJourney.departure_time?.slice(0, 5),
          arrival: selectedJourney.arrival_time?.slice(0, 5),
        }
      : undefined,
    agencyName: selectedJourney?.organization?.name,
    agencyPhone: selectedJourney?.organization?.phone,
    busRegistration: selectedJourney?.bus?.registration_number,
  })

  const handleDownload = async (ticket: Booking["tickets"][number], mode: "download" | "print") => {
    setDownloadingId(ticket.id)
    try {
      const qrWrapper = document.querySelector(`[data-qr-for="${ticket.id}"]`)
      const qrCanvas = qrWrapper?.querySelector<HTMLCanvasElement>("canvas")
      const qrDataUrl = qrCanvas?.toDataURL("image/png")
      await downloadTicketPdf({ ...buildTicketData(ticket), qrDataUrl }, mode)
    } catch (err) {
      alert((err as Error).message)
    } finally {
      setDownloadingId(null)
    }
  }

  if (confirmedBooking) {
    return (
      <div className="max-w-2xl mx-auto space-y-6">
        <div className="bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded-lg text-sm">
          Réservation confirmée — {confirmedBooking.booking_reference}
        </div>
        <div className="space-y-4">
          {confirmedBooking.tickets?.map((ticket) => (
            <div key={ticket.id} className="bg-white rounded-xl border border-gray-200 p-5 space-y-3">
              <div className="flex items-center justify-between">
                <div>
                  <p className="font-semibold text-gray-900">{ticket.passenger_name}</p>
                  <p className="text-sm text-gray-500">Siège {ticket.seat_number} · {ticket.price} USD</p>
                </div>
                <div className="bg-white p-2 border border-gray-100 rounded-lg" data-qr-for={ticket.id}>
                  <QRCodeCanvas value={ticket.qr_token} size={100} level="M" includeMargin />
                </div>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => handleDownload(ticket, "download")}
                  disabled={downloadingId === ticket.id}
                  className="flex-1 flex items-center justify-center gap-1.5 px-3 py-2 bg-emerald-50 text-emerald-700 hover:bg-emerald-100 border border-emerald-200 rounded-lg transition text-sm font-medium disabled:opacity-50"
                >
                  <Download className="h-4 w-4" /> Télécharger
                </button>
                <button
                  onClick={() => handleDownload(ticket, "print")}
                  disabled={downloadingId === ticket.id}
                  className="flex-1 flex items-center justify-center gap-1.5 px-3 py-2 bg-gray-50 text-gray-700 hover:bg-gray-100 border border-gray-200 rounded-lg transition text-sm font-medium disabled:opacity-50"
                >
                  <Printer className="h-4 w-4" /> Imprimer
                </button>
              </div>
            </div>
          ))}
        </div>
        <button
          onClick={handleNewSale}
          className="w-full py-2.5 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition"
        >
          Nouvelle vente
        </button>
      </div>
    )
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <TicketIcon className="h-6 w-6" /> Guichet
        </h1>
        <p className="text-gray-500 text-sm mt-1">Vendez et imprimez un billet directement au comptoir</p>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
          {error}
        </div>
      )}

      <div className="bg-white rounded-xl border border-gray-200 p-5 space-y-4">
        <div>
          <label className="text-sm font-medium text-gray-700">Voyage</label>
          <select
            value={journeyId}
            onChange={(e) => { setJourneyId(e.target.value); setSeats([]) }}
            className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">Sélectionner un voyage</option>
            {journeys.map((j) => (
              <option key={j.id} value={j.id}>
                {j.route?.origin} → {j.route?.destination} — {j.departure_date} {j.departure_time?.slice(0, 5)} ({j.bus?.registration_number})
              </option>
            ))}
          </select>
        </div>

        {selectedJourney && (
          <p className="text-xs text-gray-400">
            {bookedSeats.length + seats.length} / {selectedJourney.bus?.capacity ?? "?"} sièges occupés
          </p>
        )}

        <div className="grid grid-cols-3 gap-2">
          <input
            type="text"
            placeholder="N° siège"
            value={seatNumber}
            onChange={(e) => setSeatNumber(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
          />
          <input
            type="text"
            placeholder="Nom passager"
            value={passengerName}
            onChange={(e) => setPassengerName(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
          />
          <input
            type="tel"
            placeholder="Téléphone"
            value={passengerPhone}
            onChange={(e) => setPassengerPhone(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <button
          onClick={handleAddSeat}
          disabled={!journeyId}
          className="flex items-center gap-1.5 px-3 py-2 bg-blue-50 text-blue-700 hover:bg-blue-100 border border-blue-200 rounded-lg transition text-sm font-medium disabled:opacity-50"
        >
          <Plus className="h-4 w-4" /> Ajouter le passager
        </button>

        {seats.length > 0 && (
          <div className="divide-y divide-gray-100 border-t border-gray-100 pt-2">
            {seats.map((s, idx) => (
              <div key={idx} className="flex items-center justify-between py-2 text-sm">
                <span>Siège <strong>{s.seat_number}</strong> — {s.passenger_name} {s.passenger_phone && `(${s.passenger_phone})`}</span>
                <button onClick={() => handleRemoveSeat(idx)} className="text-red-500 hover:text-red-700">
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="bg-white rounded-xl border border-gray-200 p-5 space-y-4">
        <div>
          <label className="text-sm font-medium text-gray-700">Nom du client</label>
          <input
            type="text"
            value={guestName}
            onChange={(e) => setGuestName(e.target.value)}
            className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="Nom complet"
          />
        </div>
        <div>
          <label className="text-sm font-medium text-gray-700">Email (optionnel)</label>
          <input
            type="email"
            value={guestEmail}
            onChange={(e) => setGuestEmail(e.target.value)}
            className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="client@example.com"
          />
        </div>
      </div>

            <button
        onClick={handleSubmit}
        disabled={submitting}
        className="w-full py-3 bg-blue-600 text-white rounded-lg font-semibold hover:bg-blue-700 transition disabled:opacity-50"
      >
        {submitting ? "Traitement..." : "Confirmer la vente"}
      </button>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-100">
          <h2 className="font-semibold flex items-center gap-2"><Users className="h-4 w-4" /> Ventes récentes</h2>
        </div>
        {recentTickets.length === 0 ? (
          <div className="px-6 py-8 text-center text-gray-400 text-sm">Aucune vente pour l'instant</div>
        ) : (
          <div className="divide-y divide-gray-100">
            {recentTickets.map((t) => (
              <div key={t.ticket_id} className="px-6 py-3 flex items-center justify-between text-sm">
                <div>
                  <p className="font-medium text-gray-900">{t.passenger_name}</p>
                  <div className="flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-gray-500 mt-0.5">
                    {t.passenger_phone && <span className="flex items-center gap-1"><Phone className="h-3 w-3" /> {t.passenger_phone}</span>}
                    <span className="flex items-center gap-1"><MapPin className="h-3 w-3" /> {t.origin} → {t.destination}</span>
                    <span className="flex items-center gap-1"><Calendar className="h-3 w-3" /> {t.departure_date} {t.departure_time}</span>
                  </div>
                </div>
                <div className="text-right">
                  <p className="font-medium text-gray-900">Siège {t.seat_number}</p>
                  <p className="text-xs text-gray-500">{t.price} USD</p>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}