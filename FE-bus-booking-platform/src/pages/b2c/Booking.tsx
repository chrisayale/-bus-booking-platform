import { useState, useEffect, useRef, useCallback } from "react"
import { useNavigate } from "react-router-dom"
import { useSelector } from "react-redux"
import { api } from "../../api/client"
import { useAuth } from "../../hooks/useAuth"
import { useTranslate } from "../../hooks/useTranslate"
import type { RootState } from "../../store"
import type { Booking, Ticket } from "../../types"
import { ArrowLeft, Bus, User, Mail, CreditCard, CheckCircle, Download, ExternalLink, XCircle, MapPin, Clock, Compass } from "lucide-react"
import { QRCodeCanvas } from "qrcode.react"
import { downloadTicketPdf } from "../../lib/ticketPdf"

// Parse layout to get left and right column counts (e.g. "2+2" => [2, 2], "3+2" => [3, 2])
const parseLayout = (layout: string): { leftCols: number; rightCols: number } => {
  const parts = layout.split("+").map(Number)
  return { leftCols: parts[0] || 2, rightCols: parts[1] || 2 }
}

export default function Booking() {
  const { t } = useTranslate()
  const journey = useSelector((state: RootState) => state.booking.selectedJourney)
  const { isAuthenticated } = useAuth()
  const navigate = useNavigate()

  const [passengers, setPassengers] = useState([{ name: "", phone: "", seat: "" }])
  const [guestEmail, setGuestEmail] = useState("")
  const [guestName, setGuestName] = useState("")
  const [loading, setLoading] = useState(false)
  const [confirmedBooking, setConfirmedBooking] = useState<Booking | null>(null)
  const [downloadingId, setDownloadingId] = useState<string | null>(null)
  const [bookedSeats, setBookedSeats] = useState<string[]>([])
  const [seatsLoading, setSeatsLoading] = useState(true)
  const [seatsUpdated, setSeatsUpdated] = useState(false)
  const updateTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const fetchBookedSeats = useCallback(async () => {
    if (!journey || confirmedBooking) return
    try {
      const data = await api.get<{ booked_seats: string[] }>(`/journeys/${journey.id}/seats`)
      const incoming = data.booked_seats || []
      setBookedSeats((prev) => {
        if (JSON.stringify(prev) !== JSON.stringify(incoming)) {
          setSeatsUpdated(true)
        }
        return incoming
      })
    } catch {
      // Silent fail on polling errors
    } finally {
      setSeatsLoading(false)
    }
  }, [journey, confirmedBooking])

  // Initial fetch
  useEffect(() => {
    if (!journey) return
    setSeatsLoading(true) // eslint-disable-line react-hooks/set-state-in-effect
    fetchBookedSeats()
  }, [journey, fetchBookedSeats])

  // Auto-clear the update indicator after 2 seconds
  useEffect(() => {
    if (!seatsUpdated) return
    if (updateTimeoutRef.current) clearTimeout(updateTimeoutRef.current)
    updateTimeoutRef.current = setTimeout(() => setSeatsUpdated(false), 2000)
    return () => {
      if (updateTimeoutRef.current) clearTimeout(updateTimeoutRef.current)
    }
  }, [seatsUpdated])

  // Poll every 15 seconds for real-time updates
  useEffect(() => {
    if (!journey || confirmedBooking) return
    const interval = setInterval(fetchBookedSeats, 15000)
    return () => clearInterval(interval)
  }, [journey, confirmedBooking, fetchBookedSeats])

  if (!journey) {
    return (
      <div className="text-center py-20">
        <Compass className="h-12 w-12 text-gray-300 mx-auto mb-4" />
        <p className="text-gray-500">{t("booking.noJourney")}</p>
        <button onClick={() => navigate("/search")} className="mt-4 text-travel-500 hover:text-travel-600 font-medium transition-colors">
          {t("booking.searchTrips")}
        </button>
      </div>
    )
  }

  const seatConfig = journey.bus?.seat_configuration
  const seatRows = seatConfig?.rows ?? 10
  const seatCols = seatConfig?.columns ?? 4
  const seatLayout = seatConfig?.layout ?? "2+2"

  const { leftCols, rightCols } = parseLayout(seatLayout)

  const addPassenger = () => setPassengers([...passengers, { name: "", phone: "", seat: "" }])
  const removePassenger = (i: number) => setPassengers(passengers.filter((_, j) => j !== i))
  const updatePassenger = (i: number, field: string, value: string) => {
    const updated = [...passengers]
    updated[i] = { ...updated[i], [field]: value }
    setPassengers(updated)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const seats = passengers.map((p) => ({
        seat_number: p.seat,
        passenger_name: p.name,
        passenger_phone: p.phone,
      }))

      let booking: Booking
      if (!isAuthenticated) {
        const result = await api.post<{ booking: Booking; email_status: string }>("/guest/checkout", {
          journey_id: journey.id,
          guest_email: guestEmail,
          guest_name: guestName,
          seats,
        })
        booking = result.booking
      } else {
        booking = await api.post<Booking>("/bookings", {
          journey_id: journey.id,
          seats,
        })
      }
      setConfirmedBooking(booking)
    } catch (err) {
      alert((err as Error).message)
    } finally {
      setLoading(false)
    }
  }

  const handleDownloadTicket = async (ticket: Ticket) => {
    if (!confirmedBooking) return
    setDownloadingId(ticket.id)
    try {
      await downloadTicketPdf({
        ticket,
        booking: confirmedBooking,
        journey: journey
          ? {
              origin: journey.route?.origin || journey.route?.name?.split(" → ")[0] || "",
              destination: journey.route?.destination || journey.route?.name?.split(" → ")[1] || "",
              date: journey.departure_date,
              departure: journey.departure_time?.slice(0, 5),
              arrival: journey.arrival_time?.slice(0, 5),
            }
          : undefined,
      })
    } catch (err) {
      alert("Failed to download ticket: " + (err as Error).message)
    } finally {
      setDownloadingId(null)
    }
  }

  const selectedSeats = passengers.map((p) => p.seat).filter(Boolean)

  // Confirmation screen after successful booking
  if (confirmedBooking) {
    return (
      <div className="max-w-2xl mx-auto animate-fade-in-up">
        {/* Success Banner */}
        <div className="bg-gradient-to-br from-emerald-500 via-emerald-600 to-green-700 text-white p-8 rounded-2xl text-center mb-8 shadow-xl shadow-emerald-200/50">
          <div className="w-16 h-16 bg-white/20 backdrop-blur-sm rounded-full flex items-center justify-center mx-auto mb-4">
            <CheckCircle className="h-10 w-10" />
          </div>
          <h2 className="text-2xl font-bold mb-1">{t("booking.confirmed")}</h2>
          <p className="text-emerald-100 text-sm max-w-md mx-auto">{t("booking.confirmedDesc")}</p>
          <div className="mt-5 inline-flex items-center gap-2 bg-white/15 backdrop-blur-sm px-5 py-2.5 rounded-xl border border-white/10">
            <span className="text-sm opacity-80">{t("booking.reference")}:</span>
            <span className="text-lg font-bold tracking-wider">{confirmedBooking.booking_reference}</span>
          </div>
        </div>

        {/* Tickets */}
        {confirmedBooking.tickets?.map((ticket) => (
          <div key={ticket.id} className="bg-white rounded-xl border border-gray-100 overflow-hidden mb-4 shadow-md hover:shadow-lg transition-shadow">
            <div className="bg-gradient-to-r from-travel-500 to-rose-500 px-5 py-3 flex items-center justify-between text-white">
              <div className="flex items-center gap-2">
                <Bus className="h-4 w-4" />
                <span className="font-semibold text-sm">{journey.route?.name || t("booking.trip")}</span>
              </div>
              <span className="text-xs bg-white/20 px-2.5 py-1 rounded-full font-medium">
                {t("booking.seat")} {ticket.seat_number}
              </span>
            </div>

            <div className="p-5">
              <div className="grid grid-cols-3 gap-4 mb-4 text-sm">
                <div>
                  <p className="text-gray-400 text-xs uppercase tracking-wide">{t("search.date")}</p>
                  <p className="font-medium">{journey.departure_date}</p>
                </div>
                <div>
                  <p className="text-gray-400 text-xs uppercase tracking-wide">{t("booking.time")}</p>
                  <p className="font-medium flex items-center gap-1"><Clock className="h-3 w-3 text-travel-400" /> {journey.departure_time?.slice(0, 5)} → {journey.arrival_time?.slice(0, 5)}</p>
                </div>
                <div>
                  <p className="text-gray-400 text-xs uppercase tracking-wide">{t("booking.passengerName")}</p>
                  <p className="font-medium">{ticket.passenger_name}</p>
                </div>
              </div>

              {/* QR Code */}
              <div className="flex flex-col items-center py-4 border-t border-dashed border-gray-200">
                <p className="text-xs font-medium text-travel-600 mb-2">{t("myBookings.qrTicket")}</p>
                <div className="bg-white p-3 rounded-xl shadow-md border border-gray-100">
                  <QRCodeCanvas value={ticket.qr_token} size={200} level="M" includeMargin />
                </div>
                <p className="text-xs text-gray-400 mt-2">{t("myBookings.showToBoard")}</p>
              </div>

              {/* Actions */}
              <div className="flex items-center gap-3 pt-4 border-t border-gray-100">
                <button
                  onClick={() => handleDownloadTicket(ticket)}
                  disabled={downloadingId === ticket.id}
                  className="flex-1 flex items-center justify-center gap-2 bg-gradient-to-r from-emerald-500 to-green-600 text-white py-2.5 rounded-xl text-sm font-medium
                    hover:from-emerald-600 hover:to-green-700 transition-all duration-200 disabled:opacity-50 shadow-sm shadow-emerald-200/50 active:scale-[0.98]"
                >
                  <Download className="h-4 w-4" />
                  {downloadingId === ticket.id ? t("common.loading") : t("myBookings.download")}
                </button>
              </div>
            </div>
          </div>
        ))}

        {/* Navigation */}
        <div className="flex items-center gap-4 justify-center mt-6">
          {isAuthenticated && (
            <button
              onClick={() => navigate("/my-bookings")}
              className="flex items-center gap-2 text-travel-500 hover:text-travel-600 text-sm font-medium transition-colors"
            >
              <ExternalLink className="h-4 w-4" /> {t("booking.viewBookings")}
            </button>
          )}
          <button
            onClick={() => navigate("/search")}
            className="btn-ghost"
          >
            <ArrowLeft className="h-4 w-4" /> {t("booking.bookAnother")}
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="grid md:grid-cols-3 gap-8 animate-fade-in">
      {/* Seat Map */}
      <div className="md:col-span-2">
        <button onClick={() => navigate("/search")} className="btn-ghost mb-4">
          <ArrowLeft className="h-4 w-4" /> {t("booking.backToSearch")}
        </button>

        <div className="bg-white p-6 rounded-xl border border-gray-100 shadow-sm mb-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="w-9 h-9 bg-gradient-to-br from-travel-50 to-amber-50 rounded-lg flex items-center justify-center">
              <Bus className="h-5 w-5 text-travel-500" />
            </div>
            <div>
              <h2 className="font-semibold">
                {journey.route?.name || t("booking.title")}
              </h2>
              <p className="text-sm text-gray-500 flex items-center gap-2">
                <Clock className="h-3 w-3 text-travel-400" />
                {journey.departure_date} · {journey.departure_time} → {journey.arrival_time}
                <span className="text-travel-500 font-medium">| {journey.route?.base_price} {t("common.mad")}/seat</span>
              </p>
            </div>
          </div>

          <div className="relative mt-6">
            {/* Live indicator + Legend */}
            <div className="flex items-center justify-between mb-4">
              <div className={`flex items-center gap-1.5 transition-opacity duration-300 ${seatsUpdated ? "opacity-100" : "opacity-70"}`}>
                <span className={`relative flex h-2 w-2 ${seatsUpdated ? "animate-ping" : ""}`}>
                  <span className={`absolute inline-flex h-full w-full rounded-full ${seatsUpdated ? "bg-green-400" : "bg-green-500"} opacity-75`} />
                </span>
                <span className="relative inline-flex h-2 w-2 rounded-full bg-green-500" />
                <span className="text-[10px] text-green-600 font-medium uppercase tracking-wider">Live</span>
              </div>
              <div className="flex items-center gap-3 text-xs">
                <div className="flex items-center gap-1.5">
                  <div className="w-3.5 h-3.5 rounded bg-gray-100 border border-gray-200" />
                  <span className="text-gray-500">Free</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <div className="w-3.5 h-3.5 rounded bg-travel-500" />
                  <span className="text-gray-500">You</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <div className="w-3.5 h-3.5 rounded bg-red-50 border border-red-200 flex items-center justify-center">
                    <XCircle className="h-2.5 w-2.5 text-red-300" />
                  </div>
                  <span className="text-gray-500">Taken</span>
                </div>
              </div>
            </div>

            {seatsLoading ? (
              <div className="text-center py-8 text-gray-400 text-sm">{t("common.loading")}</div>
            ) : (
              <div className="flex justify-center">
                <div className="relative bg-gray-50 border-gray-300 px-5 pt-2 pb-4 shadow-inner"
                  style={{
                    borderRadius: "60px 60px 16px 16px",
                    border: "2px solid rgb(209 213 219)",
                  }}
                >
                  {/* Windshield section */}
                  <div className="relative -mx-3 -mt-2">
                    <div className="bg-gradient-to-b from-blue-100/80 to-transparent h-14 rounded-t-[50px] flex items-center justify-center"
                      style={{ borderRadius: "58px 58px 0 0" }}
                    >
                      <div className="flex items-center gap-3 text-xs text-blue-400/70">
                        <span>{seatRows * seatCols} seats</span>
                      </div>
                    </div>
                  </div>

                  {/* Passenger cabin */}
                  <div className="relative">
                    <div className="absolute left-0 top-0 bottom-0 w-0.5 bg-gray-200 rounded-full" />
                    <div className="absolute right-0 top-0 bottom-0 w-0.5 bg-gray-200 rounded-full" />

                    <div className="space-y-1.5">
                      {/* Driver row */}
                      <div className="flex items-center justify-center gap-1 mb-2">
                        <span className="w-4 text-[10px] text-gray-400 text-right mr-0.5 select-none" />
                        <div className="flex items-center gap-2 bg-amber-50 border border-amber-200 rounded-lg px-3 py-1.5 shadow-sm w-full justify-center">
                          <div className="w-5 h-5 rounded-full bg-amber-100 flex items-center justify-center">
                            <User className="w-3 h-3 text-amber-600" aria-hidden="true" />
                          </div>
                          <span className="text-xs font-medium text-amber-700">
                            {journey.bus?.driver_name || "Driver"}
                          </span>
                        </div>
                        <span className="w-4 text-[10px] text-gray-400 text-left ml-0.5 select-none" />
                      </div>

                      {Array.from({ length: seatRows }, (_, rowIdx) => {
                        const rowNum = rowIdx + 1
                        const leftSeats = Array.from({ length: leftCols }, (_, c) =>
                          `${rowNum}${String.fromCharCode(65 + c)}`
                        )
                        const rightSeats = Array.from({ length: rightCols }, (_, c) =>
                          `${rowNum}${String.fromCharCode(65 + leftCols + c)}`
                        )

                        const renderSeatBtn = (seat: string) => {
                          const isSelected = selectedSeats.includes(seat)
                          const isTaken = bookedSeats.includes(seat)
                          return (
                            <button
                              key={seat}
                              type="button"
                              disabled={isTaken}
                              onClick={() => {
                                if (isTaken) return
                                if (isSelected) {
                                  // Deselect: clear this seat from whichever passenger has it
                                  const idx = passengers.findIndex((p) => p.seat === seat)
                                  if (idx >= 0) updatePassenger(idx, "seat", "")
                                } else {
                                  // Select: assign to first passenger without a seat
                                  const emptyIdx = passengers.findIndex((p) => !p.seat)
                                  if (emptyIdx >= 0) {
                                    updatePassenger(emptyIdx, "seat", seat)
                                  }
                                }
                              }}
                              className={`w-9 h-9 rounded-lg text-xs font-medium transition-all duration-150 ${
                                isTaken
                                  ? "bg-red-50 text-red-300 border border-red-100 cursor-not-allowed opacity-70"
                                  : isSelected
                                    ? "bg-travel-500 text-white shadow-md shadow-travel-200 scale-105"
                                    : "bg-white text-gray-600 border border-gray-200 hover:border-travel-300 hover:shadow-sm hover:bg-travel-50/50 active:scale-95"
                              }`}
                            >
                              {isTaken ? <XCircle className="h-3 w-3 mx-auto" /> : seat.slice(-1)}
                            </button>
                          )
                        }

                        return (
                          <div key={rowNum} className="flex items-center justify-center gap-1">
                            <span className="w-4 text-[10px] text-gray-400 text-right mr-0.5 select-none">{rowNum}</span>
                            <div className="flex gap-1">
                              {leftSeats.map((s) => renderSeatBtn(s))}
                            </div>
                            <div className="w-4 flex items-center justify-center select-none">
                              <div className="w-0.5 h-4 bg-gray-200 rounded-full" />
                            </div>
                            <div className="flex gap-1">
                              {rightSeats.map((s) => renderSeatBtn(s))}
                            </div>
                          </div>
                        )
                      })}
                    </div>

                    <div className="flex items-center justify-center gap-6 mt-3 pt-2 border-t border-dashed border-gray-200">

                      <div className="flex items-center gap-1.5 text-[10px] text-gray-400">
                        {seatRows * seatCols - bookedSeats.length} free / {seatRows * seatCols} seats
                      </div>
                    </div>
                  </div>

                  <div className="absolute -bottom-1 left-4 right-4 h-2 bg-gray-200/50 blur-sm rounded-full" />
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Passenger Form */}
        <form onSubmit={handleSubmit} className="bg-white p-6 rounded-xl border border-gray-100 shadow-sm">
          <div className="flex items-center gap-2 mb-4">
            <User className="h-5 w-5 text-travel-500" />
            <h2 className="font-semibold">{t("booking.passengerDetails")}</h2>
          </div>

          {!isAuthenticated && (
            <div className="grid md:grid-cols-2 gap-4 mb-4 pb-4 border-b border-gray-100">
              <div>
                <label htmlFor="guest-email" className="text-sm font-medium text-gray-700">{t("booking.guestEmail")}</label>
                <div className="relative mt-1">
                  <Mail className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                  <input id="guest-email" name="guest_email" required value={guestEmail} onChange={(e) => setGuestEmail(e.target.value)}
                    className="w-full pl-10 pr-3 py-2 border border-gray-200 rounded-xl outline-none focus:ring-2 focus:ring-travel-400 focus:border-transparent bg-gray-50/50 transition-all duration-200"
                    placeholder="guest@example.com" />
                </div>
              </div>
              <div>
                <label htmlFor="guest-name" className="text-sm font-medium text-gray-700">{t("booking.guestName")}</label>
                <div className="relative mt-1">
                  <User className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                  <input id="guest-name" name="guest_name" required value={guestName} onChange={(e) => setGuestName(e.target.value)}
                    className="w-full pl-10 pr-3 py-2 border border-gray-200 rounded-xl outline-none focus:ring-2 focus:ring-travel-400 focus:border-transparent bg-gray-50/50 transition-all duration-200"
                    placeholder="John Doe" />
                </div>
              </div>
            </div>
          )}

          {passengers.map((p, i) => (
            <div key={i} className="grid md:grid-cols-3 gap-3 mb-3 p-3 bg-gradient-to-r from-travel-50/50 to-amber-50/50 rounded-xl border border-travel-100/50">
              <div>
                <label className="text-xs text-gray-500">{t("booking.seat")}</label>
                <p className="font-medium text-travel-600">{p.seat || "—"}</p>
              </div>
              <div>
                <label htmlFor={"passenger-name-" + i} className="text-xs text-gray-500">{t("booking.passengerName")}</label>
                <input id={"passenger-name-" + i} name={"passenger_name_" + i} required value={p.name} onChange={(e) => updatePassenger(i, "name", e.target.value)}
                  className="w-full px-2 py-1 border border-gray-200 rounded-lg text-sm outline-none focus:ring-1 focus:ring-travel-400 bg-white transition-all duration-200" />
              </div>
              <div className="flex items-end gap-2">
                <div className="flex-1">
                  <label htmlFor={"passenger-phone-" + i} className="text-xs text-gray-500">Phone</label>
                  <input id={"passenger-phone-" + i} name={"passenger_phone_" + i} value={p.phone} onChange={(e) => updatePassenger(i, "phone", e.target.value)}
                    className="w-full px-2 py-1 border border-gray-200 rounded-lg text-sm outline-none focus:ring-1 focus:ring-travel-400 bg-white transition-all duration-200" />
                </div>
                {passengers.length > 1 && (
                  <button type="button" onClick={() => removePassenger(i)} className="text-red-400 text-xs pb-1 hover:text-red-500 transition-colors">Remove</button>
                )}
              </div>
            </div>
          ))}
          <button type="button" onClick={addPassenger} className="text-sm text-travel-500 hover:text-travel-600 font-medium transition-colors">
            {t("booking.addPassenger")}
          </button>

          <div className="mt-6 pt-4 border-t border-gray-100 flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500">{t("booking.total")}: {passengers.length} {t("booking.tickets")}</p>
              <p className="text-xl font-bold text-gradient">
                {(journey.route?.base_price || 0) * passengers.length} {t("common.mad")}
              </p>
            </div>
            <button type="submit" disabled={loading}
              className="flex items-center gap-2 bg-gradient-to-r from-travel-500 to-rose-500 text-white px-6 py-2.5 rounded-xl font-medium
                hover:from-travel-600 hover:to-rose-600 transition-all duration-200 disabled:opacity-50 shadow-md shadow-travel-200/50 active:scale-[0.98]">
              <CreditCard className="h-4 w-4" /> {loading ? t("booking.processing") : t("booking.confirmBooking")}
            </button>
          </div>
        </form>
      </div>

      {/* Summary sidebar */}
      <div className="bg-white p-6 rounded-xl border border-gray-100 shadow-sm h-fit sticky top-24">
        <div className="flex items-center gap-2 mb-4">
          <MapPin className="h-4 w-4 text-travel-500" />
          <h3 className="font-semibold">{t("booking.tripSummary")}</h3>
        </div>
        <div className="space-y-3 text-sm text-gray-600">
          <div className="flex items-center gap-2 p-2 -mx-2 rounded-lg bg-gradient-to-r from-travel-50/50 to-amber-50/50">
            <MapPin className="h-4 w-4 text-travel-400 shrink-0" />
            <div>
              <p className="font-medium text-gray-800">{journey.route?.name || "—"}</p>
            </div>
          </div>
          <p><span className="font-medium">{t("search.date")}:</span> {journey.departure_date}</p>
          <p><span className="font-medium">{t("booking.time")}:</span> {journey.departure_time} → {journey.arrival_time}</p>
          <p><span className="font-medium">{t("booking.seats")}:</span> {selectedSeats.join(", ") || t("booking.noneSelected")}</p>
          <div className="pt-3 border-t border-gray-100">
            <p className="text-lg font-bold text-gradient">{(journey.route?.base_price || 0) * passengers.length} {t("common.mad")}</p>
          </div>
        </div>
      </div>
    </div>
  )
}
