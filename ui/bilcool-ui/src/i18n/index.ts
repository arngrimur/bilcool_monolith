import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import enCommon from './locales/en/common.json';
import enBookings from './locales/en/bookings.json';
import svCommon from './locales/sv/common.json';
import svBookings from './locales/sv/bookings.json';

i18n.use(initReactI18next).init({
  resources: {
    en: { common: enCommon, bookings: enBookings },
    sv: { common: svCommon, bookings: svBookings },
  },
  lng: 'sv',
  fallbackLng: 'sv',
  defaultNS: 'common',
  interpolation: { escapeValue: false },
});

export default i18n;
