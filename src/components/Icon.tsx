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

/** Kartu — rekening bank. Dipakai menu Master Rekening. */
export function CardIcon(props: Props) {
  return (
    <Base {...props}>
      <rect x="3" y="5.5" width="18" height="13" rx="2.5" />
      <path d="M3 10h18" />
      <path d="M6.5 14.5h3.5" />
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

/**
 * Tempat sampah — aksi hapus.
 *
 * Ia ditambahkan bersama modul Master XOL, layar pertama yang benar-benar punya tombol
 * Hapus. Bentuknya tetap garis seperti ikon lain; yang membedakan tindakan merusak dari
 * tindakan biasa adalah TEKS tombol dan konfirmasinya, bukan warna ikonnya — sekitar satu
 * dari dua belas laki-laki tidak dapat membedakan merah dari abu-abu.
 */
export function TrashIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M4 7h16" />
      <path d="M10 11v6M14 11v6" />
      <path d="M6 7l1 12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-12" />
      <path d="M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
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

/** Anak panah — penanda kelompok menu yang terbuka atau tertutup. */
export function ChevronIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="m9 6 6 6-6 6" />
    </Base>
  )
}

/** Tiga garis — pembuka menu pada layar sempit. */
export function MenuIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M4 7h16M4 12h16M4 17h16" />
    </Base>
  )
}

/** Silang — penutup menu pada layar sempit. */
export function CloseIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="m6 6 12 12M18 6 6 18" />
    </Base>
  )
}

/**
 * Tangga — dipakai menandai menu Ambang Komite.
 *
 * Bentuknya dipilih karena ia menggambarkan hal yang sebenarnya: jenjang persetujuan
 * komite adalah tangga yang dinaiki bertingkat sesuai besarnya nilai klaim, dan setiap
 * anak tangga yang terlampaui ikut menyetujui.
 */
export function StairsIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M3 20h5v-5h5v-5h5V5h3" />
      <path d="M3 20v-1M8 15h5M13 10h5" />
    </Base>
  )
}

/** Segitiga peringatan — dipakai menandai temuan pada master ambang. */
export function WarningIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M12 4.5 21 19.5H3L12 4.5Z" />
      <path d="M12 10v4M12 17h.01" />
    </Base>
  )
}

/** Timbangan — dipakai menandai menu Penjenjangan Komite. */
export function ScaleIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M12 4v16M7 20h10M4.5 8h15M12 4.5 4.5 8M12 4.5 19.5 8" />
      <path d="M2 14a2.5 2.5 0 0 0 5 0L4.5 8 2 14ZM17 14a2.5 2.5 0 0 0 5 0L19.5 8 17 14Z" />
    </Base>
  )
}

/** Rumah — Beranda. */
export function HomeIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M4 10.5 12 4l8 6.5V19a1.5 1.5 0 0 1-1.5 1.5H15v-5.5H9v5.5H5.5A1.5 1.5 0 0 1 4 19v-8.5Z" />
    </Base>
  )
}

/** Panel samping — tombol memperkecil/memperbesar menu kiri. */
export function SidebarIcon(props: Props) {
  return (
    <Base {...props}>
      <rect x="3.5" y="4.5" width="17" height="15" rx="2.5" />
      <path d="M9.5 4.5v15" />
    </Base>
  )
}

/** Baki masuk — kelompok menu INBOX. */
export function InboxIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M4 13.5 6.2 6a1.5 1.5 0 0 1 1.4-1h8.8a1.5 1.5 0 0 1 1.4 1L20 13.5V18a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 4 18v-4.5Z" />
      <path d="M4 13.5h4.5l1.2 2h4.6l1.2-2H20" />
    </Base>
  )
}

/** Grafik batang — kelompok menu REPORT. */
export function ChartIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M4 20h16" />
      <path d="M7 16v-4" />
      <path d="M12 16V7" />
      <path d="M17 16v-6" />
    </Base>
  )
}

/** Mata — kelompok menu VIEW. */
export function EyeIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12Z" />
      <circle cx="12" cy="12" r="2.75" />
    </Base>
  )
}

/** Papan klip — kelompok menu SURVEYOR. */
export function ClipboardIcon(props: Props) {
  return (
    <Base {...props}>
      <rect x="5.5" y="5" width="13" height="15.5" rx="2" />
      <path d="M9 5V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v1" />
      <path d="M9 11h6" />
      <path d="M9 15h4" />
    </Base>
  )
}

/** Unduh — tombol ekspor berkas. */
export function DownloadIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M12 4v11" />
      <path d="m7.5 10.5 4.5 4.5 4.5-4.5" />
      <path d="M5 19.5h14" />
    </Base>
  )
}

/**
 * Map — penanda baris yang MEMBUKA daftar lain.
 *
 * Dipakai pada tabel ringkas "Status Register" layar Inbox Komunikasi Cabang, tempat Pega
 * menggambar ikon map di depan setiap baris untuk menyatakan baris itu dapat dibuka.
 *
 * Artinya dibawa `aria-expanded` pada tombolnya, bukan oleh ikon ini — karena itu ia tetap
 * `aria-hidden` seperti seluruh ikon lain di berkas ini.
 */
export function FolderIcon(props: Props) {
  return (
    <Base {...props}>
      <path d="M4 7.5a1.5 1.5 0 0 1 1.5-1.5h3.3l2 2h7.7A1.5 1.5 0 0 1 20 9.5v8a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 4 17.5v-10Z" />
    </Base>
  )
}
