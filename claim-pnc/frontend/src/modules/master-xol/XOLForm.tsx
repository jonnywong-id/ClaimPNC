import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import {
  useFieldArray,
  useForm,
  useWatch,
  type Control,
  type UseFormRegister,
} from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import type { XOL } from '@/api/types'
import { ErrorCode } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { AddIcon, TrashIcon } from '@/components/Icon'
import { SelectField } from '@/components/SelectField'

import { useSaveXOL, useXOLBusinessGroup, useXOLDetail, useXOLForm, useDeleteXOLChild } from './api'
import { formatMoney } from './format'

/**
 * Batas panjang isian; sama dengan konstanta di internal/masterxol/masterxol.go.
 *
 * Angka yang sama dinyatakan dua kali — di sini dan di domain Go. Itu duplikasi yang
 * DISENGAJA: yang di sini menjawab pengguna tanpa perjalanan jaringan; yang di sana yang
 * menegakkan, karena pemanggilan langsung ke API tidak melewati layar ini sama sekali.
 *
 * Bila salah satu berubah, KEDUA tempat wajib ikut berubah.
 */
const MAX = {
  nama: 50,
  tahun: 10,
  tipe: 10,
  remark: 1000,
  bisnisID: 10,
  bisnisNama: 50,
  layerNama: 50,
  reasID: 20,
  reasNama: 100,
} as const

/**
 * Angka dibaca sebagai ANGKA, bukan dikonversi di dalam skema.
 *
 * `z.coerce.number()` sempat dipakai dan dibatalkan: di Zod 4 ia membuat tipe MASUKAN
 * skema menjadi `unknown`, sehingga resolver tidak lagi cocok dengan tipe form dan
 * `tsc --noEmit` menolaknya. Yang dipakai sebagai gantinya adalah `valueAsNumber` pada
 * `register` — jalur bawaan React Hook Form untuk `<input type="number">`, dan ia
 * mengubah nilainya sebelum skema melihatnya.
 *
 * Isian kosong menghasilkan NaN lewat jalur itu; pesannya dinyatakan di sini supaya
 * pengguna membaca kalimat yang berarti, bukan "expected number, received nan".
 */
const angka = (label: string) =>
  z
    .number({ error: `${label} harus diisi angka.` })
    .int(`${label} harus bilangan bulat.`)
    .min(0, `${label} tidak boleh negatif.`)

/**
 * Aturan yang ditegakkan di layar.
 *
 * Hanya DUA golongan, sama persis dengan yang ditegakkan server: panjang teks, dan angka
 * tidak negatif. Layar Pega tidak mewajibkan satu pun isian induk — keempatnya bertanda
 * `pyRequired=false` dan kolomnya nullable — sehingga menuntutnya di sini akan menolak
 * data yang hari ini sah.
 *
 * **Total share 100% TIDAK ada di sini**, dan itu bukan kelalaian: ia peringatan, bukan
 * penolakan. Layar lama menyimpan lebih dulu baru menampilkan pesannya, dan perilaku itu
 * ditiru (keputusan Work Owner 2026-09-20, `P-5`). Peringatannya datang dari server di
 * dalam `peringatan` pada jawaban simpan.
 */
const schema = z.object({
  nama: z.string().trim().max(MAX.nama, `Nama paling panjang ${MAX.nama} karakter.`),
  tahun: z.string().trim().max(MAX.tahun, `Tahun paling panjang ${MAX.tahun} karakter.`),
  tipe: z.string().trim().max(MAX.tipe, `Type XOL paling panjang ${MAX.tipe} karakter.`),
  kurs: angka('Kurs'),
  remark_pic: z.string().trim().max(MAX.remark, `Remark PIC paling panjang ${MAX.remark} karakter.`),

  bisnis: z.array(
    z.object({
      id: z.string().trim().max(MAX.bisnisID),
      nama: z.string().trim().max(MAX.bisnisNama),
      /** Penanda baris ini sudah ada di server. Tidak ikut dikirim. */
      tersimpan: z.boolean(),
    }),
  ),

  layer: z.array(
    z.object({
      id: z.string(),
      nama: z.string().trim().max(MAX.layerNama, `Nama layer paling panjang ${MAX.layerNama} karakter.`),
      limit: angka('Limit'),
      excess: angka('Excess'),
      tersimpan: z.boolean(),
      reas: z.array(
        z.object({
          id: z.string().trim().max(MAX.reasID),
          nama: z.string().trim().max(MAX.reasNama),
          share: angka('Share'),
          tersimpan: z.boolean(),
        }),
      ),
    }),
  ),
})

type FieldValues = z.infer<typeof schema>

const KOSONG: FieldValues = {
  nama: '',
  tahun: '',
  tipe: '',
  kurs: 0,
  remark_pic: '',
  bisnis: [],
  layer: [],
}

type Props = {
  /** Null berarti menambah; terisi berarti mengubah induk itu. */
  master: XOL | null
  tutup: () => void
}

/**
 * Form tambah dan ubah Master XOL.
 *
 * Meniru bentuk `Section/DetailXOL_sec-Section.xml`: isian induk di atas, lalu tiga grid
 * bertingkat — Nama Bisnis, Layer, dan Reas di dalam tiap layer.
 *
 * # Dua hal yang wajib disadari pengguna, dan karena itu dinyatakan di layar
 *
 *  1. **Menyimpan sekaligus mengajukan ke komite.** Tanpa syarat, persis seperti layar
 *     lama. Induk yang sudah disetujui kembali berstatus menunggu.
 *  2. **Tombol hapus pada grid menghapus SEKETIKA**, tidak menunggu Simpan — juga persis
 *     seperti layar lama, yang memanggil `DeleteFromTabelMst` langsung.
 */
export function XOLForm({ master, tutup }: Props) {
  const editing = master !== null
  const detail = useXOLDetail(master?.id ?? null)
  const option = useXOLForm()
  const save = useSaveXOL()
  const removeChild = useDeleteXOLChild()

  const {
    control,
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: KOSONG,
  })

  // Baris induk di grid TIDAK membawa anaknya — daftar sengaja tidak menariknya. Isian
  // form karena itu diisi dari detail yang dimuat terpisah, bukan dari baris grid.
  const loaded = detail.data?.xol
  useEffect(() => {
    if (!editing) {
      reset(KOSONG)
      return
    }
    if (!loaded) return
    reset(toFieldValues(loaded))
  }, [editing, loaded, reset])

  const tipe = useWatch({ control, name: 'tipe' }) ?? ''
  const kurs = useWatch({ control, name: 'kurs' }) ?? 0

  const business = useFieldArray({ control, name: 'bisnis' })
  const layer = useFieldArray({ control, name: 'layer' })

  const businessOption = useXOLBusinessGroup(tipe)

  function send(values: FieldValues) {
    save.mutate(
      {
        ...(master ? { id: master.id } : {}),
        input: {
          nama: values.nama,
          tahun: values.tahun,
          tipe: values.tipe,
          kurs: values.kurs,
          remark_pic: values.remark_pic,
          // `tersimpan` adalah penanda layar, bukan bagian kontrak — server menolak badan
          // permintaan yang memuat field tak dikenal, jadi ia wajib dibuang di sini.
          bisnis: values.bisnis.map((b) => ({ id: b.id, nama: b.nama })),
          layer: values.layer.map((l) => ({
            id: l.id,
            nama: l.nama,
            limit: l.limit,
            excess: l.excess,
            reas: l.reas.map((r) => ({ id: r.id, nama: r.nama, share: r.share })),
          })),
        },
      },
      { onSuccess: () => { if (!editing) tutup() } },
    )
  }

  /** Membuang satu baris anak: dari server bila sudah tersimpan, dari layar bila belum. */
  function dropChild(
    tersimpan: boolean,
    hapusLokal: () => void,
    hapusServer: () => void,
  ) {
    if (tersimpan && master) hapusServer()
    hapusLokal()
  }

  const memuat = editing && detail.isPending

  return (
    <section className="rounded-lg border border-slate-200 bg-white shadow-sm">
      <header className="border-b border-slate-200 px-5 py-4">
        <h2 className="text-base font-semibold text-slate-900">
          {editing ? 'Memperbaharui Data' : 'Menambah Data'}
        </h2>
        <p className="mt-1 text-xs leading-relaxed text-slate-500">
          Menyimpan sekaligus <strong>mengajukan ke komite</strong>. Induk yang sudah
          disetujui akan kembali berstatus menunggu, karena strukturnya berubah.
        </p>
      </header>

      {memuat ? (
        <p className="px-5 py-8 text-sm text-slate-500">Memuat isi master XOL…</p>
      ) : detail.isError ? (
        <div className="px-5 py-5">
          <ErrorMessage
            title="Isi master XOL gagal dimuat"
            description="Tutup form ini lalu coba buka kembali."
            tone="gangguan"
          />
        </div>
      ) : (
        <form onSubmit={handleSubmit(send)} className="space-y-6 px-5 py-5" noValidate>
          {/* ── Isian induk ────────────────────────────────────────────── */}
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <Field
              id="xol-id"
              label="ID"
              value={master?.id ?? '(dibuat sistem)'}
              readOnly
              disabled
              hint="Nomor diterbitkan sistem saat disimpan."
            />
            <Field
              id="xol-nama"
              label="Nama"
              error={errors.nama?.message}
              {...register('nama')}
            />
            <SelectField
              id="xol-tahun"
              label="Tahun"
              options={(option.data?.tahun ?? []).map((t) => ({ value: t, label: t }))}
              error={errors.tahun?.message}
              {...register('tahun')}
            />
            <Field
              id="xol-kurs"
              label="Kurs IDR"
              type="number"
              inputMode="numeric"
              error={errors.kurs?.message}
              hint="Rupiah per satu dolar. Dipakai menghitung Limit (IDR) tiap layer."
              {...register('kurs', { valueAsNumber: true })}
            />
            <SelectField
              id="xol-tipe"
              label="Type XOL"
              options={(option.data?.tipe ?? []).map((t) => ({ value: t.kode, label: t.label }))}
              error={errors.tipe?.message}
              {...register('tipe')}
            />
            <Field
              id="xol-remark"
              label="Remark PIC"
              error={errors.remark_pic?.message}
              hint="Catatan yang ikut terkirim ke komite."
              {...register('remark_pic')}
            />
          </div>

          {/* ── Grid Nama Bisnis ───────────────────────────────────────── */}
          <Panel
            judul="Nama Bisnis"
            keterangan={
              tipe === ''
                ? 'Pilih Type XOL lebih dulu — pilihan grup bisnis mengikuti jenisnya.'
                : 'Grup bisnis yang dicakup treaty ini.'
            }
            aksi={
              <Button
                tone="kedua"
                onClick={() => business.append({ id: '', nama: '', tersimpan: false })}
              >
                <AddIcon className="h-4 w-4" />
                Tambah bisnis
              </Button>
            }
          >
            {business.fields.length === 0 ? (
              <EmptyRow pesan="Belum ada grup bisnis." />
            ) : (
              <ul className="divide-y divide-slate-100">
                {business.fields.map((row, index) => (
                  <li key={row.id} className="flex flex-wrap items-end gap-3 py-3">
                    <div className="min-w-[16rem] flex-1">
                      <SelectField
                        id={`bisnis-${index}`}
                        label={`Grup bisnis ${index + 1}`}
                        options={(businessOption.data?.bisnis ?? []).map((b) => ({
                          // ID dan nama dibawa bersama dalam satu nilai: baris
                          // "TREATY INWARD" tidak punya ID, sehingga ID saja tidak cukup
                          // mengenali pilihan.
                          value: `${b.id} ${b.nama}`,
                          label: b.id ? `${b.nama} (${b.id})` : b.nama,
                        }))}
                        value={`${row.id} ${row.nama}`}
                        onChange={(event) => {
                          const [id = '', nama = ''] = event.target.value.split(' ')
                          business.update(index, { id, nama, tersimpan: row.tersimpan })
                        }}
                      />
                    </div>
                    <Button
                      tone="halus"
                      aria-label={`Hapus grup bisnis baris ${index + 1}`}
                      // Baris tersimpan yang TIDAK punya ID tidak dapat dihapus, dan itu
                      // batasan yang diwarisi apa adanya: kueri hapus lama mencocokkan
                      // `IDBUSINESS = :id`, dan pencocokan itu tidak pernah benar untuk
                      // NULL. Dua baris seperti itu memang ada di produksi.
                      disabled={row.tersimpan && row.id === ''}
                      title={
                        row.tersimpan && row.id === ''
                          ? 'Baris ini tidak punya ID di basis data sehingga tidak dapat dihapus dari layar.'
                          : undefined
                      }
                      onClick={() =>
                        dropChild(
                          row.tersimpan,
                          () => business.remove(index),
                          () =>
                            removeChild.mutate({
                              jenis: 'bisnis',
                              masterID: master?.id ?? '',
                              id: row.id,
                            }),
                        )
                      }
                    >
                      <TrashIcon className="h-3.5 w-3.5" />
                      Hapus
                    </Button>
                  </li>
                ))}
              </ul>
            )}
          </Panel>

          {/* ── Grid Layer ─────────────────────────────────────────────── */}
          <Panel
            judul="Layer"
            keterangan="Limit dan Excess dalam DOLAR. Limit (IDR) dihitung server dari Kurs IDR di atas."
            aksi={
              <Button
                tone="kedua"
                onClick={() =>
                  layer.append({ id: '', nama: '', limit: 0, excess: 0, tersimpan: false, reas: [] })
                }
              >
                <AddIcon className="h-4 w-4" />
                Tambah layer
              </Button>
            }
          >
            {layer.fields.length === 0 ? (
              <EmptyRow pesan="Belum ada layer." />
            ) : (
              <ul className="space-y-4">
                {layer.fields.map((row, index) => (
                  <li key={row.id} className="rounded-md border border-slate-200 p-4">
                    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                      <Field
                        id={`layer-nama-${index}`}
                        label="Nama Layer"
                        error={errors.layer?.[index]?.nama?.message}
                        {...register(`layer.${index}.nama`)}
                      />
                      <Field
                        id={`layer-limit-${index}`}
                        label="Limit (USD)"
                        type="number"
                        inputMode="numeric"
                        error={errors.layer?.[index]?.limit?.message}
                        {...register(`layer.${index}.limit`, { valueAsNumber: true })}
                      />
                      <Field
                        id={`layer-excess-${index}`}
                        label="Excess (USD)"
                        type="number"
                        inputMode="numeric"
                        error={errors.layer?.[index]?.excess?.message}
                        {...register(`layer.${index}.excess`, { valueAsNumber: true })}
                      />
                      <LimitIDR control={control} index={index} kurs={kurs} />
                    </div>

                    <ReasPanel
                      control={control}
                      register={register}
                      layerIndex={index}
                      layerID={row.id}
                      masterID={master?.id ?? ''}
                      onHapusTersimpan={(reasID) =>
                        removeChild.mutate({
                          jenis: 'reas',
                          masterID: master?.id ?? '',
                          layerID: row.id,
                          id: reasID,
                        })
                      }
                    />

                    <div className="mt-3 flex justify-end">
                      <Button
                        tone="halus"
                        aria-label={`Hapus layer baris ${index + 1}`}
                        onClick={() =>
                          dropChild(
                            row.tersimpan,
                            () => layer.remove(index),
                            () =>
                              removeChild.mutate({
                                jenis: 'layer',
                                masterID: master?.id ?? '',
                                id: row.id,
                              }),
                          )
                        }
                      >
                        <TrashIcon className="h-3.5 w-3.5" />
                        Hapus layer ini beserta reas-nya
                      </Button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </Panel>

          {/* ── Peringatan dan galat ───────────────────────────────────── */}
          {save.isSuccess && (save.data.peringatan?.length ?? 0) > 0 && (
            <div
              role="status"
              className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900"
            >
              <p className="font-semibold">Tersimpan, dengan catatan:</p>
              <ul className="mt-1.5 list-disc space-y-0.5 pl-5">
                {save.data.peringatan?.map((pesan) => (
                  <li key={pesan}>{pesan}</li>
                ))}
              </ul>
            </div>
          )}

          {save.isSuccess && (save.data.peringatan?.length ?? 0) === 0 && (
            <p role="status" className="text-sm font-medium text-emerald-700">
              Tersimpan dan diajukan ke komite.
            </p>
          )}

          {save.isError && <SaveError error={save.error} />}

          {removeChild.isError && (
            <p role="alert" className="text-sm font-medium text-rose-700">
              Baris gagal dihapus di server. Muat ulang layar untuk melihat keadaan yang
              sebenarnya.
            </p>
          )}

          <div className="flex gap-2 border-t border-slate-100 pt-4">
            <Button type="submit" tone="utama" disabled={save.isPending}>
              {save.isPending ? 'Menyimpan…' : 'Simpan'}
            </Button>
            <Button tone="kedua" onClick={tutup} disabled={save.isPending}>
              Tutup
            </Button>
          </div>
        </form>
      )}
    </section>
  )
}

/**
 * Limit (IDR) ditampilkan, bukan diisi.
 *
 * Nilainya dihitung server dari Limit dolar dikali kurs induk. Yang ditampilkan di sini
 * adalah hitungan yang SAMA supaya pengguna melihat akibat isiannya seketika — tetapi ia
 * tidak pernah dikirim, sehingga tidak ada kemungkinan angka layar dan angka tersimpan
 * berbeda.
 */
function LimitIDR({
  control,
  index,
  kurs,
}: {
  control: Control<FieldValues>
  index: number
  kurs: number
}) {
  const limit = useWatch({ control, name: `layer.${index}.limit` }) ?? 0
  const nilai = Number(limit) * Number(kurs)

  return (
    <div>
      <span className="block text-sm font-medium text-slate-700">Limit (IDR)</span>
      <output
        className="mt-1.5 block truncate rounded-md border border-slate-200 bg-slate-50 px-3 py-2 font-mono text-sm text-slate-600"
        title={String(nilai)}
      >
        {formatMoney(nilai)}
      </output>
      <p className="mt-1 text-xs text-slate-500">Dihitung sistem: Limit × Kurs IDR.</p>
    </div>
  )
}

/**
 * Grid Reas di dalam satu layer.
 *
 * Meniru `Section/InputDetailPanelReasGenerated-Section.xml`: tiga kolom — ID, Reasuransi,
 * dan Share (%).
 *
 * # Kenapa ID dan nama DIKETIK, bukan dipilih dari daftar
 *
 * Section yang memuat pickernya (`InputPanelReas`) **hilang dari export** (`R-16`), dan
 * pencarian sumbernya di basis data pada 2026-09-20 tidak menemukan master yang cocok:
 * dari 18 nama yang dipakai, hanya 5 ada di POOLDATA.T_REINSURER. Menebak salah satunya
 * akan membuat 13 nama yang sudah dipakai menjadi tidak dapat dipilih lagi.
 */
function ReasPanel({
  control,
  register,
  layerIndex,
  layerID,
  masterID,
  onHapusTersimpan,
}: {
  control: Control<FieldValues>
  register: UseFormRegister<FieldValues>
  layerIndex: number
  layerID: string
  masterID: string
  onHapusTersimpan: (reasID: string) => void
}) {
  const reas = useFieldArray({ control, name: `layer.${layerIndex}.reas` })
  const isi = useWatch({ control, name: `layer.${layerIndex}.reas` }) ?? []

  const total = isi.reduce((jumlah, baris) => jumlah + (Number(baris?.share) || 0), 0)
  const lengkap = total === 100

  return (
    <div className="mt-4 rounded-md bg-slate-50 p-3">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <h4 className="text-sm font-semibold text-slate-800">Reas</h4>
          <span
            className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ring-1 ${
              lengkap
                ? 'bg-emerald-50 text-emerald-800 ring-emerald-200'
                : 'bg-amber-50 text-amber-800 ring-amber-200'
            }`}
          >
            Total share {total}%
          </span>
        </div>
        <Button
          tone="kedua"
          onClick={() => reas.append({ id: '', nama: '', share: 0, tersimpan: false })}
        >
          <AddIcon className="h-4 w-4" />
          Tambah reas
        </Button>
      </div>

      {/* Total yang belum 100% ditandai, TIDAK memblokir. Layar lama pun menyimpan lebih
          dulu baru menampilkan pesannya (`P-5`). */}
      {!lengkap && (
        <p className="mb-2 text-xs text-amber-800">
          Total share belum 100%. Data tetap dapat disimpan — sama seperti layar lama —
          tetapi pembagian klaim pada layer ini belum lengkap.
        </p>
      )}

      {reas.fields.length === 0 ? (
        <EmptyRow pesan="Belum ada reasuradur pada layer ini." />
      ) : (
        <ul className="divide-y divide-slate-200">
          {reas.fields.map((row, index) => (
            <li key={row.id} className="flex flex-wrap items-end gap-3 py-2.5">
              <div className="w-40">
                <Field
                  id={`reas-id-${layerIndex}-${index}`}
                  label="ID"
                  {...register(`layer.${layerIndex}.reas.${index}.id`)}
                />
              </div>
              <div className="min-w-[14rem] flex-1">
                <Field
                  id={`reas-nama-${layerIndex}-${index}`}
                  label="Reasuransi"
                  {...register(`layer.${layerIndex}.reas.${index}.nama`)}
                />
              </div>
              <div className="w-28">
                <Field
                  id={`reas-share-${layerIndex}-${index}`}
                  label="Share (%)"
                  type="number"
                  inputMode="numeric"
                  {...register(`layer.${layerIndex}.reas.${index}.share`)}
                />
              </div>
              <Button
                tone="halus"
                aria-label={`Hapus reas baris ${index + 1}`}
                onClick={() => {
                  const tersimpan = isi[index]?.tersimpan === true
                  const reasID = isi[index]?.id ?? ''
                  if (tersimpan && masterID && layerID) onHapusTersimpan(reasID)
                  reas.remove(index)
                }}
              >
                <TrashIcon className="h-3.5 w-3.5" />
                Hapus
              </Button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function Panel({
  judul,
  keterangan,
  aksi,
  children,
}: {
  judul: string
  keterangan: string
  aksi: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <section className="rounded-md border border-slate-200 p-4">
      <div className="mb-2 flex flex-wrap items-start justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold text-slate-900">{judul}</h3>
          <p className="mt-0.5 text-xs text-slate-500">{keterangan}</p>
        </div>
        {aksi}
      </div>
      {children}
    </section>
  )
}

function EmptyRow({ pesan }: { pesan: string }) {
  return <p className="py-3 text-sm italic text-slate-400">{pesan}</p>
}

function SaveError({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Perubahan belum tersimpan. Periksa koneksi lalu tekan Simpan lagi."
        tone="gangguan"
      />
    )
  }

  if (error instanceof APIError) {
    if (error.kode === ErrorCode.validationFailed && error.detail.length > 0) {
      return (
        <div role="alert" className="rounded-md border border-rose-300 bg-rose-50 px-4 py-3">
          <p className="text-sm font-semibold text-rose-900">{error.message}</p>
          <ul className="mt-1.5 list-disc space-y-0.5 pl-5 text-sm text-rose-800">
            {error.detail.map((v, index) => (
              <li key={`${v.field}-${index}`}>{v.pesan}</li>
            ))}
          </ul>
        </div>
      )
    }

    if (error.kode === ErrorCode.xolIDTaken) {
      return (
        <ErrorMessage
          title="Nomor bentrok"
          description="Nomor yang dibuat sistem sudah dipakai. Tekan Simpan sekali lagi."
          tone="gangguan"
        />
      )
    }

    return <ErrorMessage title="Gagal menyimpan" description={error.message} tone="gangguan" />
  }

  return (
    <ErrorMessage
      title="Gagal menyimpan"
      description="Terjadi kesalahan pada sistem. Coba lagi."
      tone="gangguan"
    />
  )
}

/** Mengubah induk dari server menjadi isian form, menandai seluruh baris sebagai tersimpan. */
function toFieldValues(master: XOL): FieldValues {
  return {
    nama: master.nama,
    tahun: master.tahun,
    tipe: master.tipe,
    kurs: master.kurs,
    remark_pic: master.remark_pic,
    bisnis: (master.bisnis ?? []).map((b) => ({ id: b.id, nama: b.nama, tersimpan: true })),
    layer: (master.layer ?? []).map((l) => ({
      id: l.id,
      nama: l.nama,
      limit: l.limit,
      excess: l.excess,
      tersimpan: true,
      reas: (l.reas ?? []).map((r) => ({
        id: r.id,
        nama: r.nama,
        share: r.share,
        tersimpan: true,
      })),
    })),
  }
}
