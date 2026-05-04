import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import enCommon from './locales/en/common.json';
import enBookings from './locales/en/bookings.json';
import enWhereIs from './locales/en/where_is.json';
import svCommon from './locales/sv/common.json';
import svBookings from './locales/sv/bookings.json';
import svWhereIs from './locales/sv/where_is.json';

i18n.use(initReactI18next).init({
  resources: {
    en: { common: enCommon, bookings: enBookings, where_is: enWhereIs },
    sv: { common: svCommon, bookings: svBookings, where_is: svWhereIs },
  },
  lng: 'sv',
  fallbackLng: 'sv',
  defaultNS: 'common',
  interpolation: { escapeValue: false },
});

export default i18n;
