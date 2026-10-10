import { useEffect, useState } from 'react'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { FormField } from '@/components/FormField'
import { TextAreaField } from '@/components/TextAreaField'

import { usePUCLReasons, usePUCLSubjects, useSendToRCLPUCL, violationsFrom } from './api'

/**
 * Tiga jalur penanganan — `.ClaimData.PUCLStatus.RCL_PUCL`.
 *
 * Kodenya terbukti dari `Activity/PUCLPost-Act.xml`: tiga langkah menetapkan kategori
 * surat, dan prekondisi tiap langkah menyebut kodenya — `==2` kategori "PUCL", `==1`
 * kategori "RCL", `==3` kategori "Notification".
 */
export const TRACK_RCL = 1
export const TRACK_PUCL = 2
export const TRACK_NOTIFICATION = 3

/** Group Panel Personal Accident — When `IsPA`. */
const PANEL_PA = '002'

/** Panjang kolom POOLDATA.TC_PNC_PUCL yang menampung isiannya. */
const MAX_TEXT = 4000

/**
 * Pilihan dropdown "Nama Dokter" — `pyPromptTableList` property `NamaDokterRCL`.
 *
 * Daftar TETAP dua baris, disalin apa adanya dari property yang diserahkan Work Owner,
 * dalam urutan aslinya (`REPEATINGINDEX` 1 lalu 2) — bukan diurutkan abjad, karena itulah
 * urutan yang dilihat petugas hari ini.
 *
 * # Kenapa di sini, bukan diambil dari server
 *
 * Di Pega ia bukan data melainkan bagian dari definisi property-nya: tidak ada kueri, tidak
 * ada tabel, tidak ada yang dapat gagal. Menjadikannya panggilan jaringan menambah keadaan
 * memuat dan keadaan gagal pada sesuatu yang tidak punya keduanya — dan keadaan gagal itu
 * yang benar-benar terjadi: versi sebelumnya menariknya dari `POOLDATA.T_ACCESS_GROUP_PNC`
 * (dugaan, karena rule sumbernya hilang dari export — `R-16`), kuerinya tidak mengembalikan
 * satu baris pun, dan dropdown-nya kosong di layar.
 *
 * `id` yang disimpan (`pyStandardValue`), `nama` yang digambar (`pyLocalizedValue`). Pada
 * baris kedua keduanya BERBEDA, dan tidak boleh tertukar: yang tersimpan ke
 * `NAMADOKTERRCL_1` harus bentuk tanpa spasi, karena itulah yang dicocokkan penyaring
 * Inbox RCL. Tertukar, klaimnya hilang dari semua inbox tanpa satu pesan galat.
 *
 * Kedua nama ini hardcode di dalam rule Pega — bagian dari 24 Operator ID yang `D-15`
 * tetapkan menjadi master data (`F-4`), yang belum ada. Work Owner mengonfirmasi keduanya
 * masih berlaku (2026-10-06). Menuliskan nama Operator ID lengkap diizinkan `D-69`.
 */
const RCL_DOCTORS = [
  { id: 'WAHYUKRISTANTI', nama: 'WAHYUKRISTANTI' },
  { id: 'MARGARETHAROSAGUNAWAN', nama: 'MARGARETHA ROSA GUNAWAN' },
]

/**
 * Isian "Nama Dokter" — `pyVisible OTHER`, kondisi
 * `.ClaimData.PUCLStatus.RCL_PUCL != 2 && IsPA` (`SectionPUCL-sect.xml:2149`).
 *
 * `!= 2` berarti RCL **atau** Notification, bukan RCL saja. Jalur `0` (belum memilih)
 * secara harfiah juga memenuhinya, dan itu memang perilaku Pega; di sini isiannya baru
 * tampil setelah salah satu jalur dipilih, karena menampilkannya sebelum pengguna
 * memilih apa pun hanya menimbulkan pertanyaan.
 */
export function showDoctorName(track: number, groupPanel: string): boolean {
  if (groupPanel !== PANEL_PA) return false
  return track === TRACK_RCL || track === TRACK_NOTIFICATION
}

/**
 * Grid alasan penolakan — kontainer `IsPA && .ClaimData.PUCLStatus.RCL_PUCL != 3`
 * (`SectionPUCL-sect.xml:2342`), yang membungkus sel ke-8 berisi `BrowseReasonReject_RD`.
 *
 * Dua batas yang mudah terlewat, dan keduanya memang ada di section: grid ini **hanya
 * untuk lini PA**, dan **tidak tampil pada jalur Notification** — surat pemberitahuan
 * bukan surat penolakan, sehingga daftar alasan penolakan tidak berlaku padanya.
 */
export function showRejectReasons(track: number, groupPanel: string): boolean {
  return groupPanel === PANEL_PA && track !== TRACK_NOTIFICATION
}

type Props = {
  claimID: string
  taskID: string
  /** Group Panel polis — penentu tampilnya "Nama Dokter". */
  groupPanel: string
  onClose: () => void
}

/**
 * Modal **"Kirim ke RCL/PUCL"** — local action `KomentarRCLPUCL`
 * (`Flow Action/KomentarRCLPUCL-FA.xml`) di atas `Section/SectionPUCL-sect.xml`.
 *
 * # Susunannya mengikuti section, sel demi sel
 *
 *	1  Pilih RCL / PUCL      pxRadioButtons   WAJIB
 *	2  Catatan untuk RCL/PUCL pxTextArea      WAJIB
 *	3  Perihal               pxTextInput      dari master M_PERIHAL_RCLPUCL
 *	4  Keterangan Pembuka    pxTextArea
 *	5  Keterangan Isi        pxTextArea       WAJIB
 *	6  Keterangan Penutup    pxTextArea
 *	7  Nama Dokter           pxDropdown       `RCL_PUCL != 2 && IsPA`
 *	8  grid alasan + "Pilih"  BrowseReasonReject_RD   `IsPA && RCL_PUCL != 3`
 *	9  Kirim · Batal
 *
 * # Empat artefak pendukungnya TIDAK ADA di export (`R-16`)
 *
 * `isiDefaultPUCLRCL` (nilai bawaan saat jalur berubah), `InputKeteranganIsi` (tombol
 * Pilih), `BrowseReasonReject_RD`, dan `BrowsePerihalRCLPUCL_RD`. Yang terbaca hanyalah
 * NAMA keempatnya dan tabel yang mereka baca, sehingga perilakunya di sini diturunkan
 * dari tabelnya langsung. Dua penyimpulan yang perlu diketahui pembaca:
 *
 *   - **Tombol "Pilih" menyalin Reason Description ke Keterangan Isi.** Nama
 *     activity-nya `InputKeteranganIsi` — "isi Keterangan Isi" — dan satu-satunya kolom
 *     grid yang berupa kalimat surat adalah `REASON_DESC`. Isinya memang kalimat utuh
 *     yang siap menjadi badan surat.
 *   - **Nilai bawaan saat jalur berubah tidak ditiru.** `isiDefaultPUCLRCL` hilang, dan
 *     menebak isi ketiga keterangan akan menaruh kalimat karangan ke dalam surat yang
 *     dikirim ke tertanggung. Isiannya dibiarkan kosong untuk diisi analis.
 *
 * # Yang BELUM dibangun dari section ini
 *
 * Tombol **"Download Surat"** (sel 9, berkondisi `IsTravel`) tidak dibawa: ia membentuk
 * dokumen lewat `PUCLPost` yang juga menulis `TGL_CETAK_DOKUMEN_PUCL` — langkah yang di
 * layar Inbox RCL/PUCL dipegang tombol "Download Dokumen", dan membangunnya dua kali di
 * dua tempat akan membuat baris berpindah tab tanpa suratnya benar-benar tercetak.
 */
export function SendToRCLPUCLDialog({ claimID, taskID, groupPanel, onClose }: Props) {
  const send = useSendToRCLPUCL(claimID)
  const busy = send.isPending

  const [track, setTrack] = useState(0)
  const [note, setNote] = useState('')
  const [subject, setSubject] = useState('')
  const [opening, setOpening] = useState('')
  const [body, setBody] = useState('')
  const [closing, setClosing] = useState('')
  const [doctor, setDoctor] = useState('')
  const [search, setSearch] = useState('')
  // Isian wajib baru ditandai merah setelah Kirim ditekan sekali — menandai form yang
  // belum tersentuh sebagai salah adalah menuduh pengguna sebelum ia berbuat apa pun.
  const [attempted, setAttempted] = useState(false)

  const subjects = usePUCLSubjects(track)
  const withDoctor = showDoctorName(track, groupPanel)
  const withReasons = showRejectReasons(track, groupPanel)
  const reasons = usePUCLReasons(search, withReasons)

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape' && !busy) onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose, busy])

  // Pilihan Perihal menyempit menurut jalur, sehingga yang sudah terpilih dapat menjadi
  // tidak berlaku lagi. Mengosongkannya lebih baik daripada mengirim Perihal "Tolakan…"
  // pada surat permintaan kelengkapan dokumen.
  useEffect(() => {
    setSubject('')
  }, [track])

  const violations = violationsFrom(send.error)
  const failureOf = (field: string) => violations.find((v) => v.field === field)?.pesan
  const tooLong = (value: string) => value.trim().length > MAX_TEXT

  const missingTrack = attempted && track === 0
  const missingNote = attempted && note.trim() === ''
  const missingBody = attempted && body.trim() === ''

  function submit() {
    setAttempted(true)
    if (track === 0 || note.trim() === '' || body.trim() === '') return
    send.reset()
    send.mutate(
      {
        taskID,
        jalur: track,
        catatan: note,
        perihal: subject,
        keterangan_pembuka: opening,
        keterangan_isi: body,
        keterangan_penutup: closing,
        nama_dokter: withDoctor ? doctor : '',
      },
      { onSuccess: onClose },
    )
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-kirim-rclpucl"
    >
      <div className="max-h-full w-full max-w-3xl overflow-y-auto rounded-kartu bg-white p-6 shadow-angkat">
        <h2 id="judul-kirim-rclpucl" className="text-lg font-semibold text-slate-900">
          Kirim ke RCL/PUCL
        </h2>
        <p className="mt-1 text-sm text-slate-600">
          RCL mengirim klaim ke Inbox RCL lebih dulu; PUCL dan Notification langsung ke antrean
          Inbox RCL/PUCL.
        </p>

        {/* 1 — Pilih RCL / PUCL */}
        <fieldset className="mt-4" aria-describedby={missingTrack ? 'galat-jalur' : undefined}>
          <legend className="text-sm font-medium text-slate-700">Pilih RCL / PUCL</legend>
          <div className="mt-2 flex gap-6">
            {[
              { value: TRACK_RCL, label: 'RCL' },
              { value: TRACK_PUCL, label: 'PUCL' },
              { value: TRACK_NOTIFICATION, label: 'Notification' },
            ].map((o) => (
              <label key={o.value} className="flex items-center gap-2 text-sm text-slate-800">
                <input
                  type="radio"
                  name="jalur-rclpucl"
                  value={o.value}
                  checked={track === o.value}
                  disabled={busy}
                  onChange={() => setTrack(o.value)}
                />
                {o.label}
              </label>
            ))}
          </div>
          {missingTrack && (
            <p id="galat-jalur" className="mt-1 text-sm text-red-700">
              Pilih RCL, PUCL, atau Notification lebih dulu.
            </p>
          )}
        </fieldset>

        {/* 2 — Catatan untuk RCL/PUCL */}
        <div className="mt-4">
          <TextAreaField
            id="catatan-rclpucl"
            label="Catatan untuk RCL/PUCL"
            rows={3}
            value={note}
            disabled={busy}
            onChange={(e) => setNote(e.target.value)}
            error={
              missingNote
                ? 'Catatan untuk RCL/PUCL wajib diisi.'
                : tooLong(note)
                  ? `Paling banyak ${MAX_TEXT} karakter.`
                  : failureOf('catatan')
            }
          />
        </div>

        {/* 3 — Perihal, dari master M_PERIHAL_RCLPUCL */}
        <div className="mt-4">
          <label htmlFor="perihal-rclpucl" className="block text-sm font-medium text-slate-700">
            Perihal
          </label>
          <select
            id="perihal-rclpucl"
            className="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-slate-900 focus:border-slate-500 focus:outline-none"
            value={subject}
            disabled={busy || track === 0}
            onChange={(e) => setSubject(e.target.value)}
          >
            <option value="">{track === 0 ? '— pilih jalur lebih dulu —' : '----- PILIH -----'}</option>
            {(subjects.data?.pilihan ?? []).map((o) => (
              <option key={o.id} value={o.nama}>
                {o.nama}
              </option>
            ))}
          </select>
          {/* Yang tersimpan adalah TEKSNYA, bukan kodenya — `InputPerihalRCLPUCL_act`
              menyalin PERIHAL_NAME ke properti klaim. */}
        </div>

        {/* 4, 5, 6 — ketiga keterangan surat */}
        <div className="mt-4">
          <TextAreaField
            id="keterangan-pembuka"
            label="Keterangan Pembuka"
            rows={2}
            value={opening}
            disabled={busy}
            onChange={(e) => setOpening(e.target.value)}
            error={tooLong(opening) ? `Paling banyak ${MAX_TEXT} karakter.` : failureOf('keterangan_pembuka')}
          />
        </div>
        <div className="mt-4">
          <TextAreaField
            id="keterangan-isi"
            label="Keterangan Isi"
            rows={5}
            value={body}
            disabled={busy}
            onChange={(e) => setBody(e.target.value)}
            error={
              missingBody
                ? 'Keterangan Isi wajib diisi.'
                : tooLong(body)
                  ? `Paling banyak ${MAX_TEXT} karakter.`
                  : failureOf('keterangan_isi')
            }
            hint={withReasons ? 'Tombol Pilih pada daftar alasan di bawah mengisi kolom ini.' : undefined}
          />
        </div>
        <div className="mt-4">
          <TextAreaField
            id="keterangan-penutup"
            label="Keterangan Penutup"
            rows={2}
            value={closing}
            disabled={busy}
            onChange={(e) => setClosing(e.target.value)}
            error={tooLong(closing) ? `Paling banyak ${MAX_TEXT} karakter.` : failureOf('keterangan_penutup')}
          />
        </div>

        {/* 7 — Nama Dokter: `RCL_PUCL != 2 && IsPA`

            DROPDOWN, bukan isian bebas — `pxDropdown` pada sectionnya. Isinya `RCL_DOCTORS`
            di atas: `pyPromptTableList` property `NamaDokterRCL`, disalin apa adanya.

            Digambar langsung dari daftar itu, TANPA panggilan jaringan. Tidak ada keadaan
            memuat dan tidak ada keadaan gagal, karena di Pega pun ia bukan data melainkan
            bagian dari definisi property-nya.

            Dibiarkan "----- PILIH -----", penerimanya ditentukan penugasan tahap
            RCLDokter, persis seperti lini selain PA yang tidak menampilkan isian ini. */}
        {withDoctor && (
          <div className="mt-4">
            <label htmlFor="nama-dokter" className="block text-sm font-medium text-slate-700">
              Nama Dokter
            </label>
            <select
              id="nama-dokter"
              className="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-slate-900 focus:border-slate-500 focus:outline-none"
              value={doctor}
              disabled={busy}
              onChange={(e) => setDoctor(e.target.value)}
            >
              <option value="">----- PILIH -----</option>
              {RCL_DOCTORS.map((o) => (
                <option key={o.id} value={o.id}>
                  {o.nama}
                </option>
              ))}
            </select>
            {failureOf('nama_dokter') && (
              <p className="mt-1 text-sm text-red-700">{failureOf('nama_dokter')}</p>
            )}
            <p className="mt-1 text-xs text-slate-500">
              Dokter yang dipilih inilah yang melihat klaimnya di Inbox RCL. Dibiarkan kosong,
              klaim masuk ke inbox petugas yang ditunjuk penugasan.
            </p>
          </div>
        )}

        {/* 8 — grid alasan penolakan: `IsPA && RCL_PUCL != 3` */}
        {withReasons && (
        <div className="mt-6">
          <div className="flex items-end justify-between gap-3">
            <h3 className="text-sm font-semibold text-slate-800">Daftar Alasan</h3>
            <div className="w-64">
              <FormField
                id="cari-alasan"
                label="Cari alasan"
                value={search}
                disabled={busy}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
          </div>

          <div className="mt-2 max-h-56 overflow-y-auto rounded border border-slate-200">
            <table className="w-full border-collapse text-sm">
              <caption className="sr-only">Alasan penolakan</caption>
              <thead className="sticky top-0 bg-slate-100 text-left text-xs text-slate-700">
                <tr>
                  <th className="p-2">ID</th>
                  <th className="p-2">Reason Name</th>
                  <th className="p-2">Reason Description</th>
                  <th className="p-2" />
                </tr>
              </thead>
              <tbody>
                {reasons.isPending && (
                  <tr>
                    <td colSpan={4} className="p-2 text-xs text-slate-500">
                      Memuat…
                    </td>
                  </tr>
                )}
                {!reasons.isPending && (reasons.data?.pilihan ?? []).length === 0 && (
                  <tr>
                    <td colSpan={4} className="p-2 text-xs text-slate-500">
                      Data Tidak Ada
                    </td>
                  </tr>
                )}
                {(reasons.data?.pilihan ?? []).map((o) => (
                  <tr key={o.id} className="border-b border-slate-100 align-top">
                    <td className="p-2">{o.id}</td>
                    <td className="p-2">{o.nama}</td>
                    <td className="p-2 text-xs text-slate-600">{o.deskripsi}</td>
                    <td className="p-2">
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() => setBody(o.deskripsi)}
                        className="rounded border border-blue-300 px-2 py-1 text-xs text-blue-700 hover:bg-blue-50 disabled:opacity-60"
                      >
                        Pilih
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
        )}

        {send.isError && (
          <div className="mt-4">
            <ErrorMessage
              tone="penolakan"
              title="Klaim belum terkirim ke RCL/PUCL"
              description={
                violations.length > 0
                  ? violations.map((v) => v.pesan).join(' ')
                  : send.error instanceof Error
                    ? send.error.message
                    : 'Terjadi kesalahan.'
              }
            />
          </div>
        )}

        {/* 9 — Kirim dan Batal */}
        <div className="mt-6 flex justify-end gap-3">
          <Button tone="halus" disabled={busy} onClick={onClose}>
            Batal
          </Button>
          <Button tone="utama" disabled={busy} onClick={submit}>
            {busy ? 'Mengirim…' : 'Kirim'}
          </Button>
        </div>
      </div>
    </div>
  )
}
