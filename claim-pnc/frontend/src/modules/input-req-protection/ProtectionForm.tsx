import { useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { FormField } from '@/components/FormField'
import { SelectField, type SelectOption } from '@/components/SelectField'
import { TextAreaField } from '@/components/TextAreaField'
import { DokumenPenunjangPanel } from '@/modules/dokumen-penunjang/DokumenPenunjangPanel'

import { useClaimLookup } from './api'
import { CoveragePicker } from './CoveragePicker'
import { useCauseOfLossPanel } from './useCauseOfLossPanel'
import {
  TYPE_CHANGE_CAUSE_OF_LOSS,
  TYPE_CHANGE_LOSS_DATE,
  type CauseOfLossOption,
  type CoverageRow,
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
 * | Cause of Loss | `.ClaimDataProtect.IDMasterTONP` / `.CauseOfLossID` | tipe `8` |
 *
 * Urutan kedua properti terakhir DIKOREKSI 2026-10-05; sebelumnya tertulis terbalik. Di Pega,
 * label "Cause Of Loss Dipilih" terikat `.IDMasterTONP` (nilai LAMA) dan "Next Cause Of Loss"
 * terikat `.CauseOfLossID` (nilai BARU) — dan yang terakhir itulah yang dituliskan ke klaim
 * oleh `Activity/InsertOpenProtectionCase-Act.xml:3003`. Rinciannya di `docs/peta-penamaan.md`.
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

export function ProtectionForm({
  existing,
  typeOptions,
  onSubmit,
  onCancel,
  isSubmitting,
  failure,
}: Props) {
  const [claimNumber, setClaimNumber] = useState(existing?.nomor_klaim ?? '')
  const [type, setType] = useState(existing?.tipe_proteksi ?? '')
  const [note, setNote] = useState(existing?.keterangan ?? '')

  // HANYA nilai "sesudah" yang disimpan sebagai keadaan form. Nilai "sebelum" datang dari
  // pencarian klaim di bawah dan tidak pernah menjadi keadaan yang dapat diubah pengguna.
  const [lossDateAfter, setLossDateAfter] = useState(existing?.detail_perubahan.dol_baru ?? '')
  const [causeAfter, setCauseAfter] = useState(
    existing?.detail_perubahan.penyebab_kerugian_master ?? '',
  )

  // Baris coverage yang DIPILIH pemohon — sasaran perubahan Cause of Loss.
  //
  // Keduanya satu keadaan, bukan dua: ia diisi sekaligus oleh satu tombol Pilih, dan
  // dikosongkan sekaligus. Menyimpannya sebagai dua state terpisah membuka kemungkinan
  // terisi separuh, yang ditolak server dengan pesan yang tidak dapat diperbaiki pengguna.
  const [selected, setSelected] = useState<{ objek: string; coverage: string }>({
    objek: existing?.detail_perubahan.id_objek ?? '',
    coverage: existing?.detail_perubahan.id_coverage ?? '',
  })

  // Pencarian klaim menggantikan tombol CARI pada form Pega.
  //
  // `Activity/OpenProtection-Act.xml` mengisi No Polis, Nama Tertanggung, Object Name,
  // Branch Name, Current Date Of Loss, dan Cause Of Loss Dipilih dari klaim yang ditemukan.
  // Keenamnya TIDAK PERNAH diketik, dan di sini pun tidak.
  const claim = useClaimLookup(claimNumber)
  const found = claim.data ?? null

  const violations = failure instanceof APIError ? failure.violations() : {}

  // Galat yang TIDAK menunjuk kolom mana pun ditampilkan di atas form. Yang menunjuk kolom
  // sudah menempel di kolomnya, dan mengulangnya di atas hanya menambah kebisingan.
  const generalFailure =
    failure instanceof APIError && Object.keys(violations).length === 0 ? failure.message : null

  const changesLossDate = type === TYPE_CHANGE_LOSS_DATE
  const changesCauseOfLoss = type === TYPE_CHANGE_CAUSE_OF_LOSS

  // Seluruh turunan panel Cause Of Loss hidup di satu hook, beserta alasannya. Form hanya
  // menggambar.
  const panelCOL = useCauseOfLossPanel(claimNumber, changesCauseOfLoss, selected)

  // Keadaan pencarian klaim DITAMPILKAN, tidak lagi ditelan.
  //
  // # Cacat yang ini perbaiki
  //
  // Sebelumnya form hanya membaca `claim.data`; `claim.isError` tidak pernah diperiksa.
  // Akibatnya nomor klaim yang tidak ditemukan terlihat PERSIS SAMA dengan klaim yang
  // ditemukan tetapi datanya kosong: keenam field turunan blank, tanpa satu pun pesan.
  //
  // Itulah yang membuat "Object Name kosong terus" tidak dapat dibedakan dari "klaimnya
  // memang tidak ada" — dan pada `PENYIMPANAN=memori`, hampir setiap nomor klaim nyata
  // memang tidak ada.
  const claimSedangDicari = claimNumber.trim() !== '' && claim.isPending
  const claimTidakKetemu =
    claimNumber.trim() !== '' &&
    claim.isError &&
    claim.error instanceof APIError &&
    claim.error.status === 404
  const claimGagalDibaca = claimNumber.trim() !== '' && claim.isError && !claimTidakKetemu

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    await onSubmit({
      nomor_klaim: claimNumber,
      tipe_proteksi: type,
      keterangan: note,
      detail_perubahan: {
        dol_baru: lossDateAfter,
        penyebab_kerugian_baru: causeAfter,
        id_objek: selected.objek,
        id_coverage: selected.coverage,
      },
    })
  }

  return (
    <form onSubmit={(event) => { submit(event) }} className="space-y-5">
      {generalFailure && (
        <ErrorMessage
          title="Permintaan tidak dapat disimpan"
          description={generalFailure}
          tone="penolakan"
        />
      )}

      {/*
        Nada dibedakan, karena tindak lanjutnya berbeda: nomor yang salah ketik dapat
        diperbaiki pengguna, gangguan pembacaan tidak.
      */}
      {claimTidakKetemu && (
        <ErrorMessage
          title="Klaim tidak ditemukan"
          description={
            'Nomor klaim ini tidak ada di data klaim, sehingga No Polis, Nama Tertanggung, ' +
            'Object Name, Branch Name, Current Date Of Loss, dan Cause Of Loss tidak dapat diisi. ' +
            'Periksa kembali nomornya.'
          }
          tone="penolakan"
        />
      )}

      {claimGagalDibaca && (
        <ErrorMessage
          title="Data klaim gagal dibaca"
          description="Field yang diturunkan dari klaim belum dapat diisi. Coba lagi beberapa saat."
          tone="gangguan"
        />
      )}

      {claimSedangDicari && (
        // Tanpa penanda ini, jeda pembacaan terlihat sama dengan klaim yang tidak punya
        // datanya — keduanya menampilkan field kosong.
        <p className="text-sm text-slate-500">Mencari data klaim…</p>
      )}

      <div className="grid gap-4 sm:grid-cols-2">
        {/*
          HANYA-BACA. `Activity/OpenProtection-Act.xml` mengisi `pyWorkPage.PolicyNo` dari
          klaim yang ditemukan — di sistem lama pun ia tidak pernah diketik.
        */}
        <FormField
          id="nomor-polis"
          label="No Polis (dari klaim)"
          value={found?.nomor_polis ?? ''}
          onChange={() => {}}
          readOnly
          autoComplete="off"
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
          Isian "Referensi Klaim" DIHAPUS pada 2026-09-24.

          Sampai saat itu ada dua isian nomor klaim: No Klaim dan Referensi Klaim, meniru
          Pega yang mengisi `.PNCCaseID` dari hasil pencarian. Work Owner menegaskan
          keduanya kini berisi NILAI YANG SAMA, sehingga server menurunkan ClaimID dari
          No Klaim.

          Dua isian yang WAJIB sama tetapi diketik terpisah akan berbeda cepat atau
          lambat, dan perbedaannya tidak menghasilkan galat apa pun — hanya proteksi yang
          menunjuk dua klaim berbeda.

          Nilai ClaimID baris WARISAN tetap ditampilkan di bawah, karena di sana ia memang
          berbeda: Pega menyimpan kunci teknisnya, `ASM-FW-GCNMFW-WORK PNC-xxxx`.
        */}
        {existing?.referensi_klaim && existing.referensi_klaim !== existing.nomor_klaim && (
          <FormField
            id="referensi-klaim"
            label="ID Klaim (warisan Pega)"
            value={existing.referensi_klaim}
            onChange={() => {}}
            readOnly
            autoComplete="off"
          />
        )}
      </div>

      {changesLossDate && (
        <fieldset className="rounded-lg border border-slate-200 p-4">
          <legend className="px-1 text-sm font-medium text-slate-700">
            Detail Perubahan DOL
          </legend>

          <div className="grid gap-4 sm:grid-cols-2">
            {/*
              HANYA-BACA, dan itu inti koreksi 2026-09-24.

              Di Pega nilai ini disalin dari klaim setiap kali klaim dicari
              (`.ClaimDataProtect.BeforeDateOfLoss := TempPNCOPEN.ClaimData.DateOfLoss`).
              Implementasi pertama menjadikannya isian bebas — sehingga seseorang dapat
              meminta "ubah DOL" dengan menyebut DOL sebelum yang tidak pernah menjadi DOL
              klaim itu, dan petugas akseptasi menyetujuinya tanpa cara mengetahuinya.
            */}
            <FormField
              id="dol-sebelum"
              label="Current Date Of Loss (dari klaim)"
              type="date"
              value={found?.dol ?? ''}
              onChange={() => {}}
              readOnly
            />
            <FormField
              id="dol-baru"
              label="Next Date Of Loss"
              type="date"
              value={lossDateAfter}
              onChange={(event) => setLossDateAfter(event.target.value)}
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

          {/*
            Tabel pemilih, mengikuti layar Pega apa adanya.

            Satu klaim dapat punya banyak objek dan banyak coverage, masing-masing dengan
            Penyebab Kerugiannya sendiri. Pada klaim `PNC-1452`, `Resiko A` muncul TIGA KALI
            dengan penyebab berbeda — jadi memilih lewat nama tidak mungkin, dan panel ini
            satu-satunya cara menyatakan baris mana yang hendak diubah.
          */}
          <CoveragePicker
            rows={panelCOL.rows}
            selected={selected}
            onSelect={(row) => {
              setSelected({ objek: row.id_objek, coverage: row.id_coverage })
            }}
            memuat={panelCOL.memuatCoverage}
            gagal={panelCOL.coverageGagal}
            failure={violations['baris_coverage'] ?? null}
            pilihanHilang={panelCOL.pilihanHilang}
          />

          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            {/* HANYA-BACA — dari BARIS YANG DIPILIH, bukan dari klaim.
                Sebelum panel ini ada, nilainya diambil dari coverage PERTAMA klaim, yang
                belum tentu coverage yang hendak diubah. */}
            <FormField
              id="penyebab-kerugian"
              label="Cause Of Loss Dipilih"
              value={penyebabTerpilih(panelCOL.selectedRow)}
              onChange={() => {}}
              readOnly
              autoComplete="off"
            />

            {/*
              DROPDOWN, bukan kotak teks — ditetapkan Work Owner 2026-10-05.

              Nilainya `D_COL_ID`, bukan teks yang diketik. Teks bebas membuat salah ketik
              tersimpan apa adanya sebagai penyebab kerugian klaim, dan tidak ada yang
              menangkapnya: penyimpanan berhasil, layar normal, kodenya tidak cocok dengan
              master mana pun.
            */}
            <SelectField
              id="penyebab-kerugian-master"
              label="Next Cause Of Loss"
              options={panelCOL.options.map(pilihanPenyebab)}
              value={causeAfter}
              onChange={(event) => setCauseAfter(event.target.value)}
              required
              emptyText="--- Pilih Cause Of Loss ---"
              {...(violations['penyebab_kerugian_master']
                ? { error: violations['penyebab_kerugian_master'] }
                : {})}
            />
          </div>

          {panelCOL.pilihanGagal && (
            <p className="mt-2 text-sm text-rose-700">
              Daftar Cause Of Loss gagal dimuat. Muat ulang halaman; menyimpan tanpa memilih
              akan ditolak.
            </p>
          )}
        </fieldset>
      )}

      {(changesLossDate || changesCauseOfLoss) && (
        <div className="grid gap-4 sm:grid-cols-2">
          {/* Keduanya HANYA-BACA — dari klaim, sama alasannya dengan No Polis. */}
          <FormField
            id="nama-objek"
            label="Object Name (dari klaim)"
            value={found?.nama_objek ?? ''}
            onChange={() => {}}
            readOnly
            autoComplete="off"
          />
          <FormField
            id="nama-cabang"
            label="Branch Name (dari klaim)"
            value={found?.nama_cabang ?? ''}
            onChange={() => {}}
            readOnly
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

      {/*
        Dokumen penunjang menempel pada KLAIM, bukan pada permintaan proteksinya — karena
        itu yang diserahkan nomor klaimnya, dan karena itu pula panelnya tetap berguna
        sebelum permintaannya disimpan.

        Akibatnya satu hal harus disadari: berkas yang diunggah di sini SUDAH tersimpan di
        klaim meski form-nya kemudian dibatalkan. Itu memang perilaku yang benar — ia milik
        klaim, bukan milik draf permintaan ini — tetapi berbeda dari isian lain di layar
        yang sama.
      */}
      <DokumenPenunjangPanel nomorKlaim={found ? claimNumber : null} />

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

      {typeOptions.length === 0 && (
        // Master yang kosong DITAMPILKAN apa adanya, bukan ditutupi daftar cadangan di
        // kode. Pilihan yang berasal dari kode akan membuat master yang bermasalah tampak
        // beres — dan proteksi tersimpan dengan tipe yang tidak dikenal basis datanya.
        <p className="text-xs text-amber-700">
          Master tipe proteksi belum dapat dibaca, sehingga pilihan tipe kosong. Permintaan
          tidak dapat disimpan tanpa tipe.
        </p>
      )}
    </form>
  )
}

/**
 * Teks kolom "Cause Of Loss Dipilih" — deskripsi beserta kodenya.
 *
 * Di luar komponen, bukan ternary di dalam JSX: dua baris coverage dapat bernama sama persis
 * sementara kodenya berbeda, jadi penggabungan ini adalah aturan tampilan yang perlu terbaca
 * sebagai satu hal, bukan diselipkan ke dalam atribut.
 */
function penyebabTerpilih(row: CoverageRow | null): string {
  if (row === null) return ''
  if (row.penyebab_kerugian_id === '') return row.penyebab_kerugian
  return `${row.penyebab_kerugian} (${row.penyebab_kerugian_id})`
}

/**
 * Satu pilihan dropdown "Next Cause Of Loss".
 *
 * `LOSS_CODE` ikut ditampilkan bila ada — sebagian penyebab kerugian bernama mirip, dan kode
 * itulah yang membedakannya di mata petugas. Yang DISIMPAN tetap `kode` (`D_COL_ID`).
 */
function pilihanPenyebab(o: CauseOfLossOption): SelectOption {
  return {
    value: o.kode,
    label: o.kode_kerugian === '' ? o.nama : `${o.nama} (${o.kode_kerugian})`,
  }
}
