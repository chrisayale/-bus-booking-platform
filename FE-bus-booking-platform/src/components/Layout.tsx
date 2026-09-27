import { Link, Outlet, useNavigate, useLocation } from "react-router-dom"
import { useDispatch } from "react-redux"
import { useAuth } from "../hooks/useAuth"
import { useTranslate } from "../hooks/useTranslate"
import { logout } from "../store/authSlice"
import {
       Bus, Route, Calendar, QrCode, Search, Ticket, LayoutDashboard,
  LogOut, User, Settings, Shield, Users, DollarSign, Store
} from "lucide-react"

export default function Layout() {
    const { isAuthenticated, isOwner, isPlatformAdmin, user } = useAuth()
  const { t } = useTranslate()
  const dispatch = useDispatch()
  const navigate = useNavigate()
  const location = useLocation()

  const handleLogout = () => {
    dispatch(logout())
    navigate("/")
  }

  const isActive = (path: string) => location.pathname === path || location.pathname.startsWith(path + "/")

  return (
    <div className="min-h-screen gradient-warm">
      {/* Navbar */}
      <nav className="sticky top-0 z-50 bg-white/80 backdrop-blur-xl border-b border-gray-200/60 supports-[backdrop-filter]:bg-white/70">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16 items-center">
            {/* Logo */}
            <Link to="/" className="flex items-center gap-2 group">
              <div className="w-9 h-9 bg-gradient-to-br from-travel-500 to-rose-500 rounded-xl flex items-center justify-center
                group-hover:scale-110 transition-transform duration-200 shadow-sm">
                <Bus className="h-5 w-5 text-white" aria-hidden="true" />
              </div>
              <span className="text-lg font-bold text-gray-900">
                Ayale<span className="text-travel-500">Trans</span>
              </span>
            </Link>

            {/* Navigation */}
            <div className="hidden md:flex items-center gap-1">
                           {isAuthenticated ? (
                <>
                  {isPlatformAdmin ? (
                    <div className="flex items-center gap-1">
                      <NavLink to="/platform/pending-owners" icon={Shield} label="Demandes agences" active={isActive("/platform/pending-owners")} />
                    </div>
                  ) : isOwner ? (
                    <div className="flex items-center gap-1">
                      <NavLink to="/owner/dashboard" icon={LayoutDashboard} label={t("nav.dashboard")} active={isActive("/owner/dashboard")} />
                      <NavLink to="/owner/buses" icon={Bus} label={t("nav.fleet")} active={isActive("/owner/buses")} />
                      <NavLink to="/owner/routes" icon={Route} label={t("nav.routes")} active={isActive("/owner/routes")} />
                      <NavLink to="/owner/journeys" icon={Calendar} label={t("nav.journeys")} active={isActive("/owner/journeys")} />
                      <NavLink to="/owner/scan" icon={QrCode} label={t("nav.qrScan")} active={isActive("/owner/scan")} />
                                            <NavLink to="/owner/admins" icon={Shield} label="Admins" active={isActive("/owner/admins")} />
                                           <NavLink to="/owner/passengers" icon={Users} label="Passagers" active={isActive("/owner/passengers")} />
                                            <NavLink to="/owner/revenue" icon={DollarSign} label="Revenus" active={isActive("/owner/revenue")} />
                      <NavLink to="/owner/counter" icon={Store} label="Guichet" active={isActive("/owner/counter")} />
                    </div>
                  ) : (
                    <div className="flex items-center gap-1">
                      <NavLink to="/search" icon={Search} label={t("nav.search")} active={isActive("/search")} />
                      <NavLink to="/my-bookings" icon={Ticket} label={t("nav.myBookings")} active={isActive("/my-bookings")} />
                    </div>
                  )}
                </>
              ) : (
                <div className="flex items-center gap-1">
                  <NavLink to="/search" icon={Search} label={t("nav.search")} active={isActive("/search")} />
                </div>
              )}
            </div>

            {/* Right side */}
            <div className="flex items-center gap-2">
              {isAuthenticated ? (
                <>
                  <Link
                    to="/settings"
                    className="p-2 text-gray-400 hover:text-travel-600 hover:bg-travel-50 rounded-lg transition-all duration-200"
                    aria-label={t("nav.settings")}
                  >
                    <Settings className="h-4 w-4" aria-hidden="true" />
                  </Link>
                  <div className="flex items-center gap-2 pl-3 border-l border-gray-200">
                    <div className="w-7 h-7 bg-gradient-to-br from-travel-400 to-rose-400 rounded-full flex items-center justify-center">
                      <User className="h-3.5 w-3.5 text-white" aria-hidden="true" />
                    </div>
                    <span className="text-sm font-medium text-gray-700 hidden sm:inline">{user?.full_name}</span>
                    <button
                      onClick={handleLogout}
                      className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-all duration-200"
                      aria-label={t("nav.signOut")}
                    >
                      <LogOut className="h-4 w-4" aria-hidden="true" />
                    </button>
                  </div>
                </>
              ) : (
                <>
                  <Link to="/search" className="btn-ghost hidden sm:flex">
                    <Search className="h-4 w-4" />
                    {t("nav.search")}
                  </Link>
                  <Link
                    to="/auth"
                    className="bg-gradient-to-r from-travel-500 to-rose-500 text-white px-4 py-2 rounded-xl text-sm font-medium
                      hover:from-travel-600 hover:to-rose-600 transition-all duration-200 shadow-sm shadow-travel-200/50"
                  >
                    {t("nav.signIn")}
                  </Link>
                </>
              )}
            </div>
          </div>
        </div>
      </nav>

      {/* Mobile bottom nav */}
      {isAuthenticated && (
        <div className="md:hidden fixed bottom-0 left-0 right-0 z-50 bg-white/90 backdrop-blur-xl border-t border-gray-200 safe-area-bottom">
                    <div className="flex items-center gap-1 overflow-x-auto py-2 px-2">
                    {isOwner ? (
              <>
                <MobileNavLink to="/owner/dashboard" icon={LayoutDashboard} label={t("nav.dashboard")} active={isActive("/owner/dashboard")} />
                <MobileNavLink to="/owner/buses" icon={Bus} label={t("nav.fleet")} active={isActive("/owner/buses")} />
                <MobileNavLink to="/owner/routes" icon={Route} label={t("nav.routes")} active={isActive("/owner/routes")} />
                <MobileNavLink to="/owner/journeys" icon={Calendar} label={t("nav.journeys")} active={isActive("/owner/journeys")} />
                <MobileNavLink to="/owner/scan" icon={QrCode} label={t("nav.qrScan")} active={isActive("/owner/scan")} />
                <MobileNavLink to="/owner/admins" icon={Shield} label="Admins" active={isActive("/owner/admins")} />
                <MobileNavLink to="/owner/passengers" icon={Users} label="Passagers" active={isActive("/owner/passengers")} />
                               <MobileNavLink to="/owner/revenue" icon={DollarSign} label="Revenus" active={isActive("/owner/revenue")} />
                <MobileNavLink to="/owner/counter" icon={Store} label="Guichet" active={isActive("/owner/counter")} />
              </>
            ) : (
              <>
                <MobileNavLink to="/search" icon={Search} label={t("nav.search")} active={isActive("/search")} />
                <MobileNavLink to="/my-bookings" icon={Ticket} label={t("nav.myBookings")} active={isActive("/my-bookings")} />
                <MobileNavLink to="/settings" icon={Settings} label={t("nav.settings")} active={isActive("/settings")} />
              </>
            )}
          </div>
        </div>
      )}

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 pb-24 md:pb-8 animate-fade-in">
        <Outlet />
      </main>
    </div>
  )
}

function NavLink({
  to,
  icon: Icon,
  label,
  active,
}: {
  to: string
  icon: React.ComponentType<{ className?: string }>
  label: string
  active: boolean
}) {
  return (
    <Link
      to={to}
      className={`flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
        active
          ? "bg-travel-50 text-travel-700 shadow-sm"
          : "text-gray-600 hover:text-travel-600 hover:bg-travel-50/50"
      }`}
    >
      <Icon className="h-4 w-4" aria-hidden="true" />
      {label}
    </Link>
  )
}

function MobileNavLink({
  to,
  icon: Icon,
  label,
  active,
}: {
  to: string
  icon: React.ComponentType<{ className?: string }>
  label: string
  active: boolean
}) {
  return (
    <Link
      to={to}
      className={`flex flex-col items-center gap-0.5 px-3 py-1 rounded-lg transition-all duration-200 ${
        active ? "text-travel-600" : "text-gray-400"
      }`}
    >
      <Icon className="h-5 w-5" aria-hidden="true" />
      <span className="text-[10px] font-medium">{label}</span>
    </Link>
  )
}
