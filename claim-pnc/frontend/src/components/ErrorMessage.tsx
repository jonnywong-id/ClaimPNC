/**
 * Nada menentukan cara pesan dibaca pengguna, bukan sekadar warnanya.
 *
 * - `penolakan` — sistem menolak apa yang pengguna lakukan. Ada yang bisa ia perbaiki.
 * - `gangguan`  — sistem yang sedang bermasalah. Mengulang-ulang tidak menolong.
 *
 * Perbedaan ini penting pada layar masuk: "kata sandi salah" dan "sistem identitas
 * sedang tidak dapat dihubungi" menuntut tindak lanjut yang berbeda, dan menyamakan
 * tampilannya membuat pengguna mencoba berulang kali ke sistem yang sedang down.
 */
export type ErrorTone = 'penolakan' | 'gangguan'

type Props = {
  title: string
  description: string
  tone: ErrorTone
}

/**
 * Tiap nada dibedakan warna DAN bentuk ikonnya — lingkaran seru untuk penolakan,
 * segitiga untuk gangguan.
 *
 * Bentuk yang berbeda itu bukan hiasan: warna saja tidak terbaca oleh pengguna buta
 * warna, dan kedua nada ini menuntut tindakan yang berbeda. Bila keduanya hanya berbeda
 * merah dan kuning, sebagian pengguna tidak akan pernah melihat bedanya.
 */
const style: Record<ErrorTone, { box: string; icon: string; title: string }> = {
  penolakan: {
    box: 'border-red-200 bg-red-50/80',
    icon: 'bg-red-100 text-red-700',
    title: 'text-red-900',
  },
  gangguan: {
    box: 'border-amber-200 bg-amber-50/80',
    icon: 'bg-amber-100 text-amber-800',
    title: 'text-amber-900',
  },
}

/** ErrorMessage menampilkan satu kotak pesan kesalahan yang dapat ditindaklanjuti. */
export function ErrorMessage({ title, description, tone }: Props) {
  const g = style[tone]

  return (
    <div
      role="alert"
      className={`flex items-start gap-3 rounded-kartu border p-3.5 text-sm shadow-lembut ${g.box}`}
    >
      <span
        aria-hidden="true"
        className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full ${g.icon}`}
      >
        {tone === 'penolakan' ? (
          <svg viewBox="0 0 20 20" className="h-4 w-4 fill-current">
            <path d="M10 2a8 8 0 1 0 0 16 8 8 0 0 0 0-16Zm0 3.6a.9.9 0 0 1 .9.9v4.2a.9.9 0 1 1-1.8 0V6.5a.9.9 0 0 1 .9-.9Zm0 7.5a1 1 0 1 1 0 2 1 1 0 0 1 0-2Z" />
          </svg>
        ) : (
          <svg viewBox="0 0 20 20" className="h-4 w-4 fill-current">
            <path d="M9.1 2.6a1 1 0 0 1 1.8 0l7 12.6a1 1 0 0 1-.9 1.5H3a1 1 0 0 1-.9-1.5l7-12.6ZM10 6.8a.9.9 0 0 0-.9.9v3.5a.9.9 0 1 0 1.8 0V7.7a.9.9 0 0 0-.9-.9Zm0 6.4a1 1 0 1 0 0 2 1 1 0 0 0 0-2Z" />
          </svg>
        )}
      </span>

      <div className="min-w-0">
        <p className={`font-semibold ${g.title}`}>{title}</p>
        <p className="mt-0.5 text-slate-700">{description}</p>
      </div>
    </div>
  )
}
