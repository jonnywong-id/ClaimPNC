import { NetworkError } from '@/api/client'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useOutstandingSummary } from './api'
import type { DocumentStatusCode, DocumentStatusCount, OutstandingFilter } from './types'

/** Id panel isi tab — satu panel yang isinya berganti mengikuti tab aktif. */
export const PANEL_ID = 'panel-status-dokumen'

/** Id tombol tab, dirujuk panel lewat `aria-labelledby`. */
export function tabId(kode: string): string {
  return `tab-status-dokumen-${kode}`
}

type Props = {
  /** Penyaring layar yang sedang berlaku; jumlah tab menghormati semuanya kecuali status. */
  filter: OutstandingFilter

  /** Status yang sedang dipilih; teks kosong berarti seluruh status (tab "ALL Case"). */
  selected: DocumentStatusCode | ''
  onSelect: (status: DocumentStatusCode | '') => void
}

/**
 * Deret tab status dokumen My Inbox.
 *
 * # Apa yang digantikannya
 *
 * `Section/InboxRegister_Section-Section.xml` menampilkan status dokumen lewat TIGA hal
 * sekaligus: sembilan tab bertuliskan tebal (`:11438`, `:11626`, `:11841`, `:12023`,
 * `:12170`, `:12632`, `:12825`, `:12971`, `:13184`), sebuah donut (`:4151` —
 * `pyType=pie`, `pySubType=doughnut`), dan tabel jumlahnya.
 *
 * Ketiganya diganti **satu deret tab** atas keputusan Work Owner 2026-10-05: donut dan
 * tabel hierarkinya tidak dibawa. Angka yang dulu dibaca dari keduanya kini hidup sebagai
 * lencana pada tabnya sendiri — satu tempat, bukan tiga, dan tidak ada lagi dua sajian
 * yang bisa berbeda tanpa ada yang menyadarinya.
 *
 * Bentuknya mengikuti `SourceTab` pada Inbox Auto Claim: `role="tablist"`, bergaris bawah,
 * dan berpindah dengan panah. Yang ditiru POLANYA; modulnya tidak disentuh.
 *
 * # Tab tanpa lencana bukan tab bernilai nol
 *
 * Enam dari sembilan tab belum dapat dihitung dari `T_CLAIMLIST_ADMIN`:
 *
 *     Loss Adjuster · Internal Surveyor   Work-SurveyClaim — case type LAIN
 *     Temporary Close · Deadline          Resolved-Completed — di LUAR himpunan ini
 *     Communication                       tabel komunikasi
 *     TKA                                 kolom TKA_1 TIDAK ADA di tabel ini
 *
 * Keenamnya tetap DIGAMBAR supaya petugas Pega mengenali layarnya, tetapi tanpa lencana
 * dan tidak dapat ditekan. Memberinya angka nol akan menyatakan "tidak ada satu pun",
 * padahal yang benar "belum dihitung".
 */
export function DocumentStatusTab({ filter, selected, onSelect }: Props) {
  const summary = useOutstandingSummary(filter)

  if (summary.isError) {
    return (
      <div className="mt-5">
        <ErrorMessage
          title="Status dokumen tidak dapat dimuat"
          description={
            summary.error instanceof NetworkError
              ? 'Server Claim PNC tidak dapat dihubungi.'
              : 'Daftar klaim di bawah tetap dapat dipakai. Coba muat ulang halaman ini.'
          }
          tone="gangguan"
        />
      </div>
    )
  }

  if (summary.isPending) {
    return (
      <div
        className="mt-5 h-11 animate-pulse rounded-kontrol bg-slate-100"
        aria-label="Memuat status dokumen"
      />
    )
  }

  // `?? []` bukan kerapian: respons yang cacat atau versi backend yang lebih tua akan
  // menjatuhkan SELURUH layar bila dibaca membabi buta — dan yang hilang hanyalah deret
  // tab ini, sementara daftar klaim di bawahnya tetap berguna.
  const status = summary.data.status ?? []
  const total = summary.data.total ?? 0

  if (status.length === 0) return null

  return (
    <div className="mt-5 overflow-x-auto">
      <StatusTabList status={status} total={total} selected={selected} onSelect={onSelect} />
    </div>
  )
}

/**
 * Daftar tabnya sendiri.
 *
 * # Kenapa `role="tablist"`, bukan sekadar deretan tombol
 *
 * Dengan peran yang benar, pembaca layar mengumumkan "tab 2 dari 9" dan panah kiri/kanan
 * berpindah tab — perilaku yang diharapkan pengguna papan ketik dari sesuatu yang terlihat
 * seperti tab. Deretan tombol biasa terbaca sebagai sembilan tombol lepas tanpa hubungan.
 */
function StatusTabList({
  status,
  total,
  selected,
  onSelect,
}: {
  status: DocumentStatusCount[]
  total: number
  selected: DocumentStatusCode | ''
  onSelect: (status: DocumentStatusCode | '') => void
}) {
  // Hanya tab yang dapat dipilih yang ikut navigasi panah. Memasukkan yang mati akan
  // membuat panah berhenti di tab yang tidak dapat dibuka, dan pengguna papan ketik
  // kehilangan cara meneruskannya.
  const dapatDipilih = status.filter((s) => s.dapat_dipilih)

  /** Tab "ALL Case" berarti TANPA penyaring; kode selainnya menyaring satu status. */
  function kodeTerpilih(s: DocumentStatusCount): DocumentStatusCode | '' {
    return s.kode === 'semua' ? '' : s.kode
  }

  function aktif(s: DocumentStatusCount): boolean {
    return kodeTerpilih(s) === selected
  }

  function pindah(arah: -1 | 1) {
    if (dapatDipilih.length === 0) return
    const sekarang = dapatDipilih.findIndex((s) => aktif(s))
    if (sekarang < 0) return
    // Berputar di ujungnya, mengikuti perilaku tab yang lazim.
    const tujuan = (sekarang + arah + dapatDipilih.length) % dapatDipilih.length
    const berikut = dapatDipilih[tujuan]
    if (berikut !== undefined) onSelect(kodeTerpilih(berikut))
  }

  return (
    <div
      role="tablist"
      aria-label="Status dokumen"
      className="flex gap-1 border-b border-slate-200"
    >
      {status.map((s) => {
        const dipilih = aktif(s)

        // "ALL Case" memakai total, bukan jumlahnya sendiri: ia keadaan tanpa penyaring,
        // dan angkanya wajib cocok dengan total paginasi grid di bawahnya.
        const jumlah = s.kode === 'semua' ? total : s.jumlah

        return (
          <button
            key={s.kode}
            id={tabId(s.kode)}
            role="tab"
            type="button"
            disabled={!s.dapat_dipilih}
            aria-selected={dipilih}
            aria-controls={dipilih ? PANEL_ID : undefined}
            // Hanya tab yang aktif yang dapat di-Tab-kan; panah memindahkan yang lain.
            // Itu pola papan ketik baku untuk tablist, dan tanpa itu pengguna papan ketik
            // harus menekan Tab melewati sembilan tab sebelum sampai ke isinya.
            tabIndex={dipilih ? 0 : -1}
            /*
              Lencananya diberi nama, bukan dibiarkan terbaca sebagai angka telanjang.
              Tanpa ini pembaca layar menyebut "Complete documents0" — angkanya menempel
              tanpa jeda, dan "0" sendirian tidak menyatakan nol apa.
            */
            aria-label={
              !s.dapat_dipilih
                ? `${s.judul}, belum tersedia`
                : jumlah === null
                  ? s.judul
                  : `${s.judul}, ${jumlah} klaim`
            }
            title={s.dapat_dipilih ? undefined : 'Sumber datanya belum dimigrasikan dari Pega.'}
            onClick={() => onSelect(kodeTerpilih(s))}
            onKeyDown={(event) => {
              if (event.key === 'ArrowRight') {
                event.preventDefault()
                pindah(1)
              }
              if (event.key === 'ArrowLeft') {
                event.preventDefault()
                pindah(-1)
              }
            }}
            className={[
              '-mb-px flex items-center gap-2 border-b-2 px-4 py-2.5 text-sm font-medium whitespace-nowrap',
              'transition-[color,border-color] duration-150 ease-halus',
              'focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/20',
              !s.dapat_dipilih
                ? 'cursor-not-allowed border-transparent text-slate-400'
                : dipilih
                  ? 'border-blue-600 text-blue-700'
                  : 'border-transparent text-slate-600 hover:border-slate-300 hover:text-slate-900',
            ].join(' ')}
          >
            {s.judul}
            {/*
              Lencana hanya digambar bila jumlahnya BENAR-BENAR dihitung — konvensi yang
              sama dengan tab "Data rejected" pada Inbox Laporan Klaim.
            */}
            {jumlah !== null && (
              <span
                aria-hidden="true"
                className={[
                  'rounded-full px-1.5 py-0.5 text-xs font-medium tabular-nums',
                  dipilih ? 'bg-blue-100 text-blue-800' : 'bg-slate-100 text-slate-600',
                ].join(' ')}
              >
                {jumlah}
              </span>
            )}
          </button>
        )
      })}
    </div>
  )
}
