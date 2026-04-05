import { format } from 'date-fns';
import { sv, enGB } from 'date-fns/locale';

export function getLocale(language: string) {
  return language === 'sv' ? sv : enGB;
}

export function formatTime(date: Date | string, language: string): string {
  return format(new Date(date), 'HH:mm', { locale: getLocale(language) });
}

export function formatDate(date: Date | string, language: string): string {
  return format(new Date(date), 'd MMM yyyy', { locale: getLocale(language) });
}

export function formatMonthYear(date: Date | string, language: string): string {
  return format(new Date(date), 'MMMM yyyy', { locale: getLocale(language) });
}

export function formatMonthKey(date: Date | string): string {
  return format(new Date(date), 'yyyy-MM');
}
