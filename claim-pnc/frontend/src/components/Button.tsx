import type { ButtonHTMLAttributes } from 'react'

/**
 * Nada menentukan berat sebuah tombol, bukan sekadar warnanya.
 *
 * - `utama`   — tindakan yang dituju pengguna di layar itu. Satu saja per kelompok.
 * - `kedua`   — tindakan yang sah tetapi bukan yang utama: Ubah, Muat ulang.
 * - `halus`   — tindakan yang membatalkan atau menutup. Tidak menarik perhatian.
 *
 * Membedakannya penting pada layar master: Simpan dan Batal berdampingan, dan tombol
 * yang tampak sama berat membuat pengguna menekan yang salah.
 */
export type ButtonTone = 'utama' | 'kedua' | 'halus'

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
  tone?: ButtonTone
}

/**
 * Tiga keadaan yang WAJIB terlihat pada setiap nada, bukan hanya pada yang utama:
 *
 *   hover         — mengangkat: warna menua, bayangan melebar, tombol naik 1px
 *   active        — menekan: tombol kembali turun dan menyusut tipis (98%)
 *   focus-visible — cincin 4px beropasitas rendah, mengikuti lengkung tombol
 *
 * Gerakan naik-turun itu yang membuat tombol terasa dapat ditekan. Tanpa `active`,
 * tombol terasa "mati" karena tidak ada umpan balik antara menekan dan hasilnya muncul —
 * dan pada jaringan lambat jeda itu bisa terasa lama.
 *
 * `focus-visible`, bukan `focus`: cincin hanya muncul untuk papan ketik. Memakai `focus`
 * membuat cincin ikut muncul setiap kali tombol diklik tetikus, yang terlihat seperti
 * cacat tampilan dan akhirnya membuat orang menghapus cincinnya sama sekali — termasuk
 * bagi pengguna papan ketik yang benar-benar membutuhkannya.
 */
const toneClass: Record<ButtonTone, string> = {
  utama: [
    'bg-blue-600 text-white shadow-aksen',
    'hover:bg-blue-700 hover:shadow-angkat hover:-translate-y-px',
    'active:translate-y-0 active:scale-[0.98] active:bg-blue-800',
    'focus-visible:ring-blue-500/35',
    'disabled:hover:translate-y-0 disabled:hover:bg-blue-600 disabled:hover:shadow-aksen',
  ].join(' '),

  kedua: [
    'border border-slate-300 bg-white text-slate-700 shadow-lembut',
    'hover:border-slate-400 hover:bg-slate-50 hover:text-slate-900 hover:shadow-angkat hover:-translate-y-px',
    'active:translate-y-0 active:scale-[0.98] active:bg-slate-100',
    'focus-visible:ring-slate-400/35',
    'disabled:hover:translate-y-0 disabled:hover:border-slate-300 disabled:hover:bg-white disabled:hover:shadow-lembut',
  ].join(' '),

  halus: [
    'text-slate-600',
    'hover:bg-slate-100 hover:text-slate-900',
    'active:scale-[0.98] active:bg-slate-200',
    'focus-visible:ring-slate-400/35',
    'disabled:hover:bg-transparent',
  ].join(' '),
}

/**
 * Button baku.
 *
 * Alasan keberadaannya sama dengan Field: tanpa komponen ini, setiap layar menulis
 * ulang kelas Tailwind, keadaan nonaktif, dan cincin fokusnya sendiri — dan pada puluhan
 * layar itu berubah menjadi puluhan tafsir berbeda tentang bagaimana sebuah tombol
 * terlihat saat sedang bekerja.
 *
 * `type` sengaja berbawaan `button`. Bawaan HTML adalah `submit`, dan tombol Batal yang
 * lupa menyebut tipenya akan diam-diam mengirim form.
 */
export function Button({ tone = 'kedua', className, type = 'button', ...rest }: Props) {
  const baseClass = [
    'inline-flex items-center justify-center gap-2 rounded-kontrol px-3.5 py-2',
    'text-sm font-medium whitespace-nowrap select-none',
    'transition-[background-color,border-color,color,box-shadow,transform] duration-150 ease-halus',
    'focus:outline-none focus-visible:ring-4',
    'disabled:cursor-not-allowed disabled:opacity-55 disabled:shadow-none',
    toneClass[tone],
  ].join(' ')

  return <button type={type} className={className ? `${baseClass} ${className}` : baseClass} {...rest} />
}
