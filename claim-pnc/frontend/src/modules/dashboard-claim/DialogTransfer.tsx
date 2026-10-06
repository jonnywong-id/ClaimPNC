import { useEffect, useRef, useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'

import { useAjukanTransfer } from './api'
import type { LingkupTransfer, PermintaanTransfer } from './types'

/**
 * Dialog pengajuan Transfer.
 *
 * Melayani KEDUA tombol layar lama dengan satu bentuk, dibedakan `lingkup`:
 *
 *	baris    tombol "Transfer" pada satu baris — isian User ID Lama disembunyikan
 *	massal   "Transfer All Case By UserID" — User ID Lama wajib
 *
 * # Ia mengajukan PERMINTAAN, dan itu dinyatakan di dialognya sendiri
 *
 * `P-1` menetapkan `PC_ASSIGN_WORKLIST` masih ditulis Pega selama masa paralel. Penugasannya
 * TIDAK berpindah saat tombol ditekan, dan barisnya tetap ada di daftar.
 *
 * Tanpa keterangan itu, pengguna menekan Transfer, melihat barisnya tidak berubah, lalu
 * menekannya lagi — dan setiap penekanan mencatat satu permintaan.
 */
export function DialogTransfer({
  lingkup,
  nomorKlaim,
  klaimID,
  onTutup,
}: {
  lingkup: LingkupTransfer
  nomorKlaim?: string
  klaimID?: string
  onTutup: () => void
}) {
  const [userLama, setUserLama] = useState('')
  const [userBaru, setUserBaru] = useState('')
  const [tipePengguna, setTipePengguna] = useState('')
  const [alasan, setAlasan] = useState('')

  const ajukan = useAjukanTransfer()
  const tutupRef = useRef<HTMLButtonElement>(null)

  // Escape menutup dialog, dan fokus berpindah ke dalamnya saat dibuka.
  //
  // Keduanya perilaku yang diharapkan dari dialog mana pun; tanpa yang kedua, pengguna papan
  // tik harus menelusuri seluruh halaman di belakangnya untuk mencapai isiannya.
  useEffect(() => {
    tutupRef.current?.focus()

    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') onTutup()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onTutup])

  const pelanggaran = ajukan.error instanceof APIError ? ajukan.error.violations() : {}
  const berhasil = ajukan.isSuccess

  function kirim(event: React.FormEvent) {
    event.preventDefault()

    // Field opsional hanya DITAMBAHKAN bila terisi, bukan dikirim bernilai undefined.
    //
    // `exactOptionalPropertyTypes` menolak yang kedua, dan penolakan itu benar: `{alasan:
    // undefined}` dan `{}` tersandi menjadi JSON yang sama, tetapi hanya yang kedua yang
    // jujur menyatakan isiannya tidak diisi.
    const body: PermintaanTransfer = {
      lingkup,
      user_id_baru: userBaru,
      ...(tipePengguna ? { tipe_pengguna: tipePengguna } : {}),
      ...(alasan ? { alasan } : {}),
      ...(lingkup === 'baris'
        ? {
            ...(klaimID ? { klaim_id: klaimID } : {}),
            ...(nomorKlaim ? { nomor_klaim: nomorKlaim } : {}),
          }
        : { user_id_lama: userLama }),
    }

    ajukan.mutate(body)
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-label={lingkup === 'baris' ? 'Transfer klaim' : 'Transfer seluruh pekerjaan'}
    >
      <div className="w-full max-w-lg rounded-kartu bg-white p-6 shadow-terbang">
        <div className="mb-4 flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-slate-900">
              {lingkup === 'baris' ? 'Transfer Klaim' : 'Transfer All Case By UserID'}
            </h2>
            {lingkup === 'baris' && nomorKlaim ? (
              <p className="mt-1 text-sm text-slate-600">No Klaim {nomorKlaim}</p>
            ) : null}
          </div>
          {/*
            Tombol polos, bukan komponen Button: ia perlu `ref` untuk menerima fokus saat
            dialog dibuka, dan Button tidak meneruskan ref. Gayanya disamakan dengan
            `tone="halus"`.
          */}
          <button
            ref={tutupRef}
            type="button"
            onClick={onTutup}
            aria-label="Tutup"
            className={[
              'rounded-kontrol px-3 py-1.5 text-sm text-slate-500',
              'transition ease-halus hover:bg-slate-100 hover:text-slate-700',
              'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2',
              'focus-visible:outline-blue-600',
            ].join(' ')}
          >
            ✕
          </button>
        </div>

        {berhasil ? (
          <>
            <div className="rounded-kontrol border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-900">
              <p className="font-medium">Permintaan tercatat.</p>
              <p className="mt-1 leading-relaxed">
                Penugasannya <strong>belum berpindah</strong> — barisnya masih akan tampil di
                daftar. Pemindahan dikerjakan di Pega selama masa paralel, dan permintaan ini
                sudah masuk antreannya.
              </p>
            </div>
            <div className="mt-4 flex justify-end">
              <Button tone="utama" onClick={onTutup}>
                Tutup
              </Button>
            </div>
          </>
        ) : (
          <form onSubmit={kirim} className="space-y-4">
            {lingkup === 'massal' ? (
              <Field
                id="transfer-user-lama"
                label="User ID Lama"
                value={userLama}
                onChange={(event) => setUserLama(event.target.value)}
                error={pelanggaran['user_id_lama']}
                hint="Seluruh pekerjaan operator ini akan dipindahkan."
              />
            ) : null}

            <Field
              id="transfer-user-baru"
              label="User ID Baru"
              value={userBaru}
              onChange={(event) => setUserBaru(event.target.value)}
              error={pelanggaran['user_id_baru']}
            />

            <Field
              id="transfer-tipe-pengguna"
              label="Type User"
              value={tipePengguna}
              onChange={(event) => setTipePengguna(event.target.value)}
              error={pelanggaran['tipe_pengguna']}
              // Daftar pilihannya adalah Rule-Obj-FieldValue yang HILANG dari export
              // (`R-16`), sehingga isiannya bebas dan tidak divalidasi. Menebak daftarnya
              // berarti menolak nilai sah yang tidak kita kenal.
              hint="Opsional. Daftar pilihannya belum diterima dari Tim Pega."
            />

            <Field
              id="transfer-alasan"
              label="Alasan"
              value={alasan}
              onChange={(event) => setAlasan(event.target.value)}
              error={pelanggaran['alasan']}
              hint="Opsional. Tercatat pada jejak permintaan."
            />

            {ajukan.isError ? (
              <ErrorMessage
                tone={pelanggaran && Object.keys(pelanggaran).length > 0 ? 'penolakan' : 'gangguan'}
                title="Permintaan tidak dapat dicatat"
                description={pesanGalat(ajukan.error)}
              />
            ) : null}

            <div className="flex justify-end gap-2">
              <Button tone="kedua" type="button" onClick={onTutup}>
                Batal
              </Button>
              <Button tone="utama" type="submit" disabled={ajukan.isPending}>
                {ajukan.isPending ? 'Mengirim…' : 'Ajukan Transfer'}
              </Button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}

/** Membaca pesan galat yang layak dibaca pengguna. */
function pesanGalat(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}
