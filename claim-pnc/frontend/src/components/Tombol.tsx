import type { ButtonHTMLAttributes, ReactNode } from 'react'

/**
 * Peran tombol menentukan seberapa menonjol ia terlihat, bukan sekadar warnanya.
 *
 * - `utama`    — tindakan yang dituju pengguna saat membuka layar. Satu per layar.
 * - `sekunder` — tindakan yang wajar tetapi bukan tujuan utama.
 * - `halus`    — tindakan yang membatalkan atau mundur; tidak boleh menarik perhatian.
 *
 * Membedakan ketiganya penting pada form: "Simpan" dan "Batal" yang terlihat sama
 * membuat pengguna menekan yang salah, dan pada layar master itu berarti kehilangan
 * isian yang baru diketik.
 */
export type PeranTombol = 'utama' | 'sekunder' | 'halus'

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
  peran?: PeranTombol
  /** Ditampilkan menggantikan anak selama tindakan berjalan. */
  sedangJalan?: boolean
  teksSedangJalan?: string
  children: ReactNode
}

const kelasPeran: Record<PeranTombol, string> = {
  utama: 'bg-slate-900 text-white hover:bg-slate-700 border border-slate-900',
  sekunder: 'bg-white text-slate-700 hover:bg-slate-100 border border-slate-300',
  halus: 'bg-transparent text-slate-600 hover:bg-slate-100 border border-transparent',
}

/**
 * Tombol baku aplikasi.
 *
 * Ia menangani satu hal yang mudah terlupa bila setiap layar menulis tombolnya sendiri:
 * tombol yang sedang menjalankan tindakan WAJIB nonaktif. Tanpa itu, pengguna yang
 * menekan "Simpan" dua kali mengirim dua permintaan — dan pada layar yang menambah
 * baris, itu menghasilkan dua baris.
 */
export function Tombol({
  peran = 'sekunder',
  sedangJalan = false,
  teksSedangJalan,
  children,
  disabled,
  className,
  type = 'button',
  ...sisa
}: Props) {
  const kelas =
    'rounded px-3 py-2 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60 ' +
    kelasPeran[peran]

  return (
    <button
      type={type}
      disabled={disabled || sedangJalan}
      className={className ? `${kelas} ${className}` : kelas}
      {...sisa}
    >
      {sedangJalan && teksSedangJalan ? teksSedangJalan : children}
    </button>
  )
}
