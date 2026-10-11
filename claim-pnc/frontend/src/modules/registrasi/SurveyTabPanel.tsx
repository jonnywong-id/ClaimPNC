import { useEffect, useMemo, useState } from 'react'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import {
  useCancelSurvey,
  useSaveSurvey,
  useSurveyorOptions,
  useSurveys,
  useSurveyTab,
  useTransferSurvey,
  useTransferSurveyCommittee,
  violationsFrom,
} from './api'
import type { Claim, SurveyObject, SurveyorOption, SurveyTabResponse } from './types'

/*
 * Tab Survey tahap Choose Surveyor — `Section/TabSurvey_sect.xml`, tiga sub-tab:
 *
 *   Permintaan Survey  RequestSurvey     form baca-saja atas Permintaan Survey (T_REQ_SURVEY)
 *   Tambah Survey      InputSurvey       grid ObjectList — tidak tampil untuk GCNMFW:PncAdmin
 *   Hasil Survey       ViewHasilSurvey   survey tersimpan (T_SURVEYORLIST)
 *
 * Tombol Transfer Survei / Transfer Komite tampil menurut `CheckTypeSurveyor_act`, yang
 * dijalankan saat Tipe Surveyor diganti (TempTypeSurveyor.ObjectSurveyor: 2 → Transfer Survei,
 * 1 → Transfer Komite). Pengiriman email alur survey menyusul pada tahap kedua (Work Owner
 * 2026-10-11).
 */

const SUB_TABS = ['Permintaan Survey', 'Tambah Survey', 'Hasil Survey'] as const
type SubTab = (typeof SUB_TABS)[number]

const SURVEYOR_TYPES: { value: string; label: string }[] = [
  { value: '1', label: 'Internal Surveyor' },
  { value: '2', label: 'Loss Adjuster' },
  { value: '3', label: 'Expert' },
  { value: '4', label: 'Survey Agent' },
]

const STATUS_NAME: Record<string, string> = {
  '1': 'Menunggu Persetujuan',
  '2': 'Ditolak Komite',
  '3': 'Sedang Proses',
  '4': 'Batal Survey',
  '5': 'Selesai',
  '6': 'YA',
  '7': 'TIDAK',
}

const PAGE_SIZE = 20

/** Status terkunci — When `IsStatus` (ObjectStatus != 3 && != 1). */
function locked(o: SurveyObject): boolean {
  return o.status === '3' || o.status === '1'
}

function failureText(err: unknown): string {
  const v = violationsFrom(err)
  if (v.length > 0) return v.map((x) => x.pesan).join(' · ')
  return err instanceof Error ? err.message : 'Terjadi kesalahan pada sistem.'
}

export function SurveyTabPanel({ klaim, taskID }: { klaim: Claim; taskID: string }) {
  const tab = useSurveyTab(klaim.id)
  const tabs = SUB_TABS.filter((t) => t !== 'Tambah Survey' || tab.data?.tambah_tampil !== false)
  const [current, setCurrent] = useState<SubTab>('Tambah Survey')
  const active = tabs.includes(current) ? current : (tabs[0] ?? 'Hasil Survey')

  return (
    <div className="mt-3">
      <div role="tablist" aria-label="Sub-tab Survey" className="flex gap-1 border-b border-slate-200">
        {tabs.map((t) => (
          <button
            key={t}
            type="button"
            role="tab"
            aria-selected={active === t}
            onClick={() => setCurrent(t)}
            className={
              active === t
                ? 'border-b-2 border-blue-600 px-3 py-1.5 text-xs font-semibold text-slate-900'
                : 'px-3 py-1.5 text-xs text-slate-600 hover:text-slate-900'
            }
          >
            {t}
          </button>
        ))}
      </div>
      {tab.error && (
        <div className="mt-3">
          <ErrorMessage title="Tab Survey tidak dapat dimuat" description={failureText(tab.error)} tone="gangguan" />
        </div>
      )}
      {tab.isPending && <p className="mt-3 text-sm text-slate-500">Memuat…</p>}
      {tab.data && active === 'Permintaan Survey' && <RequestSurvey data={tab.data} />}
      {tab.data && active === 'Tambah Survey' && <InputSurvey klaim={klaim} taskID={taskID} data={tab.data} />}
      {active === 'Hasil Survey' && <SurveyResults claimID={klaim.id} pa={tab.data?.pa ?? false} />}
    </div>
  )
}

/* ── Permintaan Survey (RequestSurvey) ─────────────────────────────────────────────── */

function ReadField({ label, value }: { label: string; value: string }) {
  return (
    <label className="block text-sm">
      <span className="text-xs text-slate-600">{label}</span>
      <input readOnly value={value} className="mt-1 block w-full rounded border border-slate-200 bg-slate-50 px-2 py-1" />
    </label>
  )
}

function RequestSurvey({ data }: { data: SurveyTabResponse }) {
  const r = data.permintaan
  return (
    <div className="mt-3 space-y-3">
      <div className="grid gap-3 md:grid-cols-2">
        <ReadField label="Survey Atas Permintaan" value={r?.survey_atas_permintaan ?? ''} />
        <ReadField label="Lokasi Survey" value={r?.lokasi_survey ?? ''} />
        <ReadField label="No. Telp" value={r?.no_telp ?? ''} />
        <ReadField label="Tanggal Request Survey" value={r?.tanggal_request ? formatDate(r.tanggal_request) : ''} />
        <ReadField label="Cabang" value={r?.cabang ?? ''} />
        <ReadField label="Surveyor" value={r?.surveyor ?? ''} />
        <ReadField label="Email Surveyor" value={r?.email_surveyor ?? ''} />
        <ReadField label="Nama Object" value={r?.nama_objek ?? ''} />
      </div>
      <div className="flex gap-2">
        <Button tone="kedua" disabled title="Seluruh isian Permintaan Survey baca-saja.">
          Simpan
        </Button>
        <Button tone="kedua" disabled title="Pengiriman email alur survey menyusul pada tahap kedua.">
          Kirim Email
        </Button>
      </div>
    </div>
  )
}

/* ── Tambah Survey (InputSurvey) ───────────────────────────────────────────────────── */

function InputSurvey({ klaim, taskID, data }: { klaim: Claim; taskID: string; data: SurveyTabResponse }) {
  // Lokasi survey kosong diisi lokasi objek, seperti saat objek dibuat (`GetListObjectFire`).
  const initial = (list: SurveyObject[]) => list.map((o) => ({ ...o, lokasi_survey: o.lokasi_survey || o.lokasi_objek }))
  const [rows, setRows] = useState<SurveyObject[]>(() => initial(data.objek))
  const [caseType, setCaseType] = useState('')
  // TempTypeSurveyor.ObjectSurveyor — kosong sampai Tipe Surveyor diganti.
  const [flag, setFlag] = useState<'' | '1' | '2'>('')
  const [notice, setNotice] = useState<string | null>(null)
  const [page, setPage] = useState(0)
  const [committeeOpen, setCommitteeOpen] = useState(false)
  const [cancelRow, setCancelRow] = useState<SurveyObject | null>(null)
  const save = useSaveSurvey(klaim.id)
  const transfer = useTransferSurvey(klaim.id)

  useEffect(() => setRows(initial(data.objek)), [data.objek])

  const busy = save.isPending || transfer.isPending
  const pages = Math.max(1, Math.ceil(rows.length / PAGE_SIZE))
  const shown = rows.slice(page * PAGE_SIZE, page * PAGE_SIZE + PAGE_SIZE)

  function update(id: string, patch: Partial<SurveyObject>) {
    setRows((all) => all.map((r) => (r.objek_id === id ? { ...r, ...patch } : r)))
  }

  /** EmpetyName_svyr + CheckTypeSurveyor_act — onchange Tipe Surveyor. */
  function changeType(row: SurveyObject, type: string) {
    const cleared = { nama_surveyor: '', nama_surveyor_marine: '', login_surveyor: '', login_surveyor_marine: '' }
    setNotice(null)
    if (!row.pilih) {
      update(row.objek_id, { ...cleared, tipe_surveyor: '1' })
      setNotice('Harus Pilih Object')
      setFlag('2')
      setCaseType('1')
      return
    }
    update(row.objek_id, { ...cleared, tipe_surveyor: type })
    setCaseType(type)
    const adjuster = type === '2' || type === '3' || type === '4'
    setFlag(adjuster && !data.anggota_koasuransi ? '1' : '2')
  }

  const body = () => ({ tugas_id: taskID, tipe_surveyor_kasus: caseType, objek: rows })

  const result = transfer.data
  const failure = save.error ?? transfer.error

  return (
    <div className="mt-3">
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="bg-slate-100 text-left text-xs text-slate-700">
              <th className="p-2">
                Pilih<span className="text-orange-600">*</span>
              </th>
              <th className="p-2">Nama Objek</th>
              <th className="p-2">
                Lokasi<span className="text-orange-600">*</span>
              </th>
              <th className="p-2">Tipe Surveyor</th>
              <th className="p-2">
                Pilih Surveyor<span className="text-orange-600">*</span>
              </th>
              <th className="p-2">Status</th>
              <th className="p-2" />
            </tr>
          </thead>
          <tbody>
            {shown.length === 0 && (
              <tr>
                <td colSpan={7} className="p-2 text-xs text-slate-500">
                  Data Tidak Ada
                </td>
              </tr>
            )}
            {shown.map((r) => (
              <tr key={r.objek_id} className="border-b border-slate-100 align-top">
                <td className="p-2">
                  <input
                    type="checkbox"
                    aria-label={`Pilih ${r.nama_objek}`}
                    checked={r.pilih}
                    onChange={(e) => update(r.objek_id, { pilih: e.target.checked })}
                  />
                </td>
                <td className="p-2">{r.nama_objek}</td>
                <td className="p-2">
                  <textarea
                    aria-label={`Lokasi ${r.nama_objek}`}
                    rows={3}
                    readOnly={locked(r)}
                    value={r.lokasi_survey}
                    onChange={(e) => update(r.objek_id, { lokasi_survey: e.target.value })}
                    className="w-64 rounded border border-slate-300 px-2 py-1 read-only:bg-slate-50"
                  />
                </td>
                <td className="p-2">
                  <select
                    aria-label={`Tipe Surveyor ${r.nama_objek}`}
                    disabled={locked(r)}
                    value={r.tipe_surveyor}
                    onChange={(e) => changeType(r, e.target.value)}
                    className="rounded border border-slate-300 px-2 py-1"
                  >
                    <option value="">----- Pilih Surveyor -----</option>
                    {SURVEYOR_TYPES.map((t) => (
                      <option key={t.value} value={t.value}>
                        {t.label}
                      </option>
                    ))}
                  </select>
                </td>
                <td className="p-2">
                  <ChooseSurveyor row={r} pa={data.pa} marineHull={data.marine_hull} onChange={(p) => update(r.objek_id, p)} />
                </td>
                <td className="p-2">{STATUS_NAME[r.status] ?? r.nama_status ?? ''}</td>
                <td className="p-2">
                  {r.status === '3' && (
                    <button
                      type="button"
                      className="rounded border border-blue-600 px-2 py-1 text-xs text-blue-700"
                      onClick={() => setCancelRow(r)}
                    >
                      Batal Survei
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {pages > 1 && (
        <div className="mt-2 flex items-center gap-2 text-xs text-slate-600">
          <button type="button" disabled={page === 0} onClick={() => setPage(page - 1)} className="rounded border px-2 py-0.5 disabled:opacity-40">
            Sebelumnya
          </button>
          <span>
            Halaman {page + 1} dari {pages}
          </span>
          <button type="button" disabled={page >= pages - 1} onClick={() => setPage(page + 1)} className="rounded border px-2 py-0.5 disabled:opacity-40">
            Berikutnya
          </button>
        </div>
      )}

      {notice && (
        <div className="mt-3">
          <ErrorMessage title={notice} description="" tone="penolakan" />
        </div>
      )}
      {failure && (
        <div className="mt-3">
          <ErrorMessage title="Survey tidak dapat diproses" description={failureText(failure)} tone="penolakan" />
        </div>
      )}
      {save.isSuccess && !busy && !failure && <p className="mt-3 text-sm text-emerald-700">Data survey tersimpan.</p>}
      {result && !busy && !failure && <ActionResult surveys={result.survey_terbit} committee={result.komite_terbit} auto={result.otomatis} />}

      <div className="mt-3 flex flex-wrap gap-2">
        <Button tone="kedua" disabled={busy} onClick={() => save.mutate(body())}>
          {save.isPending ? 'Menyimpan…' : 'Simpan'}
        </Button>
        {flag === '2' && (
          <Button tone="kedua" disabled={busy} onClick={() => transfer.mutate(body())}>
            {transfer.isPending ? 'Memproses…' : 'Transfer Survei'}
          </Button>
        )}
        {flag === '1' && (
          <Button tone="kedua" disabled={busy} onClick={() => setCommitteeOpen(true)}>
            Transfer Komite
          </Button>
        )}
      </div>

      {committeeOpen && (
        <CommitteeDialog klaim={klaim} request={body()} onClose={() => setCommitteeOpen(false)} />
      )}
      {cancelRow && <CancelDialog claimID={klaim.id} taskID={taskID} row={cancelRow} onClose={() => setCancelRow(null)} />}
    </div>
  )
}

function ActionResult({ surveys, committee, auto }: { surveys: string[]; committee: string[]; auto: boolean }) {
  return (
    <div className="mt-3 space-y-1 text-sm text-emerald-700" role="status">
      {surveys.length > 0 && <p>Survey terbit: {surveys.join(', ')}.</p>}
      {committee.length > 0 && (
        <p>
          Diajukan ke Komite: {committee.join(', ')} — status Menunggu Persetujuan. Persetujuan Komite survey menunggu layanan
          link email Pega.
        </p>
      )}
      {auto && <p>Jalur otomatis: survey langsung dibuat tanpa Komite.</p>}
    </div>
  )
}

/** Kolom Pilih Surveyor — section `ChooseSurveyor`. */
function ChooseSurveyor({
  row,
  pa,
  marineHull,
  onChange,
}: {
  row: SurveyObject
  pa: boolean
  marineHull: boolean
  onChange: (p: Partial<SurveyObject>) => void
}) {
  const internal = row.tipe_surveyor === '1'
  const options = useSurveyorOptions(row.tipe_surveyor === '' ? '1' : row.tipe_surveyor, !locked(row))
  const list = options.data ?? []
  const listID = `surveyor-${row.objek_id}`

  if (pa) {
    // Investigator — dikonfigurasi selalu baca-saja.
    return (
      <label className="block text-xs">
        <span className="font-semibold text-slate-700">Investigator</span>
        <input readOnly value={row.nama_surveyor} className="mt-1 block w-56 rounded border border-slate-200 bg-slate-50 px-2 py-1 text-sm" />
      </label>
    )
  }

  if (internal) {
    // Cabang — pilihan Internal Surveyor per cabang.
    return (
      <label className="block text-xs">
        <span className="font-semibold text-slate-700">Cabang</span>
        <select
          disabled={locked(row)}
          value={row.nama_cabang ? `${row.nama_cabang}|${row.nama_surveyor}` : ''}
          onChange={(e) => {
            const o = list.find((x) => `${x.nama_cabang}|${x.nama}` === e.target.value)
            if (!o) return
            onChange({
              nama_cabang: o.nama_cabang,
              kode_cabang: o.cabang,
              email_surveyor: o.email,
              login_surveyor: o.login,
              nama_surveyor: o.nama,
              alamat_surveyor: o.alamat,
            })
          }}
          className="mt-1 block w-56 rounded border border-slate-300 px-2 py-1 text-sm"
        >
          <option value="">Ketikan kata kunci, pilih</option>
          {list.map((o) => (
            <option key={`${o.id}-${o.login}`} value={`${o.nama_cabang}|${o.nama}`}>
              {o.nama_cabang}
            </option>
          ))}
        </select>
      </label>
    )
  }

  const pick = (name: string, opts: SurveyorOption[]) => opts.find((x) => x.nama === name)
  return (
    <div className="space-y-2">
      <label className="block text-xs">
        <span className="font-semibold text-slate-700">Loss Adjuster</span>
        <input
          list={listID}
          readOnly={locked(row)}
          value={row.nama_surveyor}
          placeholder="Ketikan kata kunci, pilih"
          title="Masukkan kata kunci dan tekan panah bawah keyboard"
          onChange={(e) => {
            const o = pick(e.target.value, list)
            onChange({ nama_surveyor: e.target.value, login_surveyor: o?.login ?? '', alamat_surveyor: o?.alamat ?? '' })
          }}
          className="mt-1 block w-56 rounded border border-slate-300 bg-slate-50 px-2 py-1 text-sm"
        />
        <datalist id={listID}>
          {list.map((o) => (
            <option key={`${o.id}-${o.login}`} value={o.nama} />
          ))}
        </datalist>
      </label>
      {marineHull && (
        <label className="block text-xs">
          <span className="font-semibold text-slate-700">Surveyor Marine</span>
          <input
            readOnly={locked(row)}
            value={row.nama_surveyor_marine}
            placeholder="Ketikan kata kunci, pilih"
            onChange={(e) => onChange({ nama_surveyor_marine: e.target.value })}
            className="mt-1 block w-56 rounded border border-slate-300 px-2 py-1 text-sm"
          />
        </label>
      )}
    </div>
  )
}

/* ── Modal Batal Survei (ConfirmRejectSurvey) ──────────────────────────────────────── */

function CancelDialog({ claimID, taskID, row, onClose }: { claimID: string; taskID: string; row: SurveyObject; onClose: () => void }) {
  const cancel = useCancelSurvey(claimID)
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4" role="dialog" aria-modal="true" aria-labelledby="judul-batal-survey">
      <div className="w-full max-w-md rounded-kartu bg-white p-6 shadow-angkat">
        <h2 id="judul-batal-survey" className="text-base font-semibold text-slate-900">
          Apakah anda yakin ingin membatalkan survey ini?
        </h2>
        <div className="mt-3 space-y-2">
          <ReadField label="ID Survey" value={row.id_survey ?? ''} />
          {row.id_survey_marine && <ReadField label="ID Survey 2" value={row.id_survey_marine} />}
        </div>
        {cancel.error && (
          <div className="mt-3">
            <ErrorMessage title="Survey tidak dapat dibatalkan" description={failureText(cancel.error)} tone="penolakan" />
          </div>
        )}
        <div className="mt-4 flex justify-end gap-2">
          <Button tone="halus" disabled={cancel.isPending} onClick={onClose}>
            Tidak
          </Button>
          <Button tone="utama" disabled={cancel.isPending} onClick={() => cancel.mutate({ tugas_id: taskID, objek_id: row.objek_id }, { onSuccess: onClose })}>
            {cancel.isPending ? 'Membatalkan…' : 'Ya'}
          </Button>
        </div>
      </div>
    </div>
  )
}

/* ── Modal Transfer Komite (ClaimComitee) ──────────────────────────────────────────── */

type Nominee = { key: number; nama: string; id: string; login: string }

function localDateTime(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function CommitteeDialog({
  klaim,
  request,
  onClose,
}: {
  klaim: Claim
  request: { tugas_id: string; tipe_surveyor_kasus: string; objek: SurveyObject[] }
  onClose: () => void
}) {
  const send = useTransferSurveyCommittee(klaim.id)
  const nominatedOptions = useSurveyorOptions('nominasi')
  const [date, setDate] = useState(() => localDateTime(new Date()))
  const [circumstances, setCircumstances] = useState('')
  const [nominated, setNominated] = useState('')
  const [remarks, setRemarks] = useState('')
  const [company, setCompany] = useState(klaim.pelapor?.nama ?? '')
  const [contact, setContact] = useState(klaim.pelapor?.nama ?? '')
  const [phone, setPhone] = useState(klaim.pelapor?.telepon ?? '')
  const [email, setEmail] = useState('')
  const [manual, setManual] = useState(false)
  const [nominees, setNominees] = useState<Nominee[]>([])
  const options = useMemo(() => nominatedOptions.data ?? [], [nominatedOptions.data])

  function submit() {
    send.mutate(
      {
        ...request,
        tanggal: new Date(date).toISOString(),
        inisial: klaim.user_teknis,
        circumstances,
        nominated_adjuster: nominated,
        remarks,
        nama_perusahaan: company,
        contact_person: contact,
        office_phone: phone,
        email,
        penunjukan_manual: manual,
        nominasi: nominees.filter((n) => n.nama.trim() !== '').map((n) => ({ id: n.id, nama: n.nama, login: n.login })),
      },
      { onSuccess: onClose },
    )
  }

  const area = 'mt-1 block w-full rounded border border-slate-300 px-2 py-1 text-sm'
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8" role="dialog" aria-modal="true" aria-labelledby="judul-komite-survey">
      <div className="max-h-full w-full max-w-3xl overflow-y-auto rounded-kartu bg-white p-6 shadow-angkat">
        <h2 id="judul-komite-survey" className="text-lg font-semibold text-slate-900">
          Transfer Komite
        </h2>
        <div className="mt-4 grid gap-3 md:grid-cols-3">
          <label className="block text-sm">
            <span className="text-xs text-slate-600">Date &amp; Time</span>
            <input type="datetime-local" value={date} onChange={(e) => setDate(e.target.value)} className={area} />
          </label>
          <ReadField label="Inisial" value={klaim.user_teknis} />
          <ReadField label="Tipe Analisis" value="Survey" />
        </div>
        <div className="mt-3 space-y-3">
          <label className="block text-sm">
            <span className="text-xs text-slate-600">Circumtanses cause of Loss</span>
            <textarea rows={3} value={circumstances} onChange={(e) => setCircumstances(e.target.value)} className={area} />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-slate-600">Nominated Adjuster</span>
            <textarea rows={2} value={nominated} onChange={(e) => setNominated(e.target.value)} className={area} />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-slate-600">Remarks</span>
            <textarea rows={2} value={remarks} onChange={(e) => setRemarks(e.target.value)} className={area} />
          </label>
        </div>

        <h3 className="mt-4 text-sm font-semibold text-slate-800">Nominated Loss Adjuster</h3>
        <table className="mt-1 w-full border-collapse text-sm">
          <thead>
            <tr className="bg-slate-100 text-left text-xs text-slate-700">
              <th className="p-2">Loss Adjuster</th>
              <th className="p-2 w-16" />
            </tr>
          </thead>
          <tbody>
            {nominees.map((n) => (
              <tr key={n.key} className="border-b border-slate-100">
                <td className="p-2">
                  <input
                    list="nominasi-loss-adjuster"
                    value={n.nama}
                    placeholder="Ketikan kata kunci, pilih"
                    onChange={(e) => {
                      const o = options.find((x) => x.nama === e.target.value)
                      setNominees((all) =>
                        all.map((x) => (x.key === n.key ? { ...x, nama: e.target.value, id: o?.id ?? '', login: o?.login ?? '' } : x)),
                      )
                    }}
                    className="block w-full rounded border border-slate-300 px-2 py-1 text-sm"
                  />
                </td>
                <td className="p-2">
                  <button type="button" className="text-xs text-red-700" onClick={() => setNominees((all) => all.filter((x) => x.key !== n.key))}>
                    Hapus
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <datalist id="nominasi-loss-adjuster">
          {options.map((o) => (
            <option key={`${o.id}-${o.login}`} value={o.nama} />
          ))}
        </datalist>
        <button
          type="button"
          className="mt-2 rounded border border-slate-300 px-2 py-1 text-xs"
          onClick={() => setNominees((all) => [...all, { key: Date.now(), nama: '', id: '', login: '' }])}
        >
          + Tambah
        </button>

        <label className="mt-3 flex items-center gap-2 text-sm">
          <input type="checkbox" checked={manual} onChange={(e) => setManual(e.target.checked)} />
          Penunjukan Manual Ke Komite
        </label>

        <h3 className="mt-4 text-sm font-semibold text-slate-800">Contact Person untuk Survey</h3>
        <div className="mt-1 grid gap-3 md:grid-cols-2">
          <label className="block text-sm">
            <span className="text-xs text-slate-600">Company Name</span>
            <input value={company} onChange={(e) => setCompany(e.target.value)} className={area} />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-slate-600">Contact Person</span>
            <input value={contact} onChange={(e) => setContact(e.target.value)} className={area} />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-slate-600">Office Phone</span>
            <input value={phone} onChange={(e) => setPhone(e.target.value)} className={area} />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-slate-600">Email</span>
            <input value={email} onChange={(e) => setEmail(e.target.value)} className={area} />
          </label>
        </div>
        <p className="mt-3 text-xs text-slate-500">
          Upload Dokumen Survey dan pengiriman email Komite menyusul pada tahap kedua.
        </p>

        {send.error && (
          <div className="mt-3">
            <ErrorMessage title="Transfer Komite tidak dapat diproses" description={failureText(send.error)} tone="penolakan" />
          </div>
        )}
        <div className="mt-4 flex justify-end gap-2">
          <Button tone="halus" disabled={send.isPending} onClick={onClose}>
            Cancel
          </Button>
          <Button tone="utama" disabled={send.isPending} onClick={submit}>
            {send.isPending ? 'Memproses…' : 'Submit'}
          </Button>
        </div>
      </div>
    </div>
  )
}

/* ── Hasil Survey (ViewHasilSurvey) ────────────────────────────────────────────────── */

function surveyStatusName(sts: string): string {
  const s = sts.trim().toLowerCase()
  if (s === 'on progress') return 'Sedang Proses'
  if (s === 'batal survey') return 'Batal Survey'
  return sts || '—'
}

function SurveyResults({ claimID, pa }: { claimID: string; pa: boolean }) {
  const q = useSurveys(claimID)
  const list = q.data?.survey ?? []
  const head = pa
    ? ['Tanggal Investigasi', 'Nama Peserta', 'Lokasi Objek', 'Status']
    : ['Tanggal Pengajuan Survey', 'Nama Surveyor', 'Nama Objek', 'Lokasi Objek', 'Status']
  return (
    <div className="mt-3">
      {q.error && <ErrorMessage title="Hasil survey tidak dapat dimuat" description={failureText(q.error)} tone="gangguan" />}
      {q.isPending && <p className="text-sm text-slate-500">Memuat…</p>}
      {q.data && (
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="bg-slate-100 text-left text-xs text-slate-700">
              <th className="p-2">No</th>
              {head.map((h) => (
                <th key={h} className="p-2">
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {list.length === 0 && (
              <tr>
                <td colSpan={head.length + 1} className="p-2 text-xs text-slate-500">
                  Data Tidak Ada
                </td>
              </tr>
            )}
            {list.map((s, i) => (
              <tr key={`${s.kasus_id}-${s.urutan}-${i}`} className="border-b border-slate-100 align-top">
                <td className="p-2">{i + 1}</td>
                <td className="p-2">{formatDate(s.tanggal_survey)}</td>
                {!pa && <td className="p-2">{s.nama_surveyor || '—'}</td>}
                <td className="p-2">{s.nama_objek || '—'}</td>
                <td className="p-2">{(pa ? s.lokasi_objek : s.lokasi_survey || s.lokasi_objek) || '—'}</td>
                <td className="p-2">{surveyStatusName(s.status)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}
