import { useMemo, useState, type ReactNode } from 'react'

import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'

import {
  useEksporReas,
  useReasList,
  useReasListCounts,
  useReasMetadata,
  type ParameterDaftar,
} from './api'
import { DetailKlaimPanel } from './DetailKlaimPanel'
import { StatusSummary } from './StatusSummary'
import { bukanReasuradur, formatTanggal, pesanGalat, pesanMuat } from './pesan'
import type { Baris, Daftar } from './types'

/**
 * Layar Inbox PLA DLA — milik REASURADUR.
 *
 * Pengganti `Harness/InboxPLADLA-Harness.xml` (`MENU_ID 45`).
 *
 * # Apa yang ditampilkan layar ini, dan untuk siapa
 *
 * **Klaim yang pemberitahuannya sudah DIKIRIMKAN kepada mitra reasuransi yang sedang
 * masuk.** Login pemanggil dicocokkan ke `POOLDATA.T_REINSURER.LOGIN`, dan hasil
 * pencocokan itulah yang menentukan klaim mana yang terlihat.
 *
 * # ENAM daftar, dan di Pega tabel "Status / Jumlah" adalah navigasinya
 *
 * Seluruhnya dikendalikan satu nilai di Pega — `TempView.CityID`, yang `SetDataPLADLA`
 * terima sebagai `param.tipe`:
 *
 *	PLA                       PLA sudah dikirimkan kepada Anda, DLA belum
 *	PLA & DLA                 DLA sudah dikirimkan kepada Anda
 *	CLOSE CLAIM               klaimnya sudah selesai dan tidak menunggu penutupan
 *	NOT ANSWERED              ada pesan untuk Anda yang belum Anda jawab
 *	NOT REPLIED FROM ASM      pesan yang Anda kirim dan belum dijawab
 *	REPLIED FROM ASM          pesan yang Anda kirim dan sudah dijawab
 *
 * Layar Pega TIDAK punya bilah tab — `pyLayoutFormat>TABBED` nol kemunculan di
 * `Section/InboxDLAReas_sect-Section.xml`. Yang berpindah daftar adalah ANGKA pada tabel
 * "Status / Jumlah": selnya ber-`pyFormat=pxLink` dengan aksi `refresh` yang membawa
 * kode daftarnya. Bilah tab yang sempat dipakai di sini dicabut 2026-10-09.
 *
 * # Kenapa "DATA PLA DLA XOL KLAIM" TIDAK ada di sini
 *
 * `SetDataPLADLA` memang punya cabang `param.tipe=="7"` yang mengisi `DataKomiteXOL`, dan
 * sectionnya memang punya wadah berjudul "DATA PLA DLA XOL KLAIM". Tetapi wadah itu
 * bersyarat `pyContainerVisibleWhen = TempView.CityID==7`, dan `CityID` HANYA pernah
 * diisi dari `.CityID` sebuah baris tabel "Status / Jumlah" — literal `7` nol kemunculan
 * sebagai nilai `tipe` di seluruh export. Tabelnya sendiri berisi enam baris.
 *
 * Artinya di Pega tampilan itu **tidak pernah dapat tergambar**: ia kode mati. Ia sempat
 * dibangun di sini sebagai tab ketujuh (2026-09-28) lalu sebagai panel permanen
 * (2026-10-09 pagi), dan keduanya memberi akses kepada sesuatu yang layar lama tidak
 * pernah tampilkan. Dicabut 2026-10-09 atas keputusan Work Owner.
 *
 * # JANGAN tertukar dengan "Inbox PLA, DLA, Pre DLA" (`MENU_ID 44`)
 *
 * Penyaringnya BERLAWANAN ARAH. Layar itu menampilkan dokumen yang belum dikirim, untuk
 * petugas internal; layar ini menampilkan dokumen yang sudah dikirim, untuk penerimanya.
 *
 * # Petugas internal yang membuka layar ini DITOLAK, bukan dibiarkan melihat layar kosong
 *
 * Login yang tidak terdaftar sebagai mitra menerima pesan yang menjelaskan sebabnya dan
 * menunjuk menu yang benar. Di Pega ia hanya melihat layar kosong tanpa satu pun
 * keterangan — dan menyimpulkan sistemnya rusak.
 *
 * # Layar induk BACA-SAJA; yang menulis ada di RINCIAN
 *
 * Tombol **"Detail Claim"** membuka panel rincian klaim, dan di dalamnya ada satu operasi
 * tulis: balasan komunikasi oleh mitra reasuransi.
 *
 * Dua tombol lain pada layar Pega TIDAK dibawa, dan keduanya bukan penundaan:
 *
 *	"DLA"     berada di dalam wadah bersyarat `1==2` — tidak pernah tergambar di Pega
 *	"Detail"  hanya tampil bagi satu Operator ID yang ditulis tetap di dalam rule (`D-15`)
 */
export function InboxPLADLAReasPage() {
  const meta = useReasMetadata()

  const [tab, setTab] = useState('')
  const [page, setPage] = useState(1)

  // Kotak "Claim No" dipegang DUA KALI: yang sedang diketik, dan yang sudah dikirim.
  //
  // Layar lamanya memakai tombol "Search Data", bukan pencarian saat mengetik — dan
  // penyaringnya menyapu tabel klaim yang sama dengan modul lain.
  const [ketikan, setKetikan] = useState('')
  const [dicari, setDicari] = useState('')

  // Klaim yang rinciannya sedang terbuka. Null berarti panelnya tertutup.
  const [rincian, setRincian] = useState<Baris | null>(null)

  const daftar = meta.data?.daftar ?? []
  const aktif = tab || meta.data?.daftar_bawaan || ''
  const daftarAktif = daftar.find((item) => item.kode === aktif)

  const parameter: ParameterDaftar = useMemo(
    () => ({ tab: aktif, page, cari: dicari }),
    [aktif, page, dicari],
  )

  const siapDaftar = aktif !== ''
  const list = useReasList(parameter, siapDaftar)
  const ekspor = useEksporReas()

  // Tabel "Status / Jumlah" TIDAK menerima daftar yang sedang terbuka.
  //
  // Ia menyebut keenamnya sekaligus — itulah yang membuatnya dapat dipakai berpindah —
  // sehingga berpindah daftar tidak membuatnya dihitung ulang. Yang mengubahnya hanyalah
  // pencarian, karena angkanya memang menyusut bersama tabel di sebelahnya.
  const ringkas = useReasListCounts(dicari, siapDaftar)

  // Penolakan "bukan mitra terdaftar" dikenali dari galat daftarnya.
  const ditolak = bukanReasuradur(list.error)

  function pilihDaftar(kode: string) {
    setTab(kode)
    setPage(1)

    // Panel rincian DIBUANG saat tampilan berpindah.
    //
    // Tanpa ini, rincian klaim dari tab sebelumnya tetap terbuka di bawah daftar yang
    // sudah berganti — dan pengguna membaca rincian yang tidak ada hubungannya dengan
    // baris mana pun yang sedang ia lihat.
    setRincian(null)
  }

  function cari() {
    setDicari(ketikan.trim())
    setPage(1)
  }

  if (meta.isPending) {
    return (
      <Bingkai>
        <p className="text-sm text-slate-600">Memuat keterangan layar…</p>
      </Bingkai>
    )
  }

  if (meta.isError) {
    const pesan = pesanMuat(meta.error)
    return (
      <Bingkai>
        <ErrorMessage
          title={pesan.title}
          description={pesan.description}
          tone={pesan.tone}
        />
      </Bingkai>
    )
  }

  // Penolakan "bukan mitra terdaftar" menggantikan SELURUH isi layar, bukan digambar
  // sebagai galat di atas tabel kosong.
  //
  // Ia bukan kegagalan sementara yang layak dicoba ulang: selama pendaftarannya belum
  // diubah, jawabannya akan sama. Menggambar tabel, tombol ekspor, dan bilah tab di
  // bawahnya hanya menawarkan hal-hal yang seluruhnya akan ditolak.
  if (ditolak) {
    const pesan = pesanMuat(list.error)
    return (
      <Bingkai>
        <ErrorMessage
          title={pesan.title}
          description={pesan.description}
          tone={pesan.tone}
        />
      </Bingkai>
    )
  }

  return (
    <Bingkai>
      {daftarAktif && (
        <p className="text-sm text-slate-600">{daftarAktif.keterangan}</p>
      )}

      {/*
        Grid di KIRI, tabel "Status / Jumlah" di KANAN — susunan Pega apa adanya.

        Ia menumpuk pada layar sempit, dan itu bukan penyimpangan: `D-12` menuntut layar
        ini tetap terpakai di tablet, dan tabel dua kolom yang dipaksa berdampingan di
        lebar 768px akan memeras gridnya sampai tidak terbaca. Pada lebar desktop —
        tempat layar ini sebenarnya dipakai — susunannya sama dengan layar lama.
      */}
      <div className="flex flex-col gap-5 xl:flex-row xl:items-start">
        <div className="min-w-0 flex-1 space-y-5">
          <DataTable<Baris>
            columns={kolomDaftar(daftarAktif, rincian, setRincian)}
            rows={list.data?.baris ?? []}
            rowKey={(row) => row.kunci_klaim}
            label={`Daftar ${daftarAktif?.nama ?? ''}`}
            isLoading={list.isLoading}
            error={
              list.isError ? (
                <ErrorMessage
                  title="Daftar tidak dapat dimuat"
                  description={pesanGalat(list.error)}
                  tone="gangguan"
                />
              ) : undefined
            }
            emptyMessage={pesanKosong(dicari, daftarAktif)}
            searchLabel="Claim No"
            serverSearch={{
              value: ketikan,
              onChange: setKetikan,
              matchCount: list.data?.paginasi.total,
            }}
            pagination={{
              page: list.data?.paginasi.halaman ?? 1,
              size: list.data?.paginasi.ukuran ?? 10,
              total: list.data?.paginasi.total ?? 0,
              totalPage: list.data?.paginasi.total_halaman ?? 1,
              onPageChange: setPage,
              isLoading: list.isFetching,
            }}
            actions={
              <div className="flex flex-wrap items-center gap-2">
                <Button type="button" onClick={cari} disabled={list.isFetching}>
                  {list.isFetching ? 'Mencari…' : 'Search Data'}
                </Button>
                <Button
                  type="button"
                  tone="kedua"
                  onClick={() => { list.refetch() }}
                  disabled={list.isFetching}
                >
                  Refresh
                </Button>
                <Button
                  type="button"
                  tone="kedua"
                  disabled={ekspor.isPending}
                  onClick={() => ekspor.mutate(parameter)}
                >
                  {ekspor.isPending ? 'Menyiapkan…' : 'Export To Excel'}
                </Button>
              </div>
            }
          />

          {ekspor.error != null && (
            <ErrorMessage
              title="Berkas ekspor tidak dapat diambil"
              description={pesanGalat(ekspor.error)}
              tone="gangguan"
            />
          )}
        </div>

        <div className="w-full xl:w-96 xl:shrink-0">
          <StatusSummary
            rows={ringkas.data?.baris ?? []}
            active={aktif}
            isLoading={ringkas.isLoading}
            onSelect={pilihDaftar}
          />

          {ringkas.isError && (
            <div className="mt-3">
              <ErrorMessage
                title="Jumlah tiap daftar tidak dapat dihitung"
                description={pesanGalat(ringkas.error)}
                tone="gangguan"
              />
            </div>
          )}
        </div>
      </div>

      {rincian && (
        <DetailKlaimPanel
          kunciKlaim={rincian.kunci_klaim}
          nomorKlaim={rincian.no_klaim}
          onClose={() => setRincian(null)}
        />
      )}
    </Bingkai>
  )
}

/** Bingkai adalah judul layar beserta ruang isinya. */
function Bingkai({ children }: { children: ReactNode }) {
  return (
    <div className="space-y-5 p-6">
      <header>
        <h1 className="text-lg font-semibold text-slate-900">Inbox PLA / DLA</h1>
        <p className="mt-1 text-sm text-slate-600">
          Klaim yang pemberitahuan PLA atau DLA-nya sudah dikirimkan kepada Anda sebagai
          mitra reasuransi.
        </p>
      </header>

      {children}
    </div>
  )
}

/**
 * kolomDaftar menerjemahkan kolom yang DIKIRIM SERVER menjadi kolom DataTable.
 *
 * Alias Pega-nya menyesatkan hampir seluruhnya — `"TSI"` berarti kunci klaim,
 * `"MARKETING"` berarti nama lini bisnis, `"CURRENCY"` berarti tanggal kejadian — dan
 * menulis ulang pemetaannya di layar berarti dua tempat yang dapat bergeser.
 */
function kolomDaftar(
  daftar: Daftar | undefined,
  rincian: Baris | null,
  bukaRincian: (row: Baris | null) => void,
): Column<Baris>[] {
  if (!daftar) return []

  const kolom: Column<Baris>[] = daftar.kolom.map((item) => ({
    key: item.kunci,
    title: item.judul,
    value: (row) => nilaiSel(row, item.kunci),
    render: (row) => gambarSel(row, item.kunci, item.tanggal),
  }))

  // Apakah barisnya punya tombol rincian datang dari SERVER, bukan dicocokkan di layar.
  if (!daftar.punya_rincian) return kolom

  return [
    ...kolom,
    {
      key: 'rincian',
      title: 'Aksi',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => {
        const terbuka = rincian?.kunci_klaim === row.kunci_klaim

        return (
          <Button
            type="button"
            tone="kedua"
            aria-pressed={terbuka}
            onClick={() => bukaRincian(terbuka ? null : row)}
          >
            {terbuka ? 'Tutup' : 'Detail Claim'}
          </Button>
        )
      },
    },
  ]
}

/** nilaiSel mengambil isi satu sel sebagai TEKS — yang dicari dan diurutkan. */
function nilaiSel(row: Baris, kunci: string): string {
  const sel = (row as unknown as Record<string, unknown>)[kunci]
  return typeof sel === 'string' ? sel : ''
}

/**
 * gambarSel menggambar satu sel.
 *
 * Kolom "Status" jatuh ke KODENYA bila artinya tidak ada di master. Kode status yang
 * tidak terdaftar memang terjadi — domainnya 33 kode dan data lama memuat kode di luar
 * itu — dan sel kosong akan membuat barisnya tampak rusak.
 */
function gambarSel(row: Baris, kunci: string, tanggal: boolean) {
  if (kunci === 'status') {
    const isi = row.status || row.kode_status
    if (isi === '') return <span className="text-slate-400">—</span>
    return isi
  }

  const isi = nilaiSel(row, kunci)
  if (tanggal) return formatTanggal(isi)
  if (isi === '') return <span className="text-slate-400">—</span>
  return isi
}

/**
 * pesanKosong menjelaskan MENGAPA daftarnya kosong.
 *
 * Kalimatnya berbeda antara daftar pemberitahuan dan daftar komunikasi, dan itu bukan
 * kehalusan: keduanya kosong karena sebab yang berbeda. Satu kalimat untuk keduanya akan
 * membuat mitra yang membuka "Komunikasi Masuk" membaca keterangan tentang pemberitahuan
 * yang tidak ada hubungannya dengan tab itu.
 */
function pesanKosong(dicari: string, daftar: Daftar | undefined): string {
  if (dicari !== '') {
    return `Tidak ada klaim yang nomornya mengandung "${dicari}" pada daftar ini.`
  }

  // Sumbernya datang dari SERVER. Sebelumnya layar menyimpulkannya dari awalan kode tab,
  // dan kesimpulan itu pecah diam-diam begitu kodenya berubah mengikuti judul sebenarnya
  // — tanpa satu pun galat, hanya kalimat yang salah.
  if (daftar?.sumber === 'komunikasi') {
    return 'Belum ada klaim dengan komunikasi pada tahap ini yang menyangkut Anda.'
  }

  return 'Belum ada klaim pada tahap ini yang pemberitahuannya dikirimkan kepada Anda.'
}

