// Latar belakang kristal: bidang-bidang pucat yang tersusun seperti pecahan kuarsa.
export default function Backdrop() {
  return (
    <svg
      className="backdrop"
      viewBox="0 0 1440 900"
      preserveAspectRatio="xMidYMid slice"
      aria-hidden="true"
      focusable="false"
    >
      <polygon points="0,0 560,0 340,420" fill="#8AD6DE" fillOpacity="0.26" />
      <polygon points="0,0 340,420 0,640" fill="#DCEFF6" fillOpacity="0.55" />
      <polygon points="560,0 1100,0 780,300" fill="#CFC8F5" fillOpacity="0.3" />
      <polygon points="340,420 780,300 560,0" fill="#FFFFFF" fillOpacity="0.45" />
      <polygon points="1100,0 1440,0 1440,380" fill="#F7D3E3" fillOpacity="0.38" />
      <polygon points="780,300 1100,0 1440,380" fill="#BFEAF0" fillOpacity="0.28" />
      <polygon points="0,640 340,420 520,900 0,900" fill="#CFC8F5" fillOpacity="0.24" />
      <polygon points="520,900 340,420 900,700" fill="#8AD6DE" fillOpacity="0.2" />
      <polygon points="340,420 780,300 900,700" fill="#FFFFFF" fillOpacity="0.32" />
      <polygon points="780,300 1440,380 900,700" fill="#DCEFF6" fillOpacity="0.5" />
      <polygon points="900,700 1440,380 1440,900" fill="#F7D3E3" fillOpacity="0.3" />
      <polygon points="520,900 900,700 1440,900" fill="#CFC8F5" fillOpacity="0.2" />
    </svg>
  )
}
