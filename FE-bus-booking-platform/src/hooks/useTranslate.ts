import { useContext } from "react"
import { LanguageContext } from "../context/LanguageContext"
import en from "../locales/en.json"
import fr from "../locales/fr.json"
import ar from "../locales/ar.json"

const translations = { en, fr, ar } as const

type TranslationValue = string | Record<string, unknown>

function getNestedValue(obj: TranslationValue, path: string): string {
  const keys = path.split(".")
  let current: TranslationValue = obj
  for (const key of keys) {
    if (current && typeof current === "object" && key in current) {
      current = (current as Record<string, unknown>)[key] as TranslationValue
    } else {
      return path // fallback: return the key path
    }
  }
  return typeof current === "string" ? current : path
}

export function useTranslate() {
  const { locale, dir } = useContext(LanguageContext)

  const t = (key: string): string => {
    const langTranslations = translations[locale]
    if (!langTranslations) return key
    return getNestedValue(langTranslations as unknown as TranslationValue, key)
  }

  return { t, locale, dir }
}
