// Logo thémis — une balance stylisée (justice, équilibre), en trait.
// Hérite de currentColor ; on lui passe une taille.
export function Logo({ size = 26, className = '' }) {
  return (
    <svg
      className={className}
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {/* mât + base */}
      <path d="M16 5v22" />
      <path d="M11 27h10" />
      {/* fléau */}
      <path d="M7 9h18" />
      <circle cx="16" cy="6.5" r="1.6" fill="currentColor" stroke="none" />
      {/* plateau gauche : deux fils + coupelle */}
      <path d="M7 9l-3 6M7 9l3 6" />
      <path d="M3.2 15a3.8 2 0 0 0 7.6 0z" />
      {/* plateau droit */}
      <path d="M25 9l-3 6M25 9l3 6" />
      <path d="M21.2 15a3.8 2 0 0 0 7.6 0z" />
    </svg>
  )
}
