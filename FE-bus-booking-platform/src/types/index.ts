export interface User {
  id: string
  email: string
  full_name: string
  phone?: string
  role: "owner" | "passenger" | "platform_admin" | "agent" | "controller" | "super_admin"
  organization_id?: string
  organizations?: Organization[]
  is_verified: boolean
  created_at: string
  pin?: string
  isActive: boolean
}

export interface Bus {
  id: string
  organization_id: string
  registration_number: string
  capacity: number
  seat_configuration: SeatConfig
  is_active: boolean
  image_url?: string
  driver_name?: string
  available_seats?: number
  created_at: string
  updated_at: string
}

export interface SeatConfig {
  rows: number
  columns: number
  layout: string
}

export interface Route {
  id: string
  organization_id: string
  name: string
  origin: string
  destination: string
  base_price: number
  stops: Stop[]
  created_at: string
}

export interface Stop {
  id: string
  route_id: string
  stop_order: number
  location: string
  arrival_offset_minutes?: number
  departure_offset_minutes?: number
  stop_price: number
  qr_token?: string
  status: "active" | "checked_in" | "missed" | "cancelled" | "refunded"
  checked_in_at?: string
  journey?: Journey
  created_at: string
}

export interface Organization {
  id: string
  name: string
  email: string
  phone?: string
  logo_url?: string
  is_active: boolean
}

export interface Journey {
  id: string
  organization_id: string
  route_id: string
  bus_id: string
  departure_date: string
  departure_time: string
  arrival_date: string
  arrival_time: string
  status: "scheduled" | "boarding" | "departed" | "completed" | "cancelled"
  route?: Route
  bus?: Bus
  organization?: Organization
  created_at: string
}

export interface Booking {
  id: string
  user_id?: string
  booking_reference: string
  guest_email?: string
  guest_name?: string
  total_amount: number
  status: "pending" | "confirmed" | "cancelled" | "refunded"
  tickets: Ticket[]
  created_at: string
}

export interface Ticket {
  id: string
  booking_id: string
  journey_id: string
  seat_number: string
  passenger_name: string
  passenger_phone?: string
  boarding_stop_id?: string
  dropoff_stop_id?: string
  price: number
  qr_token: string
  status: "active" | "checked_in" | "missed" | "cancelled" | "refunded"
  checked_in_at?: string
  journey?: Journey
  created_at: string
}

export interface AuthResponse {
  access_token: string
  refresh_token: string
  user: User
}

export interface SearchParams {
  origin: string
  destination: string
  date: string
}