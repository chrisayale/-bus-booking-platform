import { useEffect, useState } from "react"
import { api } from "../../api/client"
import type { Bus, Route, Journey } from "../../types"
import { Plus, Calendar } from "lucide-react"

export default function Journeys() {
  const [journeys, setJourneys] = useState<Journey[]>([])
  const [buses, setBuses] = useState<Bus[]>([])
  const [routes, setRoutes] = useState<Route[]>([])
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({
    route_id: "", bus_id: "", departure_date: "", departure_time: "",
    arrival_date: "", arrival_time: "",
  })

  useEffect(() => {
    Promise.all([
      api.get<Journey[]>("/journeys"),
      api.get<Bus[]>("/buses"),
      api.get<Route[]>("/routes"),
    ]).then(([j, b, r]) => {
      setJourneys(j)
      setBuses(b.filter((x) => x.is_active))
      setRoutes(r)
    }).catch(console.error)
  }, [])

  const createJourney = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await api.post("/journeys", form)
      const updated = await api.get<Journey[]>("/journeys")
      setJourneys(updated)
      setShowForm(false)
    } catch (err) {
      alert((err as Error).message)
    }
  }

  const updateStatus = async (id: string, status: string) => {
    try {
      await api.patch(`/journeys/${id}/status`, { status })
      setJourneys(journeys.map((j) => (j.id === id ? { ...j, status: status as Journey["status"] } : j)))
    } catch (err) {
      alert((err as Error).message)
    }
  }

  const statusColors: Record<string, string> = {
    scheduled: "bg-yellow-100 text-yellow-700",
    boarding: "bg-blue-100 text-blue-700",
    departed: "bg-purple-100 text-purple-700",
    completed: "bg-green-100 text-green-700",
    cancelled: "bg-red-100 text-red-700",
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Journeys</h1>
        <button onClick={() => setShowForm(true)}
          className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-blue-700 transition">
          <Plus className="h-4 w-4" /> Schedule Journey
        </button>
      </div>

      {showForm && (
        <form onSubmit={createJourney} className="bg-white p-6 rounded-xl border border-gray-200 mb-6 space-y-4">
          <div className="grid md:grid-cols-2 gap-4">
            <select id="journey-route" name="route_id" required value={form.route_id} onChange={(e) => setForm({ ...form, route_id: e.target.value })}
              className="px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500">
              <option value="">Select Route</option>
              {routes.map((r) => <option key={r.id} value={r.id}>{r.name} ({r.origin}→{r.destination})</option>)}
            </select>
            <select id="journey-bus" name="bus_id" required value={form.bus_id} onChange={(e) => setForm({ ...form, bus_id: e.target.value })}
              className="px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500">
              <option value="">Select Bus</option>
              {buses.map((b) => <option key={b.id} value={b.id}>{b.registration_number} ({b.capacity} seats)</option>)}
            </select>
            <div>
              <label htmlFor="journey-dep-date" className="text-xs text-gray-500">Departure Date</label>
              <input id="journey-dep-date" name="departure_date" type="date" required value={form.departure_date} onChange={(e) => setForm({ ...form, departure_date: e.target.value })}
                className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
            <div>
              <label htmlFor="journey-dep-time" className="text-xs text-gray-500">Departure Time</label>
              <input id="journey-dep-time" name="departure_time" type="time" required value={form.departure_time} onChange={(e) => setForm({ ...form, departure_time: e.target.value })}
                className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
            <div>
              <label htmlFor="journey-arr-date" className="text-xs text-gray-500">Arrival Date</label>
              <input id="journey-arr-date" name="arrival_date" type="date" required value={form.arrival_date} onChange={(e) => setForm({ ...form, arrival_date: e.target.value })}
                className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
            <div>
              <label htmlFor="journey-arr-time" className="text-xs text-gray-500">Arrival Time</label>
              <input id="journey-arr-time" name="arrival_time" type="time" required value={form.arrival_time} onChange={(e) => setForm({ ...form, arrival_time: e.target.value })}
                className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
          </div>
          <div className="flex gap-3 justify-end">
            <button type="button" onClick={() => setShowForm(false)} className="px-4 py-2 text-sm text-gray-600">Cancel</button>
            <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition">Create Journey</button>
          </div>
        </form>
      )}

      <div className="space-y-3">
        {journeys.map((j) => (
          <div key={j.id} className="bg-white p-5 rounded-xl border border-gray-200 flex items-center justify-between">
            <div className="flex items-center gap-4">
              <Calendar className="h-8 w-8 text-blue-500" />
              <div>
                <p className="font-semibold">{j.departure_date} at {j.departure_time}</p>
                <p className="text-sm text-gray-500">
                  {j.route ? `${j.route.origin} → ${j.route.destination}` : `Route: ${j.route_id.slice(0, 8)}...`}
                  {j.bus && <span> · {j.bus.registration_number}</span>}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <span className={`text-xs px-2 py-1 rounded-full ${statusColors[j.status] || "bg-gray-100"}`}>{j.status}</span>
              {j.status === "scheduled" && (
                <button onClick={() => updateStatus(j.id, "boarding")} className="text-xs text-blue-600 hover:underline">Start Boarding</button>
              )}
              {j.status === "boarding" && (
                <button onClick={() => updateStatus(j.id, "departed")} className="text-xs text-purple-600 hover:underline">Mark Departed</button>
              )}
              {j.status === "departed" && (
                <button onClick={() => updateStatus(j.id, "completed")} className="text-xs text-green-600 hover:underline">Complete</button>
              )}
              {(j.status === "scheduled" || j.status === "boarding") && (
                <button onClick={() => updateStatus(j.id, "cancelled")} className="text-xs text-red-500 hover:underline">Cancel</button>
              )}
            </div>
          </div>
        ))}
        {journeys.length === 0 && !showForm && (
          <p className="text-gray-500 text-center py-8">No journeys scheduled yet.</p>
        )}
      </div>
    </div>
  )
}
