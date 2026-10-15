import { Button } from '@/components/Button'

/**
 * Satu isian berlabel di dalam popup rincian.
 *
 * `label` disalin APA ADANYA dari `pyLabelFieldValue` section Pega-nya (`D-13`).
 *
 * `kunci` dulu berupa JALUR di dalam dokumen JSON klaim (`ClaimData.ReportDate`). Sejak
 * migrasi lepas dari JSON — keputusan Work Owner 2026-10-12 — ia nama field pada kontrak
 * (`tanggal_lapor`). Keduanya tetap berpasangan di berkas spesifikasinya, sehingga label dan
 * sumbernya tidak dapat menyimpang diam-diam.
 *
 * `absen` menandai isian yang BELUM punya kolom di basis data. Ia dinyatakan, bukan dihapus
 * dari daftar: isian yang hilang tanpa jejak tidak akan pernah diselesaikan, sedangkan isian
 * bertanda terlihat oleh siapa pun yang membuka layarnya.
 */
export type Isian = {
  label: string
  kunci: string
  absen?: string
}

/**
 * Satu kolom grid di dalam popup.
 *
 * `absen` menandai kolom yang BELUM punya sumber, beserta sebabnya — sama seperti pada `Isian`.
 * Kolomnya tetap digambar: menghapusnya membuat perbedaan dengan layar lama hilang tanpa jejak,
 * sedangkan kolom bertanda terlihat oleh siapa pun yang membukanya.
 */
export type KolomGrid = {
  label: string
  kunci: string
  absen?: string
}

/**
 * Menggambar sekumpulan isian berlabel.
 *
 * # Tiga keadaan, dan ketiganya DIBEDAKAN
 *
 *	isian belum punya kolom   digambar bertanda beserta SEBABNYA
 *	punya kolom, kosong       tanda pisah
 *	punya kolom, terisi       nilainya
 *
 * Pembedaan pertama yang paling berharga: sel kosong yang berarti "nilainya memang belum
 * diisi" dan sel kosong yang berarti "kolomnya belum ada" tampak sama persis, dan yang kedua
 * adalah pekerjaan belum selesai yang tidak akan pernah dilaporkan siapa pun.
 */
export function Isian_({ nilai, isian }: { nilai: Record<string, string>; isian: Isian[] }) {
  return (
    <dl className="grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-2">
      {isian.map((field) => (
        <div key={field.kunci}>
          <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">
            {field.label}
          </dt>
          <dd className="mt-0.5 text-sm text-slate-900">
            {field.absen !== undefined ? (
              <span className="text-amber-700" title={field.absen}>
                belum tersambung
              </span>
            ) : (nilai[field.kunci] ?? '').trim() === '' ? (
              <span className="text-slate-400">—</span>
            ) : (
              nilai[field.kunci]
            )}
          </dd>
        </div>
      ))}
    </dl>
  )
}

/**
 * Menggambar satu grid di dalam popup.
 *
 * Daftar KOSONG dinyatakan, bukan digambar sebagai tabel berkepala tanpa isi: tabel kosong
 * terbaca sebagai layar yang gagal memuat, sedangkan klaim tanpa objek memang ada.
 */
export function GridRincian({
  baris: daftar,
  kolom,
  kosong,
  onBuka,
}: {
  /**
   * Baris yang digambar, datang dari SERVER sudah berbentuk daftar.
   *
   * Sampai 2026-10-12 komponen ini menerima dokumen JSON beserta jalur, lalu menelusurinya
   * sendiri. Setelah migrasi lepas dari JSON (keputusan Work Owner), penelusuran itu tidak
   * punya objek lagi — server mengirim tiap bagian sebagai larik tersendiri.
   *
   * Akibat yang disengaja: salah ketik nama bagian kini menjadi galat TypeScript, bukan grid
   * yang diam-diam kosong.
   */
  baris: Record<string, string>[]
  kolom: KolomGrid[]
  kosong: string

  /**
   * Membuka sub-popup untuk satu baris — padanan `pyEditAction` Pega.
   *
   * Opsional: tidak semua grid punya aksi baris. Grid tanpa aksi digambar TANPA kolom
   * tambahan sama sekali, bukan dengan tombol yang mati.
   */
  onBuka?: (baris: Record<string, string>) => void
}) {
  if (daftar.length === 0) {
    return <p className="text-sm text-slate-500">{kosong}</p>
  }

  return (
    <div className="overflow-x-auto">
      <table className="min-w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-left">
            {kolom.map((k) => (
              <th
                key={k.kunci}
                className="px-3 py-2 text-xs font-medium uppercase tracking-wide text-slate-500"
              >
                {k.label}
              </th>
            ))}
            {onBuka ? <th className="px-3 py-2" /> : null}
          </tr>
        </thead>
        <tbody>
          {daftar.map((row, index) => (
            <tr key={index} className="border-b border-slate-100 last:border-0">
              {kolom.map((k) => {
                // Kolom tanpa sumber dibedakan dari kolom yang nilainya memang kosong.
                // Keduanya tampak sama sebagai tanda pisah, dan yang pertama adalah pekerjaan
                // belum selesai yang tidak akan pernah dilaporkan siapa pun.
                if (k.absen !== undefined) {
                  return (
                    <td key={k.kunci} className="px-3 py-2">
                      <span className="text-amber-700" title={k.absen}>
                        belum tersambung
                      </span>
                    </td>
                  )
                }
                const nilai = row[k.kunci]
                const teks =
                  nilai === null || nilai === undefined || typeof nilai === 'object'
                    ? ''
                    : String(nilai)
                return (
                  <td key={k.kunci} className="px-3 py-2 text-slate-900">
                    {teks === '' ? <span className="text-slate-400">—</span> : teks}
                  </td>
                )
              })}
              {onBuka ? (
                <td className="px-3 py-2">
                  <Button tone="halus" onClick={() => onBuka(row)}>
                    Detail
                  </Button>
                </td>
              ) : null}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
