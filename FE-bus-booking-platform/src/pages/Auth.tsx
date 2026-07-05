import { useState } from "react"
import { useNavigate, Link } from "react-router-dom"
import { useDispatch, useSelector } from "react-redux"
import { useTranslate } from "../hooks/useTranslate"
import { login, register } from "../store/authSlice"
import type { AppDispatch, RootState } from "../store"
import { Mail, Lock, User, Phone, Bus, Compass, ArrowRight } from "lucide-react"

type AuthMode = "login" | "register"

export default function Auth() {
  const { t } = useTranslate()
  const [mode, setMode] = useState<AuthMode>("login")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [fullName, setFullName] = useState("")
  const [phone, setPhone] = useState("")

  const dispatch = useDispatch<AppDispatch>()
  const { isLoading, error } = useSelector((state: RootState) => state.auth)
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (mode === "login") {
      const result = await dispatch(login({ email, password }))
      if (login.fulfilled.match(result)) {
        navigate(result.payload.role === "owner" ? "/owner/dashboard" : "/search")
      }
    } else {
      const result = await dispatch(register({ email, password, full_name: fullName, phone }))
      if (register.fulfilled.match(result)) {
        setMode("login")
        alert("Account created! Please sign in.")
      }
    }
  }

  return (
    <div className="min-h-[80vh] flex items-center justify-center py-8">
      <div className="w-full max-w-md animate-fade-in-up">
        {/* Logo header */}
        <div className="text-center mb-8">
          <Link to="/" className="inline-flex items-center gap-2 group">
            <div className="w-12 h-12 bg-gradient-to-br from-travel-500 to-rose-500 rounded-2xl flex items-center justify-center
              group-hover:scale-110 transition-transform duration-200 shadow-lg shadow-travel-200/50">
              <Bus className="h-7 w-7 text-white" />
            </div>
          </Link>
          <h1 className="text-xl font-bold mt-4">
            <span className="text-gray-900">{t("auth.welcomeTo")} </span>
            <span className="text-gradient">BusBook</span>
          </h1>
          <p className="text-sm text-gray-500 mt-1">
            {mode === "login" ? t("auth.loginSubtitle") : t("auth.registerSubtitle")}
          </p>
        </div>

        <div className="bg-white/90 backdrop-blur-sm rounded-2xl p-8 shadow-xl border border-gray-100">
          {/* Tabs */}
          <div className="flex bg-gray-100/80 rounded-xl p-1 mb-6">
            <button
              className={`flex-1 py-2.5 text-sm font-medium rounded-lg transition-all duration-200 ${
                mode === "login" ? "bg-white text-travel-600 shadow-sm" : "text-gray-500 hover:text-gray-700"
              }`}
              onClick={() => setMode("login")}
            >
              {t("auth.loginTab")}
            </button>
            <button
              className={`flex-1 py-2.5 text-sm font-medium rounded-lg transition-all duration-200 ${
                mode === "register" ? "bg-white text-travel-600 shadow-sm" : "text-gray-500 hover:text-gray-700"
              }`}
              onClick={() => setMode("register")}
            >
              {t("auth.registerTab")}
            </button>
          </div>

          {/* Error */}
          {error && (
            <div className="p-3 mb-4 bg-red-50 border border-red-100 rounded-xl text-sm text-red-600 text-center">
              {error}
            </div>
          )}

          {/* Form */}
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="auth-email" className="text-sm font-medium text-gray-700">{t("auth.email")}</label>
              <div className="relative mt-1">
                <Mail className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                <input
                  id="auth-email"
                  name="email"
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full pl-10 pr-4 py-2.5 border border-gray-200 rounded-xl focus:ring-2 focus:ring-travel-400 focus:border-transparent outline-none bg-gray-50/50 transition-all duration-200"
                  placeholder="you@example.com"
                />
              </div>
            </div>

            <div>
              <label htmlFor="auth-password" className="text-sm font-medium text-gray-700">{t("auth.password")}</label>
              <div className="relative mt-1">
                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                <input
                  id="auth-password"
                  name="password"
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full pl-10 pr-4 py-2.5 border border-gray-200 rounded-xl focus:ring-2 focus:ring-travel-400 focus:border-transparent outline-none bg-gray-50/50 transition-all duration-200"
                  placeholder="••••••••"
                />
              </div>
            </div>

            {mode === "login" && (
              <div className="text-right">
                <Link to="/auth/forgot-password" className="text-sm text-travel-500 hover:text-travel-600 transition-colors">
                  {t("auth.forgotPassword")}
                </Link>
              </div>
            )}

            {mode === "register" && (
              <>
                <div>
                  <label htmlFor="auth-fullname" className="text-sm font-medium text-gray-700">{t("auth.fullName")}</label>
                  <div className="relative mt-1">
                    <User className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                    <input
                      id="auth-fullname"
                      name="full_name"
                      type="text"
                      required
                      value={fullName}
                      onChange={(e) => setFullName(e.target.value)}
                      className="w-full pl-10 pr-4 py-2.5 border border-gray-200 rounded-xl focus:ring-2 focus:ring-travel-400 focus:border-transparent outline-none bg-gray-50/50 transition-all duration-200"
                      placeholder="John Doe"
                    />
                  </div>
                </div>

                <div>
                  <label htmlFor="auth-phone" className="text-sm font-medium text-gray-700">{t("auth.phone")}</label>
                  <div className="relative mt-1">
                    <Phone className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                    <input
                      id="auth-phone"
                      name="phone"
                      type="tel"
                      value={phone}
                      onChange={(e) => setPhone(e.target.value)}
                      className="w-full pl-10 pr-4 py-2.5 border border-gray-200 rounded-xl focus:ring-2 focus:ring-travel-400 focus:border-transparent outline-none bg-gray-50/50 transition-all duration-200"
                      placeholder="+212 6XX XXX XXX"
                    />
                  </div>
                </div>
              </>
            )}

            <button
              type="submit"
              disabled={isLoading}
              className="w-full bg-gradient-to-r from-travel-500 to-rose-500 text-white py-2.5 rounded-xl font-medium
                hover:from-travel-600 hover:to-rose-600 transition-all duration-200 disabled:opacity-50
                shadow-md shadow-travel-200/50 active:scale-[0.98]"
            >
              {isLoading ? (
                <span className="flex items-center justify-center gap-2">
                  <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  {t("common.loading")}
                </span>
              ) : (
                <span className="flex items-center justify-center gap-2">
                  {mode === "login" ? t("auth.signIn") : t("auth.signUp")}
                  <ArrowRight className="h-4 w-4" />
                </span>
              )}
            </button>
          </form>
        </div>

        {/* Bottom text */}
        <p className="text-center text-xs text-gray-400 mt-6 flex items-center justify-center gap-1">
          <Compass className="h-3 w-3" />
          {t("auth.footerTagline")}
        </p>
      </div>
    </div>
  )
}
