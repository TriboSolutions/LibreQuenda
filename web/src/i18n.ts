import i18n from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'
import caAccessibility from './locales/ca/accessibility.json'
import caLegal from './locales/ca/legal.json'
import caAdmin from './locales/ca/admin.json'
import caBookings from './locales/ca/bookings.json'
import caCommon from './locales/ca/common.json'
import caCouncils from './locales/ca/councils.json'
import caEntrance from './locales/ca/entrance.json'
import caFacility from './locales/ca/facility.json'
import caHome from './locales/ca/home.json'
import caInvites from './locales/ca/invites.json'
import caNotifications from './locales/ca/notifications.json'
import caProfile from './locales/ca/profile.json'
import caReader from './locales/ca/reader.json'
import enAccessibility from './locales/en/accessibility.json'
import enLegal from './locales/en/legal.json'
import enAdmin from './locales/en/admin.json'
import enBookings from './locales/en/bookings.json'
import enCommon from './locales/en/common.json'
import enCouncils from './locales/en/councils.json'
import enEntrance from './locales/en/entrance.json'
import enFacility from './locales/en/facility.json'
import enHome from './locales/en/home.json'
import enInvites from './locales/en/invites.json'
import enNotifications from './locales/en/notifications.json'
import enProfile from './locales/en/profile.json'
import enReader from './locales/en/reader.json'
import esAccessibility from './locales/es/accessibility.json'
import esLegal from './locales/es/legal.json'
import esAdmin from './locales/es/admin.json'
import esBookings from './locales/es/bookings.json'
import esCommon from './locales/es/common.json'
import esCouncils from './locales/es/councils.json'
import esEntrance from './locales/es/entrance.json'
import esFacility from './locales/es/facility.json'
import esHome from './locales/es/home.json'
import esInvites from './locales/es/invites.json'
import esNotifications from './locales/es/notifications.json'
import esProfile from './locales/es/profile.json'
import esReader from './locales/es/reader.json'
import glAccessibility from './locales/gl/accessibility.json'
import glLegal from './locales/gl/legal.json'
import glAdmin from './locales/gl/admin.json'
import glBookings from './locales/gl/bookings.json'
import glCommon from './locales/gl/common.json'
import glCouncils from './locales/gl/councils.json'
import glEntrance from './locales/gl/entrance.json'
import glFacility from './locales/gl/facility.json'
import glHome from './locales/gl/home.json'
import glInvites from './locales/gl/invites.json'
import glNotifications from './locales/gl/notifications.json'
import glProfile from './locales/gl/profile.json'
import glReader from './locales/gl/reader.json'

export const languages = [
  { code: 'gl', label: 'Galego' },
  { code: 'es', label: 'Español' },
  { code: 'en', label: 'English' },
  { code: 'ca', label: 'Català' },
] as const

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      gl: { common: glCommon, accessibility: glAccessibility, councils: glCouncils, home: glHome, bookings: glBookings, admin: glAdmin, facility: glFacility, profile: glProfile, reader: glReader, entrance: glEntrance, invites: glInvites, notifications: glNotifications, legal: glLegal },
      es: { common: esCommon, accessibility: esAccessibility, councils: esCouncils, home: esHome, bookings: esBookings, admin: esAdmin, facility: esFacility, profile: esProfile, reader: esReader, entrance: esEntrance, invites: esInvites, notifications: esNotifications, legal: esLegal },
      en: { common: enCommon, accessibility: enAccessibility, councils: enCouncils, home: enHome, bookings: enBookings, admin: enAdmin, facility: enFacility, profile: enProfile, reader: enReader, entrance: enEntrance, invites: enInvites, notifications: enNotifications, legal: enLegal },
      ca: { common: caCommon, accessibility: caAccessibility, councils: caCouncils, home: caHome, bookings: caBookings, admin: caAdmin, facility: caFacility, profile: caProfile, reader: caReader, entrance: caEntrance, invites: caInvites, notifications: caNotifications, legal: caLegal },
    },
    supportedLngs: languages.map((l) => l.code),
    load: 'languageOnly', // es-ES -> es
    fallbackLng: 'gl', // a Galician public service
    ns: ['common', 'accessibility', 'councils', 'home', 'bookings', 'admin', 'facility', 'profile', 'reader', 'entrance', 'invites', 'notifications', 'legal'],
    defaultNS: 'common',
    interpolation: { escapeValue: false }, // React escapes
    detection: {
      order: ['querystring', 'localStorage', 'navigator'], // ?lng=gl for shared links
      lookupQuerystring: 'lng',
      lookupLocalStorage: 'lang',
      caches: ['localStorage'],
    },
  })

const syncHtmlLang = () => {
  document.documentElement.lang = i18n.resolvedLanguage ?? 'gl'
}
syncHtmlLang()
i18n.on('languageChanged', syncHtmlLang)

export default i18n
