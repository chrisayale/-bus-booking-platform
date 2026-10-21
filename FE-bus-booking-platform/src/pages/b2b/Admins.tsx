import { useEffect, useState } from "react"
import { useSelector } from "react-redux"
import { api } from "../../api/client"
import { useTranslate } from "../../hooks/useTranslate"
import { Shield, Mail, User, Lock, Phone, Plus, X, Trash2, AlertTriangle } from "lucide-react"
import type { User as UserType } from "../../types"
import type { RootState } from "../../store"

export default function Admins() {
  const { t } = useTranslate()
  const currentUser = useSelector((state: RootState) => state.auth.user)
  const [admins, setAdmins] = useState<UserType[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [fullName, setFullName] = useState("")
  const [phone, setPhone] = useState("")
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)

  const fetchAdmins = async () => {
    try {
      const data = await api.get<any>("/admin/users")
      // Ensure we always set array to state, even if API returns non-array data
      setAdmins(Array.isArray(data) ? data : [])
    } catch (err) {
      console.error("Failed to fetch admins:", err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    fetchAdmins()
  }, [])

  const handleDelete = async (adminId: string) => {
    setDeletingId(adminId)
    setError(null)
    setSuccess(null)
    try {
      await api.delete(`/admin/users/${adminId}`)
      setSuccess("Admin account removed successfully")
      setConfirmDelete(null)
      fetchAdmins()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setDeletingId(null)
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreating(true)
    setError(null)
    setSuccess(null)

    try {
      await api.post("/admin/users", {
        email,
        password,
        full_name: fullName,
        phone: phone || undefined,
      })
      setSuccess("Admin account created successfully!")
      setEmail("")
      setPassword("")
      setFullName("")
      setPhone("")
      setShowCreateForm(false)
      fetchAdmins()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setCreating(false)
    }
  }

  if (loading) {
    return <div className="text-center py-20 text-gray-500">{t("common.loading")}</div>
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Admin Management</h1>
          <p className="text-gray-500 text-sm mt-1">Create and manage admin accounts for your organization</p>
        </div>
        <button
          onClick={() => setShowCreateForm(!showCreateForm)}
          className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition text-sm font-medium"
        >
          {showCreateForm ? <X className="h-4 w-4" /> : <Plus className="h-4 w-4" />}
          {showCreateForm ? "Cancel" : "Create Admin"}
        </button>
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

      {showCreateForm && (
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <h2 className="font-semibold text-lg mb-4">New Admin Account</h2>
          <form onSubmit={handleCreate} className="space-y-4">
            <div>
              <label className="text-sm font-medium text-gray-700">{t("auth.fullName")}</label>
              <div className="relative mt-1">
                <User className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                <input
                  type="text"
                  required
                  value={fullName}
                  onChange={(e) => setFullName(e.target.value)}
                  className="w-full pl-10 pr-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  placeholder="Admin Name"
                />
              </div>
            </div>

            <div>
              <label className="text-sm font-medium text-gray-700">{t("auth.email")}</label>
              <div className="relative mt-1">
                <Mail className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full pl-10 pr-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  placeholder="admin@example.com"
                />
              </div>
            </div>

            <div>
              <label className="text-sm font-medium text-gray-700">{t("auth.password")}</label>
              <div className="relative mt-1">
                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                <input
                  type="password"
                  required
                  minLength={6}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full pl-10 pr-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  placeholder="Min. 6 characters"
                />
              </div>
            </div>

            <div>
              <label className="text-sm font-medium text-gray-700">{t("auth.phone")}</label>
              <div className="relative mt-1">
                <Phone className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                <input
                  type="tel"
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                  className="w-full pl-10 pr-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  placeholder="+212 6XX XXX XXX"
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={creating}
              className="w-full bg-blue-600 text-white py-2.5 rounded-lg font-medium hover:bg-blue-700 transition disabled:opacity-50"
            >
              {creating ? t("common.loading") : "Create Admin Account"}
            </button>
          </form>
        </div>
      )}

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-100">
          <h2 className="font-semibold">Admin Users</h2>
          <p className="text-xs text-gray-400 mt-0.5">{admins?.length ?? 0} in your organization</p>
        </div>

        {(admins?.length ?? 0) === 0 ? (
          <div className="px-6 py-12 text-center">
            <Shield className="h-10 w-10 text-gray-300 mx-auto mb-3" />
            <p className="text-gray-500 text-sm">No admin accounts yet</p>
            <p className="text-gray-400 text-xs mt-1">Create the first admin to help manage your organization</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-100">
            {admins.map((admin) => {
              const isSelf = admin.id === currentUser?.id
              const isDeleting = deletingId === admin.id
              const showConfirm = confirmDelete === admin.id

              if (showConfirm) {
                return (
                  <div key={admin.id} className="px-6 py-4 flex items-center justify-between bg-red-50">
                    <div className="flex items-center gap-3">
                      <AlertTriangle className="h-5 w-5 text-red-500" />
                      <div>
                        <p className="text-sm font-medium text-red-800">Remove {admin.full_name}?</p>
                        <p className="text-xs text-red-600">This will permanently delete this admin account.</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setConfirmDelete(null)}
                        className="px-3 py-1.5 text-sm font-medium text-gray-600 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition"
                      >
                        Cancel
                      </button>
                      <button
                        onClick={() => handleDelete(admin.id)}
                        disabled={isDeleting}
                        className="px-3 py-1.5 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition disabled:opacity-50 flex items-center gap-1"
                      >
                        {isDeleting ? t("common.loading") : "Delete"}
                      </button>
                    </div>
                  </div>
                )
              }

              return (
                <div key={admin.id} className="px-6 py-4 flex items-center justify-between hover:bg-gray-50 transition">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 bg-blue-100 rounded-full flex items-center justify-center">
                      <Shield className="h-5 w-5 text-blue-600" />
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-900">
                        {admin.full_name}
                        {isSelf && <span className="text-xs text-gray-400 ml-2">(you)</span>}
                      </p>
                      <p className="text-xs text-gray-500">{admin.email}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    {admin.phone && (
                      <span className="text-xs text-gray-400 hidden sm:inline">{admin.phone}</span>
                    )}
                    <span className="text-xs bg-blue-50 text-blue-700 px-2 py-1 rounded-full font-medium">Admin</span>
                    {!isSelf && (
                      <button
                        onClick={() => setConfirmDelete(admin.id)}
                        className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition"
                        title={`Remove ${admin.full_name}`}
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
