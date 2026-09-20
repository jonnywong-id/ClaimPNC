import type { FieldErrors, UseFormRegister } from 'react-hook-form'

import { Button } from '@/components/Button'
import { SelectField } from '@/components/SelectField'

import type { PanelFields } from './PanelForm'

/** Satu baris pada daftar yang sedang disunting; `id` diterbitkan useFieldArray. */
export type LocationRow = { id: string }

type Props = {
  /** Baris yang sedang digambar, berasal dari useFieldArray. */
  rows: LocationRow[]

  register: UseFormRegister<PanelFields>
  errors: FieldErrors<PanelFields>

  /** Pilihan Lokasi Panel, dibaca dari endpoint pilihan. */
  locationOptions: string[]

  /** Pilihan Sisi Panel beserta sebutannya. */
  sideOptions: { nilai: string; label: string }[]

  /** Pelanggaran per isian yang dilaporkan server, dikunci `lokasi.<indeks>.<isian>`. */
  violation: Record<string, string>

  isSaving: boolean
  onAdd: () => void
  onRemove: (index: number) => void
}

/**
 * LocationEditor adalah daftar lokasi pada sebuah panel — tabel anak
 * POOLDATA.LOKASI_PANEL_HE.
 *
 * # Kenapa ia komponen tersendiri
 *
 * Ini satu-satunya daftar anak yang dapat disunting di seluruh aplikasi sejauh ini;
 * seluruh layar master sebelumnya rata. Memisahkannya membuat form induknya tetap terbaca
 * sebagai daftar isian, bukan sebagai dua hal yang bercampur.
 *
 * # Dua daftar pilihan yang nilainya BENAR-BENAR diketahui
 *
 * Berbeda dari kesembilan penanda STS_* pada form induk — yang daftar pilihannya tidak ada
 * di export (`R-16`) — kedua daftar di sini terbaca utuh dari
 * `Activity/SetLokasiSisiPanel-Act.xml`. Karena itu keduanya dibuat dropdown, bukan isian
 * teks bersaran.
 *
 * # Panel TANPA lokasi adalah keadaan yang SAH
 *
 * Layar lama tidak mewajibkan satu pun baris, dan `GetLokasiSisiPanel` yang mengembalikan
 * nol baris tidak diperlakukan sebagai galat di mana pun. Daftar kosong karena itu
 * menampilkan keterangan, bukan pesan kesalahan.
 *
 * # Daftar ini TIDAK dipaginasi, dan itu mengikuti Pega
 *
 * Grid lokasi di layar lama bertanda `pyPageSize=20` TETAPI `pyPageMode="None"`
 * (`Section/BrowsePanelHEApprove-Section.xml`) — ukuran halamannya tersimpan, paginasinya
 * mati. Seluruh baris digambar sekaligus.
 *
 * Masuk akal untuk daftar yang kelima pilihan lokasinya saja: paginasi atas daftar yang
 * jarang melewati lima baris hanya menambah kontrol yang tidak pernah dipakai. Batas
 * `masterpanel.MaxLocationRows` (50) yang menahannya, bukan paginator.
 */
export function LocationEditor({
  rows,
  register,
  errors,
  locationOptions,
  sideOptions,
  violation,
  isSaving,
  onAdd,
  onRemove,
}: Props) {
  return (
    <div className="space-y-3">
      <p className="text-xs text-slate-500">
        Daftar lokasi tersimpan di tabel terpisah. Menyimpan panel akan{' '}
        <span className="font-medium">mengganti seluruh daftar ini</span> — baris yang
        dihapus di sini akan hilang dari basis data.
      </p>

      {rows.length === 0 ? (
        <p className="rounded-kontrol border border-dashed border-slate-300 px-3 py-4 text-sm text-slate-500">
          Belum ada lokasi. Panel tanpa lokasi tetap dapat disimpan.
        </p>
      ) : (
        <ul className="space-y-3">
          {rows.map((row, index) => {
            const rowError = errors.lokasi?.[index]
            const nameError =
              rowError?.lokasi_panel?.message ?? violation[`lokasi.${index}.lokasi_panel`]
            const sideError =
              rowError?.sisi_panel?.message ?? violation[`lokasi.${index}.sisi_panel`]

            return (
              <li
                key={row.id}
                className="rounded-kontrol border border-slate-200 bg-slate-50 p-3"
              >
                <div className="grid gap-3 sm:grid-cols-[1fr_1fr_auto] sm:items-end">
                  <SelectField
                    id={`lokasi-${index}-nama`}
                    label={`Lokasi panel ${index + 1}`}
                    options={locationOptions.map((one) => ({ value: one, label: one }))}
                    error={nameError}
                    disabled={isSaving}
                    {...register(`lokasi.${index}.lokasi_panel`)}
                  />
                  <SelectField
                    id={`lokasi-${index}-sisi`}
                    label="Sisi panel"
                    options={sideOptions.map((one) => ({
                      value: one.nilai,
                      label: one.label,
                    }))}
                    error={sideError}
                    disabled={isSaving}
                    {...register(`lokasi.${index}.sisi_panel`)}
                  />
                  {/*
                    Tombol hapus menyebut baris yang dihapusnya lewat aria-label.

                    Tanpa itu, pembaca layar mengumumkan sederet tombol bernama "Hapus"
                    yang sama persis, dan pengguna tidak punya cara mengetahui baris mana
                    yang akan hilang.

                    aria-label dipakai, BUKAN teks tambahan berkelas sr-only: algoritma
                    nama aksesibel memangkas setiap simpul teks lalu menyambungnya tanpa
                    pemisah, sehingga "Hapus" + " lokasi baris 1" terbaca
                    "Hapuslokasi baris 1". Label eksplisit tidak punya kelemahan itu.
                  */}
                  <Button
                    tone="halus"
                    aria-label={`Hapus lokasi baris ${index + 1}`}
                    onClick={() => onRemove(index)}
                    disabled={isSaving}
                    className="sm:mb-1"
                  >
                    Hapus
                  </Button>
                </div>
              </li>
            )
          })}
        </ul>
      )}

      {/*
        Pelanggaran yang mengenai DAFTARNYA, bukan salah satu barisnya — misalnya jumlah
        baris yang melewati batas. Ia tidak punya isian untuk ditempeli, sehingga
        ditampilkan di bawah daftar.
      */}
      {(errors.lokasi?.message ?? violation['lokasi']) && (
        <p className="text-sm text-red-700">
          {errors.lokasi?.message ?? violation['lokasi']}
        </p>
      )}

      <Button tone="kedua" onClick={onAdd} disabled={isSaving}>
        Tambah lokasi
      </Button>
    </div>
  )
}
