import { Link, useNavigate } from "react-router-dom"
import { useTranslate } from "../hooks/useTranslate"
import {
  Bus, Search, Ticket, Shield, ArrowRight, MapPin, Star,
  Compass, Coffee
} from "lucide-react"

const destinations = [
  { city: "Kinshasa", desc: "The lively capital on the Congo River", descKey: "landing.destCasaDesc", image: "🏙️", color: "from-blue-500 to-cyan-400" },
  { city: "Lubumbashi", desc: "The mining heart of Katanga", descKey: "landing.destMarrakechDesc", image: "⛏️", color: "from-orange-500 to-rose-400" },
  { city: "Goma", desc: "Between lake and volcano", descKey: "landing.destTangierDesc", image: "🌋", color: "from-sky-500 to-indigo-400" },
  { city: "Bukavu", desc: "Gateway to Lake Kivu", descKey: "landing.destFesDesc", image: "🏞️", color: "from-amber-600 to-yellow-400" },
  { city: "Kisangani", desc: "The gateway to the east", descKey: "landing.destRabatDesc", image: "🛶", color: "from-emerald-500 to-teal-400" },
  { city: "Matadi", desc: "Port city on the river", descKey: "landing.destAgadirDesc", image: "⚓", color: "from-travel-500 to-rose-500" },
]
const testimonials = [
  {
    name: "Amina B.",
    avatar: "👩🏾‍💼",
    text: "I book my weekly trips from Kinshasa to Lubumbashi with BusBook. The QR tickets save so much time at the station!",
    rating: 5,
    trip: "kinshasa → lubumbashi",
  },
  {
    name: "Youssef M.",
    avatar: "👨🏻‍💻",
    text: "The seat selection is brilliant. I always pick the window seat on the Bukavu route — the lake views are stunning.",
    rating: 5,
    trip: "Goma → Bukavu",
  },
  {
    name: "Fatima Z.",
    avatar: "👩🏾‍🌾",
    text: "Finally an app that makes bus travel easy in kisangani. My family and I use it for all our trips up north.",
    rating: 5,
    trip: "kisangani → matadi",
  },
]

const features = [
  {
    icon: Search,
    titleKey: "landing.smartSearch",
    descKey: "landing.smartSearchDesc",
    to: "/search",
  },
  {
    icon: Ticket,
    titleKey: "landing.qrTickets",
    descKey: "landing.qrTicketsDesc",
    to: "/search",
  },
  {
    icon: Shield,
    titleKey: "landing.secureBooking",
    descKey: "landing.secureBookingDesc",
    to: "/auth",
  },
]

function routePath(x1: number, y1: number, x2: number, y2: number) {
  const midX = (x1 + x2) / 2
  const midY = (y1 + y2) / 2
  const dx = x2 - x1
  const dy = y2 - y1
  const len = Math.sqrt(dx * dx + dy * dy)
  const offsetX = (-dy / len) * 35
  const offsetY = (dx / len) * 35
  const cx = midX + offsetX
  const cy = midY + offsetY
  return `M ${x1} ${y1} Q ${cx} ${cy} ${x2} ${y2}`
}

const cities = [
  { name: "Kinshasa", x: 60, y: 150 },
  { name: "Lubumbashi", x: 290, y: 240 },
  { name: "Goma", x: 330, y: 140 },
  { name: "Bukavu", x: 320, y: 175 },
  { name: "Kisangani", x: 200, y: 60 },
  { name: "Matadi", x: 40, y: 200 },
]

const routeConnections = [
  { from: 0, to: 4, delay: 0 },
  { from: 0, to: 5, delay: 0.2 },
  { from: 4, to: 2, delay: 0.4 },
  { from: 2, to: 3, delay: 0.3 },
  { from: 2, to: 1, delay: 0.5 },
  { from: 3, to: 1, delay: 0.6 },
  { from: 4, to: 1, delay: 0.7 },
  { from: 0, to: 1, delay: 0.8 },
]

function CongoMap() {
  const { t } = useTranslate()
  return (
    <div className="relative">
      {/* Map container */}
      <div className="relative bg-gradient-to-br from-gray-50/90 to-white/80 backdrop-blur-xl rounded-2xl p-5 shadow-2xl border border-white/40 overflow-hidden">
        {/* Decorative map background */}
        <div className="absolute inset-0 opacity-[0.03]">
          <svg viewBox="0 0 400 280" className="w-full h-full">
            <defs>
              <pattern id="grid" width="20" height="20" patternUnits="userSpaceOnUse">
                <path d="M 20 0 L 0 0 0 20" fill="none" stroke="#f97316" strokeWidth="0.5" />
              </pattern>
            </defs>
            <rect width="400" height="280" fill="url(#grid)" />
          </svg>
        </div>

        {/* SVG Map */}
        <svg viewBox="0 0 400 280" className="relative w-full h-full">
          <defs>
            {/* Gradient for route lines */}
            <linearGradient id="routeGrad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="#f97316" />
              <stop offset="50%" stopColor="#f43f5e" />
              <stop offset="100%" stopColor="#8b5cf6" />
            </linearGradient>
            {/* Glow filter for dots */}
            <filter id="glow">
              <feGaussianBlur stdDeviation="3" result="coloredBlur" />
              <feMerge>
                <feMergeNode in="coloredBlur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
            {/* Soft glow for route lines */}
            <filter id="routeGlow">
              <feGaussianBlur stdDeviation="1.5" result="blur" />
              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
            <style>{`
              @keyframes route-appear {
                from { opacity: 0; transform: scaleY(0); }
                to { opacity: 1; transform: scaleY(1); }
              }
              @keyframes dash-move {
                to { stroke-dashoffset: -20; }
              }
              @keyframes dot-pulse {
                0%, 100% { transform: scale(1); opacity: 1; }
                50% { transform: scale(1.375); opacity: 0.7; }
              }
              @keyframes ring-pulse {
                0%, 100% { transform: scale(1); opacity: 0.08; }
                50% { transform: scale(1.4); opacity: 0.12; }
              }
              .route-line {
                animation: route-appear 0.6s ease-out backwards, dash-move 1s linear infinite;
              }
              .city-dot {
                animation: dot-pulse 2.5s ease-in-out infinite;
              }
              .city-ring {
                animation: ring-pulse 2.5s ease-in-out infinite;
              }
            `}</style>
          </defs>

          {/* Decorative background curves (topographic-like) */}
          <ellipse cx="200" cy="140" rx="160" ry="110" fill="none" stroke="#f97316" strokeWidth="0.5" opacity="0.06" />
          <ellipse cx="200" cy="140" rx="120" ry="80" fill="none" stroke="#f97316" strokeWidth="0.5" opacity="0.08" />
          <ellipse cx="200" cy="140" rx="80" ry="50" fill="none" stroke="#f97316" strokeWidth="0.5" opacity="0.1" />

          {/* Route lines with staggered appearance and continuous dash animation */}
          {routeConnections.map((conn, idx) => {
            const from = cities[conn.from]
            const to = cities[conn.to]
            const path = routePath(from.x, from.y, to.x, to.y)
            const delay = conn.delay
            return (
              <g key={idx}>
                {/* Glow behind line */}
                <path
                  d={path}
                  fill="none"
                  stroke="url(#routeGrad)"
                  strokeWidth="5"
                  opacity="0.08"
                  filter="url(#routeGlow)"
                  className="route-line"
                  style={{ animationDelay: `${delay}s, ${delay}s` }}
                />
                {/* Dashed animated route line */}
                <path
                  d={path}
                  fill="none"
                  stroke="url(#routeGrad)"
                  strokeWidth="2"
                  strokeDasharray="6 4"
                  opacity="0.65"
                  strokeLinecap="round"
                  className="route-line"
                  style={{ animationDelay: `${delay}s, ${delay}s` }}
                />
                {/* Subtle under-line for visibility */}
                <path
                  d={path}
                  fill="none"
                  stroke="#f97316"
                  strokeWidth="1"
                  opacity="0.15"
                  strokeLinecap="round"
                  className="route-line"
                  style={{ animationDelay: `${delay}s, ${delay}s` }}
                />
              </g>
            )
          })}

          {/* City dots with pulse animation */}
          {cities.map((city, idx) => (
            <g key={idx}>
              {/* Outer glow ring */}
              <circle
                cx={city.x}
                cy={city.y}
                fill="#f97316"
                className="city-ring"
                style={{ animationDelay: `${idx * 0.15}s` }}
              />
              {/* Inner dot */}
              <circle
                cx={city.x}
                cy={city.y}
                fill="#f97316"
                filter="url(#glow)"
                className="city-dot"
                style={{ animationDelay: `${idx * 0.15}s` }}
              />
              {/* White center */}
              <circle cx={city.x} cy={city.y} r="1.5" fill="white" />
              {/* City label */}
              <text
                x={city.x}
                y={city.y - 10}
                textAnchor="middle"
                className="text-[7px] font-medium fill-gray-600 city-label"
                style={{ animationDelay: `${idx * 0.15}s` }}
              >
                {city.name}
              </text>
            </g>
          ))}

          {/* Compass rose */}
          <g transform="translate(340, 30)" opacity="0.25" className="city-label" style={{ animationDelay: "0.5s" }}>
            <line x1="0" y1="-12" x2="0" y2="12" stroke="#f97316" strokeWidth="1" />
            <line x1="-12" y1="0" x2="12" y2="0" stroke="#f97316" strokeWidth="1" opacity="0.5" />
            <polygon points="0,-12 -3,-4 3,-4" fill="#f97316" />
            <text x="0" y="-16" textAnchor="middle" className="text-[6px] font-bold fill-travel-500">N</text>
          </g>
        </svg>

        {/* Glowing bus icon overlay */}
        <div className="absolute top-4 right-4 w-10 h-10 bg-gradient-to-br from-travel-400 to-rose-400 rounded-xl flex items-center justify-center shadow-lg shadow-travel-200/40 animate-float">
          <Bus className="h-5 w-5 text-white" />
        </div>

        {/* Bottom stats bar */}
        <div className="mt-3 flex items-center justify-center gap-4 text-xs text-gray-400 border-t border-gray-100 pt-3">
          <div className="flex items-center gap-1">
            <span className="w-1.5 h-1.5 rounded-full bg-travel-400" />
            <span>{t("landing.statsRoutes")}</span>
          </div>
          <div className="flex items-center gap-1">
            <span className="w-1.5 h-1.5 rounded-full bg-rose-400" />
            <span>{t("landing.statsBuses")}</span>
          </div>
          <div className="flex items-center gap-1">
            <MapPin className="h-3 w-3 text-travel-400" />
            <span>{t("landing.popularDestinations")}</span>
          </div>
        </div>
      </div>
    </div>
  )
}

const defaultOrigins: Record<string, string> = {
  Lubumbashi: "Kinshasa",
  Goma: "Kinshasa",
  Bukavu: "Goma",
  Kisangani: "Kinshasa",
  Matadi: "Kinshasa",
  Kinshasa: "Lubumbashi",
}

export default function Landing() {
  const { t } = useTranslate()
  const navigate = useNavigate()

  return (
    <div className="overflow-hidden">
      {/* ===== HERO ===== */}
      <section className="relative min-h-[80vh] flex items-center overflow-hidden">
        {/* Animated gradient background */}
        <div className="absolute inset-0 bg-gradient-to-br from-amber-50 via-orange-50/50 to-rose-50/30" />
        <div className="absolute inset-0 bg-gradient-to-tr from-ocean-50/40 via-transparent to-travel-50/40" />

        {/* Animated floating orbs */}
        <div className="absolute top-20 left-10 w-72 h-72 bg-gradient-to-br from-travel-200/30 to-rose-200/20 rounded-full blur-3xl animate-float" />
        <div className="absolute bottom-20 right-10 w-96 h-96 bg-gradient-to-br from-ocean-200/30 to-cyan-200/20 rounded-full blur-3xl animate-float animate-delay-200" />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-gradient-to-br from-travel-100/10 to-rose-100/10 rounded-full blur-3xl animate-pulse-soft" />

        {/* Content */}
        <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-20 w-full">
          <div className="grid lg:grid-cols-2 gap-12 items-center">
            {/* Left column - Text */}
            <div className="space-y-8">
              {/* Badge */}
              <div className="inline-flex items-center gap-2 bg-travel-100/80 backdrop-blur-sm text-travel-700 px-4 py-1.5 rounded-full text-sm font-medium animate-fade-in-up">
                <Compass className="h-4 w-4" />
                <span>{t("landing.heroBadge")}</span>
              </div>

              {/* Heading */}
              <h1 className="text-5xl md:text-6xl lg:text-7xl font-black leading-tight animate-fade-in-up animate-delay-100">
                <span className="text-gray-900">{t("landing.heroTitle1")}</span>
                <br />
                <span className="text-gradient">{t("landing.heroTitle2")}</span>
                <br />
                <span className="text-gray-700">{t("landing.heroTitle3")}</span>
              </h1>

              {/* Subtitle */}
              <p className="text-lg md:text-xl text-gray-600 max-w-xl leading-relaxed animate-fade-in-up animate-delay-200">
                {t("landing.heroSubtitle")}
              </p>

              {/* CTA Buttons */}
              <div className="flex flex-wrap gap-4 animate-fade-in-up animate-delay-300">
                <Link
                  to="/search"
                  className="group btn-primary text-base px-8 py-3.5 shadow-lg shadow-ocean-200/50 hover:shadow-xl hover:shadow-ocean-200/60 active:scale-[0.98] transition-all"
                >
                  <Search className="h-5 w-5" />
                  {t("landing.searchTrips")}
                  <ArrowRight className="h-4 w-4 group-hover:translate-x-1 transition-transform" />
                </Link>
                <Link
                  to="/auth"
                  className="btn-secondary text-base px-8 py-3.5"
                >
                  <Shield className="h-5 w-5" />
                  {t("landing.ownerPortal")}
                </Link>
              </div>

              {/* Quick stats */}
              <div className="flex items-center gap-6 text-sm text-gray-500 animate-fade-in-up animate-delay-500">
                <div className="flex items-center gap-1.5">
                  <MapPin className="h-4 w-4 text-travel-400" />
                  <span>{t("landing.statsRoutes")}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <Bus className="h-4 w-4 text-travel-400" />
                  <span>{t("landing.statsBuses")}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <Star className="h-4 w-4 text-travel-400" />
                  <span>{t("landing.statsRating")}</span>
                </div>
              </div>
            </div>

            {/* Right column - Map Visual */}
            <div className="relative hidden lg:flex items-center justify-center animate-fade-in animate-delay-300">
              <div className="relative w-full max-w-md">
               <CongoMap />
              </div>
            </div>
          </div>
        </div>

        {/* Bottom fade */}
        <div className="absolute bottom-0 left-0 right-0 h-32 bg-gradient-to-t from-gray-50/80 to-transparent" />
      </section>

      {/* ===== DESTINATIONS ===== */}
      <section className="relative py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-12">
            <div className="inline-flex items-center gap-2 bg-travel-50 text-travel-700 px-4 py-1.5 rounded-full text-sm font-medium mb-4">
              <MapPin className="h-4 w-4" />
              <span>{t("landing.popularDestinations")}</span>
            </div>
            <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-3">
              {t("landing.destinationsTitle")}
            </h2>
            <p className="text-gray-500 max-w-xl mx-auto">
              {t("landing.destinationsSubtitle")}
            </p>
          </div>

          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
            {destinations.map((dest, idx) => (
              <button
                key={dest.city}
                onClick={() => navigate("/search", { state: { origin: defaultOrigins[dest.city] || "Kinshasa", destination: dest.city } })}
                className="group relative overflow-hidden rounded-xl bg-white border border-gray-100 p-4 text-center
                  hover:shadow-lg hover:-translate-y-1 transition-all duration-300 cursor-pointer text-left"
                style={{ animationDelay: `${idx * 80}ms` }}
              >
                <div className={`w-12 h-12 mx-auto mb-3 rounded-lg bg-gradient-to-br ${dest.color} flex items-center justify-center text-xl
                  group-hover:scale-110 transition-transform duration-300`}>
                  {dest.image}
                </div>
                <h3 className="font-semibold text-gray-800 text-sm">{dest.city}</h3>
                <p className="text-xs text-gray-400 mt-0.5">{dest.descKey ? t(dest.descKey) : dest.desc}</p>
              </button>
            ))}
          </div>
        </div>
      </section>

      {/* ===== FEATURES ===== */}
      <section className="py-16 bg-gradient-to-b from-transparent via-travel-50/30 to-amber-50/20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-12">
            <div className="inline-flex items-center gap-2 bg-ocean-50 text-ocean-700 px-4 py-1.5 rounded-full text-sm font-medium mb-4">
              <Coffee className="h-4 w-4" />
              <span>{t("landing.whyBusBook")}</span>
            </div>
            <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-3">
              {t("landing.featuresTitle")}
            </h2>
            <p className="text-gray-500 max-w-xl mx-auto">
              {t("landing.featuresSubtitle")}
            </p>
          </div>

          <div className="grid md:grid-cols-3 gap-6">
            {features.map((f, idx) => (
              <Link
                key={idx}
                to={f.to}
                className="group bg-white p-6 rounded-xl border border-gray-100 hover:shadow-xl hover:border-travel-200 hover:-translate-y-1 transition-all duration-300"
              >
                <div className="w-12 h-12 bg-gradient-to-br from-travel-50 to-amber-50 rounded-xl flex items-center justify-center mb-4
                  group-hover:from-travel-500 group-hover:to-amber-500 transition-all duration-300" aria-hidden="true">
                  <f.icon className="h-6 w-6 text-travel-600 group-hover:text-white transition-colors duration-300" aria-hidden="true" />
                </div>
                <h3 className="text-lg font-semibold mb-1 group-hover:text-travel-600 transition-colors">{t(f.titleKey)}</h3>
                <p className="text-gray-500 text-sm leading-relaxed">{t(f.descKey)}</p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      {/* ===== TESTIMONIALS ===== */}
      <section className="py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-12">
            <div className="inline-flex items-center gap-2 bg-amber-50 text-amber-700 px-4 py-1.5 rounded-full text-sm font-medium mb-4">
              <Star className="h-4 w-4" />
              <span>{t("landing.testimonialsBadge")}</span>
            </div>
            <h2 className="text-3xl md:text-4xl font-bold text-gray-900 mb-3">
              {t("landing.testimonialsTitle")}
            </h2>
            <p className="text-gray-500 max-w-xl mx-auto">
              {t("landing.testimonialsSubtitle")}
            </p>
          </div>

          <div className="grid md:grid-cols-3 gap-6">
            {testimonials.map((item, idx) => (
              <div
                key={idx}
                className="bg-white rounded-xl p-6 border border-gray-100 shadow-sm hover:shadow-lg transition-all duration-300"
              >
                <div className="flex items-center gap-1 mb-3">
                  {Array.from({ length: item.rating }, (_, i) => (
                    <Star key={i} className="h-4 w-4 fill-amber-400 text-amber-400" />
                  ))}
                </div>
                <p className="text-gray-600 text-sm leading-relaxed mb-4 italic">
                  "{item.text}"
                </p>
                <div className="flex items-center gap-3 pt-3 border-t border-gray-100">
                  <span className="text-2xl">{item.avatar}</span>
                  <div>
                    <p className="font-semibold text-sm text-gray-800">{item.name}</p>
                    <p className="text-xs text-gray-400 flex items-center gap-1">
                      <Bus className="h-3 w-3" /> {item.trip}
                    </p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ===== CTA ===== */}
      <section className="pb-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="relative overflow-hidden rounded-2xl bg-gradient-to-br from-travel-500 via-rose-500 to-purple-600 p-8 md:p-14 text-center text-white">
            {/* Animated background elements */}
            <div className="absolute top-0 right-0 w-64 h-64 bg-white/5 rounded-full -translate-y-1/2 translate-x-1/2" />
            <div className="absolute bottom-0 left-0 w-48 h-48 bg-white/5 rounded-full translate-y-1/2 -translate-x-1/2" />
            <div className="absolute top-1/2 left-1/4 w-32 h-32 bg-white/5 rounded-full animate-float" />

            <div className="relative">
              <h2 className="text-3xl md:text-4xl font-bold mb-4">{t("landing.ctaTitle")}</h2>
              <p className="text-white/80 mb-8 max-w-xl mx-auto text-lg">
                {t("landing.ctaDesc")}
              </p>
              <Link
                to="/search"
                className="inline-flex items-center gap-2 bg-white text-travel-600 px-8 py-3.5 rounded-xl font-semibold
                  hover:bg-travel-50 hover:scale-105 active:scale-95 transition-all duration-200 shadow-lg"
              >
                <Bus className="h-5 w-5" />
                {t("landing.getStarted")}
                <ArrowRight className="h-4 w-4" />
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* ===== FOOTER ===== */}
      <footer className="border-t border-gray-200 bg-white/60 backdrop-blur-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="flex flex-col md:flex-row items-center justify-between gap-4 text-sm text-gray-500">
            <div className="flex items-center gap-2">
              <Bus className="h-5 w-5 text-travel-500" />
              <span className="font-semibold text-gray-700">Ayale<span className="text-travel-500">Trans</span></span>
              <span className="hidden sm:inline">&middot; {t("landing.footerTagline")}</span>
            </div>
            <div className="flex items-center gap-4">
              <span>© {new Date().getFullYear()} AyaleTrans</span>
              <span className="text-gray-300">|</span>
              <span>{t("landing.footerMadeIn")}</span>
            </div>
          </div>
        </div>
      </footer>
    </div>
  )
}
