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
  note: string
  tone: ErrorTone
}

const classes: Record<ErrorTone, string> = {
  penolakan: 'border-red-200 bg-red-50 text-red-800',
  gangguan: 'border-amber-200 bg-amber-50 text-amber-900',
}

/** ErrorMessage menampilkan satu kotak pesan kesalahan yang dapat ditindaklanjuti. */
export function ErrorMessage({ title, note, tone }: Props) {
  return (
    <div role="alert" className={`rounded border p-3 text-sm ${classes[tone]}`}>
      <p className="font-medium">{title}</p>
      <p className="mt-1">{note}</p>
    </div>
  )
}
