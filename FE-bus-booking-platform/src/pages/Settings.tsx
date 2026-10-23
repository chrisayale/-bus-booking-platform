import { useState, useEffect, useContext } from "react"
import { useNavigate } from "react-router-dom"
import { useDispatch } from "react-redux"
import { api } from "../api/client"
import { useTranslate } from "../hooks/useTranslate"
import { LanguageContext } from "../context/LanguageContext"
import { logout } from "../store/authSlice"

import { User, Lock, Shield, AlertTriangle, Save, Globe, Check } from "lucide-react"

interface Profile {
  id: string
  email: string
  full_name: string
  phone?: string
  role: string
  is_verified: boolean
}

const languages = [
  { code: "en", label: "English", flag: "🇬🇧" },
  { code: "fr", label: "Français", flag: "🇫🇷" },
  { code: "ar", label: "العربية", flag: "🇲🇦" },
]

export default function Settings() {
  const { t } = useTranslate()
  const { locale, setLocale } = useContext(LanguageContext)
  const [profile, setProfile] = useState<Profile | null>(null)
  const [fullName, setFullName] = useState("")
  const [phone, setPhone] = useState("")
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null)
  const dispatch = useDispatch()
  const navigate = useNavigate()

  useEffect(() => {
    api.get<Profile>("/user/profile")
      .then((p) => {
        setProfile(p)
        setFullName(p.full_name)
        setPhone(p.phone || "")
      })
      .catch(() => navigate("/auth"))
      .finally(() => setLoading(false))
  }, [navigate])

  const updateProfile = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setMessage(null)
    try {
      const updated = await api.put<Profile>("/user/profile", { full_name: fullName, phone })
      setProfile(updated)
      setMessage({ type: "success", text: t("settings.profileUpdated") })
    } catch (err) {
      setMessage({ type: "error", text: (err as Error).message })
    } finally {
      setSaving(false)
    }
  }

  const changePassword = async (e: React.FormEvent) => {
    e.preventDefault()
    if (newPassword !== confirmPassword) {
      setMessage({ type: "error", text: t("settings.passwordMismatch") })
      return
    }
    setSaving(true)
    setMessage(null)
    try {
      await api.put("/user/password", {
        current_password: currentPassword,
        new_password: newPassword,
      })
      setMessage({ type: "success", text: t("settings.passwordChanged") })
      setCurrentPassword("")
      setNewPassword("")
      setConfirmPassword("")
    } catch (err) {
      setMessage({ type: "error", text: (err as Error).message })
    } finally {
      setSaving(false)
    }
  }

  const deleteAccount = async () => {
    if (!confirm(t("settings.deleteConfirm1"))) return
    if (!confirm(t("settings.deleteConfirm2"))) return
    try {
      await api.delete("/user/account")
      dispatch(logout())
      navigate("/")
    } catch (err) {
      setMessage({ type: "error", text: (err as Error).message })
    }
  }

  if (loading) {
    return <div className="text-center py-12 text-gray-500">{t("common.loading")}</div>
  }

  return (
    <div className="max-w-2xl mx-auto space-y-8">
      <h1 className="text-2xl font-bold">{t("settings.title")}</h1>

      {message && (
        <div className={`p-4 rounded-lg text-sm ${
          message.type === "success"
            ? "bg-green-50 text-green-700 border border-green-200"
            : "bg-red-50 text-red-700 border border-red-200"
        }`}>
          {message.text}
        </div>
      )}

      {/* Profile Section */}
      <form onSubmit={updateProfile} className="bg-white p-6 rounded-xl border border-gray-200 space-y-4">
        <div className="flex items-center gap-2 mb-4">
          <User className="h-5 w-5 text-travel-500" />
          <h2 className="text-lg font-semibold">{t("settings.profile")}</h2>
        </div>

        <div className="grid md:grid-cols-2 gap-4">
          <div>
            <label htmlFor="profile-email" className="text-sm font-medium text-gray-700">{t("settings.email")}</label>
            <input id="profile-email" name="email" disabled value={profile?.email || ""}
              className="mt-1 w-full px-3 py-2 bg-gray-50 border border-gray-300 rounded-lg text-gray-500 cursor-not-allowed" />
          </div>
          <div>
            <label htmlFor="profile-role" className="text-sm font-medium text-gray-700">{t("settings.role")}</label>
            <input id="profile-role" name="role" disabled value={profile?.role === "owner" ? t("auth.roleOwner") : t("auth.rolePassenger")}
              className="mt-1 w-full px-3 py-2 bg-gray-50 border border-gray-300 rounded-lg text-gray-500 cursor-not-allowed" />
          </div>
          <div>
            <label htmlFor="profile-name" className="text-sm font-medium text-gray-700">{t("settings.fullName")}</label>
            <input id="profile-name" name="full_name" value={fullName} onChange={(e) => setFullName(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          <div>
            <label htmlFor="profile-phone" className="text-sm font-medium text-gray-700">{t("settings.phone")}</label>
            <input id="profile-phone" name="phone" value={phone} onChange={(e) => setPhone(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
        </div>

        <div className="flex items-center gap-2 text-sm">
          <Shield className="h-4 w-4 text-gray-400" />
          <span className={profile?.is_verified ? "text-green-600" : "text-yellow-600"}>
            {profile?.is_verified ? t("auth.email") + " verified" : t("auth.email") + " not verified"}
          </span>
        </div>

        <button type="submit" disabled={saving}
          className="flex items-center gap-2 bg-gradient-to-r from-travel-500 to-rose-500 text-white px-4 py-2 rounded-xl text-sm font-medium hover:from-travel-600 hover:to-rose-600 transition-all duration-200 disabled:opacity-50 shadow-sm shadow-travel-200/50 active:scale-[0.98]">
          <Save className="h-4 w-4" /> {saving ? t("settings.saving") : t("settings.saveProfile")}
        </button>
      </form>

      {/* Password Section */}
      <form onSubmit={changePassword} className="bg-white p-6 rounded-xl border border-gray-200 space-y-4">
        <div className="flex items-center gap-2 mb-4">
          <Lock className="h-5 w-5 text-travel-500" />
          <h2 className="text-lg font-semibold">{t("settings.changePassword")}</h2>
        </div>

        <div>
          <label htmlFor="settings-current-pw" className="text-sm font-medium text-gray-700">{t("settings.currentPassword")}</label>
          <input id="settings-current-pw" name="current_password" type="password" required value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <div className="grid md:grid-cols-2 gap-4">
          <div>
            <label htmlFor="settings-new-pw" className="text-sm font-medium text-gray-700">{t("settings.newPassword")}</label>
            <input id="settings-new-pw" name="new_password" type="password" required minLength={6} value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Min. 6 characters" />
          </div>
          <div>
            <label htmlFor="settings-confirm-pw" className="text-sm font-medium text-gray-700">{t("settings.confirmPassword")}</label>
            <input id="settings-confirm-pw" name="confirm_password" type="password" required value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
        </div>

        <button type="submit" disabled={saving}
          className="bg-gradient-to-r from-travel-500 to-rose-500 text-white px-4 py-2 rounded-xl text-sm font-medium hover:from-travel-600 hover:to-rose-600 transition-all duration-200 disabled:opacity-50 shadow-sm shadow-travel-200/50 active:scale-[0.98]">
          {saving ? t("settings.updating") : t("settings.changePassword")}
        </button>
      </form>

      {/* Language Settings */}
      <div className="bg-white p-6 rounded-xl border border-gray-200 space-y-4">
        <div className="flex items-center gap-2 mb-4">
          <Globe className="h-5 w-5 text-travel-500" />
          <h2 className="text-lg font-semibold">{t("settings.language")}</h2>
        </div>
        <div className="grid grid-cols-3 gap-3">
          {languages.map((lang) => (
            <button
              key={lang.code}
              onClick={() => { setLocale(lang.code as "en" | "fr" | "ar"); }}
              className={`flex items-center justify-between p-3 border rounded-xl transition ${
                locale === lang.code
                  ? "border-travel-500 bg-travel-50"
                  : "border-gray-200 hover:border-travel-300"
              }`}
            >
              <div className="flex items-center gap-2">
                <span className="text-lg">{lang.flag}</span>
                <span className="text-sm font-medium">{lang.label}</span>
              </div>
              {locale === lang.code && <Check className="h-4 w-4 text-travel-500" />}
            </button>
          ))}
        </div>
        <p className="text-xs text-gray-400">{t("settings.languageNote")}</p>
      </div>

      {/* Danger Zone */}
      <div className="bg-white p-6 rounded-xl border border-red-200 space-y-4">
        <div className="flex items-center gap-2 mb-4">
          <AlertTriangle className="h-5 w-5 text-red-500" />
          <h2 className="text-lg font-semibold text-red-700">{t("settings.dangerZone")}</h2>
        </div>
        <p className="text-sm text-gray-600">
          {t("settings.deleteWarning")}
        </p>
        <button onClick={deleteAccount}
          className="bg-red-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-red-700 transition">
          {t("settings.deleteAccount")}
        </button>
      </div>
    </div>
  )
}
