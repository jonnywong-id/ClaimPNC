import { useEffect } from 'react'

import { formatDate } from '@/components/format'

import { useInsuredProfile } from './api'
import type { Claim, InsuredAddress } from './types'

const NOT_BUILT = 'Proses tombol ini belum dibangun.'

const UNSAVED_NOTE = 'Not saved yet: no column for this field in T_CLAIM_PNC.'

/**
 * Bagian atas tab Register untuk PA — kontainer `IsPA` pada `Section/InputRegisterDetail`:
 *
 *   DATA TERTANGGUNG KLAIM   No Polis (+ Cari Polis), Nama Tertanggung, Nama Sumbis, Nama Bisnis,
 *                            Tanggal Mulai Polis, Akhir Polis — baca saja; No KTP (wajib IsPA)
 *   InputAddress_PNC_Klaim   Alamat · Telephone dan Email · Pengkinian Data
 *
 * Alamat dan telepon dibaca dari CIF polis (`Policy.CIFData`, GET .../tertanggung) — di Pega
 * `pyWorkPage.AddressList`, disalin `GetDataTertartanggungFromASMTelfFax`. No KTP diisi bawaan
 * dari CIF bila masih kosong, seperti langkah 12 activity itu.
 */
export function InsuredDataSection({
  klaim,
  idCard,
  onIDCard,
  phone,
  onPhone,
  email,
  onEmail,
}: Readonly<{
  klaim: Claim
  idCard: string
  onIDCard: (value: string) => void
  phone: string
  onPhone: (value: string) => void
  email: string
  onEmail: (value: string) => void
}>) {
  const profile = useInsuredProfile(klaim.id)
  const defaultIDCard = profile.data?.no_ktp ?? ''
  useEffect(() => {
    if (idCard === '' && defaultIDCard !== '') onIDCard(defaultIDCard)
  }, [idCard, defaultIDCard, onIDCard])

  const address: InsuredAddress | undefined = profile.data?.alamat?.[0]

  return (
    <section className="rounded border border-slate-200 p-4" aria-label="DATA TERTANGGUNG KLAIM">
      <h2 className="text-sm font-semibold uppercase text-slate-800 underline">DATA TERTANGGUNG KLAIM</h2>

      <div className="mt-3 grid gap-4 sm:grid-cols-2">
        <div className="space-y-3">
          <div className="flex items-end gap-3">
            <ReadField label="No Polis" value={klaim.polis.nomor} />
            <button type="button" disabled title={NOT_BUILT}
              className="mb-1 rounded border border-blue-300 px-2 py-1 text-sm text-blue-700 disabled:opacity-60">
              Cari Polis
            </button>
          </div>
          <ReadField label="Nama Tertanggung" value={klaim.polis.nama_tertanggung} />
          <ReadField label="Nama Sumbis" value={klaim.polis.nama_sumbis ?? ''} />
          <ReadField label="Nama Bisnis" value={klaim.polis.nama_bisnis ?? ''} />
        </div>
        <div className="space-y-3">
          <ReadField label="Tanggal Mulai Polis" value={formatDate(klaim.polis.mulai_pertanggungan)} />
          <ReadField label="Akhir Polis" value={formatDate(klaim.polis.akhir_pertanggungan)} />
          <div>
            <label htmlFor="no_ktp" className="block text-sm font-medium text-slate-700">
              No KTP <span className="text-red-600">*</span>
            </label>
            <input id="no_ktp" value={idCard} onChange={(e) => onIDCard(e.target.value)}
              className="mt-1 w-full rounded border border-dashed border-slate-300 px-3 py-2 text-slate-900 focus:border-slate-500 focus:outline-none" />
            <p className="mt-1 text-xs text-amber-700">{UNSAVED_NOTE}</p>
          </div>
        </div>
      </div>

      <div className="mt-6 grid gap-6 rounded border border-slate-200 p-4 lg:grid-cols-2">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 underline">Alamat</h3>
          {profile.isPending && <p className="mt-2 text-xs text-slate-500">Memuat…</p>}
          {profile.isError && <p className="mt-2 text-xs text-red-700">Data alamat tertanggung tidak dapat dimuat.</p>}
          <div className="mt-2 grid gap-3 sm:grid-cols-2">
            <ReadField label="Type Alamat" value={(address?.nama_jenis ?? '').toUpperCase()} />
            <div className="sm:col-span-2">
              <ReadField label="Alamat" value={address?.alamat ?? ''} />
            </div>
            <ReadField label="Kota" value={address?.nama_kota || address?.kota || ''} />
            <ReadField label="Kecamatan" value={address?.nama_kecamatan || address?.kecamatan || ''} />
            <ReadField label="Kelurahan" value={address?.nama_kelurahan || address?.kelurahan || ''} />
            <ReadField label="Kode Pos" value={address?.kode_pos ?? ''} />
          </div>
        </div>

        <div>
          <h3 className="text-sm font-semibold text-slate-800 underline">Telephone dan Email</h3>
          <table className="mt-2 w-full border-collapse text-sm">
            <caption className="sr-only">Telephone dan Email</caption>
            <thead>
              <tr className="bg-slate-100 text-left text-xs text-slate-700">
                <th className="p-2" />
                <th className="p-2">Jenis Telepon-Fax</th>
                <th className="p-2">Kode Telepone/Fax</th>
                <th className="p-2">Nomor Telepone/Email</th>
                <th className="p-2">Nomor Extension</th>
              </tr>
            </thead>
            <tbody>
              {(address?.telepon ?? []).length === 0 ? (
                <tr>
                  <td colSpan={5} className="p-2 text-xs text-slate-500">Data Tidak Ada</td>
                </tr>
              ) : (
                address!.telepon.map((t, i) => (
                  <tr key={`${t.jenis}-${i}`} className="border-b border-slate-100">
                    <td className="p-2 text-xs text-slate-500">{i + 1}</td>
                    <td className="p-2">{t.nama_jenis || '—'}</td>
                    <td className="p-2">{t.kode || '(Kode Area)'}</td>
                    <td className="p-2">{t.nomor || '—'}</td>
                    <td className="p-2">{t.ekstensi || ''}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>

          <h3 className="mt-6 text-sm font-semibold text-slate-800 underline">Pengkinian Data</h3>
          <p className="mt-2 text-sm">
            <span className="font-semibold">Wajib</span> <span className="text-red-600">*</span>{' '}
            <i>No HP harus diisi. Harap Masukan No. Hp yang terhubung dengan WA</i>
          </p>
          <div className="mt-2 grid gap-3">
            <UnsavedInput id="pengkinian_hp" label="No. HP" value={phone} onChange={onPhone} />
            <UnsavedInput id="pengkinian_email" label="Email" value={email} onChange={onEmail} />
          </div>
        </div>
      </div>
    </section>
  )
}

function ReadField({ label, value }: Readonly<{ label: string; value: string }>) {
  return (
    <div className="min-w-0 flex-1">
      <p className="text-xs font-medium text-slate-600">{label}</p>
      <p className="mt-0.5 whitespace-pre-wrap break-words text-sm text-slate-900">{value || '—'}</p>
    </div>
  )
}

function UnsavedInput({ id, label, value, onChange }: Readonly<{ id: string; label: string; value: string; onChange: (v: string) => void }>) {
  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">{label}</label>
      <input id={id} value={value} onChange={(e) => onChange(e.target.value)}
        className="mt-1 w-full rounded border border-dashed border-slate-300 px-3 py-2 text-slate-900 focus:border-slate-500 focus:outline-none" />
      <p className="mt-1 text-xs text-amber-700">{UNSAVED_NOTE}</p>
    </div>
  )
}
