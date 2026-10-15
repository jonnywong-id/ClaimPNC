import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type MaskingBulkRow } from '@/api/types'
import { Field } from '@/components/Field'
import { NumberField } from '@/components/NumberField'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Button } from '@/components/Button'

import { useBranchOptions, useOperators, useSaveMaskingBulk } from './api'
import { ModuleChecklist } from './ModuleChecklist'

/** Batas isian; sama dengan konstanta di domain Go. */
const MAX_QUOTA = 1_000_000

/**
 * Satu baris Template Akses yang sedang diisi.
 *
 * `dipakai` menandai baris yang ikut disimpan. Ia ada karena tabelnya menampilkan SELURUH
 * petugas cabang — di portal ASM cabang terbesar memuat 30 orang — dan petugas hanya
 * memberi kewenangan kepada sebagian.
 */
type DraftRow = MaskingBulkRow & { dipakai: boolean; nama: string }

function emptyRow(login: string, nama: string): DraftRow {
  return {
    dipakai: false,
    login,
    nama,
    // Satu-satunya nilai MODUL yang ada di data produksi. Ia SARAN, bukan batasan —
    // daftar pilihannya di Pega dipasok rule yang hilang dari export (`R-16`).
    modul: 'PNCSearchKlaim',
    sub_modul: '',
    maks_cari: 0,
    maks_lihat: 0,
    lihat_ktp: false,
    lihat_email: false,
    lihat_notelp: false,
  }
}

type Props = { onClose: () => void }

/**
 * Form Tambah Master Masking — berbentuk DAFTAR, bukan satu baris.
 *
 * Inilah bentuk form "INPUT DATA MASKING" di layar lama, dan ia sengaja berbeda dari form
 * Ubah:
 *
 *	CABANG *                       satu pilihan, wajib
 *	┌──────────────────────────────────────────────┐
 *	│ NAMA USER │ LOGIN │ TEMPLATE AKSES            │
 *	└──────────────────────────────────────────────┘
 *	SIMPAN
 *
 * Alurnya: pilih cabang → tabel terisi petugas cabang itu
 * (`RDB List/GetDataLogin-SQL.xml`, disaring `PYPOSITION` ke empat jabatan) → isi Template
 * Akses untuk petugas yang dikehendaki → satu tombol SIMPAN menyimpan seluruhnya.
 *
 * Penyimpanannya berulang per baris, sama seperti
 * `Activity/InsermaskingDataKlaimPnc_-Act.xml` yang mengulang `TempLogin.pxResults` —
 * sehingga sebagian baris dapat tersimpan meski sebagian lain ditolak.
 *
 * # Kenapa bukan satu form per orang
 *
 * Karena bukan itu yang dikerjakan petugas. Kewenangan masking diberikan per CABANG untuk
 * beberapa orang sekaligus; memaksanya satu per satu berarti mengulang pemilihan cabang
 * dan pengisian modul yang sama tiga puluh kali.
 */
export function MaskingAddForm({ onClose }: Props) {
  const save = useSaveMaskingBulk()

  const [branchKeyword, setBranchKeyword] = useState('')
  const [branchID, setBranchID] = useState('')
  const [branchName, setBranchName] = useState('')
  const branches = useBranchOptions(branchKeyword)

  const operators = useOperators(branchID)
  const [row, setRow] = useState<DraftRow[]>([])
  const [touchedBranch, setTouchedBranch] = useState('')

  // Tabel disusun ulang setiap cabang berganti. Dikerjakan saat render, bukan di efek,
  // supaya isian lama milik cabang sebelumnya tidak sempat terlihat di cabang baru.
  if (operators.data && touchedBranch !== branchID) {
    setTouchedBranch(branchID)
    setRow(operators.data.pengguna.map((p) => emptyRow(p.login, p.nama)))
  }

  function patch(login: string, change: Partial<DraftRow>) {
    setRow((current) => current.map((r) => (r.login === login ? { ...r, ...change } : r)))
  }

  const chosen = row.filter((r) => r.dipakai)

  function send() {
    save.mutate(
      {
        cabang: branchID,
        baris: chosen.map(({ dipakai: _dipakai, nama: _nama, ...rest }) => rest),
      },
      {
        onSuccess: (hasil) => {
          // Form ditutup HANYA bila seluruh baris tersimpan.
          //
          // Bila sebagian ditolak, ia dibiarkan terbuka beserta laporan per barisnya —
          // menutupnya akan membuang satu-satunya tempat petugas dapat membaca baris mana
          // yang gagal, dan isian yang sudah ia ketik ikut hilang.
          if (hasil.ditolak === 0) onClose()
        },
      },
    )
  }

  const result = save.data

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault()
        send()
      }}
      noValidate
      className="overflow-hidden rounded-kartu border border-slate-200 border-l-4 border-l-blue-500 bg-white shadow-angkat"
      aria-label="Input data masking"
    >
      <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
        <h3 className="text-base font-semibold text-slate-900">Input Data Masking</h3>
        <p className="mt-1 text-sm text-slate-600">
          Pilih cabang, lalu isi Template Akses untuk petugas yang diberi kewenangan.
        </p>
      </div>

      <div className="space-y-5 p-5">
        {save.isError && <SaveErrorMessage error={save.error} />}

        {/*
          Hasil per baris dilaporkan apa adanya — sebagian dapat tersimpan dan sebagian
          ditolak, persis seperti perulangan di sistem lama.

          Ia digambar sebagai panel tersendiri, bukan lewat ErrorMessage: keberhasilan
          sebagian bukan galat, dan menampilkannya bernada galat akan membuat petugas
          mengira tidak ada yang tersimpan padahal sebagian sudah.
        */}
        {result && (
          <div
            role="status"
            className={`rounded-kartu border px-4 py-3 text-sm ${
              result.ditolak === 0
                ? 'border-emerald-200 bg-emerald-50 text-emerald-900'
                : 'border-amber-200 bg-amber-50 text-amber-900'
            }`}
          >
            <p className="font-medium">
              {result.tersimpan} tersimpan, {result.ditolak} ditolak
            </p>
            {result.ditolak > 0 && (
              <ul className="mt-2 space-y-1 text-xs">
                {result.hasil
                  .filter((h) => !h.tersimpan)
                  .map((h) => (
                    <li key={h.login}>
                      <span className="font-mono">{h.login}</span> — {h.pesan}
                    </li>
                  ))}
              </ul>
            )}
          </div>
        )}

        <div className="max-w-md">
          <Field
            id="cariCabang"
            label="Cabang"
            placeholder="Ketik nama atau kode cabang"
            value={branchKeyword}
            onChange={(event) => setBranchKeyword(event.target.value)}
            hint={branchID ? `Terpilih: ${branchName} (${branchID})` : 'Wajib dipilih.'}
          />
          <div className="mt-2 max-h-40 overflow-y-auto rounded-kontrol border border-slate-200">
            {branches.isPending && (
              <p className="px-3 py-2 text-xs text-slate-500">Memuat cabang…</p>
            )}
            {branches.data?.cabang.map((branch) => (
              <button
                key={branch.kode}
                type="button"
                onClick={() => {
                  setBranchID(branch.kode)
                  setBranchName(branch.nama)
                }}
                className={`flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-sm hover:bg-slate-50 ${
                  branchID === branch.kode
                    ? 'bg-blue-50 font-medium text-blue-800'
                    : 'text-slate-700'
                }`}
              >
                <span>{branch.nama}</span>
                <span className="font-mono text-xs text-slate-500">{branch.kode}</span>
              </button>
            ))}
          </div>
        </div>

        {/* Tabel petugas. Kolomnya sama dengan layar lama: NAMA USER, LOGIN, TEMPLATE
            AKSES. */}
        <div className="overflow-hidden rounded-kartu border border-slate-200">
          <table className="w-full text-left text-sm">
            <thead className="bg-slate-50 text-xs font-semibold uppercase text-slate-600">
              <tr>
                <th scope="col" className="w-12 px-3 py-2"></th>
                <th scope="col" className="px-3 py-2">
                  Nama User
                </th>
                <th scope="col" className="px-3 py-2">
                  Login
                </th>
                <th scope="col" className="px-3 py-2">
                  Template Akses
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {branchID === '' && (
                <tr>
                  <td colSpan={4} className="px-3 py-6 text-center text-sm text-slate-500">
                    Pilih cabang lebih dulu.
                  </td>
                </tr>
              )}
              {branchID !== '' && operators.isPending && (
                <tr>
                  <td colSpan={4} className="px-3 py-6 text-center text-sm text-slate-500">
                    Memuat petugas…
                  </td>
                </tr>
              )}
              {branchID !== '' && operators.data && row.length === 0 && (
                <tr>
                  <td colSpan={4} className="px-3 py-6 text-center text-sm text-slate-500">
                    Data Tidak Ada
                  </td>
                </tr>
              )}
              {row.map((r) => (
                <tr key={r.login} className={r.dipakai ? 'bg-blue-50/30' : undefined}>
                  <td className="px-3 py-3 align-top">
                    <input
                      type="checkbox"
                      aria-label={`Beri kewenangan kepada ${r.login}`}
                      className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-2 focus:ring-blue-500/30"
                      checked={r.dipakai}
                      onChange={(event) => patch(r.login, { dipakai: event.target.checked })}
                    />
                  </td>
                  <td className="px-3 py-3 align-top text-slate-900">{r.nama}</td>
                  <td className="px-3 py-3 align-top font-mono text-xs text-slate-700">
                    {r.login}
                  </td>
                  <td className="px-3 py-3">
                    <TemplateAkses row={r} disabled={!r.dipakai} onChange={patch} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="flex items-center justify-between gap-2 border-t border-slate-100 bg-slate-50/70 px-5 py-4">
        <p className="text-xs text-slate-600">
          {chosen.length === 0
            ? 'Belum ada petugas yang dipilih.'
            : `${chosen.length} petugas akan disimpan.`}
        </p>
        <div className="flex gap-2">
          <Button tone="kedua" onClick={onClose} disabled={save.isPending}>
            Batal
          </Button>
          <Button
            tone="utama"
            type="submit"
            disabled={save.isPending || branchID === '' || chosen.length === 0}
          >
            {save.isPending ? 'Menyimpan…' : 'Simpan'}
          </Button>
        </div>
      </div>
    </form>
  )
}

/**
 * Template Akses satu petugas.
 *
 * Isinya sepadan dengan section `TemplateAksesMasking` pada layar lama: MODUL, SUB MODUL,
 * MAX CARI DATA, MAX LIHAT DATA, dan ketiga kewenangan melihat.
 *
 * Seluruh isian dinonaktifkan sampai barisnya dicentang — supaya terbaca jelas bahwa baris
 * yang tidak dicentang tidak akan disimpan, bukan disimpan dengan nilai kosong.
 */
function TemplateAkses({
  row,
  disabled,
  onChange,
}: {
  row: DraftRow
  disabled: boolean
  onChange: (login: string, change: Partial<DraftRow>) => void
}) {
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {/* MODUL dan SUB MODUL berupa KOTAK CENTANG, sama seperti kolom TEMPLATE AKSES di
          layar lama. */}
      <div className="sm:col-span-2">
        <ModuleChecklist
          idPrefix={row.login}
          modul={row.modul}
          subModul={row.sub_modul}
          disabled={disabled}
          onChange={(change) => onChange(row.login, change)}
        />
      </div>
      <NumberField
        id={`makscari-${row.login}`}
        label="Max Cari Data"
        max={MAX_QUOTA}
        disabled={disabled}
        value={row.maks_cari}
        onValueChange={(maks_cari) => onChange(row.login, { maks_cari })}
      />
      <NumberField
        id={`makslihat-${row.login}`}
        label="Max Lihat Data"
        max={MAX_QUOTA}
        disabled={disabled}
        value={row.maks_lihat}
        onValueChange={(maks_lihat) => onChange(row.login, { maks_lihat })}
      />

      <fieldset className="sm:col-span-2">
        <legend className="text-xs font-medium text-slate-700">
          Boleh melihat data pribadi tanpa disamarkan
        </legend>
        <div className="mt-1.5 flex flex-wrap gap-3">
          {(
            [
              ['lihat_ktp', 'KTP'],
              ['lihat_email', 'Email'],
              ['lihat_notelp', 'Notelp'],
            ] as const
          ).map(([key, label]) => (
            <label
              key={key}
              className={`flex items-center gap-2 text-sm ${
                disabled ? 'text-slate-400' : 'text-slate-700'
              }`}
            >
              <input
                type="checkbox"
                disabled={disabled}
                className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-2 focus:ring-blue-500/30"
                checked={row[key]}
                onChange={(event) => onChange(row.login, { [key]: event.target.checked })}
                aria-label={`${label} untuk ${row.login}`}
              />
              {label}
            </label>
          ))}
        </div>
      </fieldset>
    </div>
  )
}

function SaveErrorMessage({ error }: { error: unknown }) {
  const message = saveMessage(error)
  return <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
}

function saveMessage(error: unknown): { title: string; description: string; tone: ErrorTone } {
  if (error instanceof NetworkError) {
    return {
      title: 'Tidak dapat menghubungi server',
      description: 'Data belum tersimpan. Periksa koneksi lalu tekan Simpan lagi.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.maskingNoRowChosen:
        return {
          title: 'Belum ada petugas yang dipilih',
          description: 'Centang minimal satu petugas lalu isi Template Akses-nya.',
          tone: 'penolakan',
        }
      case ErrorCode.maskingBranchUnknown:
        return {
          title: 'Cabang tidak dikenal',
          description: 'Pilih cabang dari daftar, jangan mengetik kodenya sendiri.',
          tone: 'penolakan',
        }
      default:
        return { title: 'Gagal menyimpan', description: error.message, tone: 'gangguan' }
    }
  }
  return {
    title: 'Gagal menyimpan',
    description: 'Terjadi kesalahan pada sistem. Coba simpan lagi.',
    tone: 'gangguan',
  }
}
