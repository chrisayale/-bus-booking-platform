import { useEffect, useState } from "react"
import { api } from "../../api/client"
import type { DashboardStats } from "../../types/analytics"
import { Bus, Route, Calendar, Users, TrendingUp, MapPin, Play, XCircle, Loader } from "lucide-react"
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts"

function formatDate(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" })
}

function formatTime(iso: string): string {
  // TIME values come as 0000-01-01T18:30:00Z — extract HH:MM
  const match = iso.match(/T(\d{2}:\d{2})/)
  return match ? match[1] : iso.slice(0, 5)
}

export default function OwnerDashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [updatingJourneys, setUpdatingJourneys] = useState<Set<string>>(new Set())

  const handleJourneyAction = async (journeyId: string, status: string) => {
    setUpdatingJourneys((prev) => new Set(prev).add(journeyId))
    try {
      await api.patch(`/journeys/${journeyId}/status`, { status })
      // Refresh stats to reflect the change
      const updated = await api.get<DashboardStats>("/analytics/dashboard")
      setStats(updated)
    } catch (err) {
      alert("Failed to update journey: " + (err as Error).message)
    } finally {
      setUpdatingJourneys((prev) => {
        const next = new Set(prev)
        next.delete(journeyId)
        return next
      })
    }
  }

  useEffect(() => {
    api.get<DashboardStats>("/analytics/dashboard")
      .then(setStats)
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return <div className="text-center py-20 text-gray-500">Loading dashboard...</div>
  }

  if (!stats) {
    return <div className="text-center py-20 text-gray-500">Failed to load analytics. Make sure you have data.</div>
  }

  const summaryCards = [
    { icon: Bus, label: "Total Buses", value: stats.total_buses, sub: `${stats.active_buses} active`, color: "bg-blue-500" },
    { icon: Route, label: "Routes", value: stats.total_routes, sub: `${stats.total_journeys} total journeys`, color: "bg-green-500" },
    { icon: Calendar, label: "Upcoming Journeys", value: stats.upcoming_journeys, sub: "scheduled", color: "bg-purple-500" },
    { icon: Users, label: "Seat Occupancy", value: `${Math.round(stats.seat_occupancy)}%`, sub: `${stats.active_tickets} active tickets`, color: "bg-orange-500" },
    { icon: TrendingUp, label: "Total Revenue", value: `${Math.round(stats.total_revenue)} USD`, sub: `${stats.total_bookings} bookings`, color: "bg-emerald-500" },
    { icon: MapPin, label: "Popular Routes", value: stats.popular_routes.length, sub: "top routes", color: "bg-rose-500" },
  ]

  const ticketStatusData = [
    { name: "Active", value: stats.active_tickets, fill: "#3b82f6" },
    { name: "Checked In", value: stats.checked_in, fill: "#22c55e" },
    { name: "Missed", value: stats.missed_tickets, fill: "#f97316" },
    { name: "Cancelled", value: stats.canceled_tickets, fill: "#ef4444" },
  ]

  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-bold">Dashboard</h1>

      {/* Summary Cards */}
      <div className="grid md:grid-cols-3 lg:grid-cols-6 gap-4">
        {summaryCards.map((c) => (
          <div key={c.label} className="bg-white rounded-xl p-4 shadow-sm border border-gray-100" role="region" aria-label={c.label}>
            <div className={`w-9 h-9 ${c.color} rounded-lg flex items-center justify-center mb-2`} aria-hidden="true">
              <c.icon className="h-4 w-4 text-white" aria-hidden="true" />
            </div>
            <p className="text-lg font-bold">{c.value}</p>
            <p className="text-xs text-gray-500">{c.label}</p>
            <p className="text-xs text-gray-400">{c.sub}</p>
          </div>
        ))}
      </div>

      <div className="grid md:grid-cols-2 gap-6">
        {/* Monthly Revenue Chart */}
        <div className="bg-white p-6 rounded-xl border border-gray-200" role="region" aria-label="Monthly revenue chart">
          <h2 className="font-semibold mb-4">Monthly Revenue (6 months)</h2>
          {stats.monthly_revenue.length > 0 ? (
            <div role="img" aria-label="Bar chart showing monthly revenue over the last 6 months">
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={stats.monthly_revenue}>
                <XAxis dataKey="month" tick={{ fontSize: 12 }} />
                <YAxis tick={{ fontSize: 12 }} />
                <Tooltip formatter={(value) => [`${Math.round(Number(value ?? 0))} USD`, "Revenue"]} />
                <Bar dataKey="amount" fill="#3b82f6" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
            </div>
          ) : (
            <p className="text-gray-400 text-sm py-12 text-center">No revenue data yet</p>
          )}
        </div>

        {/* Ticket Status Breakdown */}
        <div className="bg-white p-6 rounded-xl border border-gray-200" role="region" aria-label="Ticket status breakdown">
          <h2 className="font-semibold mb-4">Ticket Status Breakdown</h2>
          {stats.active_tickets + stats.checked_in + stats.missed_tickets + stats.canceled_tickets > 0 ? (
            <div className="space-y-3">
              {ticketStatusData.map((item) => {
                const total = Math.max(1, stats.active_tickets + stats.checked_in + stats.missed_tickets + stats.canceled_tickets)
                const pct = Math.round((item.value / total) * 100)
                return (
                  <div key={item.name} className="flex items-center gap-3">
                    <div className="w-3 h-3 rounded-full" style={{ backgroundColor: item.fill }} aria-hidden="true" />
                    <span className="text-sm text-gray-600 flex-1">{item.name}</span>
                    <span className="text-sm font-medium">{item.value}</span>
                    <div className="w-32 bg-gray-100 rounded-full h-2" role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100} aria-label={`${item.name}: ${pct}%`}>
                      <div
                        className="h-2 rounded-full transition-all"
                        style={{
                          width: `${pct}%`,
                          backgroundColor: item.fill,
                        }}
                      />
                    </div>
                  </div>
                )})
              }
            </div>
          ) : (
            <p className="text-gray-400 text-sm py-12 text-center">No ticket data yet</p>
          )}
        </div>

        {/* Popular Routes */}
        <div className="bg-white p-6 rounded-xl border border-gray-200" role="region" aria-label="Popular routes">
          <h2 className="font-semibold mb-4">Popular Routes</h2>
          {stats.popular_routes.length > 0 ? (
            <div className="space-y-3">
              {stats.popular_routes.map((route, i) => (
                <div key={i} className="flex items-center justify-between py-2 border-b border-gray-50 last:border-0">
                  <div>
                    <p className="text-sm font-medium">{route.route_name}</p>
                    <p className="text-xs text-gray-400">{route.origin} → {route.destination}</p>
                  </div>
                  <span className="text-sm font-bold text-blue-600">{route.ticket_count} tickets</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-gray-400 text-sm py-12 text-center">No route data yet</p>
          )}
        </div>

        {/* Recent Journeys */}
        <div className="bg-white p-6 rounded-xl border border-gray-200" role="region" aria-label="Recent journeys">
          <h2 className="font-semibold mb-4">Recent Journeys</h2>
          {stats.recent_journeys.length > 0 ? (
            <div className="space-y-2">
              {stats.recent_journeys.map((j) => (
                <div key={j.id} className="flex items-center justify-between py-2 border-b border-gray-50 last:border-0 gap-2">
                  <div className="flex items-center gap-3 min-w-0">
                    <Calendar className="h-4 w-4 text-gray-400 shrink-0" aria-hidden="true" />
                    <div>
                      <p className="text-sm font-medium">{formatDate(j.departure_date)} · {formatTime(j.departure_time)}</p>
                      <p className="text-xs text-gray-400">{j.booked_seats}/{j.total_seats} booked</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    {j.status === "scheduled" && (
                      <>
                        <button
                          onClick={() => handleJourneyAction(j.id, "boarding")}
                          disabled={updatingJourneys.has(j.id)}
                          className="flex items-center gap-1 text-xs px-2.5 py-1 rounded-lg bg-blue-50 text-blue-700 hover:bg-blue-100 transition disabled:opacity-50"
                          title="Start boarding"
                        >
                          {updatingJourneys.has(j.id) ? <Loader className="h-3 w-3 animate-spin" /> : <Play className="h-3 w-3" />}
                          Board
                        </button>
                        <button
                          onClick={() => handleJourneyAction(j.id, "cancelled")}
                          disabled={updatingJourneys.has(j.id)}
                          className="flex items-center gap-1 text-xs px-2.5 py-1 rounded-lg bg-red-50 text-red-700 hover:bg-red-100 transition disabled:opacity-50"
                          title="Cancel journey"
                        >
                          {updatingJourneys.has(j.id) ? <Loader className="h-3 w-3 animate-spin" /> : <XCircle className="h-3 w-3" />}
                          Cancel
                        </button>
                      </>
                    )}
                    <span className={`text-xs px-2 py-0.5 rounded-full ${
                      j.status === "completed" ? "bg-green-100 text-green-700" :
                      j.status === "departed" ? "bg-purple-100 text-purple-700" :
                      j.status === "boarding" ? "bg-blue-100 text-blue-700" :
                      j.status === "cancelled" ? "bg-red-100 text-red-700" :
                      "bg-yellow-100 text-yellow-700"
                    }`}>{j.status}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-gray-400 text-sm py-12 text-center">No journeys yet</p>
          )}
        </div>
      </div>
    </div>
  )
}
