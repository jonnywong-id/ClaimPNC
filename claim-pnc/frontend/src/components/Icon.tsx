import type { SVGProps } from 'react'

/**
 * Ikon garis, digambar langsung sebagai SVG.
 *
 * # Kenapa tidak memakai pustaka ikon
 *
 * Aplikasi ini berjalan di VM on-premise tanpa jaminan akses internet, dan satu-satunya
 * artefak produksinya adalah binary Go berisi SPA tersemat (`ADR-0002`). Menambah
 * pustaka ikon berarti menambah dependensi yang harus dipelajari tim, dipantau
 * keamanannya, dan ikut membesarkan bundel — untuk delapan bentuk yang seluruhnya
 * beberapa baris path.
 *
 * Alasan yang sama membuat aplikasi ini memakai font sistem, bukan Google Fonts: berkas
 * yang diambil dari internet tidak akan sampai di jaringan tertutup, dan huruf yang
 * gagal dimuat mengubah seluruh tata letak.
 *
 * # Aturan pemakaian
 *
 * Seluruh ikon `aria-hidden` secara bawaan. Ikon TIDAK PERNAH menjadi satu-satunya
 * penjelas sebuah kontrol — tombol yang hanya berisi ikon wajib membawa `aria-label`,
 * karena pembaca layar tidak dapat menebak arti sebuah bentuk.
 */
type Props = SVGProps<SVGSVGElement>

function Base({ children, ...rest }: Props & { children: React.ReactNode }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.75}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      {...rest}
    >
      {children}
    </svg>
  )
}

/** Perisai — lambang aplikasi. Asuransi adalah perlindungan; bentuknya menyatakan itu. */
export function ShieldIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M12 3 5 6v5.5c0 4.2 2.9 7.9 7 9.5 4.1-1.6 7-5.3 7-9.5V6l-7-3Z" />
      <path d="m9 12 2.2 2.2L15.5 10" />
    </Base>
  )
}

export function SearchIcon(props: Props) {
  return (
    <Base {...props}>
      <circle cx="11" cy="11" r="6.5" />
      <path d="m16 16 4 4" />
    </Base>
  )
}

export function AddIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M12 6v12M6 12h12" />
    </Base>
  )
}

export function EditIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M4 20h4l10-10a2.8 2.8 0 0 0-4-4L4 16v4Z" />
      <path d="m13.5 6.5 4 4" />
    </Base>
  )
}

export function ReloadIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M20 12a8 8 0 1 1-2.6-5.9" />
      <path d="M20 4v4h-4" />
    </Base>
  )
}

export function LogoutIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M15 4h3a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-3" />
      <path d="M10 8 6 12l4 4M6 12h9" />
    </Base>
  )
}

export function LockIcon(props: Props) {
  return (
    <Base {...props}>
      <rect x="5" y="10.5" width="14" height="10" rx="2.5" />
      <path d="M8.5 10.5V8a3.5 3.5 0 1 1 7 0v2.5" />
    </Base>
  )
}

export function ListIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M8 7h12M8 12h12M8 17h12" />
      <path d="M4 7h.01M4 12h.01M4 17h.01" />
    </Base>
  )
}

export function EmptyBoxIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M3.5 8.5 12 4l8.5 4.5v7L12 20l-8.5-4.5v-7Z" />
      <path d="M3.5 8.5 12 13l8.5-4.5M12 13v7" />
    </Base>
  )
}
