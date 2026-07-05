/* eslint-disable react-refresh/only-export-components */

import { createContext, useState, useEffect, useCallback, type ReactNode } from "react"

export type Locale = "en" | "fr" | "ar"

interface LanguageContextType {
  locale: Locale
  setLocale: (locale: Locale) => void
  dir: "ltr" | "rtl"
}

export const LanguageContext = createContext<LanguageContextType>({
  locale: "en",
  setLocale: () => {},
  dir: "ltr",
})

export function LanguageProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => {
    const saved = localStorage.getItem("lang")
    if (saved === "fr" || saved === "ar" || saved === "en") return saved
    return "en"
  })

  const setLocale = useCallback((newLocale: Locale) => {
    setLocaleState(newLocale)
    localStorage.setItem("lang", newLocale)
    document.documentElement.dir = newLocale === "ar" ? "rtl" : "ltr"
    document.documentElement.lang = newLocale
  }, [])

  const dir = locale === "ar" ? "rtl" : "ltr"

  // Sync dir attribute on mount and locale change
  useEffect(() => {
    document.documentElement.dir = dir
    document.documentElement.lang = locale
  }, [locale, dir])

  return (
    <LanguageContext.Provider value={{ locale, setLocale, dir }}>
      {children}
    </LanguageContext.Provider>
  )
}
