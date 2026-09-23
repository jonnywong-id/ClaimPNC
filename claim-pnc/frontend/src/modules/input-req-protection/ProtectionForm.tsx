import { useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { FormField } from '@/components/FormField'
import { SelectField, type SelectOption } from '@/components/SelectField'
import { TextAreaField } from '@/components/TextAreaField'

import {
  PROTECTION_TYPE_LABEL,
  TYPE_CHANGE_CAUSE_OF_LOSS,
  TYPE_CHANGE_LOSS_DATE,
  type ChangeDetail,
  type ProtectionDetail,
  type ProtectionFields,
} from './types'

/**
 * Form Input Open Protection.
 *
 * Migrasi dari **`Section/InputProtectionSection-Section.xml`**, section yang dimuat flow
 * action `InputProtectionFlow` pada langkah pertama `CreateProtection_Flow`.
 *
 * | Isian | Asal di Pega | Wajib |
 * |---|---|---|
 * | No Polis | `.PolicyNo` | ya |
 * | No Klaim | `.CaseID` | ya |
 * | Tipe Proteksi | `.TypeProtection` | ya |
 * | Keterangan | `.Keterangan` | ya |
 * | Current / Next Date Of Loss | `.ClaimDataProtect.BeforeDateOfLoss` / `.DateOfLoss` | tipe `7` |
 * | Cause of Loss | `.ClaimDataProtect.CauseOfLossID` / `.IDMasterTONP` | tipe `8` |
 *
 * Keempat isian wajib itu berasal dari prakondisi
 * `Activity/InsertOpenProtectionCase-Act.xml:921`.
 *
 * # Panel detail perubahan muncul mengikuti tipe
 *
 * `.TypeProtection==7` memunculkan "Detail Perubahan DOL" (`:2728`), `==8` memunculkan
 * "Detail Perubahan Cause Of Loss" (`:3638`). Keduanya saling meniadakan, persis seperti di
 * layar lama.
 *
 * # Validasi tidak diulang di sini
 *
 * Aturan wajib-isi ditegakkan BACKEND, dan pesannya ditempelkan ke kolomnya lewat
 * `APIError.violations()`. Menyalinnya ke sini akan membuat aturan hidup di dua tempat —
 * dan yang di peramban tidak dapat diuji bersama aturan lainnya.
 *
 * Yang dikerjakan layar hanyalah menandai isian wajib dengan `required`, supaya peramban
 * memberi tahu lebih awal tanpa perjalanan jaringan.
 */
type Props = {
  /** Terisi saat menyunting; kosong saat membuat baru. */
  existing?: ProtectionDetail | undefined

  /** Daftar tipe proteksi yang dapat dipilih. */
  typeOptions: SelectOption[]

  onSubmit: (values: ProtectionFields) => Promise<void>
  onCancel: () => void

  isSubmitting: boolean

  /** Galat dari backend; pelanggaran per kolom dibaca dari sini. */
  failure: unknown
}

const EMPTY_CHANGE_DETAIL: ChangeDetail = {
  dol_sebelum: '',
  dol_baru: '',
  penyebab_kerugian: '',
  penyebab_kerugian_master: '',
  nama_objek: '',
  nama_cabang: '',
}

export function ProtectionForm({
  existing,
  typeOptions,
  onSubmit,
  onCancel,
  isSubmitting,
  failure,
}: Props) {
  const [policyNumber, setPolicyNumber] = useState(existing?.nomor_polis ?? '')
  const [claimNumber, setClaimNumber] = useState(existing?.nomor_klaim ?? '')
  const [claimReference, setClaimReference] = useState(existing?.referensi_klaim ?? '')
  const [type, setType] = useState(existing?.tipe_proteksi ?? '')
  const [note, setNote] = useState(existing?.keterangan ?? '')
  const [detail, setDetail] = useState<ChangeDetail>(
    existing?.detail_perubahan ?? EMPTY_CHANGE_DETAIL,
  )

  const violations = failure instanceof APIError ? failure.violations() : {}

  // Galat yang TIDAK menunjuk kolom mana pun ditampilkan di atas form. Yang menunjuk kolom
  // sudah menempel di kolomnya, dan mengulangnya di atas hanya menambah kebisingan.
  const generalFailure =
    failure instanceof APIError && Object.keys(violations).length === 0 ? failure.message : null

  const changesLossDate = type === TYPE_CHANGE_LOSS_DATE
  const changesCauseOfLoss = type === TYPE_CHANGE_CAUSE_OF_LOSS

  function patchDetail(patch: Partial<ChangeDetail>) {
    setDetail((current) => ({ ...current, ...patch }))
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    await onSubmit({
      nomor_polis: policyNumber,
      nomor_klaim: claimNumber,
      referensi_klaim: claimReference,
      tipe_proteksi: type,
      keterangan: note,
      detail_perubahan: detail,
    })
  }

  return (
    <form onSubmit={(event) => void submit(event)} className="space-y-5">
      {generalFailure && (
        <ErrorMessage
          title="Permintaan tidak dapat disimpan"
          description={generalFailure}
          tone="penolakan"
        />
      )}

      <div className="grid gap-4 sm:grid-cols-2">
        <FormField
          id="nomor-polis"
          label="No Polis"
          value={policyNumber}
          onChange={(event) => setPolicyNumber(event.target.value)}
          required
          autoComplete="off"
          {...(violations['no_polis'] ? { failure: violations['no_polis'] } : {})}
        />

        <SelectField
          id="tipe-proteksi"
          label="Tipe Proteksi"
          options={typeOptions}
          value={type}
          onChange={(event) => setType(event.target.value)}
          required
          emptyText="--- Pilih ---"
          {...(violations['tipe_proteksi'] ? { error: violations['tipe_proteksi'] } : {})}
        />
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <FormField
          id="nomor-klaim"
          label="No Klaim"
          value={claimNumber}
          onChange={(event) => setClaimNumber(event.target.value)}
          required
          autoComplete="off"
          {...(violations['no_klaim'] ? { failure: violations['no_klaim'] } : {})}
        />

        {/*
          Referensi klaim adalah hasil PENCARIAN, bukan ketikan bebas.
          `Activity/ValidationInputProtection-Act.xml` menolak penyimpanan saat ia kosong,
          dengan pesan "Silakan Tulis dan Cari Ulang No Klaim".

          Pencarian klaimnya sendiri BELUM ADA di sini: ia menembak modul klaim, yang
          rutenya belum tersedia. Sampai itu ada, isian ini diisi manual dan keterbatasan
          itu dinyatakan di layar — bukan disembunyikan dengan tombol cari yang tidak
          berfungsi.
        */}
        <FormField
          id="referensi-klaim"
          label="Referensi Klaim (hasil pencarian)"
          value={claimReference}
          onChange={(event) => setClaimReference(event.target.value)}
          autoComplete="off"
          {...(violations['no_klaim'] ? { failure: violations['no_klaim'] } : {})}
        />
      </div>

      {changesLossDate && (
        <fieldset className="rounded-lg border border-slate-200 p-4">
          <legend className="px-1 text-sm font-medium text-slate-700">
            Detail Perubahan DOL
          </legend>

          <div className="grid gap-4 sm:grid-cols-2">
            <FormField
              id="dol-sebelum"
              label="Current Date Of Loss"
              type="date"
              value={detail.dol_sebelum}
              onChange={(event) => patchDetail({ dol_sebelum: event.target.value })}
            />
            <FormField
              id="dol-baru"
              label="Next Date Of Loss"
              type="date"
              value={detail.dol_baru}
              onChange={(event) => patchDetail({ dol_baru: event.target.value })}
              required
              {...(violations['dol_baru'] ? { failure: violations['dol_baru'] } : {})}
            />
          </div>
        </fieldset>
      )}

      {changesCauseOfLoss && (
        <fieldset className="rounded-lg border border-slate-200 p-4">
          <legend className="px-1 text-sm font-medium text-slate-700">
            Detail Perubahan Cause Of Loss
          </legend>

          <div className="grid gap-4 sm:grid-cols-2">
            <FormField
              id="penyebab-kerugian"
              label="Cause Of Loss Sebelumnya"
              value={detail.penyebab_kerugian}
              onChange={(event) => patchDetail({ penyebab_kerugian: event.target.value })}
              required
              autoComplete="off"
              {...(violations['penyebab_kerugian']
                ? { failure: violations['penyebab_kerugian'] }
                : {})}
            />
            <FormField
              id="penyebab-kerugian-master"
              label="Cause Of Loss Dipilih"
              value={detail.penyebab_kerugian_master}
              onChange={(event) =>
                patchDetail({ penyebab_kerugian_master: event.target.value })
              }
              required
              autoComplete="off"
              {...(violations['penyebab_kerugian_master']
                ? { failure: violations['penyebab_kerugian_master'] }
                : {})}
            />
          </div>
        </fieldset>
      )}

      {(changesLossDate || changesCauseOfLoss) && (
        <div className="grid gap-4 sm:grid-cols-2">
          <FormField
            id="nama-objek"
            label="Object Name"
            value={detail.nama_objek}
            onChange={(event) => patchDetail({ nama_objek: event.target.value })}
            autoComplete="off"
          />
          <FormField
            id="nama-cabang"
            label="Branch Name"
            value={detail.nama_cabang}
            onChange={(event) => patchDetail({ nama_cabang: event.target.value })}
            autoComplete="off"
          />
        </div>
      )}

      <TextAreaField
        id="keterangan"
        label="Keterangan"
        value={note}
        onChange={(event) => setNote(event.target.value)}
        rows={3}
        required
        {...(violations['keterangan'] ? { error: violations['keterangan'] } : {})}
      />

      <div className="flex flex-wrap items-center gap-3">
        <Button tone="utama" type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Menyimpan…' : existing ? 'Simpan Perubahan' : 'Simpan'}
        </Button>
        <Button tone="halus" onClick={onCancel} disabled={isSubmitting}>
          Batal
        </Button>

        {existing && (
          <span className="text-xs text-slate-500">
            No Proteksi <span className="font-mono">{existing.nomor_proteksi}</span>
          </span>
        )}
      </div>

      <p className="text-xs text-slate-500">
        Tipe proteksi yang belum berlabel ditampilkan sebagai kodenya. Daftar lengkapnya
        belum diserahkan — lihat {Object.keys(PROTECTION_TYPE_LABEL).length} tipe yang sudah
        dikenali.
      </p>
    </form>
  )
}
