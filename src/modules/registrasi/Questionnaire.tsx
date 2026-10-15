import { useState } from 'react'

const UNSAVED_NOTE = 'Not saved yet: no column for this field in T_CLAIM_PNC.'

/** Isian `Section/QuestionnaireClaim` — `.ClaimData.QuestionnaireData.*`, tanpa kondisi tampil. */
const FIELDS = [
  { id: 'interest_name', label: 'Nama Kepentingan', kind: 'textarea' },
  { id: 'airline_insurance', label: 'Maskapai Asuransi', kind: 'textarea' },
  { id: 'premium_paid', label: 'Premi Dibayar', kind: 'radio' },
  { id: 'policy_cover', label: 'Polis Dibayar', kind: 'textarea' },
  { id: 'underwriting_action', label: 'Tindakan Underwriting', kind: 'textarea' },
  { id: 'loss_protection', label: 'Hilangnya Perlindungan', kind: 'radio' },
] as const

type FieldID = (typeof FIELDS)[number]['id']

/**
 * Tab Kuisioner — `Section/QuestionnaireClaim` (`!IsTravel`). Keenam isiannya disimpan Pega di
 * `ClaimData.QuestionnaireData` (dokumen kasus), dan T_CLAIM_PNC tidak punya kolom untuknya, sehingga
 * di sini tampil tetapi belum tersimpan.
 *
 * Premi Dibayar dan Hilangnya Perlindungan adalah radio button (`pxRadioButtons`) yang pilihannya
 * berasal dari definisi properti `PremiumPaid` / `LossProtection` — tidak ada di export. Sampai
 * diterima, keduanya isian teks.
 */
export function QuestionnaireTab() {
  const [values, setValues] = useState<Record<FieldID, string>>({
    interest_name: '', airline_insurance: '', premium_paid: '', policy_cover: '', underwriting_action: '', loss_protection: '',
  })
  const set = (id: FieldID, value: string) => setValues((current) => ({ ...current, [id]: value }))
  const className =
    'mt-1 w-full rounded border border-dashed border-slate-300 px-3 py-2 text-slate-900 focus:border-slate-500 focus:outline-none'

  return (
    <section className="mt-3 rounded border border-slate-200 p-4" aria-label="Kuisioner">
      <div className="grid gap-4 sm:grid-cols-2">
        {FIELDS.map((f) => (
          <div key={f.id}>
            <label htmlFor={`kuisioner_${f.id}`} className="block text-sm font-medium text-slate-700">{f.label}</label>
            {f.kind === 'textarea' ? (
              <textarea id={`kuisioner_${f.id}`} rows={3} className={className} value={values[f.id]}
                onChange={(e) => set(f.id, e.target.value)} />
            ) : (
              <input id={`kuisioner_${f.id}`} className={className} value={values[f.id]}
                onChange={(e) => set(f.id, e.target.value)} />
            )}
            <p className="mt-1 text-xs text-amber-700">
              {UNSAVED_NOTE}
              {f.kind === 'radio' && ' The radio options (property rule) are not in the Pega export.'}
            </p>
          </div>
        ))}
      </div>
    </section>
  )
}
