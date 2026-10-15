import { useNavigate, useParams } from 'react-router-dom'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

/**
 * Tujuan sementara tombol "Lihat Detail Klaim".
 *
 * # Kenapa berkas ini ada
 *
 * Work Owner memutuskan 2026-09-20 tombol "Lihat Detail Klaim" pada layar Inbox Admin
 * DIBANGUN. Layar tujuannya adalah menu `MENU_ID 75` "View Claim" (`PNCViewClaim`) — modul
 * tersendiri yang belum dibangun.
 *
 * Tiga pilihan ada, dan dua di antaranya buruk:
 *
 *   - Tombol yang menuju rute tidak terdaftar → pengguna terlempar ke beranda tanpa
 *     penjelasan, dan tampak seperti kerusakan.
 *   - Tombol yang mati → tidak memenuhi keputusan Work Owner, dan menyembunyikan bahwa
 *     jalurnya sudah lengkap sampai ke kunci klaimnya.
 *   - Tujuan yang menyatakan keadaan sebenarnya → yang dipakai di sini.
 *
 * # Kenapa di app/, bukan di modul mana pun
 *
 * Karena ia bukan milik Inbox Admin: modul mana pun yang kelak membuka rincian klaim akan
 * menuju rute yang sama. Ia juga bukan milik modul View Claim, sebab modul itu belum ada
 * dan berkas ini justru akan DIHAPUS saat modul itu lahir.
 *
 * # Yang TIDAK dilakukan di sini
 *
 * Tidak ada satu pun permintaan ke server. Menampilkan sebagian data klaim di layar
 * sementara akan menciptakan kontrak yang harus dipelihara, untuk layar yang seluruh
 * bentuknya belum dirancang.
 */
export function ViewClaimPlaceholder() {
  const { referensi } = useParams<{ referensi: string }>()
  const navigate = useNavigate()

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">View Claim</h1>
        <p className="mt-1 text-sm text-slate-600">Rincian satu klaim.</p>
      </header>

      <div className="mt-4">
        <ErrorMessage
          title="Layar rincian klaim belum dibangun"
          description={
            'Layar ini adalah menu "View Claim", modul tersendiri yang belum termasuk ' +
            'dalam pekerjaan yang sudah selesai. Klaim yang Anda pilih sudah dikenali dan ' +
            'akan langsung terbuka di sini begitu modulnya ada.'
          }
          tone="gangguan"
        />
      </div>

      {referensi && (
        <p className="mt-4 text-sm text-slate-600">
          Klaim yang dipilih:{' '}
          <span className="font-mono text-slate-900">{decodeURIComponent(referensi)}</span>
        </p>
      )}

      <div className="mt-6">
        <Button tone="kedua" onClick={() => navigate(-1)}>
          Kembali ke antrean
        </Button>
      </div>
    </div>
  )
}
