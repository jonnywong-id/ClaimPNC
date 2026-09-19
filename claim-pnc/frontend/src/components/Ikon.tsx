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

function Dasar({ children, ...sisa }: Props & { children: React.ReactNode }) {
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
      {...sisa}
    >
      {children}
    </svg>
  )
}

/** Perisai — lambang aplikasi. Asuransi adalah perlindungan; bentuknya menyatakan itu. */
export function IkonPerisai(props: Props) {
  return (
    <Dasar {...props}>
      <path d="M12 3 5 6v5.5c0 4.2 2.9 7.9 7 9.5 4.1-1.6 7-5.3 7-9.5V6l-7-3Z" />
      <path d="m9 12 2.2 2.2L15.5 10" />
    </Dasar>
  )
}

export function IkonCari(props: Props) {
  return (
    <Dasar {...props}>
      <circle cx="11" cy="11" r="6.5" />
      <path d="m16 16 4 4" />
    </Dasar>
  )
}

export function IkonTambah(props: Props) {
  return (
    <Dasar {...props}>
      <path d="M12 6v12M6 12h12" />
    </Dasar>
  )
}

export function IkonUbah(props: Props) {
  return (
    <Dasar {...props}>
      <path d="M4 20h4l10-10a2.8 2.8 0 0 0-4-4L4 16v4Z" />
      <path d="m13.5 6.5 4 4" />
    </Dasar>
  )
}

export function IkonMuatUlang(props: Props) {
  return (
    <Dasar {...props}>
      <path d="M20 12a8 8 0 1 1-2.6-5.9" />
      <path d="M20 4v4h-4" />
    </Dasar>
  )
}

export function IkonKeluar(props: Props) {
  return (
    <Dasar {...props}>
      <path d="M15 4h3a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-3" />
      <path d="M10 8 6 12l4 4M6 12h9" />
    </Dasar>
  )
}

export function IkonKunci(props: Props) {
  return (
    <Dasar {...props}>
      <rect x="5" y="10.5" width="14" height="10" rx="2.5" />
      <path d="M8.5 10.5V8a3.5 3.5 0 1 1 7 0v2.5" />
    </Dasar>
  )
}

export function IkonDaftar(props: Props) {
  return (
    <Dasar {...props}>
      <path d="M8 7h12M8 12h12M8 17h12" />
      <path d="M4 7h.01M4 12h.01M4 17h.01" />
    </Dasar>
  )
}

export function IkonKotakKosong(props: Props) {
  return (
    <Dasar {...props}>
      <path d="M3.5 8.5 12 4l8.5 4.5v7L12 20l-8.5-4.5v-7Z" />
      <path d="M3.5 8.5 12 13l8.5-4.5M12 13v7" />
    </Dasar>
  )
}

/**
 * Tangga — dipakai menandai menu Ambang Komite.
 *
 * Bentuknya dipilih karena ia menggambarkan hal yang sebenarnya: jenjang persetujuan
 * komite adalah tangga yang dinaiki bertingkat sesuai besarnya nilai klaim, dan setiap
 * anak tangga yang terlampaui ikut menyetujui.
 */
export function IkonTangga(props: Props) {
  return (
    <Dasar {...props}>
      <path d="M3 20h5v-5h5v-5h5V5h3" />
      <path d="M3 20v-1M8 15h5M13 10h5" />
    </Dasar>
  )
}

/** Segitiga peringatan — dipakai menandai temuan pada master ambang. */
export function IkonPeringatan(props: Props) {
  return (
    <Dasar {...props}>
      <path d="M12 4.5 21 19.5H3L12 4.5Z" />
      <path d="M12 10v4M12 17h.01" />
    </Dasar>
  )
}

/** Timbangan — dipakai menandai menu Penjenjangan Komite. */
export function IkonTimbangan(props: Props) {
  return (
    <Dasar {...props}>
      <path d="M12 4v16M7 20h10M4.5 8h15M12 4.5 4.5 8M12 4.5 19.5 8" />
      <path d="M2 14a2.5 2.5 0 0 0 5 0L4.5 8 2 14ZM17 14a2.5 2.5 0 0 0 5 0L19.5 8 17 14Z" />
    </Dasar>
  )
}
