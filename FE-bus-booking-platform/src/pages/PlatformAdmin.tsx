import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { useDispatch } from "react-redux"
import { api } from "../api/client"
import { setTokens } from "../api/client"
import { fetchProfile } from "../store/authSlice"
import type { AppDispatch } from "../store"
import { Building2, Mail, Phone, Check, X, Power, LogIn, DollarSign, RefreshCw } from "lucide-react"

interface PendingAgency {
  id: string
  agency_name: string
  manager_name: string
  address: string
  contact_phone: string
  email: string
  created_at: string
}

interface Agency {
  id: string
  name: string
  email: string
  phone?: string
  is_active: boolean
  created_at: string
}

export default function PlatformAdmin() {
  const navigate = useNavigate()
  const dispatch = useDispatch<AppDispatch>()
  const [pending, setPending] = useState<PendingAgency[]>([])
  const [agencies, setAgencies] = useState<Agency[]>([])
  const [loading, setLoading] = useState(true)
  const [processingId, setProcessingId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [exchangeRate, setExchangeRate] = useState<number | null>(null)
  const [exchangeRateLoading, setExchangeRateLoading] = useState(false)
  const [exchangeRateError, setExchangeRateError] = useState<string | null>(null)
  const [exchangeRateSuccess, setExchangeRateSuccess] = useState<string | null>(null)

  const fetchAll = async () => {
    try {
      const [pendingData, agenciesData] = await Promise.all([
        api.get<any>("/platform-admin/pending-agencies"),
        api.get<any>("/platform-admin/agencies"),
      ])
      // Ensure we always set arrays to state, even if API returns non-array data
      setPending(Array.isArray(pendingData) ? pendingData : [])
      setAgencies(Array.isArray(agenciesData) ? agenciesData : [])
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setLoading(false)
    }
  }

  const fetchExchangeRate = async () => {
    setExchangeRateLoading(true)
    setExchangeRateError(null)
    setExchangeRateSuccess(null)
    try {
      const data = await api.get<any>("/admin/exchange-rate")
      // Ensure we have an object with the expected property
      if (data && typeof data === 'object' && 'usd_to_cdf' in data && typeof data.usd_to_cdf === 'number') {
        setExchangeRate(data.usd_to_cdf)
      } else {
        // Handle invalid response format
        setExchangeRateError("Invalid exchange rate data format")
        setExchangeRate(null)
      }
    } catch (err) {
      setExchangeRateError((err as Error).message)
    } finally {
      setExchangeRateLoading(false)
    }
  }

  const updateExchangeRate = async (rate: number) => {
    setExchangeRateLoading(true)
    setExchangeRateError(null)
    setExchangeRateSuccess(null)
    try {
      await api.put("/admin/exchange-rate", { usd_to_cdf: rate })
      setExchangeRateSuccess("Taux de change mis à jour avec succès")
      setExchangeRate(rate)
    } catch (err) {
      setExchangeRateError((err as Error).message)
    } finally {
      setExchangeRateLoading(false)
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    fetchAll()
    // Fetch exchange rate on load
    fetchExchangeRate()
  }, [])

  const handleApprove = async (id: string) => {
    setProcessingId(id)
    setError(null)
    setSuccess(null)
    try {
      await api.put(`/platform-admin/pending-agencies/${id}/approve`, {})
      setSuccess("Agence approuvée avec succès")
      fetchAll()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setProcessingId(null)
    }
  }

  const handleReject = async (id: string) => {
    setProcessingId(id)
    setError(null)
    setSuccess(null)
    try {
      await api.put(`/platform-admin/pending-agencies/${id}/reject`, {})
      setSuccess("Agence rejetée")
      fetchAll()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setProcessingId(null)
    }
  }

  const handleToggleActive = async (agency: Agency) => {
    setProcessingId(agency.id)
    setError(null)
    setSuccess(null)
    try {
      await api.put(`/platform-admin/agencies/${agency.id}/active`, { is_active: !agency.is_active })
      setSuccess(agency.is_active ? "Agence désactivée" : "Agence activée")
      fetchAll()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setProcessingId(null)
    }
  }

  const handleImpersonate = async (agency: Agency) => {
    setProcessingId(agency.id)
    setError(null)
    try {
      const data = await api.post<{ access_token: string; refresh_token: string }>(`/platform-admin/agencies/${agency.id}/impersonate`, {})
      setTokens(data.access_token, data.refresh_token)
      await dispatch(fetchProfile())
      navigate("/owner/dashboard")
    } catch (err) {
      setError((err as Error).message)
      setProcessingId(null)
    }
  }

  if (loading) {
    return <div className="text-center py-20 text-gray-500">Chargement...</div>
  }

  return (
    <div className="space-y-6 max-w-4xl mx-auto p-6">
      <div>
        <h1 className="text-2xl font-bold">Gestion des agences</h1>
        <p className="text-gray-500 text-sm mt-1">Approuvez les nouvelles demandes et gérez les agences existantes</p>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
          {error}
        </div>
      )}

      {success && (
        <div className="bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded-lg text-sm">
          {success}
        </div>
      )}

      {/* Exchange Rate Management Section */}
      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
          <h2 className="font-semibold">Taux de change USD/CDF</h2>
          <div className="flex items-center gap-2 text-sm">
            {exchangeRateLoading && (
              <RefreshCw className="h-4 w-4 text-blue-500 animate-spin" />
            )}
          </div>
        </div>

        {exchangeRateError && (
          <div className="px-6 py-3 bg-red-50 border border-red-200 text-red-700 rounded-b-lg">
            {exchangeRateError}
          </div>
        )}

        {exchangeRateSuccess && (
          <div className="px-6 py-3 bg-green-50 border border-green-200 text-green-700 rounded-b-lg">
            {exchangeRateSuccess}
          </div>
        )}

        <div className="px-6 py-4">
          <div className="space-y-4">
            <div className="flex items-center gap-4">
              <DollarSign className="h-5 w-5 text-gray-400" />
              <div>
                <p className="text-sm font-medium text-gray-900">
                  Taux de change actuel (1 USD = X CDF)
                </p>
                {exchangeRate === null ? (
                  <p className="text-xs text-gray-500">Chargement...</p>
                ) : (
                  <p className="text-2xl font-bold text-gray-900">
                    {exchangeRate.toFixed(2)}
                  </p>
                )}
              </div>
            </div>

            <div className="flex items-center gap-4">
              <label className="flex items-center gap-2 text-sm font-medium text-gray-700">
                Nouveau taux:
                <input
                  type="number"
                  min="0"
                  step="0.01"
                  value={exchangeRate !== null ? exchangeRate : 0}
                  onChange={(e) => {
                    const value = parseFloat(e.target.value)
                    setExchangeRate(isNaN(value) ? null : value)
                  }}
                  className="w-32 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  disabled={exchangeRateLoading}
                />
              </label>

              <button
                onClick={() => {
                  if (exchangeRate !== null && !exchangeRateLoading) {
                    updateExchangeRate(exchangeRate)
                  }
                }}
                disabled={exchangeRateLoading || exchangeRate === null}
                className="flex items-center gap-1 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
              >
                {exchangeRateLoading ? "Enregistrement..." : "Sauvegarder"}
                <RefreshCw className="h-4 w-4" />
              </button>
            </div>

            <div className="text-xs text-gray-500">
              Ce taux est utilisé pour convertir les prix entre USD et CDF dans tout le système.
            </div>
          </div>
        </div>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-100">
          <h2 className="font-semibold">Demandes en attente</h2>
          <p className="text-xs text-gray-400 mt-0.5">{pending?.length ?? 0} demande(s)</p>
        </div>

        {(pending?.length ?? 0) === 0 ? (
          <div className="px-6 py-12 text-center">
            <Building2 className="h-10 w-10 text-gray-300 mx-auto mb-3" />
            <p className="text-gray-500 text-sm">Aucune demande en attente</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-100">
            {pending.map((agency) => (
              <div key={agency.id} className="px-6 py-4 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 bg-orange-100 rounded-full flex items-center justify-center">
                    <Building2 className="h-5 w-5 text-orange-600" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-gray-900">{agency.agency_name}</p>
                    <p className="text-xs text-gray-500">Manager: {agency.manager_name}</p>
                    <p className="text-xs text-gray-500">Address: {agency.address}</p>
                    <p className="text-xs text-gray-500 flex items-center gap-1">
                      <Phone className="h-3 w-3" /> {agency.contact_phone}
                    </p>
                    <p className="text-xs text-gray-500 flex items-center gap-1">
                      <Mail className="h-3 w-3" /> {agency.email}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => handleApprove(agency.id)}
                    disabled={processingId === agency.id}
                    className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-white bg-green-600 rounded-lg hover:bg-green-700 transition disabled:opacity-50"
                  >
                    <Check className="h-4 w-4" /> Approuver
                  </button>
                  <button
                    onClick={() => handleReject(agency.id)}
                    disabled={processingId === agency.id}
                    className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition disabled:opacity-50"
                  >
                    <X className="h-4 w-4" /> Rejeter
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-100">
          <h2 className="font-semibold">Agences</h2>
          <p className="text-xs text-gray-400 mt-0.5">{agencies?.length ?? 0} agence(s)</p>
        </div>

        {(agencies?.length ?? 0) === 0 ? (
          <div className="px-6 py-12 text-center">
            <Building2 className="h-10 w-10 text-gray-300 mx-auto mb-3" />
            <p className="text-gray-500 text-sm">Aucune agence enregistrée</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-100">
            {agencies.map((agency) => (
              <div key={agency.id} className="px-6 py-4 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className={`w-10 h-10 rounded-full flex items-center justify-center ${agency.is_active ? "bg-green-100" : "bg-gray-100"}`}>
                    <Building2 className={`h-5 w-5 ${agency.is_active ? "text-green-600" : "text-gray-400"}`} />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-gray-900">{agency.name}</p>
                    <p className="text-xs text-gray-500 flex items-center gap-1">
                      <Mail className="h-3 w-3" /> {agency.email}
                    </p>
                    {agency.phone && (
                      <p className="text-xs text-gray-500 flex items-center gap-1">
                        <Phone className="h-3 w-3" /> {agency.phone}
                      </p>
                    )}
                  </div>
                </div>
                                <div className="flex items-center gap-3">
                  <span className={`text-xs px-2 py-1 rounded-full font-medium ${agency.is_active ? "bg-green-50 text-green-700" : "bg-gray-100 text-gray-500"}`}>
                    {agency.is_active ? "Active" : "Désactivée"}
                  </span>
                  {agency.is_active && (
                    <button
                      onClick={() => handleImpersonate(agency)}
                      disabled={processingId === agency.id}
                      className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
                    >
                      <LogIn className="h-4 w-4" /> Se connecter
                    </button>
                  )}
                  <button
                    onClick={() => handleToggleActive(agency)}
                    disabled={processingId === agency.id}
                    className={`flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-white rounded-lg transition disabled:opacity-50 ${
                      agency.is_active ? "bg-red-600 hover:bg-red-700" : "bg-green-600 hover:bg-green-700"
                    }`}
                  >
                    <Power className="h-4 w-4" /> {agency.is_active ? "Désactiver" : "Activer"}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}