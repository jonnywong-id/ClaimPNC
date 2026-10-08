import { type ReactNode, useEffect, useRef, useState } from 'react'

import { APIError, simpanBerkas } from '@/api/client'

import { DateInput } from '@/components/DateField'
import { centsToRupiah, formatDateTimeWIB, rupiahToCents } from '@/components/format'
import { useEscapeToClose } from '@/components/shared/useEscapeToClose'

import { type LODType, useAcceptSettlement, useAcceptanceDefaults, useDocuments, useLODTypes, usePrintLOD, violationsFrom } from './api'
import { DLADialog } from './DLADialog'
import { PaymentType, type Receiver, type Settlement } from './types'

/**
 * Tombol kolom Akseptasi grid Adjustment — Print LOD, Persetujuan / Akseptasi, Print DLA
 * (`Section/ShowAdjustment_sect.xml`).
 *
 * # Kapan tampil dan kapan mati — disalin dari section apa adanya
 *
 * Ketiganya tampil hanya bila `.AcceptanceStatus == 1` (adjustment disetujui komite).
 *
 *   Print LOD                mati bila `.AcceptationStatusLOD != ''` · Bonding / BondingKBG ·
 *                            Group Panel 002 / 005 · Type 3 / 4        (local action PrintLODUP)
 *   Persetujuan / Akseptasi  mati bila `.AcceptationStatusLOD == '0'` · `.AcceptedNo != ''` ·
 *                            `.AcceptanceStatus == '2'` · `.IsKomiteSetuju == '0'`
 *                                                                      (local action AcceptationLOD)
 *   Print DLA                mati bila `.AcceptationStatusLOD != '1'` · `.AcceptanceStatus == '2'`
 *                                                                      (local action PrintDLA)
 *
 * `.IsKomiteSetuju` tidak disertakan: ia hanya hidup di clipboard (diisi
 * `ProteksiKomiteSetuju` dengan syarat yang tidak terbaca) dan tidak punya kolom.
 *
 * # Prosesnya — keputusan Work Owner 2026-09-29
 *
 * - **Print LOD**: dialog "Email LOD" (flow action `PrintLODUP`, section `PrintLODdanEmail`)
 *   dengan Tipe PDF (`SetTypePDFAdjustment`). Hanya jenis yang ada contoh PDF-nya yang dapat
 *   dicetak. Send Email menunggu (keputusan 2026-09-29): activity `DownloadProposeAdjustment`
 *   dan badan email `SendAutoLodKeTertanggungPA` tidak ada di export.
 * - **Persetujuan / Akseptasi**: form `AcceptationLOD_Sect`; Simpan menjalankan
 *   `SetAdjustmentAcceptation` di backend (usecase AcceptSettlement). PA menunggu Transfer Kasir.
 * - **Print DLA**: hanya dapat dilakukan setelah akseptasi (keputusan Work Owner), karena daftar
 *   DLA diterbitkan saat nomor akseptasi terbit — ikut menunggu proses akseptasi.
 *
 * Transfer Kasir TIDAK di sini: di Pega tombolnya ada di detail adjustment, di samping Nomor
 * Akseptasi (`InputAdjustment_sect`) — lihat AcceptanceNumber.
 *
 * Teks tambahan di luar Pega berbahasa Inggris (`D-80`); label mengikuti Pega.
 */
export function AcceptanceButtons({
  claimID,
  taskID,
  object,
  coverage,
  adjustment,
  line,
  groupPanel,
  businessType,
  receivers,
}: Readonly<{
  claimID: string
  taskID: string
  /** Objek, jaminan, dan adjustment berbasis 1. */
  object: number
  coverage: number
  adjustment: number
  line: Settlement
  groupPanel: string
  businessType: string
  receivers: Receiver[]
}>) {
  const [open, setOpen] = useState<'lod' | 'accept' | 'dla' | null>(null)
  const rules = acceptanceRules(line, groupPanel, businessType)
  if (!rules.visible) return null

  const address = { tugas_id: taskID, objek: object, jaminan: coverage, adjustment }

  return (
    <div className="space-y-1">
      {/* Satu baris, tidak dilipat: Print DLA selalu di kanan Persetujuan / Akseptasi. */}
      <div className="flex flex-nowrap items-center gap-2">
        <ActionButton label="Print LOD" disabled={rules.printLODDisabled} onClick={() => setOpen('lod')} />
        <ActionButton
          label="Persetujuan / Akseptasi"
          disabled={rules.acceptDisabled}
          onClick={() => setOpen('accept')}
        />
        <ActionButton
          label="Print DLA"
          disabled={rules.printDLADisabled}
          onClick={() => setOpen('dla')}
        />
      </div>
      {open === 'lod' && (
        <PrintLODDialog claimID={claimID} address={address} initialType={line.tipe_pdf_lod ?? ''} onClose={() => setOpen(null)} />
      )}
      {open === 'dla' && <DLADialog claimID={claimID} address={address} onClose={() => setOpen(null)} />}
      {open === 'accept' && (
        <AcceptationForm
          claimID={claimID}
          address={address}
          line={line}
          groupPanel={groupPanel}
          receivers={receivers}
          onClose={() => setOpen(null)}
        />
      )}
    </div>
  )
}

function ActionButton({ label, disabled, onClick }: Readonly<{ label: string; disabled: boolean; onClick: () => void }>) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className="whitespace-nowrap rounded border border-blue-400 px-2 py-0.5 text-xs text-blue-700 hover:bg-blue-50 disabled:border-slate-200 disabled:text-slate-400 disabled:hover:bg-transparent"
    >
      {label}
    </button>
  )
}

function failureText(failure: unknown): string {
  const violations = violationsFrom(failure)
  if (violations.length > 0) return violations.map((v) => v.pesan).join(' ')
  if (failure instanceof APIError || failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}

/**
 * Popup adalah lapisan modal kedua form di atas grid Adjustment — pola yang sama dengan
 * PLADialog: latar gelap menutup layar, Escape menutup kecuali sedang memproses.
 */
function Popup({
  title,
  titleID,
  busy,
  onClose,
  children,
  wide = false,
}: Readonly<{
  title: string
  titleID: string
  busy: boolean
  onClose: () => void
  children: ReactNode
  /** Lebar modal AcceptationLOD, yang memuat dua kolom Remark | Berita Acara. */
  wide?: boolean
}>) {
  useEscapeToClose(onClose, busy)

  return (
    <dialog
      open
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      aria-modal="true"
      aria-labelledby={titleID}
    >
      <div className={`max-h-full w-full ${wide ? 'max-w-3xl' : 'max-w-xl'} space-y-3 overflow-y-auto rounded-kartu bg-white p-6 text-xs text-slate-700 shadow-angkat`}>
        <h2 id={titleID} className="text-lg font-semibold text-slate-900">
          {title}
        </h2>
        {children}
      </div>
    </dialog>
  )
}

/**
 * Dialog local action PrintLODUP ("Email LOD"), section `PrintLODdanEmail`: Email LOD,
 * Nama Tertanggung (bukan PA), dan Catatan — diisi awal seperti `SetDataEmailTertanggung`.
 * Ketiganya hanya dipakai Send Email; Print LOD mengunduh PDF menurut Tipe PDF.
 */
function PrintLODDialog({
  claimID,
  address,
  initialType,
  onClose,
}: Readonly<{
  claimID: string
  address: { tugas_id: string; objek: number; jaminan: number; adjustment: number }
  /** Tipe LOD yang dipilih pada dropdown kolom Adjustment (PDFTYPE); kosong bila belum. */
  initialType: string
  onClose: () => void
}>) {
  const types = useLODTypes(claimID)
  const print = usePrintLOD(claimID)
  const [choice, setChoice] = useState(initialType)
  const [email, setEmail] = useState('')
  const [insured, setInsured] = useState('')
  const [note, setNote] = useState('')
  const load = types.mutate
  const { tugas_id, objek, jaminan, adjustment } = address

  useEffect(() => {
    load(
      { tugas_id, objek, jaminan, adjustment },
      {
        onSuccess: (dialog) => {
          setEmail(dialog.email_lod)
          setInsured(dialog.nama_tertanggung)
        },
      },
    )
  }, [load, tugas_id, objek, jaminan, adjustment])
  const list: LODType[] = types.data?.tipe ?? []
  const chosen = list.find((t) => t.id === choice)

  return (
    <Popup title="Email LOD" titleID="judul-email-lod" busy={print.isPending} onClose={onClose}>
      {types.isPending && <p className="text-xs text-slate-500">Memuat…</p>}
      {types.isError && (
        <p role="alert" className="text-xs text-red-700">
          {failureText(types.error)}
        </p>
      )}
      {types.isSuccess && (
        <label className="block text-xs text-slate-700">
          <span>Tipe PDF</span>
          <select
            value={choice}
            onChange={(e) => setChoice(e.target.value)}
            className="mt-1 block w-full rounded border border-slate-300 px-2 py-1 text-sm"
          >
            <option value="">-- Pilih --</option>
            {list.map((t) => (
              <option key={t.id} value={t.id}>
                {t.nama}
              </option>
            ))}
          </select>
        </label>
      )}
      {types.isSuccess && (
        <>
          <label className="block text-xs text-slate-700">
            <span>Email LOD</span>
            <textarea value={email} onChange={(e) => setEmail(e.target.value)} rows={2} className={field} />
          </label>
          {/* Nama Tertanggung tampil bila GroupPanel != 002; Print LOD sendiri mati untuk PA. */}
          <label className="block text-xs text-slate-700">
            <span>Nama Tertanggung</span>
            <textarea value={insured} onChange={(e) => setInsured(e.target.value)} rows={2} className={field} />
          </label>
          <label className="block text-xs text-slate-700">
            <span>Catatan</span>
            <textarea value={note} onChange={(e) => setNote(e.target.value)} rows={2} className={field} />
          </label>
        </>
      )}
      {chosen && !chosen.tersedia && (
        <output className="block text-xs text-amber-700">
          The template for this LOD type is not available yet: no sample PDF has been provided.
        </output>
      )}
      {print.isSuccess && (
        <output className="block text-xs text-green-700">
          LOD downloaded.
        </output>
      )}
      {print.isError && (
        <p role="alert" className="text-xs text-red-700">
          {failureText(print.error)}
        </p>
      )}
      <div className="flex gap-2">
        <button
          type="button"
          disabled={!chosen?.tersedia || print.isPending}
          onClick={() =>
            print.mutate({ ...address, tipe_pdf: choice }, { onSuccess: (file) => simpanBerkas(file) })
          }
          className="rounded bg-blue-600 px-3 py-1 text-xs text-white disabled:opacity-60"
        >
          {print.isPending ? 'Mencetak…' : 'Print LOD'}
        </button>
        <button
          type="button"
          disabled
          title="Sending the LOD email is not available yet: the DownloadProposeAdjustment activity and the email body are not in the Pega export."
          className="rounded bg-blue-600 px-3 py-1 text-xs text-white disabled:opacity-60"
        >
          Send Email
        </button>
        <button type="button" onClick={onClose} className="rounded border border-slate-300 px-3 py-1 text-xs">
          Cancel
        </button>
      </div>
    </Popup>
  )
}

const field = 'mt-1 block w-full rounded border border-slate-300 px-2 py-1 text-sm'

/**
 * Form `Section/AcceptationLOD_Sect.xml` (versi 01-03-13) dalam modal flow action
 * `AcceptationLOD`; Submit menjalankan `SetAdjustmentAcceptation`. Urutan dan label mengikuti
 * section (label = caption `pyLabelFor`) dan tampilan Pega:
 *
 *   Tipe PDF · Tanggal Cetak LOD              baca saja — diisi Print LOD (`.PDFType`,
 *                                             `.PrintDateLOD`)
 *   Tanggal Terima LOD · Tanggal Boleh Bayar  wajib bila bukan Travel (`!IsTravel`)
 *   Persetujuan Tertanggung                   Setuju (1) / Tidak Setuju (0)
 *   Nilai LOD                                 wajib bila IsNonMbu (003 / 004 / 006 / 009)
 *   Penerima Klaim                            dropdown `.Receiver`, wajib bila Setuju
 *   Nama Komite Akseptasi · Remark | Berita Acara · Unggah Dokumen Persetujuan LOD
 *
 * "Remark" (`.RemarkAccepted`) adalah isian yang di section berlabel lapangan "Penerima
 * Klaim" tetapi ber-caption "Remark"; yang tampil di Pega caption-nya. "No Invoice" tidak ada:
 * dihapus pada versi section ini ("remove noinvoice & faktur pajak").
 * Belum dibangun: Tipe Akseptasi Klaim (sumber `TipeAkseptasi.pxResults` tidak ada di export),
 * Remark To Leader (alur Transfer Kasir PA yang menunggu), dan bagian SLIK. Isian awal flow
 * action (Nilai LOD, Nama Komite, Penerima Klaim terpilih) tidak dibuat: pra-prosesnya tidak
 * ada di export.
 */
function AcceptationForm({
  claimID,
  address,
  line,
  groupPanel,
  receivers,
  onClose,
}: Readonly<{
  claimID: string
  address: { tugas_id: string; objek: number; jaminan: number; adjustment: number }
  line: Settlement
  groupPanel: string
  receivers: Receiver[]
  onClose: () => void
}>) {
  const travel = groupPanel === '005'
  // When IsNonMbu: Group Panel 003 / 004 / 006 / 009.
  const nonMBU = ['003', '004', '006', '009'].includes(groupPanel)
  const save = useAcceptSettlement(claimID)
  const documents = useDocuments(claimID)
  const defaults = useAcceptanceDefaults(claimID, address)
  const [form, setForm] = useState({
    tipe: '',
    persetujuan: '',
    tanggalTerimaLOD: '',
    tanggalBolehBayar: '',
    nilaiLOD: '',
    penerima: '',
    komite: '',
    remark: '',
    beritaAcara: '',
  })
  // `id` hanya kunci render yang stabil per baris unggahan; tidak ikut dikirim ke server.
  const [files, setFiles] = useState<{ id: number; berkas: File | null; jenisDokumen: string }[]>([])
  const nextFileID = useRef(0)

  // AcceptationLOD_PreAct: Tipe Akseptasi, Nama Komite Akseptasi, dan Nilai LOD sudah terisi
  // saat form dibuka. Diisi sekali, dan hanya isian yang masih kosong — isian yang sudah
  // diubah petugas tidak ditimpa.
  const initial = defaults.data
  useEffect(() => {
    if (!initial) return
    setForm((f) => ({
      ...f,
      tipe: f.tipe || initial.tipe_akseptasi,
      komite: f.komite || initial.nama_komite_akseptasi,
      nilaiLOD: f.nilaiLOD || (initial.nilai_lod_sen === undefined ? '' : centsToRupiah(initial.nilai_lod_sen)),
    }))
  }, [initial])
  const [invalid, setInvalid] = useState<string | null>(null)
  const set = (key: keyof typeof form) => (e: { target: { value: string } }) =>
    setForm({ ...form, [key]: e.target.value })
  const types = (documents.data?.kategori ?? []).flatMap((k) =>
    k.dokumen.map((d) => ({ id: d.id, nama: `${k.nama} — ${d.nama}` })),
  )

  const submit = () => {
    const cents = form.nilaiLOD.trim() === '' ? undefined : rupiahToCents(form.nilaiLOD)
    if (cents !== undefined && Number.isNaN(cents)) {
      setInvalid('Nilai LOD is not a valid amount.')
      return
    }
    setInvalid(null)
    save.mutate(
      {
        isian: {
          ...address,
          tipe_akseptasi: form.tipe,
          persetujuan_tertanggung: form.persetujuan,
          tanggal_terima_lod: form.tanggalTerimaLOD,
          tanggal_boleh_bayar: form.tanggalBolehBayar,
          ...(cents === undefined ? {} : { nilai_lod_sen: cents }),
          penerima: form.penerima,
          nama_komite_akseptasi: form.komite,
          catatan_penerima: form.remark,
          berita_acara: form.beritaAcara,
        },
        berkas: files.flatMap((f) => (f.berkas ? [{ berkas: f.berkas, jenisDokumen: f.jenisDokumen }] : [])),
      },
      { onSuccess: onClose },
    )
  }

  const caption = 'block text-xs font-semibold text-slate-800'
  const required = <span className="text-amber-600"> *</span>

  return (
    <Popup title="AcceptationLOD" titleID="judul-akseptasi-lod" busy={save.isPending} onClose={onClose} wide>
      <div className="space-y-3">
        {(defaults.data?.peringatan ?? []).map((w) => (
          <p key={w} role="alert" className="rounded border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
            {w}
          </p>
        ))}
        <label className={caption}>
          <span>Tipe Akseptasi Klaim</span>
          <select value={form.tipe} onChange={set('tipe')} className={field}>
            <option value="">-- Pilih --</option>
            {(defaults.data?.pilihan_tipe_akseptasi ?? []).map((o) => (
              <option key={o.id} value={o.id}>
                {o.nama}
              </option>
            ))}
          </select>
        </label>
        <div>
          <span className={caption}>Tipe PDF</span>
          <p className="text-sm text-slate-900">{line.nama_tipe_pdf_lod || '—'}</p>
        </div>
        <div>
          <span className={caption}>Tanggal Cetak LOD</span>
          <p className="text-sm text-slate-900">{formatDateTimeWIB(line.tanggal_cetak_lod) || '—'}</p>
        </div>
        {!travel && (
          <>
            <label className={caption}>
              Tanggal Terima LOD{required}
              <DateInput value={form.tanggalTerimaLOD} onChange={(v) => setForm({ ...form, tanggalTerimaLOD: v })} className={field} />
            </label>
            <label className={caption}>
              Tanggal Boleh Bayar{required}
              <DateInput value={form.tanggalBolehBayar} onChange={(v) => setForm({ ...form, tanggalBolehBayar: v })} className={field} />
            </label>
          </>
        )}
        <fieldset>
          <legend className={caption}>Persetujuan Tertanggung{required}</legend>
          <div className="mt-1 grid grid-cols-2 text-sm">
            {/* Nilai `.AcceptationStatusLOD` di data: 1 dan 0 (STATUSAKSEPTASILOD); label
                mengikuti tampilan Pega — rule property-nya tidak ada di export. */}
            {[
              ['1', 'Setuju'],
              ['0', 'Tidak Setuju'],
            ].map(([value, label]) => (
              <label key={value} className="flex items-center gap-1">
                <input
                  type="radio"
                  name="persetujuan"
                  value={value}
                  checked={form.persetujuan === value}
                  onChange={set('persetujuan')}
                />
                {label}
              </label>
            ))}
          </div>
        </fieldset>
        {nonMBU && (
          <label className={caption}>
            Nilai LOD{required}
            <input inputMode="decimal" value={form.nilaiLOD} onChange={set('nilaiLOD')} className={field} />
          </label>
        )}
        <label className={caption}>
          Penerima Klaim{form.persetujuan === '1' && required}
          <select value={form.penerima} onChange={set('penerima')} className={field}>
            <option value="">-- Pilih --</option>
            {receivers.map((r) => (
              <option key={r.id} value={r.id}>
                {r.nama}
              </option>
            ))}
          </select>
        </label>
        {!travel && (
          <label className={caption}>
            <span>Nama Komite Akseptasi</span>
            <input value={form.komite} onChange={set('komite')} className={field} />
          </label>
        )}
        <div className="grid gap-3 sm:grid-cols-2">
          {!travel && (
            <label className={caption}>
              <span>Remark</span>
              <textarea value={form.remark} onChange={set('remark')} rows={4} className={field} />
            </label>
          )}
          <label className={caption}>
            <span>Berita Acara</span>
            <textarea value={form.beritaAcara} onChange={set('beritaAcara')} rows={4} className={field} />
          </label>
        </div>
        {!travel && (
          <fieldset className="space-y-2">
            <legend className="text-sm font-semibold text-slate-900">Unggah Dokumen Persetujuan LOD</legend>
            {files.map((f, i) => (
              <div key={f.id} className="flex flex-wrap items-center gap-2">
                <select
                  aria-label={`Jenis dokumen ${i + 1}`}
                  value={f.jenisDokumen}
                  onChange={(e) =>
                    setFiles(files.map((x, j) => (j === i ? { ...x, jenisDokumen: e.target.value } : x)))
                  }
                  className="rounded border border-slate-300 px-2 py-1"
                >
                  <option value="">-- Pilih --</option>
                  {types.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.nama}
                    </option>
                  ))}
                </select>
                <input
                  type="file"
                  aria-label={`Berkas ${i + 1}`}
                  onChange={(e) => {
                    const picked = e.target.files?.[0] ?? null
                    setFiles(files.map((x, j) => (j === i ? { ...x, berkas: picked } : x)))
                  }}
                />
                <button type="button" onClick={() => setFiles(files.filter((_, j) => j !== i))} className="text-red-700">
                  Hapus
                </button>
              </div>
            ))}
            <div className="text-center">
              <button
                type="button"
                onClick={() => {
                  nextFileID.current += 1
                  setFiles([...files, { id: nextFileID.current, berkas: null, jenisDokumen: '' }])
                }}
                className="rounded border border-blue-500 px-2 py-0.5 text-blue-700"
              >
                Select file(s)
              </button>
            </div>
          </fieldset>
        )}
        {invalid && (
          <p role="alert" className="text-red-700">
            {invalid}
          </p>
        )}
        {save.isError && (
          <p role="alert" className="text-red-700">
            {failureText(save.error)}
          </p>
        )}
      </div>
      <div className="-mx-6 -mb-6 mt-4 flex justify-between bg-slate-200 px-6 py-3">
        <button
          type="button"
          disabled={save.isPending}
          onClick={onClose}
          className="rounded bg-slate-400 px-6 py-1.5 text-sm text-white disabled:opacity-60"
        >
          Cancel
        </button>
        <button
          type="button"
          disabled={save.isPending}
          onClick={submit}
          className="rounded bg-orange-500 px-6 py-1.5 text-sm text-white disabled:opacity-60"
        >
          {save.isPending ? 'Menyimpan…' : 'Submit'}
        </button>
      </div>
    </Popup>
  )
}

export type AcceptanceRules = {
  visible: boolean
  printLODDisabled: boolean
  acceptDisabled: boolean
  printDLADisabled: boolean
}

/** Aturan tampil/mati ketiga tombol — lihat komentar AcceptanceButtons. */
export function acceptanceRules(
  line: Settlement,
  groupPanel: string,
  businessType: string,
): AcceptanceRules {
  const status = (line.status_akseptasi ?? '').trim()
  const lod = (line.status_akseptasi_lod ?? '').trim()
  const accepted = (line.nomor_akseptasi ?? '').trim()
  const type = (line.tipe_pembayaran ?? '').trim()
  const bonding = businessType === 'Bonding' || businessType === 'BondingKBG'

  return {
    visible: status === '1',
    printLODDisabled:
      lod !== '' ||
      bonding ||
      groupPanel === '002' ||
      groupPanel === '005' ||
      type === PaymentType.Salvage ||
      type === PaymentType.AdjusterFee,
    acceptDisabled: lod === '0' || accepted !== '' || status === '2',
    printDLADisabled: lod !== '1' || status === '2',
  }
}
