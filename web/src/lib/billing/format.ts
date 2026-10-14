// Money is integer cents; dates are YYYY-MM-DD and shown as M/D/YYYY.

export function usd(cents: number | null | undefined): string {
  if (cents === null || cents === undefined) return 'N/A';
  const neg = cents < 0, c = Math.abs(cents);
  const whole = Math.floor(c / 100).toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  return `${neg ? '-' : ''}$${whole}.${String(c % 100).padStart(2, '0')}`;
}

export const plain = (cents: number) => usd(cents).replace('$', '');

// parseMoney reads "$1,234.5" or "1234.56" without floating point; null if invalid.
export function parseMoney(text: string): number | null {
  const s = text.replace(/[$,\s]/g, '');
  const m = s.match(/^(\d{1,12})(?:\.(\d{0,2}))?$/) ?? s.match(/^()\.(\d{1,2})$/);
  if (!m) return null;
  return Number(m[1] || '0') * 100 + Number((m[2] ?? '').padEnd(2, '0'));
}

export const moneyText = (cents: number) => (cents / 100).toFixed(2);

export function us(date: string | undefined | null): string {
  const m = (date ?? '').match(/^(\d{4})-(\d{2})-(\d{2})/);
  return m ? `${+m[2]}/${+m[3]}/${m[1]}` : '';
}

const monthNames = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'];
export function longDate(date: string): string {
  const m = date.match(/^(\d{4})-(\d{2})-(\d{2})/);
  return m ? `${monthNames[+m[2] - 1]} ${m[3]}, ${m[1]}` : date;
}

// stamp shows an activity time as "3/1/2027, 7:47 am".
export function stamp(at: string): string {
  const m = at.match(/^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2})/);
  if (!m) return us(at);
  const h = +m[2], ampm = h < 12 ? 'am' : 'pm';
  return `${us(m[1])}, ${h % 12 || 12}:${m[3]} ${ampm}`;
}

// created shows a payment time as "1/1/2027 05:04 AM".
export function created(at: string): string {
  const m = at.match(/^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2})/);
  if (!m) return us(at);
  const h = +m[2];
  return `${us(m[1])} ${String(h % 12 || 12).padStart(2, '0')}:${m[3]} ${h < 12 ? 'AM' : 'PM'}`;
}

export function addDays(date: string, n: number): string {
  const d = new Date(date + 'T00:00:00Z');
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
}

export function csv(rows: string[][]): string {
  return rows.map(r => r.map(c => /[",\n]/.test(c) ? `"${c.replace(/"/g, '""')}"` : c).join(',')).join('\r\n') + '\r\n';
}

// initials of a person or company name for the menu avatar.
export const initials = (name: string) => name.split(/[\s,&]+/).filter(w => /^[A-Za-z]/.test(w)).slice(0, 2).map(w => w[0].toUpperCase()).join('') || '·';
