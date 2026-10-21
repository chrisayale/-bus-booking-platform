import { useEffect } from "react"
import { useDispatch, useSelector } from "react-redux"
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { Provider } from "react-redux"
import { store } from "./store"
import type { AppDispatch, RootState } from "./store"
import { useAuth } from "./hooks/useAuth"
import { LanguageProvider } from "./context/LanguageContext"
import { fetchProfile } from "./store/authSlice"
import Layout from "./components/Layout"
import Landing from "./pages/Landing"
import Auth from "./pages/Auth"
import ForgotPassword from "./pages/ForgotPassword"
import ResetPassword from "./pages/ResetPassword"
import Settings from "./pages/Settings"
import OwnerDashboard from "./pages/b2b/Dashboard"
import Buses from "./pages/b2b/Buses"
import RoutesPage from "./pages/b2b/Routes"
import Journeys from "./pages/b2b/Journeys"
import QRScan from "./pages/b2b/QRScan"
import Admins from "./pages/b2b/Admins"
import Passengers from "./pages/b2b/Passengers"
import Revenue from "./pages/b2b/Revenue"
import Counter from "./pages/b2b/Counter"
import PlatformAdmin from "./pages/PlatformAdmin"
import AgentManagement from "./pages/AgentManagement"
import Search from "./pages/b2c/Search"
import Booking from "./pages/b2c/Booking"
import MyBookings from "./pages/b2c/MyBookings"

function ProtectedRoute({ children, ownerOnly = false, platformAdminOnly = false }: { children: React.ReactNode; ownerOnly?: boolean; platformAdminOnly?: boolean }) {
  const { isAuthenticated, isOwner, isPlatformAdmin, isLoading } = useAuth()
  if (isLoading) return <div className="flex items-center justify-center min-h-screen"><div className="w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full animate-spin" /></div>
  if (!isAuthenticated) return <Navigate to="/auth" replace />
  if (ownerOnly && !isOwner) return <Navigate to="/search" replace />
  if (platformAdminOnly && !isPlatformAdmin) return <Navigate to="/search" replace />
  return <>{children}</>
}

function AppRoutes() {
  const dispatch = useDispatch<AppDispatch>()
  const { isAuthenticated, user } = useSelector((state: RootState) => state.auth)

  // On page refresh, restore user profile from backend
  useEffect(() => {
    if (isAuthenticated && !user) {
      dispatch(fetchProfile())
    }
  }, [dispatch, isAuthenticated, user])

  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<Landing />} />
        <Route path="/auth" element={<Auth />} />
        <Route path="/auth/forgot-password" element={<ForgotPassword />} />
        <Route path="/auth/reset-password" element={<ResetPassword />} />
        <Route path="/search" element={<Search />} />
        <Route path="/booking" element={<Booking />} />
        <Route path="/settings" element={<ProtectedRoute><Settings /></ProtectedRoute>} />
        <Route path="/my-bookings" element={<ProtectedRoute><MyBookings /></ProtectedRoute>} />
        <Route path="/owner/dashboard" element={<ProtectedRoute ownerOnly><OwnerDashboard /></ProtectedRoute>} />
        <Route path="/owner/buses" element={<ProtectedRoute ownerOnly><Buses /></ProtectedRoute>} />
        <Route path="/owner/routes" element={<ProtectedRoute ownerOnly><RoutesPage /></ProtectedRoute>} />
        <Route path="/owner/journeys" element={<ProtectedRoute ownerOnly><Journeys /></ProtectedRoute>} />
        <Route path="/owner/scan" element={<ProtectedRoute ownerOnly><QRScan /></ProtectedRoute>} />
        <Route path="/owner/admins" element={<ProtectedRoute ownerOnly><Admins /></ProtectedRoute>} />
        <Route path="/owner/passengers" element={<ProtectedRoute ownerOnly><Passengers /></ProtectedRoute>} />
        <Route path="/owner/revenue" element={<ProtectedRoute ownerOnly><Revenue /></ProtectedRoute>} />
        <Route path="/owner/counter" element={<ProtectedRoute ownerOnly><Counter /></ProtectedRoute>} />
        <Route path="/platform/pending-owners" element={<ProtectedRoute platformAdminOnly><PlatformAdmin /></ProtectedRoute>} />
        <Route path="/platform/agents" element={<ProtectedRoute platformAdminOnly><AgentManagement /></ProtectedRoute>} />
      </Route>
    </Routes>
  )
}

export default function App() {
  return (
    <Provider store={store}>
      <LanguageProvider>
        <BrowserRouter>
          <AppRoutes />
        </BrowserRouter>
      </LanguageProvider>
    </Provider>
  )
}
