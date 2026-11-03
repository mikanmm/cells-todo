// Permata berfacet: enam bidang yang bisa "mengkristal" satu per satu.
const FACETS = [
  { points: '7,3 21,3 17,11 11,11', alpha: 0.55 }, // meja
  { points: '7,3 11,11 1,11', alpha: 0.8 }, // mahkota kiri
  { points: '21,3 27,11 17,11', alpha: 0.95 }, // mahkota kanan
  { points: '1,11 11,11 14,26', alpha: 0.72 }, // paviliun kiri
  { points: '11,11 17,11 14,26', alpha: 0.5 }, // paviliun tengah
  { points: '17,11 27,11 14,26', alpha: 0.9 }, // paviliun kanan
]

// Warna isi permata dan warna garis tepinya (garis dibuat lebih gelap agar
// kotak centang tetap terlihat jelas di atas latar terang).
const TONES = {
  low: { fill: 'var(--aqua)', line: '#2e97a6' },
  medium: { fill: 'var(--amethyst)', line: '#6f5fd0' },
  high: { fill: 'var(--rose)', line: '#c9487f' },
}

function GemSvg({ tone, filled, size, className = '' }) {
  const colors = TONES[tone] || { fill: tone, line: tone }
  return (
    <svg
      className={`gem ${className}`}
      viewBox="0 0 28 28"
      width={size}
      height={size}
      aria-hidden="true"
      focusable="false"
      style={{ '--gem': colors.fill, '--gem-line': colors.line }}
    >
      {FACETS.map((f, i) => {
        const on = filled > i
        return (
          <polygon
            key={f.points}
            points={f.points}
            style={{
              fillOpacity: on ? f.alpha : 0.1,
              stroke: on ? 'rgba(255, 255, 255, 0.85)' : undefined,
              transitionDelay: on ? `${i * 60}ms` : '0ms',
            }}
          />
        )
      })}
    </svg>
  )
}

// Kotak centang tugas. Menyala penuh (6 facet) saat tugas selesai.
export function GemCheck({ priority, done, label, onToggle }) {
  return (
    <button
      type="button"
      role="checkbox"
      aria-checked={done}
      aria-label={label}
      className={`gem-check ${done ? 'is-done' : ''}`}
      onClick={onToggle}
    >
      <GemSvg tone={priority} filled={done ? 6 : 0} size={30} />
    </button>
  )
}

// Permata progres: jumlah facet yang terisi mengikuti persentase tugas selesai.
export function ProgressGem({ done, total }) {
  const filled = total === 0 ? 0 : done === total ? 6 : Math.min(5, Math.floor((done / total) * 6))
  return <GemSvg tone="var(--teal)" filled={filled} size={112} className="gem-progress" />
}

// Dekorasi statis untuk halaman masuk.
export function GemStatic({ tone, size }) {
  return <GemSvg tone={tone} filled={6} size={size} />
}

export function GemDot({ tone }) {
  return <GemSvg tone={tone} filled={6} size={14} className="gem-dot" />
}
