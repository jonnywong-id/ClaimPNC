import type { Stage } from './types'

type Props = {
  /** Seluruh tahap alur, dipakai menerjemahkan pengenal menjadi nama. */
  tahap: Stage[]
  /** Rangkaian pengenal tahap dari tahap sekarang sampai akhir. */
  jalur: string[]
  /** Pengenal tahap tempat klaim berada sekarang. */
  currentStage: string
}

/**
 * StagePath menunjukkan DI MANA klaim berada dan APA yang menantinya.
 *
 * # Kenapa layar ini ada
 *
 * Tahapan klaim di Pega terbentuk dari Flow dan Flow Action, dan petugas terbiasa dengan
 * urutannya. Bila layar baru tidak menampilkan tahapan yang sama, petugas kehilangan
 * orientasi — bukan karena datanya salah, melainkan karena mereka tidak lagi tahu sedang
 * di mana (`TKT-U4-001`).
 *
 * # Kenapa ia menyebut dirinya perkiraan
 *
 * Alur Register bercabang di empat tempat, dan cabangnya dipilih dari data klaim — serta,
 * pada satu keputusan, dari peran orang yang menekan tombol. Jalur yang ditampilkan di
 * sini adalah jalur MENURUT DATA HARI INI. Ia dapat berubah bila datanya berubah, dan
 * itu memang perilaku sistem lama; yang tidak boleh terjadi adalah petugas mengira ini
 * janji.
 */
export function StagePath({ tahap, jalur, currentStage }: Props) {
  if (jalur.length === 0) return null

  const stageNames = new Map(tahap.map((t) => [t.id, t.nama]))

  return (
    <nav aria-label="Tahapan klaim" className="rounded border border-slate-200 bg-slate-50 p-4">
      <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">
        Tahapan klaim
      </h2>

      <ol className="mt-3 flex flex-wrap items-center gap-x-2 gap-y-2">
        {jalur.map((id, order) => {
          const current = id === currentStage
          return (
            <li key={id} className="flex items-center gap-2">
              <span
                aria-current={current ? 'step' : undefined}
                className={
                  current
                    ? 'rounded-full bg-slate-900 px-3 py-1 text-sm font-medium text-white'
                    : 'rounded-full border border-slate-300 bg-white px-3 py-1 text-sm text-slate-600'
                }
              >
                {stageNames.get(id) ?? id}
              </span>
              {order < jalur.length - 1 && (
                <span aria-hidden="true" className="text-slate-400">
                  →
                </span>
              )}
            </li>
          )
        })}
      </ol>

      <p className="mt-3 text-xs text-slate-500">
        Tahapan sesudah tahap berjalan adalah perkiraan menurut data klaim saat ini.
        Mengubah lini bisnis atau penanda RCL/PUCL dapat mengubahnya.
      </p>
    </nav>
  )
}
