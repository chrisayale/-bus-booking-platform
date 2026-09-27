import { useEffect, useState } from "react"
import { api } from "../../api/client"
import { DollarSign, Bus, Calendar } from "lucide-react"

interface BusRevenue {
  bus_id: string
  registration_number: string
  ticket_count: number
  revenue: number
}

interface RevenueReport {
  total_revenue: number
  by_bus: BusRevenue[]
}

function firstDayOfMonth() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-01`
}

function today() {
  return new Date().toISOString().split("T")[0]
}

export default function Revenue() {
  const [startDate, setStartDate] = useState(firstDayOfMonth())
  const [endDate, setEndDate] = useState(today())
  const [report, setReport] = useState<RevenueReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchReport = async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await api.get<RevenueReport>(
        `/analytics/revenue?start_date=${startDate}&end_date=${endDate}`
      )
      setReport(data)
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    fetchReport()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Revenus</h1>
        <p className="text-gray-500 text-sm mt-1">Revenu combiné et par bus sur une période donnée</p>
      </div>

      <div className="flex flex-wrap items-end gap-3 bg-white p-4 rounded-xl border border-gray-200">
        <div>
          <label className="text-xs font-medium text-gray-500 flex items-center gap-1"><Calendar className="h-3 w-3" /> Du</label>
          <input
            type="date"
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
            className="mt-1 px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <div>
          <label className="text-xs font-medium text-gray-500 flex items-center gap-1"><Calendar className="h-3 w-3" /> Au</label>
          <input
            type="date"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            className="mt-1 px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <button
          onClick={fetchReport}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition"
        >
          Appliquer
        </button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
          {error}
        </div>
      )}

      {loading ? (
        <div className="text-center py-20 text-gray-500">Chargement...</div>
      ) : report && (
        <>
          <div className="bg-white rounded-xl border border-gray-200 p-6 flex items-center gap-4">
            <div className="w-12 h-12 bg-green-100 rounded-full flex items-center justify-center">
              <DollarSign className="h-6 w-6 text-green-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">Revenu combiné (toute l'agence)</p>
              <p className="text-3xl font-bold text-gray-900">{report.total_revenue.toFixed(2)} USD</p>
            </div>
          </div>

          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            <div className="px-6 py-4 border-b border-gray-100">
              <h2 className="font-semibold">Revenu par bus</h2>
            </div>
            {(report.by_bus?.length ?? 0) === 0 ? (
              <div className="px-6 py-12 text-center">
                <Bus className="h-10 w-10 text-gray-300 mx-auto mb-3" />
                <p className="text-gray-500 text-sm">Aucune donnée pour cette période</p>
              </div>
            ) : (
              <div className="divide-y divide-gray-100">
                {report.by_bus.map((b) => (
                  <div key={b.bus_id} className="px-6 py-4 flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-blue-100 rounded-full flex items-center justify-center">
                        <Bus className="h-5 w-5 text-blue-600" />
                      </div>
                      <div>
                        <p className="text-sm font-medium text-gray-900">{b.registration_number}</p>
                        <p className="text-xs text-gray-500">{b.ticket_count} billet(s)</p>
                      </div>
                    </div>
                    <p className="text-lg font-bold text-gray-900">{b.revenue.toFixed(2)} USD</p>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  )
}