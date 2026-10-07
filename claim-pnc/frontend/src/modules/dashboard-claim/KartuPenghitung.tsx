import type { Kartu, Tile } from './types'

/**
 * Satu kartu penghitung pada Dashboard Claim.
 *
 * # Kenapa tombol, bukan div yang dapat diklik
 *
 * Kartu ini MEMILIH tile yang ditelusuri, sehingga ia kendali — bukan hiasan. Elemen
 * `button` memberi fokus papan tik, tanggapan Enter dan Spasi, serta peran yang terbaca
 * pembaca layar tanpa satu baris pun ARIA tambahan.
 *
 * `div` ber-onClick akan tampak sama bagi yang memakai tetikus, dan tidak dapat dipakai sama
 * sekali oleh yang tidak.
 */
export function KartuPenghitung({
  kartu,
  terpilih,
  memuat,
  onPilih,
}: Readonly<{
  kartu: Kartu
  terpilih: boolean
  memuat: boolean
  onPilih: (tile: Tile) => void
}>) {
  return (
    <button
      type="button"
      onClick={() => onPilih(kartu.tile)}
      aria-pressed={terpilih}
      className={[
        'flex flex-col items-start gap-1 rounded-kartu border p-5 text-left',
        'transition ease-halus',
        'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2',
        'focus-visible:outline-blue-600',
        terpilih
          ? 'border-blue-300 bg-blue-50 shadow-angkat'
          : 'border-slate-200 bg-white shadow-lembut hover:-translate-y-0.5 hover:border-blue-200 hover:shadow-angkat',
      ].join(' ')}
    >
      <span className="text-sm font-medium text-slate-600">{kartu.judul}</span>

      {memuat ? (
        // Rangka berdenyut, bukan angka nol.
        //
        // Angka nol saat memuat tidak dapat dibedakan dari nol yang benar — dan pada layar
        // manajerial, nol yang keliru terbaca sebagai "tidak ada pekerjaan".
        <span
          aria-hidden="true"
          className="mt-1 h-9 w-20 animate-pulse rounded bg-slate-200"
        />
      ) : (
        <span className="mt-1 text-3xl font-semibold tabular-nums text-slate-900">
          {kartu.jumlah.toLocaleString('id-ID')}
        </span>
      )}

      <span className="text-xs text-slate-500">
        {kartu.bentuk === 'survei' ? 'baris survei' : 'klaim'}
      </span>
    </button>
  )
}
