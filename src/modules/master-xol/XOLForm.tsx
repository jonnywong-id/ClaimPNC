import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useState } from 'react'
import {
  useFieldArray,
  useForm,
  useWatch,
  type Control,
  type UseFormRegister,
} from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import type { XOL, XOLReinsurerOption } from '@/api/types'
import { ErrorCode } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { AddIcon, TrashIcon } from '@/components/Icon'

import {
  useDeleteXOLChild,
  useSaveXOL,
  useXOLBusinessGroup,
  useXOLDetail,
  useXOLForm,
  useXOLReinsurerSearch,
} from './api'

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
 * Isian angka disimpan sebagai TEKS di dalam form, dan baru menjadi angka saat dikirim.
 *
 * # Kenapa teks, bukan angka
 *
 * Versi pertama menyimpannya sebagai angka dengan nilai awal `0`, dan itu cacat yang
 * terlihat langsung oleh pengguna: kotaknya sudah berisi `0`, sehingga mengetik `1`
 * menghasilkan **`01`**, bukan `1`. Angka nol itu sendiri juga karangan — layar Pega
 * menampilkan kotak **kosong**, bukan nol (`D-13`).
 *
 * Menyimpannya sebagai teks menyelesaikan keduanya sekaligus: nilai awalnya `''` sehingga
 * kotaknya benar-benar kosong, dan tidak ada nol yang perlu dihapus lebih dulu.
 *
 * `z.coerce.number()` sempat dicoba dan dibatalkan: di Zod 4 ia membuat tipe MASUKAN skema
 * menjadi `unknown`, sehingga resolver tidak lagi cocok dengan tipe form dan
 * `tsc --noEmit` menolaknya.
 *
 * # Kosong diterima, dan itu mengikuti Pega
 *
 * Keempat kolomnya nullable dan isiannya `pyRequired=false`; layar lama menyimpan isian
 * kosong tanpa keluhan. Di sini kosong dikirim sebagai `0`, karena muatan JSON bertipe
 * angka — dan pembacaan balik memperlakukan NULL sebagai `0` juga, sehingga keduanya
 * tampil sama di layar.
 */
const angka = (label: string) =>
  z
    .string()
    .trim()
    .refine(
      (nilai) => nilai === '' || /^\d+$/.test(nilai),
      `${label} harus bilangan bulat tidak negatif.`,
    )

/** Teks isian angka menjadi muatan JSON. Kosong berarti nol — lihat catatan `angka`. */
function keAngka(nilai: string): number {
  const bersih = nilai.trim()
  return bersih === '' ? 0 : Number(bersih)
}

/** Muatan JSON menjadi teks isian. Dipakai saat memuat induk yang sudah tersimpan. */
function keTeks(nilai: number): string {
  return String(nilai)
}

/**
 * Aturan yang ditegakkan di layar.
 *
 * Hanya DUA golongan, sama persis dengan yang ditegakkan server: panjang teks, dan angka
 * tidak negatif. Layar Pega tidak mewajibkan satu pun isian — keempatnya bertanda
 * `pyRequired=false` dan kolomnya nullable — sehingga menuntutnya di sini akan menolak
 * data yang hari ini sah.
 *
 * **Total share 100% TIDAK ada di sini**, dan itu bukan kelalaian: ia peringatan, bukan
 * penolakan. Layar lama menyimpan lebih dulu baru menampilkan pesannya, dan perilaku itu
 * ditiru (keputusan Work Owner 2026-09-20, `P-5`).
 */
const schema = z.object({
  // Nama TIDAK punya isian di layar — lihat catatan di XOLForm. Ia tetap ada di sini
  // supaya nilai yang sudah tersimpan ikut terbawa saat menyimpan, bukan terhapus.
  nama: z.string().trim().max(MAX.nama),

  tahun: z.string().trim().max(MAX.tahun, `Tahun paling panjang ${MAX.tahun} karakter.`),
  tipe: z.string().trim().max(MAX.tipe, `Type XOL paling panjang ${MAX.tipe} karakter.`),
  kurs: angka('Kurs IDR'),
  remark_pic: z
    .string()
    .trim()
    .max(MAX.remark, `Remark PIC paling panjang ${MAX.remark} karakter.`),

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
      nama: z
        .string()
        .trim()
        .max(MAX.layerNama, `Nama Layer paling panjang ${MAX.layerNama} karakter.`),
      limit: angka('Limit (USD)'),
      excess: angka('Excess (USD)'),
      tersimpan: z.boolean(),
      reas: z.array(
        z.object({
          id: z.string().trim().max(MAX.reasID),
          nama: z.string().trim().max(MAX.reasNama),
          share: angka('Share (%)'),
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
  kurs: '',
  remark_pic: '',
  bisnis: [],
  layer: [],
}

/** Kelas isian, dijaga sama di seluruh form supaya tidak terlihat dirakit dari dua tempat. */
const INPUT =
  'w-full rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-900 shadow-sm ' +
  'focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-100'

const LABEL = 'block text-sm font-medium text-slate-700'

type Props = {
  /** Null berarti menambah; terisi berarti mengubah induk itu. */
  master: XOL | null
  /**
   * Menutup form.
   *
   * `kabar` diisi hanya saat penyimpanan berhasil TANPA catatan — halaman menampilkannya
   * di atas daftar. Ia tidak dapat ditampilkan di dalam form, karena form itu sendiri yang
   * pergi pada saat yang sama.
   */
  tutup: (kabar?: string) => void
}

/**
 * Form Master XOL — "UPDATE DATA XOL".
 *
 * Bentuknya SAMA PERSIS dengan `Section/DetailXOL_sec-Section.xml`, dan seluruh teksnya
 * diambil apa adanya dari sana (`D-13`):
 *
 *	judul panel  UPDATE DATA XOL
 *	isian        Tahun · Kurs IDR · Type XOL · Remark PIC
 *	panel anak   Detail Group Bisnis · Detail Layer
 *	tombol       Tambah · Hapus · Simpan
 *	kosong       Data Tidak Ada
 *	dropdown     --Pilih--
 *
 * # Dua hal yang SENGAJA tidak ada, karena layar Pega pun tidak punya
 *
 *  1. **Isian Nama.** Kolom `NAMA` ada di tabel dan terisi di produksi ("Section 1" dan
 *     seterusnya), tetapi form Pega tidak menampilkannya. Nilainya karena itu dibawa
 *     diam-diam: dimuat saat membuka, dikirim kembali apa adanya saat menyimpan.
 *     Menghilangkannya dari badan permintaan akan MENGOSONGKAN nama yang sudah ada.
 *  2. **Isian ID.** Nomor diterbitkan sistem dan tidak pernah diketik.
 *
 * # Dua hal yang wajib disadari pengguna
 *
 *  1. **Menyimpan sekaligus mengajukan ke komite.** Tanpa syarat, persis seperti dua
 *     langkah terakhir `InsertUpdateMasterXOL`. Induk yang sudah disetujui kembali
 *     berstatus menunggu.
 *  2. **Tombol Hapus pada grid menghapus SEKETIKA**, tidak menunggu Simpan — juga persis
 *     seperti layar lama, yang memanggil `DeleteFromTabelMst` langsung.
 */
export function XOLForm({ master, tutup }: Props) {
  const editing = master !== null
  const detail = useXOLDetail(master?.id ?? null)

  /**
   * Nomor induk yang sedang disunting form ini.
   *
   * Ia state, bukan sekadar `master?.id`, karena **berubah di tengah jalan**: begitu
   * penambahan berhasil, nomornya baru diterbitkan server dan form berpindah dari
   * "menambah" menjadi "mengubah". Tanpa ini, menekan Simpan sekali lagi akan mengirim
   * POST untuk kedua kalinya — dan membuat induk kembar.
   *
   * Nilainya aman dimulai dari prop karena layar memberi `key` pada form, sehingga
   * berpindah baris membuatnya dirakit ulang dari awal.
   */
  const [currentID, setCurrentID] = useState(master?.id ?? '')
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

  const business = useFieldArray({ control, name: 'bisnis' })
  const layer = useFieldArray({ control, name: 'layer' })

  /**
   * Nilai baris dibaca lewat `useWatch`, BUKAN dari `fields` milik useFieldArray.
   *
   * Ini bukan pilihan gaya. `useFieldArray` menambahkan `id`-nya sendiri — kunci React
   * berupa UUID — dan kunci itu **menimpa** field bernama `id` milik data kita. Memakai
   * `fields[i].id` sebagai nomor layer karena itu menghasilkan UUID, bukan `10001`, dan
   * permintaan hapus akan menunjuk baris yang tidak ada.
   *
   * Uji `menghapus baris anak yang sudah tersimpan` yang menangkapnya: URL yang terkirim
   * berbunyi `/layer/249eccb3-…/reas/…`. `fields[i].id` karena itu hanya dipakai sebagai
   * `key` React, dan nilai sesungguhnya selalu datang dari sini.
   */
  const businessValue = useWatch({ control, name: 'bisnis' }) ?? []
  const layerValue = useWatch({ control, name: 'layer' }) ?? []

  const businessOption = useXOLBusinessGroup(tipe)

  function send(values: FieldValues) {
    save.mutate(
      {
        ...(currentID ? { id: currentID } : {}),
        input: {
          nama: values.nama,
          tahun: values.tahun,
          tipe: values.tipe,
          kurs: keAngka(values.kurs),
          remark_pic: values.remark_pic,
          // `tersimpan` adalah penanda layar, bukan bagian kontrak — server menolak badan
          // permintaan yang memuat field tak dikenal, jadi ia wajib dibuang di sini.
          bisnis: values.bisnis.map((b) => ({ id: b.id, nama: b.nama })),
          layer: values.layer.map((l) => ({
            id: l.id,
            nama: l.nama,
            limit: keAngka(l.limit),
            excess: keAngka(l.excess),
            reas: l.reas.map((r) => ({ id: r.id, nama: r.nama, share: keAngka(r.share) })),
          })),
        },
      },
      {
        onSuccess: (hasil) => {
          // Dua hal sekaligus, dan keduanya perlu.
          //
          // Nomornya dicatat supaya penyimpanan berikutnya menjadi PUT, bukan POST kedua
          // yang membuat induk kembar. Dan isian diisi ulang dari jawaban server supaya
          // nomor layer yang baru diterbitkan ikut masuk — tanpa itu, menekan Simpan dua
          // kali akan menyisipkan lapisan yang sama untuk kedua kalinya.
          setCurrentID(hasil.xol.id)
          reset(toFieldValues(hasil.xol))

          // Form ditutup HANYA bila tidak ada catatan.
          //
          // Bila ada, ia wajib tetap terbuka: catatannya menempel pada baris yang baru saja
          // disunting, dan menutup form akan membuangnya sebelum sempat dibaca. Pengguna
          // menutupnya sendiri lewat Tutup setelah membacanya — atau memperbaiki share lalu
          // menyimpan lagi, dan form menutup diri saat catatannya habis.
          if ((hasil.peringatan?.length ?? 0) === 0) {
            tutup(`Tersimpan dan diajukan ke komite — ID ${hasil.xol.id}.`)
          }
        },
      },
    )
  }

  /** Membuang satu baris anak: dari server bila sudah tersimpan, dari layar bila belum. */
  function dropChild(tersimpan: boolean, hapusLokal: () => void, hapusServer: () => void) {
    if (tersimpan && currentID) hapusServer()
    hapusLokal()
  }

  const memuat = editing && detail.isPending

  return (
    <section className="rounded-lg border border-slate-200 bg-white shadow-sm">
      <header className="border-b border-slate-200 px-5 py-4">
        {/* Judulnya diambil apa adanya dari `pyCaption UPDATE DATA XOL`. Pega memakai judul
            yang sama untuk menambah dan mengubah; keduanya tidak dibedakan di sini. */}
        <h2 className="text-base font-semibold tracking-wide text-slate-900">UPDATE DATA XOL</h2>
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
          {/* ── Empat isian, urutannya persis layar Pega ──────────────── */}
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div>
              <label className={LABEL} htmlFor="xol-tahun">
                Tahun
              </label>
              <select id="xol-tahun" className={`${INPUT} mt-1.5`} {...register('tahun')}>
                <option value="">--Pilih--</option>
                {(option.data?.tahun ?? []).map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
              <FieldError pesan={errors.tahun?.message} />
            </div>

            <div>
              <label className={LABEL} htmlFor="xol-kurs">
                Kurs IDR
              </label>
              {/* `type="text"` dengan `inputMode="numeric"`, bukan `type="number"`.
                  Kotak angka bawaan peramban menelan masukan yang tidak sah secara diam-diam
                  — "-5" dan "1,5" sampai ke form sebagai kosong — sehingga pengguna tidak
                  pernah membaca alasan penolakannya. Di sini nilainya sampai utuh dan
                  skema `angka` yang menjawabnya dengan kalimat. */}
              <input
                id="xol-kurs"
                type="text"
                inputMode="numeric"
                className={`${INPUT} mt-1.5`}
                {...register('kurs')}
              />
              <FieldError pesan={errors.kurs?.message} />
            </div>

            <div>
              <label className={LABEL} htmlFor="xol-tipe">
                Type XOL
              </label>
              <select id="xol-tipe" className={`${INPUT} mt-1.5`} {...register('tipe')}>
                <option value="">--Pilih--</option>
                {(option.data?.tipe ?? []).map((t) => (
                  <option key={t.kode} value={t.kode}>
                    {t.label}
                  </option>
                ))}
              </select>
              <FieldError pesan={errors.tipe?.message} />
            </div>

            <div>
              <label className={LABEL} htmlFor="xol-remark">
                Remark PIC
              </label>
              {/* Textarea, bukan input satu baris — layar Pega memakai `pxTextArea`. */}
              <textarea
                id="xol-remark"
                rows={3}
                className={`${INPUT} mt-1.5`}
                {...register('remark_pic')}
              />
              <FieldError pesan={errors.remark_pic?.message} />
            </div>
          </div>

          {/* ── Detail Group Bisnis ───────────────────────────────────── */}
          <Panel
            judul="Detail Group Bisnis"
            aksi={
              <Button
                tone="kedua"
                aria-label="Tambah Nama Bisnis"
                onClick={() => business.append({ id: '', nama: '', tersimpan: false })}
              >
                <AddIcon className="h-4 w-4" />
                Tambah
              </Button>
            }
          >
            <GridHead kolom={['Nama Bisnis', '']} lebar={['', '7rem']} />
            {business.fields.length === 0 ? (
              <EmptyRow kolom={2} />
            ) : (
              <tbody className="divide-y divide-slate-100">
                {business.fields.map((row, index) => {
                  // Nilainya dari useWatch; `row` hanya menyumbang kunci React.
                  const isi = businessValue[index] ?? { id: '', nama: '', tersimpan: false }
                  return (
                  <tr key={row.id}>
                    <td className="px-3 py-2">
                      <select
                        aria-label={`Nama Bisnis baris ${index + 1}`}
                        className={INPUT}
                        value={`${isi.id}\u0000${isi.nama}`}
                        onChange={(event) => {
                          const [id = '', nama = ''] = event.target.value.split('\u0000')
                          business.update(index, { id, nama, tersimpan: isi.tersimpan })
                        }}
                      >
                        <option value="\u0000">--Pilih--</option>
                        {(businessOption.data?.bisnis ?? []).map((b) => (
                          // ID dan nama dibawa bersama dalam satu nilai: baris
                          // "TREATY INWARD" tidak punya ID, sehingga ID saja tidak cukup
                          // mengenali pilihan.
                          <option key={`${b.id}-${b.nama}`} value={`${b.id}\u0000${b.nama}`}>
                            {b.nama}
                          </option>
                        ))}
                      </select>
                    </td>
                    <td className="px-3 py-2 text-right">
                      <Button
                        tone="halus"
                        aria-label={`Hapus Nama Bisnis baris ${index + 1}`}
                        // Baris tersimpan yang TIDAK punya ID tidak dapat dihapus, dan itu
                        // batasan yang diwarisi apa adanya: kueri hapus lama mencocokkan
                        // `IDBUSINESS = :id`, dan pencocokan itu tidak pernah benar untuk
                        // NULL. Dua baris seperti itu memang ada di produksi.
                        disabled={isi.tersimpan && isi.id === ''}
                        title={
                          isi.tersimpan && isi.id === ''
                            ? 'Baris ini tidak punya ID di basis data sehingga tidak dapat dihapus dari layar.'
                            : undefined
                        }
                        onClick={() =>
                          dropChild(
                            isi.tersimpan,
                            () => business.remove(index),
                            () =>
                              removeChild.mutate({
                                jenis: 'bisnis',
                                masterID: currentID,
                                id: isi.id,
                              }),
                          )
                        }
                      >
                        <TrashIcon className="h-3.5 w-3.5" />
                        Hapus
                      </Button>
                    </td>
                  </tr>
                  )
                })}
              </tbody>
            )}
          </Panel>

          {/* ── Detail Layer ──────────────────────────────────────────── */}
          <Panel
            judul="Detail Layer"
            aksi={
              <Button
                tone="kedua"
                aria-label="Tambah Layer"
                onClick={() =>
                  layer.append({
                    id: '',
                    nama: '',
                    limit: '',
                    excess: '',
                    tersimpan: false,
                    reas: [],
                  })
                }
              >
                <AddIcon className="h-4 w-4" />
                Tambah
              </Button>
            }
          >
            <GridHead
              kolom={['ID Layer', 'Nama Layer', 'Limit (USD)', 'Excess (USD)', '']}
              lebar={['8rem', '', '10rem', '10rem', '7rem']}
            />
            {layer.fields.length === 0 ? (
              <EmptyRow kolom={5} />
            ) : (
              <tbody className="divide-y divide-slate-100">
                {layer.fields.map((row, index) => {
                  // Nomor layer SELALU dari useWatch — `row.id` adalah kunci React milik
                  // useFieldArray, bukan nomor yang diterbitkan server.
                  const isi = layerValue[index]
                  const layerID = isi?.id ?? ''
                  return (
                    <LayerRow
                      key={row.id}
                      control={control}
                      register={register}
                      index={index}
                      layerID={layerID}
                      masterID={currentID}
                      errorNama={errors.layer?.[index]?.nama?.message}
                      errorLimit={errors.layer?.[index]?.limit?.message}
                      errorExcess={errors.layer?.[index]?.excess?.message}
                      onHapus={() =>
                        dropChild(
                          isi?.tersimpan === true,
                          () => layer.remove(index),
                          () =>
                            removeChild.mutate({
                              jenis: 'layer',
                              masterID: currentID,
                              id: layerID,
                            }),
                        )
                      }
                      onHapusReas={(reasID) =>
                        removeChild.mutate({
                          jenis: 'reas',
                          masterID: currentID,
                          layerID,
                          id: reasID,
                        })
                      }
                    />
                  )
                })}
              </tbody>
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

          {/* Tidak ada kabar "berhasil" di sini: penyimpanan tanpa catatan MENUTUP form, dan
              kabarnya muncul di halaman daftar — tempat ia masih terbaca setelah form pergi. */}

{save.isError && <SaveError error={save.error} />}

          {removeChild.isError && (
            <p role="alert" className="text-sm font-medium text-rose-700">
              Baris gagal dihapus di server. Muat ulang layar untuk melihat keadaan yang
              sebenarnya.
            </p>
          )}

          <div className="flex justify-end gap-2 border-t border-slate-100 pt-4">
            {/* `() => tutup()`, bukan `tutup` — tanpa pembungkus ini, objek klik terkirim
                sebagai kabar dan halaman menampilkannya sebagai pesan berhasil. */}
            <Button tone="kedua" onClick={() => tutup()} disabled={save.isPending}>
              Tutup
            </Button>
            <Button type="submit" tone="utama" disabled={save.isPending}>
              {save.isPending ? 'Menyimpan…' : 'Simpan'}
            </Button>
          </div>
        </form>
      )}
    </section>
  )
}

/**
 * Satu baris Layer beserta grid Reas-nya.
 *
 * Reas berada DI DALAM baris layer, bukan sebagai panel tersendiri: ia memang anak dari
 * lapisan, dan `UpdateMasterXOL` pun memuatnya sebagai `ObjectCoverageList` di bawah tiap
 * `ObjectList`. Grid-nya baru muncul setelah lapisannya ada — sama seperti layar lama,
 * yang hanya menampilkan "Data Tidak Ada" selama belum ada lapisan.
 */
function LayerRow({
  control,
  register,
  index,
  layerID,
  masterID,
  errorNama,
  errorLimit,
  errorExcess,
  onHapus,
  onHapusReas,
}: {
  control: Control<FieldValues>
  register: UseFormRegister<FieldValues>
  index: number
  layerID: string
  masterID: string
  errorNama?: string | undefined
  errorLimit?: string | undefined
  errorExcess?: string | undefined
  onHapus: () => void
  onHapusReas: (reasID: string) => void
}) {
  const id = useWatch({ control, name: `layer.${index}.id` }) ?? ''

  return (
    <>
      <tr>
        <td className="px-3 py-2">
          {/* ID Layer diterbitkan sistem; ditampilkan, tidak pernah diketik. */}
          <span className="font-mono text-xs text-slate-500">{id || '(baru)'}</span>
        </td>
        <td className="px-3 py-2">
          <input
            aria-label={`Nama Layer baris ${index + 1}`}
            className={INPUT}
            {...register(`layer.${index}.nama`)}
          />
          <FieldError pesan={errorNama} />
        </td>
        <td className="px-3 py-2">
          <input
            aria-label={`Limit (USD) baris ${index + 1}`}
            type="text"
            inputMode="numeric"
            className={INPUT}
            {...register(`layer.${index}.limit`)}
          />
          <FieldError pesan={errorLimit} />
        </td>
        <td className="px-3 py-2">
          <input
            aria-label={`Excess (USD) baris ${index + 1}`}
            type="text"
            inputMode="numeric"
            className={INPUT}
            {...register(`layer.${index}.excess`)}
          />
          <FieldError pesan={errorExcess} />
        </td>
        <td className="px-3 py-2 text-right align-top">
          <Button tone="halus" aria-label={`Hapus Layer baris ${index + 1}`} onClick={onHapus}>
            <TrashIcon className="h-3.5 w-3.5" />
            Hapus
          </Button>
        </td>
      </tr>
      <tr>
        <td colSpan={5} className="px-3 pb-3">
          <ReasGrid
            control={control}
            register={register}
            layerIndex={index}
            layerID={layerID}
            masterID={masterID}
            onHapusTersimpan={onHapusReas}
          />
        </td>
      </tr>
    </>
  )
}

/**
 * Grid Reas — ID, Reasuransi, Share (%).
 *
 * Ketiga judulnya diambil apa adanya dari
 * `Section/InputDetailPanelReasGenerated-Section.xml`.
 *
 * # Kenapa ID dan nama DIKETIK, bukan dipilih dari daftar
 *
 * Section yang memuat pickernya (`InputPanelReas`) **hilang dari export** (`R-16`), dan
 * pencarian sumbernya di basis data pada 2026-09-20 tidak menemukan master yang cocok:
 * dari 18 nama yang dipakai, hanya 5 ada di POOLDATA.T_REINSURER. Menebak salah satunya
 * akan membuat 13 nama yang sudah dipakai menjadi tidak dapat dipilih lagi.
 */
/**
 * Kotak "NAMA REASURANSI" beserta tombol "Cari".
 *
 * # Apa yang disalin dari Pega, dan apa yang tidak
 *
 * Teks dan susunannya mengikuti layar Pega: label **NAMA REASURANSI**, kotak berisyarat
 * `nama`, tombol **Cari**. Yang TIDAK dapat disalin adalah bentuk hasilnya — section
 * pencarian itu **tidak ada di export** (`R-16`), sehingga tidak diketahui apakah Pega
 * menampilkannya sebagai daftar, sebagai baris yang langsung masuk grid, atau sebagai
 * jendela terpisah. Di sini hasilnya berupa daftar yang dapat diklik.
 *
 * # Kenapa ditembakkan tombol, bukan ketikan
 *
 * Layar lama memang bertombol, dan menirunya sekaligus menghindari satu permintaan per
 * huruf ke master yang besar. Enter diperlakukan sama dengan menekan Cari.
 *
 * Isian bebas pada kolom ID dan Reasuransi TETAP ada. Pencarian ini menambah jalan, bukan
 * menggantinya — reasuradur yang belum ada di master masih dapat diketik, persis seperti
 * sebelum kotak ini ada.
 */
function CariReas({
  layerIndex,
  onPilih,
}: {
  layerIndex: number
  onPilih: (pilihan: XOLReinsurerOption) => void
}) {
  const [ketikan, setKetikan] = useState('')

  /** Kata kunci yang SUDAH ditekan Cari. Hanya ini yang menembak server. */
  const [kunci, setKunci] = useState('')

  const hasil = useXOLReinsurerSearch(kunci)

  function cari() {
    setKunci(ketikan.trim())
  }

  const nomor = layerIndex + 1
  const daftar = hasil.data?.reas ?? []

  return (
    <div className="mb-3 rounded-md border border-slate-200 bg-white p-3">
      <label
        className="block text-xs font-semibold uppercase tracking-wide text-slate-700"
        htmlFor={`cari-reas-${layerIndex}`}
      >
        NAMA REASURANSI
      </label>

      <div className="mt-1.5 flex flex-wrap gap-2">
        <input
          id={`cari-reas-${layerIndex}`}
          aria-label={`Nama Reasuransi layer baris ${nomor}`}
          className={`${INPUT} flex-1 min-w-[12rem]`}
          placeholder="nama"
          value={ketikan}
          onChange={(e) => setKetikan(e.target.value)}
          onKeyDown={(e) => {
            // Enter TIDAK boleh menembus ke form induk — di sana ia berarti Simpan, dan
            // mencari reasuradur akan diam-diam menyimpan seluruh master.
            if (e.key !== 'Enter') return
            e.preventDefault()
            cari()
          }}
        />
        <Button
          tone="kedua"
          aria-label={`Cari Reasuransi layer baris ${nomor}`}
          onClick={cari}
          disabled={ketikan.trim() === '' || hasil.isFetching}
        >
          {hasil.isFetching ? 'Mencari…' : 'Cari'}
        </Button>
      </div>

      {hasil.isError && (
        <p role="alert" className="mt-2 text-xs font-medium text-rose-700">
          Pencarian reasuransi gagal. Nama dan ID masih dapat diketik langsung di grid.
        </p>
      )}

      {kunci !== '' && !hasil.isFetching && !hasil.isError && daftar.length === 0 && (
        <p role="status" className="mt-2 text-xs text-slate-500">
          Data Tidak Ada
        </p>
      )}

      {daftar.length > 0 && (
        <ul className="mt-2 max-h-48 divide-y divide-slate-100 overflow-y-auto rounded-md border border-slate-200">
          {daftar.map((baris) => (
            <li key={baris.id}>
              <button
                type="button"
                onClick={() => onPilih(baris)}
                className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-sky-50"
              >
                <span className="text-slate-800">{baris.nama}</span>
                <span className="shrink-0 font-mono text-xs text-slate-500">{baris.id}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function ReasGrid({
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
    <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-700">Reas</h4>
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
          aria-label={`Tambah Reas layer baris ${layerIndex + 1}`}
          onClick={() => reas.append({ id: '', nama: '', share: '', tersimpan: false })}
        >
          <AddIcon className="h-4 w-4" />
          Tambah
        </Button>
      </div>

      <CariReas
        layerIndex={layerIndex}
        onPilih={(pilihan) => {
          // Reasuradur yang SUDAH ada di lapisan ini tidak ditambahkan dua kali. Tabel
          // MST_XOL_REAS tidak punya kunci utama, sehingga baris kembar benar-benar dapat
          // tersimpan — dan sudah terjadi di produksi pada tabel sekerabat.
          const sudahAda = isi.some((baris) => (baris?.id ?? '') === pilihan.id)
          if (sudahAda) return
          reas.append({ id: pilihan.id, nama: pilihan.nama, share: '', tersimpan: false })
        }}
      />

      <table className="w-full table-fixed border-collapse text-sm">
        <GridHead
          kolom={['ID', 'Reasuransi', 'Share (%)', '']}
          lebar={['9rem', '', '8rem', '7rem']}
        />
        {reas.fields.length === 0 ? (
          <EmptyRow kolom={4} />
        ) : (
          <tbody className="divide-y divide-slate-200">
            {reas.fields.map((row, index) => (
              <tr key={row.id}>
                <td className="px-2 py-2">
                  <input
                    aria-label={`ID reas baris ${index + 1}`}
                    className={INPUT}
                    {...register(`layer.${layerIndex}.reas.${index}.id`)}
                  />
                </td>
                <td className="px-2 py-2">
                  <input
                    aria-label={`Reasuransi baris ${index + 1}`}
                    className={INPUT}
                    {...register(`layer.${layerIndex}.reas.${index}.nama`)}
                  />
                </td>
                <td className="px-2 py-2">
                  <input
                    aria-label={`Share baris ${index + 1}`}
                    type="text"
                    inputMode="numeric"
                    className={INPUT}
                    {...register(`layer.${layerIndex}.reas.${index}.share`)}
                  />
                </td>
                <td className="px-2 py-2 text-right">
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
                </td>
              </tr>
            ))}
          </tbody>
        )}
      </table>
    </div>
  )
}

/** Panel bergrid: judul di kiri, tombol Tambah di kanan — tata letak layar Pega. */
function Panel({
  judul,
  aksi,
  children,
}: {
  judul: string
  aksi: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <section>
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-sm font-semibold text-slate-900">{judul}</h3>
        {aksi}
      </div>
      <div className="overflow-x-auto rounded-md border border-slate-200">
        <table className="w-full table-fixed border-collapse text-sm">{children}</table>
      </div>
    </section>
  )
}

function GridHead({ kolom, lebar }: { kolom: string[]; lebar: string[] }) {
  return (
    <thead className="bg-slate-50">
      <tr>
        {kolom.map((judul, index) => (
          <th
            key={judul || `kosong-${index}`}
            scope="col"
            className="px-3 py-2 text-left text-xs font-semibold text-slate-700"
            style={lebar[index] ? { width: lebar[index] } : undefined}
          >
            {judul}
          </th>
        ))}
      </tr>
    </thead>
  )
}

/** Teks kosongnya "Data Tidak Ada" — diambil apa adanya dari layar Pega. */
function EmptyRow({ kolom }: { kolom: number }) {
  return (
    <tbody>
      <tr>
        <td colSpan={kolom} className="px-3 py-4 text-sm italic text-slate-400">
          Data Tidak Ada
        </td>
      </tr>
    </tbody>
  )
}

function FieldError({ pesan }: { pesan?: string | undefined }) {
  if (!pesan) return null
  return (
    <p role="alert" className="mt-1 text-xs font-medium text-rose-700">
      {pesan}
    </p>
  )
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
    kurs: keTeks(master.kurs),
    remark_pic: master.remark_pic,
    bisnis: (master.bisnis ?? []).map((b) => ({ id: b.id, nama: b.nama, tersimpan: true })),
    layer: (master.layer ?? []).map((l) => ({
      id: l.id,
      nama: l.nama,
      limit: keTeks(l.limit),
      excess: keTeks(l.excess),
      tersimpan: true,
      reas: (l.reas ?? []).map((r) => ({
        id: r.id,
        nama: r.nama,
        share: keTeks(r.share),
        tersimpan: true,
      })),
    })),
  }
}
