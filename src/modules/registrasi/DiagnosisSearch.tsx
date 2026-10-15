import { useState } from 'react'

import { Button } from '@/components/Button'
import { DataTable } from '@/components/DataTable'
import { Field } from '@/components/Field'

import { useDiagnosisSearch } from './api'
import type { DiagnosisOption } from './types'

/**
 * "Detail Diagnosa" modal Transfer Claim ke Komite (PA) — isian "Cari Kode / Desc Diagnose"
 * (`TempKodeDiagnosa.NoteKasir`), tombol Cari, dan grid Kode Diagnose / Desc Diagnose dengan
 * tombol Pilih.
 *
 * Cari menjalankan `CariKodeDiagnosKlaimPA` tanpa flag → `GetKodeDiagnosaKlaimPa` (kode persis
 * atau sebagian deskripsi, master `sm.m_diagnosis`). Pilih menjalankannya dengan flag 1:
 * `.CodeDiagnose` dan `.DescDiagnose` diisi baris itu.
 */
export function DiagnosisSearch({
  disabled,
  onPick,
}: {
  disabled: boolean
  onPick: (d: DiagnosisOption) => void
}) {
  const [text, setText] = useState('')
  const [term, setTerm] = useState('')
  const search = useDiagnosisSearch(term)

  return (
    <div className="mt-3 rounded border border-slate-200 p-3">
      <h4 className="text-sm font-semibold text-slate-800">Detail Diagnosa</h4>
      <div className="mt-2 flex items-end gap-2">
        <div className="flex-1">
          <Field
            id="komite-cari-diagnosa"
            label="Cari Kode / Desc Diagnose"
            value={text}
            disabled={disabled}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                setTerm(text.trim())
              }
            }}
          />
        </div>
        <Button tone="kedua" disabled={disabled || text.trim() === ''} onClick={() => setTerm(text.trim())}>
          Cari
        </Button>
      </div>
      {term !== '' && (
        <div className="mt-2">
          <DataTable<DiagnosisOption>
            label="Hasil pencarian diagnosa"
            hideSearch
            showHeaderWhenEmpty
            isLoading={search.isPending}
            error={search.isError ? 'Kode diagnosa tidak dapat dicari.' : undefined}
            emptyMessage="Kode diagnosa tidak ditemukan."
            rows={search.data?.pilihan ?? []}
            rowKey={(d) => `${d.kode}|${d.deskripsi}`}
            columns={[
              { key: 'kode', title: 'Kode Diagnose', value: (d) => d.kode },
              { key: 'deskripsi', title: 'Desc Diagnose', value: (d) => d.deskripsi },
              {
                key: 'pilih',
                title: '',
                noSort: true,
                alignRight: true,
                value: () => '',
                render: (d) => (
                  <Button tone="kedua" disabled={disabled} onClick={() => onPick(d)}>
                    Pilih
                  </Button>
                ),
              },
            ]}
          />
        </div>
      )}
    </div>
  )
}
