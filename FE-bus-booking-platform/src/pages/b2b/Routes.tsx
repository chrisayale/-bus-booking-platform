import { useEffect, useState } from "react"
import { api } from "../../api/client"
import type { Route } from "../../types"
import { Plus, Trash2, MapPin } from "lucide-react"

export default function Routes() {
  const [routes, setRoutes] = useState<Route[]>([])
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({
    name: "",
    origin: "",
    destination: "",
    base_price: 50,
    stops: [{ location: "", stop_order: 0, stop_price: 0 }],
  })

  useEffect(() => {
    api.get<any>("/routes").then((data) => setRoutes(Array.isArray(data) ? data : [])).catch(console.error)
  }, [])

  const createRoute = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await api.post("/routes", form)
      const updated = await api.get<any>("/routes")
      setRoutes(Array.isArray(updated) ? updated : [])
      setShowForm(false)
      setForm({ name: "", origin: "", destination: "", base_price: 50, stops: [{ location: "", stop_order: 0, stop_price: 0 }] })
    } catch (err) {
      alert((err as Error).message)
    }
  }

  const deleteRoute = async (id: string) => {
    if (!confirm("Delete this route?")) return
    try {
      await api.delete(`/routes/${id}`)
      setRoutes((routes ?? []).filter((r) => r.id !== id))
    } catch (err) {
      alert((err as Error).message)
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Routes</h1>
        <button
          onClick={() => setShowForm(true)}
          className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-blue-700 transition"
        >
          <Plus className="h-4 w-4" /> Add Route
        </button>
      </div>

      {showForm && (
        <form onSubmit={createRoute} className="bg-white p-6 rounded-xl border border-gray-200 mb-6 space-y-4">
          <div className="grid md:grid-cols-2 gap-4">
            <div>
              <label htmlFor="route-name" className="text-sm font-medium text-gray-700">Route Name</label>
              <input id="route-name" name="name" required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
                className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500" placeholder="Kinshasa → Lubumbashi" />
            </div>
            <div>
              <label htmlFor="route-price" className="text-sm font-medium text-gray-700">Base Price (USD)</label>
              <input id="route-price" name="base_price" type="number" required min={0} value={form.base_price} onChange={(e) => setForm({ ...form, base_price: parseFloat(e.target.value) })}
                className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
            <div>
              <label htmlFor="route-origin" className="text-sm font-medium text-gray-700">Origin</label>
              <input id="route-origin" name="origin" required value={form.origin} onChange={(e) => setForm({ ...form, origin: e.target.value })}
                className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500" placeholder="Kinshasa" />
            </div>
            <div>
              <label htmlFor="route-destination" className="text-sm font-medium text-gray-700">Destination</label>
              <input id="route-destination" name="destination" required value={form.destination} onChange={(e) => setForm({ ...form, destination: e.target.value })}
                className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500" placeholder="Lubumbashi" />
            </div>
          </div>

          {/* Stops */}
          <div>
            <label className="text-sm font-medium text-gray-700">Intermediary Stops</label>
            {form.stops.map((stop, i) => (
              <div key={i} className="flex items-center gap-2 mt-2">
                <MapPin className="h-4 w-4 text-gray-400" />
                <input id={"stop-location-" + i} name={"stop_location_" + i} value={stop.location} onChange={(e) => {
                  const stops = [...form.stops]; stops[i].location = e.target.value; setForm({ ...form, stops })
                }} className="flex-1 px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500" placeholder="Stop location" />
                <button type="button" onClick={() => setForm({ ...form, stops: form.stops.filter((_, j) => j !== i) })}
                  className="p-1 text-red-400 hover:text-red-600"><Trash2 className="h-4 w-4" /></button>
              </div>
            ))}
            <button type="button" onClick={() => setForm({ ...form, stops: [...form.stops, { location: "", stop_order: form.stops.length, stop_price: 0 }] })}
              className="mt-2 text-sm text-blue-600 hover:text-blue-800">+ Add stop</button>
          </div>

          <div className="flex gap-3 justify-end">
            <button type="button" onClick={() => setShowForm(false)} className="px-4 py-2 text-sm text-gray-600">Cancel</button>
            <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition">Save Route</button>
          </div>
        </form>
      )}

      <div className="space-y-4">
        {routes?.map((route) => (
          <div key={route.id} className="bg-white p-5 rounded-xl border border-gray-200">
            <div className="flex items-start justify-between">
              <div>
                <h3 className="font-semibold text-lg">{route.name}</h3>
                <p className="text-sm text-gray-500">{route.origin} → {route.destination}</p>
                <p className="text-sm font-medium text-blue-600">{route.base_price} USD base</p>
                {route.stops?.length > 0 && (
                  <div className="flex items-center gap-2 mt-2 text-xs text-gray-500">
                    <MapPin className="h-3 w-3" /> {route.stops?.length ?? 0} stops
                  </div>
                )}
              </div>
              <button onClick={() => deleteRoute(route.id)} className="p-1 text-gray-400 hover:text-red-500"><Trash2 className="h-4 w-4" /></button>
            </div>
          </div>
        ))}
        {(routes?.length ?? 0) === 0 && !showForm && (
          <p className="text-gray-500 text-center py-8">No routes yet. Create your first route!</p>
        )}
      </div>
    </div>
  )
}
