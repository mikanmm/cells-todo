const pad = (n) => String(n).padStart(2, '0')

export function toDateStr(d) {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function todayStr() {
  return toDateStr(new Date())
}

// Selisih hari antara tanggal "YYYY-MM-DD" dan hari ini (negatif = sudah lewat).
export function dayDiff(dateStr) {
  const [y, m, d] = dateStr.split('-').map(Number)
  const target = new Date(y, m - 1, d)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  return Math.round((target - today) / 86400000)
}

const shortDate = new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short' })
const weekday = new Intl.DateTimeFormat('id-ID', { weekday: 'long' })
const longDate = new Intl.DateTimeFormat('id-ID', {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  year: 'numeric',
})

export function dueLabel(dateStr) {
  const diff = dayDiff(dateStr)
  const [y, m, d] = dateStr.split('-').map(Number)
  const date = new Date(y, m - 1, d)
  if (diff === 0) return 'Hari ini'
  if (diff === 1) return 'Besok'
  if (diff === -1) return 'Kemarin'
  if (diff < 0) return `${-diff} hari lalu`
  if (diff < 7) return weekday.format(date)
  return shortDate.format(date)
}

export function todayLong() {
  return longDate.format(new Date())
}
