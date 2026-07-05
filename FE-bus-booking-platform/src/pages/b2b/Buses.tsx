import { useEffect, useState } from "react"
import { api } from "../../api/client"
import type { Bus } from "../../types"
import { Plus, Trash2 } from "lucide-react"

export default function Buses() {
  const [buses, setBuses] = useState<Bus[]>([])
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({ registration_number: "", capacity: 40, driver_name: "" })

  useEffect(() => {
    api.get<Bus[]>("/buses").then(setBuses).catch(console.error)
  }, [])

  const createBus = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await api.post("/buses", form)
      const updated = await api.get<Bus[]>("/buses")
      setBuses(updated)
      setShowForm(false)
      setForm({ registration_number: "", capacity: 40, driver_name: "" })
    } catch (err) {
      alert((err as Error).message)
    }
  }

  const deleteBus = async (id: string) => {
    if (!confirm("Delete this bus?")) return
    try {
      await api.delete(`/buses/${id}`)
      setBuses(buses.filter((b) => b.id !== id))
    } catch (err) {
      alert((err as Error).message)
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Fleet Management</h1>
        <button
          onClick={() => setShowForm(true)}
          className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-blue-700 transition"
        >
          <Plus className="h-4 w-4" /> Add Bus
        </button>
      </div>

      {showForm && (
        <form onSubmit={createBus} className="bg-white p-6 rounded-xl border border-gray-200 mb-6 space-y-4">
          <div className="grid md:grid-cols-3 gap-4">
            <div>
              <label htmlFor="bus-reg" className="text-sm font-medium text-gray-700">Registration Number</label>
              <input
                id="bus-reg"
                name="registration_number"
                required
                value={form.registration_number}
                onChange={(e) => setForm({ ...form, registration_number: e.target.value })}
                className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                placeholder="1234-A-5678"
              />
            </div>
            <div>
              <label htmlFor="bus-capacity" className="text-sm font-medium text-gray-700">Capacity (seats)</label>
              <input
                id="bus-capacity"
                name="capacity"
                type="number"
                required
                min={1}
                value={form.capacity}
                onChange={(e) => setForm({ ...form, capacity: parseInt(e.target.value) })}
                className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
              />
            </div>
            <div>
              <label htmlFor="bus-driver" className="text-sm font-medium text-gray-700">Driver Name</label>
              <input
                id="bus-driver"
                name="driver_name"
                value={form.driver_name}
                onChange={(e) => setForm({ ...form, driver_name: e.target.value })}
                className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                placeholder="e.g. Ahmed El Mansouri"
              />
            </div>
          </div>
          <div className="flex gap-3 justify-end">
            <button type="button" onClick={() => setShowForm(false)} className="px-4 py-2 text-sm text-gray-600 hover:text-gray-800">
              Cancel
            </button>
            <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition">
              Save Bus
            </button>
          </div>
        </form>
      )}

      <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
        {buses.map((bus) => (
          <div key={bus.id} className="bg-white p-5 rounded-xl border border-gray-200 hover:shadow-sm transition">
            <div className="flex items-start justify-between">
              <div>
                <h3 className="font-semibold text-lg">{bus.registration_number}</h3>
                <p className="text-sm text-gray-500">{bus.capacity} seats</p>
                {bus.driver_name && (
                  <p className="text-xs text-amber-600 mt-1 flex items-center gap-1">
                    <span>🚌</span> Driver: {bus.driver_name}
                  </p>
                )}
              </div>
              <button onClick={() => deleteBus(bus.id)} className="p-1 text-gray-400 hover:text-red-500">
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
            <span className={`inline-block mt-3 text-xs px-2 py-1 rounded-full ${bus.is_active ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>
              {bus.is_active ? "Active" : "Inactive"}
            </span>
          </div>
        ))}
        {buses.length === 0 && !showForm && (
          <p className="text-gray-500 col-span-full text-center py-8">No buses yet. Add your first bus!</p>
        )}
      </div>
    </div>
  )
}
