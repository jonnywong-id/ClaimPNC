import { useState, type ReactNode } from 'react'

import { useSession } from '@/app/session'

import { DocumentTab, ProgressTab } from './EstimateTabs'
import { QuestionnaireTab } from './Questionnaire'
import { TabList, ownerNotice } from './SurveyorForm'
import type { Claim, Task } from './types'

/** Group Panel Travel — When `IsTravel`. */
const PANEL_TRAVEL = '005'

const NOT_BUILT = 'Proses tombol ini belum dibangun.'

const TABS = ['Register', 'Kuisioner', 'Unggah Dokumen', 'Progress Claim & Komunikasi'] as const
type Tab = (typeof TABS)[number]

/**
 * Bingkai tahap Input Register — flow action InputRegister, `Section/InputRegister-sect.xml`:
 *
 *   Register                      !IsPNCReceive && TempData.StatusClaim != 1   InputRegisterDetail
 *   Kuisioner                     !IsTravel && !IsPNCReceive && ...            QuestionnaireClaim
 *   Unggah Dokumen                !IsTravel && ...                             UploadDocument
 *   Progress Claim & Komunikasi   TempData.StatusClaim != 1                    PNCProgressKomunikasi_Sec
 *
 * Berlaku untuk semua lini (Work Owner, 2026-10-03), menggantikan bingkai ClaimSurvey_sect.
 * `IsPNCReceive` (peran penerima dokumen) tidak membuka tahap ini, sehingga tidak diuji di sini.
 *
 * Tab Register memuat formulir `InputRegisterDetail` (FormRegister); tab Kuisioner `QuestionnaireClaim`.
 * Keduanya tetap terpasang saat berpindah tab, supaya isian yang belum disimpan tidak hilang.
 */
export function InputRegisterFrame({ klaim, tugas, register }: Readonly<{ klaim: Claim; tugas: Task; register: ReactNode }>) {
  const identity = useSession((state) => state.user?.identitas ?? '')
  const notice = ownerNotice(tugas, identity, true)
  const travel = klaim.polis.lini === PANEL_TRAVEL
  const tabs = TABS.filter((t) => !travel || (t !== 'Kuisioner' && t !== 'Unggah Dokumen'))
  const [tab, setTab] = useState<Tab>('Register')

  return (
    <section className="mt-6 rounded border border-slate-200 p-4" aria-label="InputRegister">
      <h2 className="text-sm text-slate-700">InputRegister</h2>
      {notice && (
        <p className="mt-2 rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900" role="status">
          {notice} Layar ini dapat dibaca, tetapi isiannya tidak dapat disimpan.
        </p>
      )}

      <TabList items={tabs} current={tab} onSelect={setTab} label="Tab InputRegister" />

      <div hidden={tab !== 'Register'}>
        <div className="mt-3 flex flex-wrap justify-end gap-2">
          {['Detail Premi', 'Detail Polis', 'Riwayat Klaim'].map((label) => (
            <button
              key={label}
              type="button"
              disabled
              title={NOT_BUILT}
              className="rounded border border-blue-300 px-2 py-1 text-sm text-blue-700 disabled:opacity-60"
            >
              {label}
            </button>
          ))}
        </div>
        {register}
      </div>
      {!travel && (
        <div hidden={tab !== 'Kuisioner'}>
          <QuestionnaireTab />
        </div>
      )}
      {tab === 'Unggah Dokumen' && <DocumentTab claimID={klaim.id} line={klaim.polis.lini} />}
      {tab === 'Progress Claim & Komunikasi' && <ProgressTab claimID={klaim.id} />}
    </section>
  )
}
