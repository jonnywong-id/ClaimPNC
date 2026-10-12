import { useLODTypeOptions, useSetLODType, violationsFrom } from './api'
import type { Settlement } from './types'

/**
 * Dropdown Tipe LOD di kolom Adjustment — `.PDFType` pada `Section/ShowAdjustment_sect.xml`.
 *
 * Tampil bila `GroupPanel != '002' && GroupPanel != '005' && BusinessType != 'Bonding'`. Pilihan
 * langsung disimpan ke PDFTYPE dan dipakai dialog Print LOD sebagai pilihan awal. Dropdown
 * digambar pada SETIAP baris, seperti Pega; ia dapat diubah sampai Persetujuan / Akseptasi
 * diisi, sesudahnya terkunci dan hanya menampilkan pilihannya.
 *
 * Activity yang Pega jalankan saat pilihan berubah (`ValidationAdjustment`,
 * `SetShareAsmWhenPilihAdjustment`) tidak ada di export dan tidak dibawa.
 */
export function LODTypeSelect({
  claimID,
  taskID,
  object,
  coverage,
  adjustment,
  line,
  groupPanel,
  businessType,
  lockedReason,
}: {
  claimID: string
  taskID: string
  /** Objek, jaminan, dan adjustment berbasis 1. */
  object: number
  coverage: number
  adjustment: number
  line: Settlement
  groupPanel: string
  businessType: string
  lockedReason: string | null
}) {
  const visible = lodTypeVisible(groupPanel, businessType)
  const editable = visible && lodTypeEditable(line) && lockedReason === null
  const options = useLODTypeOptions(claimID, taskID, visible)
  const save = useSetLODType(claimID)
  if (!visible) return null

  const current = line.tipe_pdf_lod ?? ''
  const list = [...(options.data?.tipe ?? [])]
  // Baris terkunci tetap menampilkan pilihannya meski daftar belum termuat.
  if (current !== '' && !list.some((t) => t.id === current)) {
    list.push({ id: current, nama: line.nama_tipe_pdf_lod || current, tersedia: true })
  }
  const violations = violationsFrom(save.error)
  return (
    <div className="mb-1">
      <select
        aria-label={`Tipe LOD adjustment ${adjustment}`}
        value={current}
        disabled={!editable || options.isPending || save.isPending}
        title={editable ? undefined : 'Tipe LOD tidak dapat diubah setelah Persetujuan / Akseptasi.'}
        onChange={(e) => {
          save.reset()
          save.mutate({ tugas_id: taskID, objek: object, jaminan: coverage, adjustment, tipe_pdf: e.target.value })
        }}
        className="block w-full max-w-xs rounded border border-slate-300 bg-white px-1.5 py-0.5 text-xs text-slate-800 disabled:bg-slate-100 disabled:text-slate-600"
      >
        <option value="">-- Pilih Tipe LOD --</option>
        {list.map((t) => (
          <option key={t.id} value={t.id}>
            {t.nama}
          </option>
        ))}
      </select>
      {save.error && (
        <p role="alert" className="mt-0.5 text-xs text-red-700">
          {violations.length > 0 ? violations.map((v) => v.pesan).join(' ') : 'Tipe LOD belum dapat disimpan.'}
        </p>
      )}
    </div>
  )
}

/** Syarat tampil `.PDFType` ShowAdjustment_sect. */
export function lodTypeVisible(groupPanel: string, businessType: string): boolean {
  return groupPanel !== '002' && groupPanel !== '005' && businessType !== 'Bonding'
}

/** Dapat diubah sampai Persetujuan / Akseptasi diisi. */
export function lodTypeEditable(line: Settlement): boolean {
  return (line.status_akseptasi_lod ?? '').trim() === '' && (line.nomor_akseptasi ?? '').trim() === ''
}
