import { useId, useState } from 'react'

import type {
  CauseOfLossActiveOption,
  CauseOfLossBusiness,
  CauseOfLossDetail,
} from '@/api/types'
import { Button } from '@/components/Button'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

import { useCauseOfLossMasterSearch } from './api'
import { BusinessPicker } from './BusinessPicker'

/** Sama dengan `detailpenyebab.MinLookupKeyword` di backend. */
const MIN_KEYWORD = 2

export type DetailFormValues = {
  id_lama: string
  id_master: string
  nama_master: string
  deskripsi_kerugian: string
  kode_kehilangan: string
  status_aktif: string
  bisnis: CauseOfLossBusiness[]
}

type Props = {
  /** Baris yang sedang disunting; null berarti menambah baris baru. */
  existing: CauseOfLossDetail | null
  options: CauseOfLossActiveOption[]
  /** Pesan galat per isian, dari jawaban 422. */
  fieldError: Record<string, string>
  busy: boolean
  onSubmit: (values: DetailFormValues) => void
  onCancel: () => void
}

/** Nilai awal form — dari baris yang disunting, atau kosong untuk baris baru. */
function initialValues(existing: CauseOfLossDetail | null): DetailFormValues {
  if (existing === null) {
    return {
      id_lama: '',
      id_master: '',
      nama_master: '',
      deskripsi_kerugian: '',
      kode_kehilangan: '',
      // Baris baru diawali Aktif. Layar lama tidak menentukan nilai awal apa pun —
      // isiannya lahir kosong — tetapi baris yang baru dibuat memang dimaksudkan berlaku,
      // dan membiarkannya kosong membuat setiap baris baru lahir "Belum diisi".
      //
      // Ini SELISIH TERENCANA, dan satu-satunya yang diambil atas nilai awal.
      status_aktif: '1',
      bisnis: [],
    }
  }
  return {
    id_lama: existing.id_lama,
    id_master: existing.id_master,
    nama_master: existing.nama_master,
    deskripsi_kerugian: existing.deskripsi_kerugian,
    kode_kehilangan: existing.kode_kehilangan,
    status_aktif: existing.status_aktif,
    bisnis: existing.bisnis,
  }
}

/**
 * Form Detail Penyebab Kerugian — padanan panel "Memperbaharui Data" di layar lama.
 *
 * # Kelima isiannya, dan asal masing-masing
 *
 * Seluruhnya dibaca dari `Section/BrowseDetailCauseOfLoss-Section.xml`:
 *
 *	ID                    `.D_COL_ID`     :1475  pxTextInput, pyReadOnly=true
 *	ID Master Kerugian    `.M_COL_ID`     :1631  autocomplete, pyPrompt=.COL_DESC
 *	Deskripsi Kerugian    `.DESCRIPTION`  :1954  pxTextInput
 *	Status Aktif          `.STS_AKTIF`    :2123  pxDropdown
 *	Kode Kehilangan       `.LOSS_CODE`    :2356  pxTextInput
 *	Bisnis                `.BISNISID`     :3251  repeating grid + autocomplete
 *
 * # TIDAK ADA satu pun isian wajib, dan itu bukan kelalaian
 *
 * `Activity/CNMInsertDetailCauseOfLoss_act` tidak memeriksa apa pun — nol
 * `Property-Set-Messages`, nol precondition, dan nol isian ber-`pyRequired=true`. Baris
 * yang seluruh isiannya kosong pun tersimpan.
 *
 * Perilakunya ditiru (`P-5`). Preseden di aplikasi ini sudah ada: pada Master Pasal
 * Kerugian, tiga aturan yang diusulkan ditolak Work Owner pada 2026-09-19 dengan jawaban
 * "coba jalankan secara as is".
 *
 * Yang dipasang di sini hanyalah satu penjagaan yang TIDAK menolak isian pengguna: Status
 * Aktif berupa dropdown, sehingga nilai di luar kedua pilihannya tidak dapat dikirim dari
 * layar ini.
 *
 * # ID tidak dapat diketik, dan itu ditampilkan apa adanya
 *
 * Ia diterbitkan sequence `D_CAUSE_SEQ` dengan kode situs di depannya
 * (`Database/PEGA_D_CAUSE_OF_LOSS.prc:19`). Pada baris baru ia belum ada, dan isiannya
 * menyatakan demikian alih-alih menampilkan kotak kosong yang tampak dapat diisi.
 */
export function DetailForm({
  existing,
  options,
  fieldError,
  busy,
  onSubmit,
  onCancel,
}: Props) {
  const idField = useId()
  const [values, setValues] = useState<DetailFormValues>(() => initialValues(existing))

  const [masterKeyword, setMasterKeyword] = useState('')
  const [masterOpen, setMasterOpen] = useState(false)
  const masterLookup = useCauseOfLossMasterSearch(masterKeyword)

  const cleanMasterKeyword = masterKeyword.trim()
  const shortMasterKeyword =
    cleanMasterKeyword.length > 0 && cleanMasterKeyword.length < MIN_KEYWORD

  function change<K extends keyof DetailFormValues>(key: K, value: DetailFormValues[K]) {
    setValues((current) => ({ ...current, [key]: value }))
  }

  return (
    <form
      className="space-y-4"
      onSubmit={(e) => {
        e.preventDefault()
        onSubmit(values)
      }}
    >
      {/*
        ID hanya ditampilkan. Pada baris baru belum ada nilainya sama sekali — dan
        menampilkan kotak kosong yang tampak dapat diisi justru menyesatkan.
      */}
      <div>
        <span className="block text-sm font-medium text-slate-700">ID</span>
        <p className="mt-1.5 rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-900">
          {existing === null ? (
            <span className="text-slate-500">Diterbitkan sistem setelah disimpan.</span>
          ) : (
            existing.id
          )}
        </p>
      </div>

      {/* ID Master Kerugian — autocomplete ke POOLDATA.V_M_CAUSE_OF_LOSS. */}
      <div>
        <Field
          id={`${idField}-master`}
          label="ID Master Kerugian"
          type="search"
          autoComplete="off"
          placeholder="Cari master penyebab kerugian…"
          value={masterKeyword}
          error={fieldError['id_master']}
          hint={
            values.id_master === ''
              ? `Ketik minimal ${MIN_KEYWORD} huruf, lalu pilih dari daftar. Boleh dikosongkan.`
              : undefined
          }
          onChange={(e) => {
            setMasterKeyword(e.target.value)
            setMasterOpen(true)
          }}
          onFocus={() => setMasterOpen(true)}
        />

        {values.id_master !== '' && (
          <p className="mt-1.5 flex flex-wrap items-center gap-2 rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-2 text-sm">
            <span className="min-w-0 flex-1 text-slate-900">
              {values.nama_master === '' ? (
                <>
                  <span className="text-slate-400">(sebutan tidak ditemukan)</span>
                  <span
                    className="ml-2 rounded-full bg-amber-50 px-2 py-0.5 text-xs text-amber-800 ring-1 ring-amber-200"
                    title="Kode induk ini tidak ada di master penyebab kerugian portal ini."
                  >
                    induk tidak ada
                  </span>
                </>
              ) : (
                values.nama_master
              )}
              <span className="ml-2 text-xs text-slate-500">{values.id_master}</span>
            </span>
            <Button
              tone="halus"
              onClick={() => {
                change('id_master', '')
                change('nama_master', '')
              }}
            >
              Kosongkan
            </Button>
          </p>
        )}

        {masterOpen && !shortMasterKeyword && cleanMasterKeyword !== '' && (
          <div className="mt-2 max-h-56 overflow-y-auto rounded-kontrol border border-slate-200 bg-white shadow-lembut">
            {masterLookup.isError ? (
              <p className="px-3 py-3 text-sm text-red-700" role="alert">
                Pencarian gagal. Coba beberapa saat lagi.
              </p>
            ) : masterLookup.isFetching ? (
              <p className="px-3 py-3 text-sm text-slate-500">Mencari…</p>
            ) : (masterLookup.data?.master ?? []).length === 0 ? (
              <p className="px-3 py-3 text-sm text-slate-500">
                Tidak ada master penyebab kerugian yang cocok.
              </p>
            ) : (
              <ul>
                {(masterLookup.data?.master ?? []).map((option) => (
                  <li key={option.id}>
                    <button
                      type="button"
                      onClick={() => {
                        setValues((current) => ({
                          ...current,
                          id_master: option.id,
                          nama_master: option.nama,
                        }))
                        setMasterKeyword('')
                        setMasterOpen(false)
                      }}
                      className={[
                        'flex w-full items-baseline gap-2 px-3 py-2 text-left text-sm',
                        'transition-colors duration-150 ease-halus',
                        'hover:bg-blue-50 focus:bg-blue-50 focus:outline-none',
                      ].join(' ')}
                    >
                      <span className="min-w-0 flex-1 text-slate-900">{option.nama}</span>
                      <span className="shrink-0 text-xs text-slate-500">{option.id}</span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>

      <Field
        id={`${idField}-deskripsi`}
        label="Deskripsi Kerugian"
        value={values.deskripsi_kerugian}
        error={fieldError['deskripsi_kerugian']}
        onChange={(e) => change('deskripsi_kerugian', e.target.value)}
      />

      <Field
        id={`${idField}-kode`}
        label="Kode Kehilangan"
        value={values.kode_kehilangan}
        error={fieldError['kode_kehilangan']}
        onChange={(e) => change('kode_kehilangan', e.target.value)}
      />

      <SelectField
        id={`${idField}-status`}
        label="Status Aktif"
        value={values.status_aktif}
        options={options.map((option) => ({ value: option.kode, label: option.label }))}
        error={fieldError['status_aktif']}
        // Baris lama dapat memuat Status Aktif kosong — kolomnya tidak punya constraint
        // NOT NULL yang diketahui (`R-08`). Pilihan kosong dibiarkan agar baris semacam itu
        // dapat disimpan ulang tanpa dipaksa memilih, yang berarti mengubah data yang tidak
        // diminta siapa pun untuk diubah.
        emptyText="— belum diisi —"
        onChange={(e) => change('status_aktif', e.target.value)}
      />

      <BusinessPicker
        value={values.bisnis}
        onChange={(bisnis) => change('bisnis', bisnis)}
        disabled={busy}
      />

      <div className="flex flex-wrap gap-2 pt-2">
        <Button type="submit" tone="utama" disabled={busy}>
          {busy ? 'Menyimpan…' : 'Simpan'}
        </Button>
        <Button onClick={onCancel} disabled={busy}>
          Batal
        </Button>
      </div>
    </form>
  )
}
