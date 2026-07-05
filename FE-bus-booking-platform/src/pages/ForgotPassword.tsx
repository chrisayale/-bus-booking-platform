import { useState } from "react"
import { Link } from "react-router-dom"
import { api } from "../api/client"
import { Mail, ArrowLeft, CheckCircle } from "lucide-react"

export default function ForgotPassword() {
  const [email, setEmail] = useState("")
  const [sent, setSent] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError("")
    try {
      await api.post("/auth/forgot-password", { email })
      setSent(true)
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-[80vh] flex items-center justify-center py-8">
      <div className="w-full max-w-md animate-fade-in-up">
        <div className="bg-white/90 backdrop-blur-sm rounded-2xl p-8 shadow-xl border border-gray-100">
          <Link to="/auth" className="flex items-center gap-1 text-sm text-gray-500 mb-6 hover:text-travel-600 transition-colors">
            <ArrowLeft className="h-4 w-4" /> Back to sign in
          </Link>

          <h1 className="text-xl font-bold mb-2 text-gray-900">Forgot Password?</h1>
          <p className="text-sm text-gray-500 mb-6">Enter your email and we'll send you a reset link.</p>

          {sent ? (
            <div className="text-center py-6">
              <CheckCircle className="h-12 w-12 text-green-500 mx-auto mb-3" />
              <p className="text-green-700 font-medium">Check your email</p>
              <p className="text-sm text-gray-500 mt-1">If an account with that email exists, you'll receive a reset link shortly.</p>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              {error && <p className="text-red-500 text-sm">{error}</p>}
              <div>
                <label htmlFor="forgot-email" className="text-sm font-medium text-gray-700">Email</label>
                <div className="relative mt-1">
                  <Mail className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                  <input
                    id="forgot-email"
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
              <button
                type="submit"
                disabled={loading}
                className="w-full bg-gradient-to-r from-travel-500 to-rose-500 text-white py-2.5 rounded-xl font-medium hover:from-travel-600 hover:to-rose-600 transition-all duration-200 disabled:opacity-50 shadow-md shadow-travel-200/50 active:scale-[0.98]"
              >
                {loading ? "Sending..." : "Send Reset Link"}
              </button>
            </form>
          )}
        </div>
      </div>
    </div>
  )
}
