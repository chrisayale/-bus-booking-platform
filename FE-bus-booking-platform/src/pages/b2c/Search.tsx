import { useState, useEffect } from "react"
import { useNavigate, useLocation } from "react-router-dom"
import { useDispatch, useSelector } from "react-redux"
import { useTranslate } from "../../hooks/useTranslate"
import { searchTrips, setSelectedJourney } from "../../store/bookingSlice"
import type { AppDispatch, RootState } from "../../store"
import { Search as SearchIcon, MapPin, Calendar, Clock, ArrowRight, Bus, Compass } from "lucide-react"

export default function Search() {
  const { t } = useTranslate()
  const location = useLocation()
  const initialState = location.state as { origin?: string; destination?: string } | null
  const [origin, setOrigin] = useState(initialState?.origin || "")
  const [destination, setDestination] = useState(initialState?.destination || "")
  const [hasSearched, setHasSearched] = useState(false)
  const [date, setDate] = useState(new Date().toISOString().split("T")[0])
  const dispatch = useDispatch<AppDispatch>()
  const { searchResults, isSearching } = useSelector((state: RootState) => state.booking)
  const navigate = useNavigate()

  // Auto-search when navigated from landing page with pre-filled values
  useEffect(() => {
    if (initialState?.origin && initialState?.destination) {
      setHasSearched(true) // eslint-disable-line react-hooks/set-state-in-effect
      dispatch(searchTrips({ origin: initialState.origin, destination: initialState.destination, date }))
    }
    if (initialState?.origin || initialState?.destination) {
      window.history.replaceState(null, "")
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    setHasSearched(true)
    dispatch(searchTrips({ origin, destination, date }))
  }

  const handleBook = (journey: (typeof searchResults)[0]) => {
    dispatch(setSelectedJourney(journey))
    navigate("/booking")
  }

  return (
    <div className="animate-fade-in">
      {/* Page header */}
      <div className="text-center mb-8">
        <div className="inline-flex items-center gap-2 bg-travel-50 text-travel-700 px-4 py-1.5 rounded-full text-sm font-medium mb-3">
          <Bus className="h-4 w-4" />
          <span>{t("search.findTrip")}</span>
        </div>
        <h1 className="text-2xl font-bold text-gray-900">{t("search.title")}</h1>
      </div>

      {/* Search Form */}
      <form onSubmit={handleSearch} className="card-travel p-6 mb-8">
        <div className="grid md:grid-cols-4 gap-4">
          <div>
            <label htmlFor="origin" className="text-sm font-medium text-gray-700">{t("search.origin")}</label>
            <div className="relative mt-1">
              <MapPin className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
              <input id="origin" required value={origin} onChange={(e) => setOrigin(e.target.value)}
                className="w-full pl-10 pr-3 py-2.5 border border-gray-200 rounded-xl outline-none focus:ring-2 focus:ring-travel-400 focus:border-transparent bg-gray-50/50 transition-all duration-200"
                placeholder={t("search.originPlaceholder")} />
            </div>
          </div>
          <div>
            <label htmlFor="destination" className="text-sm font-medium text-gray-700">{t("search.destination")}</label>
            <div className="relative mt-1">
              <MapPin className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
              <input id="destination" required value={destination} onChange={(e) => setDestination(e.target.value)}
                className="w-full pl-10 pr-3 py-2.5 border border-gray-200 rounded-xl outline-none focus:ring-2 focus:ring-travel-400 focus:border-transparent bg-gray-50/50 transition-all duration-200"
                placeholder={t("search.destinationPlaceholder")} />
            </div>
          </div>
          <div>
            <label htmlFor="travel-date" className="text-sm font-medium text-gray-700">{t("search.date")}</label>
            <div className="relative mt-1">
              <Calendar className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
              <input id="travel-date" type="date" required value={date} onChange={(e) => setDate(e.target.value)}
                className="w-full pl-10 pr-3 py-2.5 border border-gray-200 rounded-xl outline-none focus:ring-2 focus:ring-travel-400 focus:border-transparent bg-gray-50/50 transition-all duration-200" />
            </div>
          </div>
          <div>
            <label className="text-sm font-medium text-gray-700 invisible">Search</label>
            <button type="submit" disabled={isSearching}
              className="w-full mt-1 flex items-center justify-center gap-2 bg-gradient-to-r from-travel-500 to-rose-500 text-white py-2.5 rounded-xl font-medium
                hover:from-travel-600 hover:to-rose-600 transition-all duration-200 disabled:opacity-50 shadow-sm shadow-travel-200/50 active:scale-[0.98]">
              <SearchIcon className="h-4 w-4" />
              {isSearching ? t("search.searching") : t("search.search")}
            </button>
          </div>
        </div>
      </form>

      {/* Results */}
      <div className="space-y-4">
        {(searchResults || []).map((j, idx) => (
          <div
            key={j.id}
            className="card-travel p-5 hover:-translate-y-0.5"
            style={{ animation: `fade-in-up 0.4s ease-out ${idx * 80}ms both` }}
          >
            <div className="flex items-center justify-between flex-wrap gap-4">
              <div className="flex items-center gap-4">
                <div className="w-12 h-12 bg-gradient-to-br from-travel-50 to-amber-50 rounded-xl flex items-center justify-center">
                  <Bus className="h-6 w-6 text-travel-500" />
                </div>
                <div>
                  <div className="flex items-center gap-2 text-lg font-semibold">
                    <span>{j.route?.origin || t("search.origin")}</span>
                    <div className="flex items-center gap-1 text-travel-400">
                      <ArrowRight className="h-4 w-4" />
                      <div className="h-px w-6 bg-gradient-to-r from-travel-300 to-rose-300 hidden sm:block" />
                    </div>
                    <span>{j.route?.destination || t("search.destination")}</span>
                  </div>
                  <div className="flex items-center gap-4 mt-1 text-sm text-gray-500">
                    <span className="flex items-center gap-1"><Clock className="h-3 w-3 text-travel-400" /> {j.departure_time}</span>
                    <span className="flex items-center gap-1"><Bus className="h-3 w-3 text-travel-400" /> {j.bus?.registration_number || "—"}</span>
                  </div>
                </div>
              </div>
              <div className="text-right flex items-center gap-4 flex-wrap">
                <div>
                  <p className="text-2xl font-bold text-gradient">{j.route?.base_price || "—"} USD</p>
                  <p className="text-xs text-gray-400">{j.bus?.available_seats ?? "?"} {t("search.seatsLeft")}</p>
                </div>
                <button onClick={() => handleBook(j)}
                  className="bg-gradient-to-r from-travel-500 to-rose-500 text-white px-5 py-2 rounded-xl text-sm font-medium
                    hover:from-travel-600 hover:to-rose-600 transition-all duration-200 shadow-sm shadow-travel-200/50 active:scale-95">
                  {t("search.select")}
                </button>
              </div>
            </div>
          </div>
        ))}
        {(searchResults || []).length === 0 && !isSearching && (
          <div className="text-center py-16">
            <Compass className="h-12 w-12 text-gray-300 mx-auto mb-4" />
            <p className="text-gray-500">{hasSearched ? t("search.noResults") : t("search.noSearchYet")}</p>
          </div>
        )}
        {isSearching && (
          <div className="text-center py-12">
            <div className="inline-flex items-center gap-2 text-travel-500">
              <svg className="animate-spin h-5 w-5" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              {t("search.searching")}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
