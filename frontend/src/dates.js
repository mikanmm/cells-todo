const jakartaDate = new Intl.DateTimeFormat('en-CA', {
  timeZone: 'Asia/Jakarta', year: 'numeric', month: '2-digit', day: '2-digit',
})

const pad = (n) => String(n).padStart(2, '0')

export function toDateStr(d) {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function todayStr() {
  return jakartaDate.format(new Date())
}

// Days between a YYYY-MM-DD date and today; negative values indicate past dates.
export function dayDiff(dateStr) {
  return (Date.parse(dateStr) - Date.parse(todayStr())) / 86400000
}

const shortDate = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short' })
const weekday = new Intl.DateTimeFormat('en-GB', { weekday: 'long' })
const longDate = new Intl.DateTimeFormat('en-GB', {
  timeZone: 'Asia/Jakarta',
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  year: 'numeric',
})

export function dueLabel(dateStr) {
  const diff = dayDiff(dateStr)
  const [y, m, d] = dateStr.split('-').map(Number)
  const date = new Date(y, m - 1, d)
  if (diff === 0) return 'Today'
  if (diff === 1) return 'Tomorrow'
  if (diff === -1) return 'Yesterday'
  if (diff < 0) return `${-diff} days ago`
  if (diff < 7) return weekday.format(date)
  return shortDate.format(date)
}

export function todayLong() {
  return longDate.format(new Date())
}
