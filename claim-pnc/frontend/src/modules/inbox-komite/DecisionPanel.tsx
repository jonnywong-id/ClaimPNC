import { useEffect, useRef, useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { KomiteDecisionKind, type KomiteCase, type KomiteDecisionKind as Kind } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { TextAreaField } from '@/components/TextAreaField'
import { formatRupiah } from '@/lib/money'

type Props = {
  item: KomiteCase
  working: boolean
  error: unknown
  onClose: () => void
  onSubmit: (decision: Kind, note: string) => void
}

/** Batas panjang catatan; sama dengan `komite.MaxNoteLength` di server. */
const MAX_NOTE = 1000

const CHOICES: { kind: Kind; label: string; hint: string }[] = [
  {
    kind: KomiteDecisionKind.approve,
    label: 'Setuju',
    hint: 'Meneruskan ke jenjang berikutnya, atau menyelesaikan bila ini jenjang terakhir.',
  },
  {
    kind: KomiteDecisionKind.reject,
    label: 'Tolak',
    hint: 'Menghentikan SELURUH komite, bukan hanya jenjang ini.',
  },
  {
    kind: KomiteDecisionKind.return,
    label: 'Kembalikan',
    hint: 'Mengembalikan kepada pengaju untuk diperbaiki. Komite berhenti.',
  },
]

/**
 * Panel keputusan komite.
 *
 * Menggantikan flow action `ViewTransferDtl` pada `Flow/Komite_Flow.xml` — layar "Lihat
 * Detail Transfer" tempat anggota komite memberi keputusan.
 *
 * # Kenapa panel di dalam halaman, bukan dialog yang menutupi layar
 *
 * Keputusannya diambil dengan MEMBACA angka: nilai klaim, nilai ASM share, dan penilaian
 * AI. Dialog yang menutupi tabel memaksa anggota komite mengingat angka yang baru saja ia
 * lihat, dan angka yang diingat adalah angka yang dapat salah diingat.
 *
 * # Ketiga keputusan ditampilkan bersamaan, beserta akibatnya
 *
 * Bukan dropdown. Akibat ketiganya BERBEDA JAUH — satu meneruskan, dua menghentikan
 * seluruh komite — dan perbedaan itu harus terbaca sebelum tombolnya ditekan, bukan
 * sesudahnya. Di sistem lama ketiganya adalah nilai `AcceptStatus` yang dipilih tanpa satu
 * pun keterangan.
 *
 * # Konfirmasi hanya untuk yang menghentikan
 *
 * Tolak dan kembalikan menutup komite dan tidak dapat ditarik kembali (`ADR-0012`).
 * Persetujuan tidak menutup apa pun selama masih ada jenjang berikutnya, sehingga
 * memintanya berkonfirmasi hanya menambah satu klik pada pekerjaan yang paling sering
 * dilakukan.
 *
 * Konfirmasinya DI DALAM halaman, bukan `window.confirm`. Dialog bawaan peramban tidak
 * dapat memuat nomor kasus dengan penekanan yang benar, tampil berbeda di setiap
 * peramban, dan tidak dapat diuji sama sekali — sementara yang dikonfirmasi di sini
 * adalah penghentian klaim yang tidak dapat ditarik kembali.
 */
export function DecisionPanel({ item, working, error, onClose, onSubmit }: Props) {
  const [decision, setDecision] = useState<Kind | null>(null)
  const [note, setNote] = useState('')
  const [touched, setTouched] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const headingRef = useRef<HTMLHeadingElement>(null)

  // Fokus dipindahkan ke panel saat ia terbuka, supaya pengguna papan ketik dan pembaca
  // layar tidak tertinggal di tabel — panelnya muncul di luar urutan baca mereka.
  useEffect(() => {
    headingRef.current?.focus()
  }, [item.nomor_case])

  // Isian dikosongkan saat berpindah kasus. Tanpa ini, catatan penolakan untuk satu kasus
  // akan terbawa ke kasus berikutnya yang dibuka — dan tercatat permanen di sana.
  useEffect(() => {
    setDecision(null)
    setNote('')
    setTouched(false)
    setConfirming(false)
  }, [item.nomor_case])

  const terminal = decision === KomiteDecisionKind.reject || decision === KomiteDecisionKind.return
  const noteMissing = terminal && note.trim() === ''
  const noteTooLong = note.length > MAX_NOTE

  // Mengganti pilihan MEMBATALKAN konfirmasi yang sedang berjalan. Tanpa ini, seseorang
  // yang menekan "Simpan" pada Tolak lalu berpindah ke Setuju akan menemukan tombol
  // konfirmasi penghentian masih menunggu — dan menekannya mengira ia menyetujui.
  function choose(kind: Kind) {
    setDecision(kind)
    setConfirming(false)
  }

  function submit() {
    setTouched(true)
    if (decision === null || noteMissing || noteTooLong) return

    // Keputusan yang menghentikan komite menempuh satu langkah tambahan, dan langkah itu
    // menyebutkan nomor kasusnya — supaya yang dihentikan bukan kasus yang salah.
    if (terminal && !confirming) {
      setConfirming(true)
      return
    }
    onSubmit(decision, note.trim())
  }

  return (
    <section
      aria-labelledby="judul-keputusan-komite"
      className="rounded-kartu border border-slate-200 bg-white p-5 shadow-sm"
    >
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h2
            id="judul-keputusan-komite"
            ref={headingRef}
            tabIndex={-1}
            className="text-base font-semibold text-slate-900 outline-none"
          >
            Keputusan komite — {item.nomor_case}
          </h2>
          <p className="mt-1 text-sm text-slate-600">
            Klaim {item.nomor_klaim || '—'} · {item.nama_tertanggung || 'tanpa nama tertanggung'}
          </p>
        </div>
        <Button tone="halus" onClick={onClose}>
          Tutup
        </Button>
      </header>

      <dl className="mt-4 grid gap-3 sm:grid-cols-3">
        <Figure label="Nilai klaim" value={formatRupiah(item.nilai_klaim)} />
        <Figure label="Nilai ASM share" value={formatRupiah(item.nilai_asm_share)} />
        <Figure label="Nilai OR ASM" value={formatRupiah(item.nilai_or_asm)} />
      </dl>

      {/*
        Penilaian AI ditampilkan sebagai KETERANGAN, bukan sebagai anjuran. Ia satu bahan
        pertimbangan di antara yang lain, dan layar tidak boleh menyiratkan bahwa
        menyetujui berarti mengikutinya.

        Kasus TANPA penilaian AI menyebutkannya apa adanya. Membiarkannya kosong akan
        terbaca seperti "AI menolak", dan itu arti yang sama sekali berbeda.
      */}
      <div className="mt-4 rounded-kartu border border-slate-200 bg-slate-50 p-3">
        <p className="text-xs font-medium uppercase tracking-wide text-slate-500">Penilaian AI</p>
        {item.ada_penilaian_ai ? (
          <>
            <p className="mt-1 text-sm font-medium text-slate-900">{item.jawaban_ai || '—'}</p>
            {item.note_ai_diterima && (
              <p className="mt-1 text-sm text-slate-600">{item.note_ai_diterima}</p>
            )}
            {item.note_ai_ditolak && (
              <p className="mt-1 text-sm text-slate-600">{item.note_ai_ditolak}</p>
            )}
          </>
        ) : (
          <p className="mt-1 text-sm text-slate-600">
            Kasus ini belum dinilai AI. Itu tidak menghalangi keputusan Anda.
          </p>
        )}
      </div>

      <fieldset className="mt-5">
        <legend className="text-xs font-medium uppercase tracking-wide text-slate-500">
          Keputusan
        </legend>
        <div className="mt-2 grid gap-2 sm:grid-cols-3">
          {CHOICES.map((choice) => {
            const selected = decision === choice.kind
            return (
              <label
                key={choice.kind}
                className={[
                  'cursor-pointer rounded-kartu border p-3 text-left',
                  'transition-[border-color,background-color] duration-150 ease-halus',
                  selected
                    ? 'border-blue-600 bg-blue-50 ring-1 ring-blue-600'
                    : 'border-slate-200 hover:border-slate-300 hover:bg-slate-50',
                ].join(' ')}
              >
                <span className="flex items-center gap-2">
                  <input
                    type="radio"
                    name="keputusan-komite"
                    value={choice.kind}
                    checked={selected}
                    onChange={() => choose(choice.kind)}
                    className="h-4 w-4"
                  />
                  <span className="text-sm font-medium text-slate-900">{choice.label}</span>
                </span>
                <span className="mt-1 block text-xs leading-relaxed text-slate-600">
                  {choice.hint}
                </span>
              </label>
            )
          })}
        </div>
        {touched && decision === null && (
          <p className="mt-2 text-sm text-red-700">Pilih salah satu keputusan lebih dulu.</p>
        )}
      </fieldset>

      <div className="mt-4">
        <TextAreaField
          id="catatan-komite"
          label={terminal ? 'Catatan (wajib)' : 'Catatan'}
          value={note}
          rows={3}
          onChange={(e) => setNote(e.target.value)}
          error={
            touched && noteMissing
              ? 'Catatan wajib diisi supaya alasannya dapat ditindaklanjuti.'
              : noteTooLong
                ? `Catatan paling panjang ${MAX_NOTE} karakter.`
                : undefined
          }
          hint={
            terminal
              ? 'Pengaju membaca catatan ini untuk tahu apa yang harus diperbaiki.'
              : 'Boleh dikosongkan pada persetujuan.'
          }
        />
      </div>

      {error !== null && error !== undefined && (
        <div className="mt-4">
          <DecisionError error={error} />
        </div>
      )}

      {confirming && (
        <div
          role="alert"
          className="mt-4 rounded-kartu border border-amber-300 bg-amber-50 p-3 text-sm leading-relaxed text-amber-900"
        >
          Kasus <strong>{item.nomor_case}</strong> akan{' '}
          {decision === KomiteDecisionKind.reject ? 'ditolak' : 'dikembalikan'}. Komite
          berhenti seluruhnya dan keputusan ini <strong>tidak dapat ditarik kembali</strong>.
          Tekan sekali lagi untuk melanjutkan.
        </div>
      )}

      <div className="mt-5 flex flex-wrap gap-2">
        <Button tone="utama" onClick={submit} disabled={working}>
          {working
            ? 'Menyimpan…'
            : confirming
              ? 'Ya, lanjutkan'
              : 'Simpan keputusan'}
        </Button>
        <Button
          tone="kedua"
          onClick={confirming ? () => setConfirming(false) : onClose}
          disabled={working}
        >
          Batal
        </Button>
      </div>

      <p className="mt-3 text-xs leading-relaxed text-slate-500">
        Keputusan tercatat permanen beserta nama dan waktunya, dan tidak dapat diubah
        maupun dihapus. Perubahan pikiran dinyatakan dengan keputusan baru.
      </p>
    </section>
  )
}

function Figure({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-kartu border border-slate-200 p-3">
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className="mt-1 text-sm font-semibold tabular-nums text-slate-900">{value}</dd>
    </div>
  )
}

/**
 * Galat saat menyimpan keputusan.
 *
 * Konflik (`409`) dibedakan dari kegagalan lain dan diberi tindakan yang benar: muat
 * ulang. Ia terjadi saat keputusan sudah tercatat dari tab lain, dan menyuruh pengguna
 * "coba lagi" di sana akan membuatnya menekan tombol berulang kali pada sesuatu yang
 * memang tidak akan berubah.
 */
function DecisionError({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Keputusan belum tersimpan. Periksa koneksi lalu coba lagi."
        tone="gangguan"
      />
    )
  }
  if (error instanceof APIError && error.status === 409) {
    return (
      <ErrorMessage
        title="Keputusan sudah tercatat"
        description={error.message}
        tone="penolakan"
      />
    )
  }
  return (
    <ErrorMessage
      title="Keputusan gagal disimpan"
      description={error instanceof Error ? error.message : 'Terjadi kesalahan pada sistem.'}
      tone="penolakan"
    />
  )
}
