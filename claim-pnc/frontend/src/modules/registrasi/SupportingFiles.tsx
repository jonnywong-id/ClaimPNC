import { useEffect, useState } from 'react'

import { APIError } from '@/api/client'

import { type SettlementAddress, useDocuments, useUploadSupportingFiles, violationsFrom } from './api'

type Row = { berkas: File | null; jenisDokumen: string; catatan: string }

const emptyRow = (): Row => ({ berkas: null, jenisDokumen: '', catatan: '' })

/**
 * Tombol "Unggah File Penunjang" di bawah rincian satu baris adjustment, tab Adjustment &
 * Akseptasi (`Section/InputAdjustment_sect.xml:82677`).
 *
 * Di Pega tombol ini membuka local action `UploadDokumen_Adj` dalam modal: daftar berkas
 * beserta jenis dokumen dan catatannya, disimpan oleh `SetUploadDokumenAdjustment_Act`
 * sebagai lampiran klaim. Penyimpanannya berdiri sendiri — tidak menunggu Submit akseptasi —
 * sehingga tombolnya tetap tersedia setelah klaim diakseptasi atau dibayar.
 */
export function SupportingFilesButton({ claimID, address }: { claimID: string; address: SettlementAddress }) {
  const [open, setOpen] = useState(false)
  const [done, setDone] = useState<number | null>(null)

  return (
    <div className="mt-5 flex flex-wrap items-center gap-3">
      <button
        type="button"
        onClick={() => {
          setDone(null)
          setOpen(true)
        }}
        className="rounded border border-blue-500 px-3 py-1 text-xs text-blue-700 hover:bg-blue-50"
      >
        Unggah File Penunjang
      </button>
      {done !== null && (
        <span role="status" className="text-xs text-emerald-700">
          {done} berkas penunjang tersimpan.
        </span>
      )}
      {open && (
        <SupportingFilesDialog
          claimID={claimID}
          address={address}
          onClose={() => setOpen(false)}
          onSaved={(count) => {
            setDone(count)
            setOpen(false)
          }}
        />
      )}
    </div>
  )
}

function SupportingFilesDialog({
  claimID,
  address,
  onClose,
  onSaved,
}: {
  claimID: string
  address: SettlementAddress
  onClose: () => void
  onSaved: (count: number) => void
}) {
  const documents = useDocuments(claimID)
  const upload = useUploadSupportingFiles(claimID)
  const [rows, setRows] = useState<Row[]>([emptyRow()])
  const [invalid, setInvalid] = useState<string | null>(null)
  const busy = upload.isPending

  // Jenis dokumen dari checklist lini bisnis klaim — sama dengan Unggah Dokumen Persetujuan LOD.
  const types = (documents.data?.kategori ?? []).flatMap((k) =>
    k.dokumen.map((d) => ({ id: d.id, nama: `${k.nama} — ${d.nama}` })),
  )

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape' && !busy) onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose, busy])

  const update = (i: number, patch: Partial<Row>) => setRows(rows.map((r, j) => (j === i ? { ...r, ...patch } : r)))

  const submit = () => {
    const chosen = rows.filter((r) => r.berkas !== null)
    if (chosen.length === 0) {
      setInvalid('Pilih minimal satu berkas.')
      return
    }
    if (chosen.some((r) => r.jenisDokumen === '')) {
      setInvalid('Pilih jenis dokumen untuk setiap berkas.')
      return
    }
    setInvalid(null)
    upload.mutate(
      {
        alamat: address,
        berkas: chosen.map((r) => ({ berkas: r.berkas as File, jenisDokumen: r.jenisDokumen, catatan: r.catatan })),
      },
      { onSuccess: () => onSaved(chosen.length) },
    )
  }

  const field = 'mt-1 w-full rounded border border-slate-300 px-2 py-1 text-xs'

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-file-penunjang"
    >
      <div className="max-h-full w-full max-w-3xl space-y-3 overflow-y-auto rounded-kartu bg-white p-6 text-xs text-slate-700 shadow-angkat">
        <h2 id="judul-file-penunjang" className="text-lg font-semibold text-slate-900">
          Unggah File Penunjang
        </h2>

        {documents.isError && (
          <p role="alert" className="text-red-700">
            Daftar jenis dokumen tidak dapat dimuat.
          </p>
        )}

        <div className="space-y-3">
          {rows.map((row, i) => (
            <div key={i} className="grid gap-2 rounded border border-slate-200 p-3 sm:grid-cols-[1fr_1fr_auto]">
              <label className="block font-semibold text-slate-800">
                Berkas
                <input
                  type="file"
                  aria-label={`Berkas ${i + 1}`}
                  disabled={busy}
                  onChange={(e) => update(i, { berkas: e.target.files?.[0] ?? null })}
                  className="mt-1 block w-full text-xs"
                />
              </label>
              <label className="block font-semibold text-slate-800">
                Jenis Dokumen
                <select
                  aria-label={`Jenis dokumen ${i + 1}`}
                  value={row.jenisDokumen}
                  disabled={busy}
                  onChange={(e) => update(i, { jenisDokumen: e.target.value })}
                  className={field}
                >
                  <option value="">-- Pilih --</option>
                  {types.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.nama}
                    </option>
                  ))}
                </select>
              </label>
              <div className="flex items-end">
                <button
                  type="button"
                  disabled={busy || rows.length === 1}
                  onClick={() => setRows(rows.filter((_, j) => j !== i))}
                  className="text-red-700 disabled:text-slate-300"
                >
                  Hapus
                </button>
              </div>
              <label className="block font-semibold text-slate-800 sm:col-span-3">
                Catatan
                <input
                  aria-label={`Catatan ${i + 1}`}
                  value={row.catatan}
                  disabled={busy}
                  onChange={(e) => update(i, { catatan: e.target.value })}
                  className={field}
                />
              </label>
            </div>
          ))}
          <div className="text-center">
            <button
              type="button"
              disabled={busy}
              onClick={() => setRows([...rows, emptyRow()])}
              className="rounded border border-blue-500 px-2 py-0.5 text-blue-700"
            >
              Select file(s)
            </button>
          </div>
        </div>

        {invalid && (
          <p role="alert" className="text-red-700">
            {invalid}
          </p>
        )}
        {upload.isError && (
          <p role="alert" className="text-red-700">
            {failureText(upload.error)}
          </p>
        )}

        <div className="-mx-6 -mb-6 mt-4 flex justify-between bg-slate-200 px-6 py-3">
          <button
            type="button"
            disabled={busy}
            onClick={onClose}
            className="rounded bg-slate-400 px-6 py-1.5 text-sm text-white disabled:opacity-60"
          >
            Cancel
          </button>
          <button
            type="button"
            disabled={busy}
            onClick={submit}
            className="rounded bg-orange-500 px-6 py-1.5 text-sm text-white disabled:opacity-60"
          >
            {busy ? 'Mengunggah…' : 'Submit'}
          </button>
        </div>
      </div>
    </div>
  )
}

function failureText(failure: unknown): string {
  const violations = violationsFrom(failure)
  if (violations.length > 0) return violations.map((v) => v.pesan).join(' ')
  if (failure instanceof APIError || failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}
