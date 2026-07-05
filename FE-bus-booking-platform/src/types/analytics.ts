export interface DashboardStats {
  total_buses: number
  active_buses: number
  total_routes: number
  total_journeys: number
  upcoming_journeys: number
  total_bookings: number
  total_revenue: number
  active_tickets: number
  checked_in: number
  missed_tickets: number
  canceled_tickets: number
  seat_occupancy: number
  monthly_revenue: MonthlyRevenue[]
  popular_routes: PopularRoute[]
  recent_journeys: JourneySummary[]
}

export interface MonthlyRevenue {
  month: string
  year: number
  amount: number
}

export interface PopularRoute {
  route_name: string
  origin: string
  destination: string
  ticket_count: number
}

export interface JourneySummary {
  id: string
  departure_date: string
  departure_time: string
  status: string
  booked_seats: number
  total_seats: number
}
