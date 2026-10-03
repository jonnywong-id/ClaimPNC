import { Fragment, useState, type ReactNode } from 'react'

import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { useDocuments, useProgressRecords, useSurveys, useUploadDocument } from './api'

/*
 * Tiga tab pendamping tahap Input Estimasi. Sub-section aslinya — TabSurvey,
 * UploadDocument, PNCProgressKomunikasi_Sec — tidak ada di export, sehingga susunannya
 * mengikuti section tampilan yang ada: ViewHasilSurvey, ViewUploadDocument,
 * ViewShowKomunikasi.
 *
 * Tombol Unggah Dokumen sudah berjalan (layanan penyimpanan + DATA_ATTACHFILE). Tombol
 * lain yang menulis (ajukan survey, input progres, kirim dan jawab pesan) tampil tetapi
 * mati sampai modulnya dibangun.
 */

const NOT_BUILT = 'Modul ini belum dibangun.'

/** formatMoment menampilkan waktu RFC 3339 (WIB) sebagai `1 Juni 2026 14:05`. */
function formatMoment(iso: string): string {
  if (!iso) return '—'
  const [date = '', rest = ''] = iso.split('T')
  const time = rest.slice(0, 5)
  return time ? `${formatDate(date)} ${time}` : formatDate(date)
}

function Table({ caption, head, empty, children }: { caption: string; head: string[]; empty: boolean; children: ReactNode }) {
  return (
    <table className="mt-2 w-full border-collapse text-sm">
      <caption className="mb-1 text-left text-xs font-semibold text-slate-700">{caption}</caption>
      <thead>
        <tr className="bg-slate-100 text-left text-xs text-slate-700">
          {head.map((h) => (
            <th key={h} className="p-2">
              {h}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {empty ? (
          <tr>
            <td colSpan={head.length} className="p-2 text-xs text-slate-500">
              Data Tidak Ada
            </td>
          </tr>
        ) : (
          children
        )}
      </tbody>
    </table>
  )
}

function Loading({ state }: { state: { isPending: boolean; error: unknown } }) {
  if (state.error) {
    const description = state.error instanceof Error ? state.error.message : 'Terjadi kesalahan pada sistem.'
    return (
      <div className="mt-3">
        <ErrorMessage title="Data tidak dapat dimuat" description={description} tone="gangguan" />
      </div>
    )
  }
  if (state.isPending) return <p className="mt-3 text-sm text-slate-500">Memuat…</p>
  return null
}

const surveyTypeName: Record<string, string> = { '1': 'Survey Internal', '2': 'Loss Adjuster' }

/**
 * Tab Investigasi (PA) — `Section/TabInvestigasi-sect.xml`. Sub-tab "Input Investigasi" bersyarat
 * `1=2` sehingga tidak pernah tampil; yang tampil "Hasil investigasi" = `ViewHasilSurvey`, grid
 * berkondisi `isPA_PNC` atas `ClaimData.SurveyResults` (T_SURVEYORLIST, sumber yang sama dengan
 * tab Survey).
 */
export function InvestigationTab({ claimID }: { claimID: string }) {
  const q = useSurveys(claimID)
  const list = q.data?.survey ?? []
  return (
    <div className="mt-3">
      <div role="tablist" aria-label="Sub-tab Investigasi" className="flex gap-1 border-b border-slate-200">
        <button type="button" role="tab" aria-selected className="border-b-2 border-blue-600 px-3 py-1.5 text-xs font-semibold text-slate-900">
          Hasil investigasi
        </button>
      </div>
      <Loading state={q} />
      {q.data && (
        <Table caption="Hasil investigasi" head={['Tanggal Investigasi', 'Nama Peserta', 'Lokasi Objek', 'Status']} empty={list.length === 0}>
          {list.map((s) => (
            <tr key={`${s.kasus_id}-${s.urutan}`} className="border-b border-slate-100 align-top">
              <td className="p-2">{formatDate(s.tanggal_survey)}</td>
              <td className="p-2">{s.nama_objek || '—'}</td>
              <td className="p-2">{s.lokasi_objek || '—'}</td>
              <td className="p-2">{s.status || '—'}</td>
            </tr>
          ))}
        </Table>
      )}
    </div>
  )
}

export function SurveyTab({ claimID }: { claimID: string }) {
  const q = useSurveys(claimID)
  const list = q.data?.survey ?? []
  return (
    <div className="mt-3">
      <button type="button" disabled title={NOT_BUILT} className="rounded border border-slate-300 px-3 py-1 text-sm disabled:opacity-60">
        Ajukan Survey
      </button>
      <Loading state={q} />
      {q.data && (
        <Table
          caption="Hasil Survey"
          head={['No. Survey', 'Jenis', 'Nama Objek', 'Lokasi Objek', 'Tanggal Survey', 'Nama Surveyor', 'Status', 'Keterangan']}
          empty={list.length === 0}
        >
          {list.map((s) => (
            <tr key={`${s.kasus_id}-${s.urutan}`} className="border-b border-slate-100 align-top">
              <td className="p-2">{s.kasus_id.replace(/^ASM-FW-GCNMFW-WORK /, '')}</td>
              <td className="p-2">{surveyTypeName[s.tipe] ?? (s.tipe || '—')}</td>
              <td className="p-2">{s.nama_objek || '—'}</td>
              <td className="p-2">{s.lokasi_objek || s.lokasi_survey || '—'}</td>
              <td className="p-2">{formatDate(s.tanggal_survey)}</td>
              <td className="p-2">{s.nama_surveyor || '—'}</td>
              <td className="p-2">{s.status || '—'}</td>
              <td className="p-2">{s.keterangan || '—'}</td>
            </tr>
          ))}
        </Table>
      )}
    </div>
  )
}

/** Ekstensi yang dikenali layanan penyimpanan (`dokumenpenunjang.TipeMedia`). */
const ACCEPTED_FILES = '.png,.jpg,.jpeg,.avif,.txt,.doc,.docx,.pdf,.eml,.rar,.zip,.csv,.xls,.xlsx,.ppt,.pptx'

/**
 * UploadPanel adalah isian unggah satu baris checklist — pengganti section `UploadDocument`
 * yang tidak ada di export. Berkasnya ke layanan penyimpanan, barisnya ke DATA_ATTACHFILE.
 */
function UploadPanel({ claimID, typeID, typeName, onDone }: { claimID: string; typeID: string; typeName: string; onDone: () => void }) {
  const upload = useUploadDocument(claimID)
  const [file, setFile] = useState<File | null>(null)
  const [note, setNote] = useState('')
  const message = upload.error instanceof Error ? upload.error.message : ''

  return (
    <div className="space-y-2 rounded border border-blue-200 bg-blue-50/60 p-3 text-sm">
      <p className="font-semibold text-slate-800">Unggah Dokumen: {typeName}</p>
      <label className="block">
        <span className="text-xs text-slate-600">Berkas</span>
        <input
          type="file"
          accept={ACCEPTED_FILES}
          aria-label="Berkas"
          className="mt-1 block w-full text-sm"
          onChange={(e) => setFile(e.target.files?.[0] ?? null)}
        />
      </label>
      <label className="block">
        <span className="text-xs text-slate-600">Catatan</span>
        <input
          type="text"
          aria-label="Catatan"
          maxLength={255}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          className="mt-1 block w-full rounded border border-slate-300 px-2 py-1"
        />
      </label>
      {message && <ErrorMessage title="Dokumen tidak dapat diunggah" description={message} tone="gangguan" />}
      <div className="flex gap-2">
        <button
          type="button"
          disabled={!file || upload.isPending}
          onClick={() => file && upload.mutate({ jenisDokumen: typeID, berkas: file, catatan: note }, { onSuccess: onDone })}
          className="rounded bg-blue-700 px-3 py-1 text-white disabled:opacity-60"
        >
          {upload.isPending ? 'Mengunggah…' : 'Unggah'}
        </button>
        <button type="button" onClick={onDone} disabled={upload.isPending} className="rounded border border-slate-300 px-3 py-1">
          Batal
        </button>
      </div>
    </div>
  )
}

/** Group Panel Personal Accident dan Travel — kondisi sub-tab InputRegister-sect.xml. */
const LINE_PA = '002'
const LINE_TRAVEL = '005'

/**
 * Sub-tab Unggah Dokumen menurut `Section/InputRegister-sect.xml`:
 *
 *   PENDAFTARAN · SURVEI · DOKUMEN LAIN-LAIN · PEMBAYARAN · KOMITE   !IsTravel
 *   DOKUMEN TRAVEL                                                IsTravel
 *   SALVAGE                                                       !IsTravelPA
 *
 * DOKUMEN TRAVEL disusun Pega per item jaminan dari Report Definition
 * `BrowseDocumentTravel_Rd`, yang tidak ada di export; tabnya tampil dengan keterangan itu.
 */
function visibleCategory(kode: string, line: string): boolean {
  if (line === LINE_TRAVEL) return false
  if (kode === 'SALVAGE') return line !== LINE_PA
  return true
}

const TRAVEL_TAB = 'DOKUMEN TRAVEL'

export function DocumentTab({ claimID, line = '' }: { claimID: string; line?: string }) {
  const q = useDocuments(claimID)
  const [open, setOpen] = useState<string | null>(null)
  const [viewing, setViewing] = useState<string | null>(null)
  const categories = (q.data?.kategori ?? []).filter((c) => visibleCategory(c.kode, line))
  const tabs = line === LINE_TRAVEL ? [TRAVEL_TAB] : categories.map((c) => c.nama)
  const [picked, setPicked] = useState<string | null>(null)
  const current = picked && tabs.includes(picked) ? picked : tabs[0] ?? ''
  const category = categories.find((c) => c.nama === current)
  const files = q.data?.berkas ?? []

  return (
    <div className="mt-3">
      <Loading state={q} />
      {q.data && (
        <div role="tablist" aria-label="Kategori dokumen" className="flex flex-wrap gap-1 border-b border-slate-200">
          {tabs.map((t) => (
            <button
              key={t}
              type="button"
              role="tab"
              aria-selected={current === t}
              onClick={() => setPicked(t)}
              className={['px-3 py-1.5 text-xs', current === t ? 'border-b-2 border-blue-600 font-semibold text-slate-900' : 'text-slate-600'].join(' ')}
            >
              {t}
            </button>
          ))}
        </div>
      )}
      {q.data && current === TRAVEL_TAB && (
        <p className="mt-3 rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900" role="status">
          Travel documents are listed per coverage item by the Report Definition BrowseDocumentTravel_Rd, which is not in the Pega export.
        </p>
      )}
      {category && (
        <Table
          caption={category.nama}
          head={['Kategori', 'Wajib Unggah', 'Minimal Unggah', 'Unggah Dokumen', 'Lihat dokumen', 'Ubah Kategori Dok']}
          empty={category.dokumen.length === 0}
        >
          {category.dokumen.map((d) => {
            const key = `${category.kode}:${d.id}`
            const own = files.filter((b) => b.sub_kategori === d.id)
            return (
              <Fragment key={d.id}>
                <tr className="border-b border-slate-100">
                  <td className="p-2">{d.nama}</td>
                  <td className="p-2">{d.wajib ? 'Ya' : 'Tidak'}</td>
                  <td className="p-2">{d.minimal || '0'}</td>
                  <td className="p-2">
                    <button
                      type="button"
                      onClick={() => setOpen(open === key ? null : key)}
                      className="rounded border border-blue-300 px-2 text-blue-700"
                    >
                      Unggah Dokumen
                    </button>
                  </td>
                  <td className="p-2">
                    <button
                      type="button"
                      onClick={() => setViewing(viewing === key ? null : key)}
                      className="rounded border border-blue-300 px-2 text-blue-700"
                    >
                      Lihat dokumen ({d.terunggah})
                    </button>
                  </td>
                  <td className="p-2">
                    <button type="button" disabled title={NOT_BUILT} className="rounded border border-slate-300 px-2 text-slate-600 disabled:opacity-60">
                      Ubah Kategori Dok
                    </button>
                  </td>
                </tr>
                {open === key && (
                  <tr>
                    <td colSpan={6} className="p-2">
                      <UploadPanel claimID={claimID} typeID={d.id} typeName={d.nama} onDone={() => setOpen(null)} />
                    </td>
                  </tr>
                )}
                {viewing === key && (
                  <tr>
                    <td colSpan={6} className="p-2 text-xs text-slate-700">
                      {own.length === 0
                        ? 'Belum ada berkas untuk jenis dokumen ini.'
                        : own.map((b) => `${b.nama || '(tanpa nama)'} · ${b.diunggah_oleh || '—'} · ${formatMoment(b.diunggah_pada)}`).join(' | ')}
                    </td>
                  </tr>
                )}
              </Fragment>
            )
          })}
        </Table>
      )}
      {q.data && (
        <Table caption="Berkas yang sudah diunggah" head={['Nama Berkas', 'Jenis', 'Catatan', 'Diunggah Oleh', 'Diunggah Pada']} empty={q.data.berkas.length === 0}>
          {q.data.berkas.map((b) => (
            <tr key={b.id} className="border-b border-slate-100">
              <td className="p-2">{b.nama || '(tanpa nama)'}</td>
              <td className="p-2">{b.jenis_berkas || '—'}</td>
              <td className="p-2">{b.catatan || '—'}</td>
              <td className="p-2">{b.diunggah_oleh || '—'}</td>
              <td className="p-2">{formatMoment(b.diunggah_pada)}</td>
            </tr>
          ))}
        </Table>
      )}
    </div>
  )
}

export function ProgressTab({ claimID }: { claimID: string }) {
  const q = useProgressRecords(claimID)
  return (
    <div className="mt-3">
      <div className="flex gap-2">
        <button type="button" disabled title={NOT_BUILT} className="rounded border border-slate-300 px-3 py-1 text-sm disabled:opacity-60">
          Input Progress Claim
        </button>
        <button type="button" disabled title={NOT_BUILT} className="rounded border border-slate-300 px-3 py-1 text-sm disabled:opacity-60">
          Kirim Pesan
        </button>
      </div>
      <Loading state={q} />
      {q.data && (
        <>
          <Table
            caption="Progress Claim"
            head={['Tanggal Input', 'Status Progress', 'Detail Progress', 'Keterangan', 'Follow Up Berikutnya', 'User']}
            empty={q.data.progres.length === 0}
          >
            {q.data.progres.map((p) => (
              <tr key={p.urutan} className="border-b border-slate-100 align-top">
                <td className="p-2">{formatMoment(p.tanggal_input)}</td>
                <td className="p-2">{p.status_1_nama || p.status_1 || '—'}</td>
                <td className="p-2">{p.status_2_nama || p.status_2 || '—'}</td>
                <td className="p-2">{p.keterangan || '—'}</td>
                <td className="p-2">{formatMoment(p.tindak_lanjut)}</td>
                <td className="p-2">{p.diinput_oleh || '—'}</td>
              </tr>
            ))}
          </Table>
          <Table caption="Komunikasi" head={['Case ID', 'Tanggal', 'Pengirim', 'Pesan', 'Jawaban', 'Dijawab Oleh']} empty={q.data.komunikasi.length === 0}>
            {q.data.komunikasi.map((k) => (
              <tr key={`${k.kasus_id}-${k.id}`} className="border-b border-slate-100 align-top">
                <td className="p-2">{k.kasus_id.replace(/^ASM-FW-GCNMFW-WORK /, '')}</td>
                <td className="p-2">{formatMoment(k.tanggal)}</td>
                <td className="p-2">{k.pengirim || '—'}</td>
                <td className="p-2 whitespace-pre-wrap">{k.pesan || '—'}</td>
                <td className="p-2 whitespace-pre-wrap">{k.balasan || '—'}</td>
                <td className="p-2">{k.penjawab || '—'}</td>
              </tr>
            ))}
          </Table>
        </>
      )}
    </div>
  )
}
