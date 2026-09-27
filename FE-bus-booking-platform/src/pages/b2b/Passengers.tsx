import { useEffect, useState } from "react"
import { api } from "../../api/client"
import { Users, Phone, Bus as BusIcon, MapPin, Calendar } from "lucide-react"

interface TicketWithDetails {
  ticket_id: string
  passenger_name: string
  passenger_phone?: string
  seat_number: string
  price: number
  ticket_status: string
  journey_id: string
  departure_date: string
  departure_time: string
  bus_id: string
  registration_number: string
  origin: string
  destination: string
  booking_reference: string
  booking_status: string
  booked_at: string
}

export default function Passengers() {
  const [tickets, setTickets] = useState<TicketWithDetails[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busFilter, setBusFilter] = useState<string>("")
  const [journeyFilter, setJourneyFilter] = useState<string>("")

  const fetchTickets = async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      if (busFilter) params.set("bus_id", busFilter)
      if (journeyFilter) params.set("journey_id", journeyFilter)
      const query = params.toString() ? `?${params.toString()}` : ""
      const data = await api.get<TicketWithDetails[]>(`/tickets${query}`)
      setTickets(data)
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    fetchTickets()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [busFilter, journeyFilter])

  const uniqueBuses = Array.from(
    new Map(tickets.map((t) => [t.bus_id, t.registration_number])).entries()
  )
  const uniqueJourneys = Array.from(
    new Map(
      tickets.map((t) => [
        t.journey_id,
        `${t.origin} → ${t.destination} (${t.departure_date} ${t.departure_time})`,
      ])
    ).entries()
  )

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Passagers</h1>
        <p className="text-gray-500 text-sm mt-1">Liste de tous les passagers ayant réservé sur vos voyages</p>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
          {error}
        </div>
      )}

      <div className="flex flex-wrap gap-3">
        <select
          value={busFilter}
          onChange={(e) => setBusFilter(e.target.value)}
          className="px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="">Tous les bus</option>
          {uniqueBuses.map(([id, reg]) => (
            <option key={id} value={id}>{reg}</option>
          ))}
        </select>

        <select
          value={journeyFilter}
          onChange={(e) => setJourneyFilter(e.target.value)}
          className="px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="">Tous les voyages</option>
          {uniqueJourneys.map(([id, label]) => (
            <option key={id} value={id}>{label}</option>
          ))}
        </select>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-100">
          <h2 className="font-semibold">Réservations</h2>
          <p className="text-xs text-gray-400 mt-0.5">{tickets?.length ?? 0} billet(s)</p>
        </div>

        {loading ? (
          <div className="text-center py-20 text-gray-500">Chargement...</div>
        ) : (tickets?.length ?? 0) === 0 ? (
          <div className="px-6 py-12 text-center">
            <Users className="h-10 w-10 text-gray-300 mx-auto mb-3" />
            <p className="text-gray-500 text-sm">Aucun passager trouvé</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-100">
            {tickets.map((t) => (
              <div key={t.ticket_id} className="px-6 py-4 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 bg-blue-100 rounded-full flex items-center justify-center">
                    <Users className="h-5 w-5 text-blue-600" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-gray-900">{t.passenger_name}</p>
                    <div className="flex flex-wrap gap-x-4 gap-y-0.5 text-xs text-gray-500 mt-0.5">
                      {t.passenger_phone && (
                        <span className="flex items-center gap-1"><Phone className="h-3 w-3" /> {t.passenger_phone}</span>
                      )}
                      <span className="flex items-center gap-1"><MapPin className="h-3 w-3" /> {t.origin} → {t.destination}</span>
                      <span className="flex items-center gap-1"><BusIcon className="h-3 w-3" /> {t.registration_number}</span>
                      <span className="flex items-center gap-1"><Calendar className="h-3 w-3" /> {t.departure_date} {t.departure_time}</span>
                    </div>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-sm font-medium text-gray-900">Siège {t.seat_number}</p>
                  <p className="text-xs text-gray-500">{t.price} USD</p>
                  <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${
                    t.ticket_status === "active" ? "bg-green-50 text-green-700" : "bg-gray-100 text-gray-500"
                  }`}>
                    {t.ticket_status}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}