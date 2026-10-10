import { Button } from '@/components/Button'

import { ambil, ambilDaftar } from './dokumen'

/**
 * Satu isian berlabel di dalam popup rincian.
 *
 * `jalur` menunjuk ke dalam dokumen JSON klaim; `label` disalin APA ADANYA dari
 * `pyLabelFieldValue` section Pega-nya. Keduanya berpasangan di berkas spesifikasinya,
 * sehingga label dan jalur tidak dapat menyimpang diam-diam.
 */
export type Isian = {
  label: string
  jalur: string
}

/** Satu kolom grid di dalam popup. */
export type KolomGrid = {
  label: string
  kunci: string
}

/**
 * Menggambar sekumpulan isian berlabel.
 *
 * # Tiga keadaan, dan ketiganya DIBEDAKAN
 *
 *	jalur tidak ada di dokumen   digambar bertanda, bukan sebagai sel kosong
 *	ada tetapi kosong            tanda pisah
 *	ada dan terisi               nilainya
 *
 * Pembedaan pertama yang paling berharga saat menelusuri: sel kosong yang berarti "isiannya
 * memang belum diisi" dan sel kosong yang berarti "jalurnya salah" tampak sama persis, dan
 * yang kedua adalah cacat yang tidak akan pernah dilaporkan siapa pun.
 */
export function Isian_({
  dokumen,
  isian,
}: {
  dokumen: Record<string, unknown>
  isian: Isian[]
}) {
  return (
    <dl className="grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-2">
      {isian.map((field) => {
        const hasil = ambil(dokumen, field.jalur)
        return (
          <div key={field.jalur}>
            <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">
              {field.label}
            </dt>
            <dd className="mt-0.5 text-sm text-slate-900">
              {!hasil.ada ? (
                <span
                  className="text-amber-700"
                  title={`Jalur ${field.jalur} tidak ada di dokumen klaim ini`}
                >
                  —
                </span>
              ) : hasil.nilai === '' ? (
                <span className="text-slate-400">—</span>
              ) : (
                hasil.nilai
              )}
            </dd>
          </div>
        )
      })}
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
  dokumen,
  jalur,
  kolom,
  kosong,
  onBuka,
}: {
  dokumen: Record<string, unknown>
  jalur: string
  kolom: KolomGrid[]
  kosong: string

  /**
   * Membuka sub-popup untuk satu baris — padanan `pyEditAction` Pega.
   *
   * Opsional: tidak semua grid punya aksi baris. Grid tanpa aksi digambar TANPA kolom
   * tambahan sama sekali, bukan dengan tombol yang mati.
   */
  onBuka?: (baris: Record<string, unknown>) => void
}) {
  const baris = ambilDaftar(dokumen, jalur)

  if (baris.length === 0) {
    return <p className="text-sm text-slate-500">{kosong}</p>
  }

  return (
    <div className="overflow-x-auto">
      <table className="min-w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-left">
            {kolom.map((k) => (
              <th key={k.kunci} className="px-3 py-2 text-xs font-medium uppercase tracking-wide text-slate-500">
                {k.label}
              </th>
            ))}
            {onBuka ? <th className="px-3 py-2" /> : null}
          </tr>
        </thead>
        <tbody>
          {baris.map((row, index) => (
            <tr key={index} className="border-b border-slate-100 last:border-0">
              {kolom.map((k) => {
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
