import { useState } from "react"
import { useSearchParams, useNavigate, Link } from "react-router-dom"
import { api } from "../api/client"
import { Lock, CheckCircle, ArrowLeft } from "lucide-react"

export default function ResetPassword() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get("token") || ""
  const navigate = useNavigate()

  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [success, setSuccess] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (password !== confirmPassword) {
      setError("Passwords do not match")
      return
    }
    if (password.length < 6) {
      setError("Password must be at least 6 characters")
      return
    }

    setLoading(true)
    setError("")
    try {
      await api.post("/auth/reset-password", { token, new_password: password })
      setSuccess(true)
      setTimeout(() => navigate("/auth"), 2000)
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setLoading(false)
    }
  }

  if (!token) {
    return (
      <div className="min-h-[80vh] flex items-center justify-center">
        <div className="bg-white/90 backdrop-blur-sm rounded-2xl p-8 shadow-xl border border-gray-100 w-full max-w-md text-center">
          <p className="text-red-500 mb-4">Invalid reset link.</p>
          <Link to="/auth/forgot-password" className="text-travel-500 hover:text-travel-600 hover:underline font-medium">Request a new one</Link>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-[80vh] flex items-center justify-center py-8">
      <div className="w-full max-w-md animate-fade-in-up">
        <div className="bg-white/90 backdrop-blur-sm rounded-2xl p-8 shadow-xl border border-gray-100">
          <Link to="/auth" className="flex items-center gap-1 text-sm text-gray-500 mb-6 hover:text-travel-600 transition-colors">
            <ArrowLeft className="h-4 w-4" /> Back to sign in
          </Link>

          <h1 className="text-xl font-bold mb-2 text-gray-900">Reset Password</h1>
          <p className="text-sm text-gray-500 mb-6">Enter your new password.</p>

          {success ? (
            <div className="text-center py-6">
              <CheckCircle className="h-12 w-12 text-green-500 mx-auto mb-3" />
              <p className="text-green-700 font-medium">Password reset successfully!</p>
              <p className="text-sm text-gray-500 mt-1">Redirecting to sign in...</p>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              {error && <p className="text-red-500 text-sm">{error}</p>}
              <div>
                <label htmlFor="reset-password" className="text-sm font-medium text-gray-700">New Password</label>
                <div className="relative mt-1">
                  <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                  <input
                    id="reset-password"
                    name="new_password"
                    type="password"
                    required
                    minLength={6}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="w-full pl-10 pr-4 py-2.5 border border-gray-200 rounded-xl focus:ring-2 focus:ring-travel-400 focus:border-transparent outline-none bg-gray-50/50 transition-all duration-200"
                    placeholder="Min. 6 characters"
                  />
                </div>
              </div>
              <div>
                <label htmlFor="reset-confirm" className="text-sm font-medium text-gray-700">Confirm Password</label>
                <div className="relative mt-1">
                  <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                  <input
                    id="reset-confirm"
                    name="confirm_password"
                    type="password"
                    required
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    className="w-full pl-10 pr-4 py-2.5 border border-gray-200 rounded-xl focus:ring-2 focus:ring-travel-400 focus:border-transparent outline-none bg-gray-50/50 transition-all duration-200"
                    placeholder="Repeat password"
                  />
                </div>
              </div>
              <button
                type="submit"
                disabled={loading}
                className="w-full bg-gradient-to-r from-travel-500 to-rose-500 text-white py-2.5 rounded-xl font-medium hover:from-travel-600 hover:to-rose-600 transition-all duration-200 disabled:opacity-50 shadow-md shadow-travel-200/50 active:scale-[0.98]"
              >
                {loading ? "Resetting..." : "Reset Password"}
              </button>
            </form>
          )}
        </div>
      </div>
    </div>
  )
}
