import { useState, type ReactNode } from 'react'

import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { TextAreaField } from '@/components/TextAreaField'

import {
  useBalasKomunikasi,
  useDokumenPemberitahuan,
  useRincianKlaim,
  useTindakanBelumTersedia,
  useUnduhDokumen,
} from './api'
import { formatTanggal, pesanGalat, pesanMuat } from './pesan'
import type {
  BarisDokumen,
  BarisKomunikasi,
  BarisPemberitahuan,
  Kolom,
} from './types'

/**
 * Panel rincian satu klaim — tombol **"Detail Claim"**.
 *
 * # Asalnya di sistem lama
 *
 * Di Pega ia POPUP: `showHarness` ke `ViewDetailClaimReas` dengan `pyTarget = popup`. Di
 * sini ia panel yang terbuka DI BAWAH daftarnya, mengikuti preseden "Detail Salvage" pada
 * modul Inbox Salvage.
 *
 * Perbedaannya disengaja. Popup memaksa pengguna menutupnya sebelum dapat membandingkan
 * dengan baris lain, dan pada layar yang dibaca mitra dari luar jaringan kantor, jendela
 * yang terbuka di atas jendela lain lebih sering diblokir peramban daripada dibuka.
 *
 * # Empat bagian, dan satu di antaranya MENULIS
 *
 *	kepala klaim          keterangan klaimnya
 *	grid PLA · grid DLA   pemberitahuan MILIK PEMANGGIL yang sudah terkirim
 *	dokumen               dibuka lewat tombol "Dokumen" pada salah satu baris di atas
 *	riwayat komunikasi    beserta tombol "Balas Pesan"
 */
type Props = {
  /** Kunci objek kerja klaim — `T_CLAIM_PNC.CLAIMID`. */
  kunciKlaim: string

  /** Nomor klaim, dipakai judul panel supaya pengguna tahu apa yang sedang dibuka. */
  nomorKlaim: string

  onClose: () => void
}

/** Pilihan dokumen yang sedang dibuka — nomor pemberitahuan beserta jenisnya. */
type PilihanDokumen = { nomor: string; jenis: string }

export function DetailKlaimPanel({ kunciKlaim, nomorKlaim, onClose }: Props) {
  const rincian = useRincianKlaim(kunciKlaim, true)
  const [dokumen, setDokumen] = useState<PilihanDokumen | null>(null)

  const isi = rincian.data

  return (
    <section
      aria-label={`Rincian klaim ${nomorKlaim}`}
      className="space-y-4 rounded-kartu border border-slate-300 bg-slate-50 p-4"
    >
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold text-slate-900">
            Rincian Klaim {nomorKlaim}
          </h2>
          <p className="mt-1 text-sm text-slate-600">
            Pemberitahuan dan dokumen yang dikirimkan kepada Anda atas klaim ini,
            beserta riwayat komunikasinya.
          </p>
        </div>

        <Button type="button" tone="kedua" onClick={onClose}>
          Tutup Rincian
        </Button>
      </header>

      {rincian.isPending && (
        <p className="text-sm text-slate-600">Memuat rincian klaim…</p>
      )}

      {rincian.isError && (
        <ErrorMessage {...pesanMuat(rincian.error)} />
      )}

      {isi && (
        <>
          <KepalaKlaim isi={isi.klaim} />

          <GridPemberitahuan
            judul="PLA"
            kolom={isi.kolom_pla}
            baris={isi.pla}
            dipilih={dokumen}
            onPilih={setDokumen}
            pesanKosong={
              'Belum ada PLA yang dikirimkan kepada Anda atas klaim ini.'
            }
          />

          <GridPemberitahuan
            judul="DLA"
            kolom={isi.kolom_dla}
            baris={isi.dla}
            dipilih={dokumen}
            onPilih={setDokumen}
            pesanKosong={
              'Belum ada DLA yang dikirimkan kepada Anda atas klaim ini.'
            }
          />

          {dokumen && (
            <PanelDokumen
              kunciKlaim={kunciKlaim}
              pilihan={dokumen}
              kolom={isi.kolom_dokumen}
              onClose={() => setDokumen(null)}
            />
          )}

          <PanelKomunikasi
            kunciKlaim={kunciKlaim}
            kolom={isi.kolom_komunikasi}
            baris={isi.komunikasi}
          />
        </>
      )}
    </section>
  )
}

/** KepalaKlaim menggambar keterangan klaim di kepala panel. */
function KepalaKlaim({ isi }: { isi: NonNullable<ReturnType<typeof useRincianKlaim>['data']>['klaim'] }) {
  return (
    <dl className="grid grid-cols-1 gap-x-6 gap-y-3 rounded-lg border border-slate-200 bg-white p-4 sm:grid-cols-2 lg:grid-cols-4">
      <Isian label="Claim No" nilai={isi.no_klaim} />
      <Isian label="Policy No" nilai={isi.no_polis} />
      <Isian label="Insured" nilai={isi.nama_tertanggung} />
      <Isian label="Business Name" nilai={isi.nama_bisnis} />
      <Isian label="DOL" nilai={formatTanggal(isi.tanggal_kejadian)} />
      <Isian label="Register Date" nilai={formatTanggal(isi.tanggal_register)} />
      <Isian label="PIC ASM" nilai={isi.pic_teknik} />
      <Isian
        label="Claim Progress"
        nilai={isi.status || isi.kode_status}
      />
    </dl>
  )
}

/** Isian adalah satu label beserta isinya. */
function Isian({ label, nilai }: { label: string; nilai: string }) {
  const bersih = nilai.trim()

  return (
    <div className="min-w-0">
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">
        {label}
      </dt>
      <dd className="mt-0.5 break-words text-sm text-slate-900">
        {bersih === '' ? <span className="text-slate-400">—</span> : bersih}
      </dd>
    </div>
  )
}

/**
 * GridPemberitahuan menggambar grid PLA atau grid DLA.
 *
 * Kolomnya datang dari SERVER, termasuk kolom "No Akseptasi" yang hanya ada pada DLA.
 * Tombol "Dokumen" ditambahkan di sini karena ia bukan kolom data melainkan aksi.
 */
function GridPemberitahuan({
  judul,
  kolom,
  baris,
  dipilih,
  onPilih,
  pesanKosong,
}: {
  judul: string
  kolom: Kolom[]
  baris: BarisPemberitahuan[]
  dipilih: PilihanDokumen | null
  onPilih: (pilihan: PilihanDokumen) => void
  pesanKosong: string
}) {
  const tindakan = useTindakanBelumTersedia()

  const kolomTabel: Column<BarisPemberitahuan>[] = [
    ...kolom.map((item) => ({
      key: item.kunci,
      title: item.judul,
      value: (row: BarisPemberitahuan) => selPemberitahuan(row, item.kunci),
      render: (row: BarisPemberitahuan) =>
        item.tanggal
          ? formatTanggal(selPemberitahuan(row, item.kunci))
          : isiAtauStrip(selPemberitahuan(row, item.kunci)),
    })),
    {
      key: 'dokumen',
      title: 'Dokumen',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button
          type="button"
          tone="kedua"
          onClick={() => onPilih({ nomor: row.nomor, jenis: row.jenis })}
          aria-pressed={
            dipilih?.nomor === row.nomor && dipilih?.jenis === row.jenis
          }
        >
          Dokumen
        </Button>
      ),
    },
  ]

  return (
    <div className="space-y-2">
      <DataTable<BarisPemberitahuan>
        columns={kolomTabel}
        rows={baris}
        rowKey={(row) => `${row.jenis}-${row.nomor}`}
        title={judul}
        label={`Daftar ${judul}`}
        emptyMessage={pesanKosong}
        actions={
          <Button
            type="button"
            tone="kedua"
            disabled={tindakan.isPending}
            onClick={() =>
              tindakan.mutate(
                judul === 'PLA' ? 'unduh-semua-pla' : 'unduh-semua-dla',
              )
            }
          >
            Download ALL {judul}
          </Button>
        }
      />

      {/* Alasan tombol yang belum dibangun digambar TEPAT di bawah tombolnya, bukan di
          kaki halaman: alasan yang jauh dari tombolnya tidak terbaca sebagai jawaban
          atas penekanan barusan. */}
      {tindakan.error != null && (
        <ErrorMessage
          title={`Tombol "Download ALL ${judul}" belum tersedia`}
          description={pesanGalat(tindakan.error)}
          tone="penolakan"
        />
      )}
    </div>
  )
}

/** PanelDokumen menggambar dokumen satu nomor pemberitahuan. */
function PanelDokumen({
  kunciKlaim,
  pilihan,
  kolom,
  onClose,
}: {
  kunciKlaim: string
  pilihan: PilihanDokumen
  kolom: Kolom[]
  onClose: () => void
}) {
  const dokumen = useDokumenPemberitahuan(
    kunciKlaim,
    pilihan.nomor,
    pilihan.jenis,
  )
  const unduh = useUnduhDokumen(kunciKlaim)

  const kolomTabel: Column<BarisDokumen>[] = [
    ...kolom.map((item) => ({
      key: item.kunci,
      title: item.judul,
      value: (row: BarisDokumen) => selDokumen(row, item.kunci),
      render: (row: BarisDokumen) =>
        isiAtauStrip(selDokumen(row, item.kunci)),
    })),
    {
      key: 'aksi',
      title: 'Berkas',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button
          type="button"
          tone="kedua"
          disabled={unduh.isPending}
          onClick={() => unduh.mutate({ id: row.id, nama: row.nama })}
        >
          {unduh.isPending ? 'Menyiapkan…' : 'View Document'}
        </Button>
      ),
    },
  ]

  return (
    <div className="space-y-2">
      <DataTable<BarisDokumen>
        columns={kolomTabel}
        rows={dokumen.data?.baris ?? []}
        rowKey={(row) => row.id}
        title={`Dokumen ${pilihan.jenis} ${pilihan.nomor}`}
        label={`Dokumen ${pilihan.jenis} ${pilihan.nomor}`}
        isLoading={dokumen.isLoading}
        error={
          dokumen.isError ? (
            <ErrorMessage
              title="Dokumen tidak dapat dimuat"
              description={pesanGalat(dokumen.error)}
              tone="gangguan"
            />
          ) : undefined
        }
        emptyMessage={
          'Tidak ada dokumen yang dikirimkan kepada Anda pada nomor ini.'
        }
        actions={
          <Button type="button" tone="kedua" onClick={onClose}>
            Tutup Dokumen
          </Button>
        }
      />

      {unduh.error != null && (
        <ErrorMessage
          title="Berkas tidak dapat diunduh"
          description={pesanGalat(unduh.error)}
          tone="gangguan"
        />
      )}
    </div>
  )
}

/**
 * PanelKomunikasi menggambar riwayat komunikasi beserta form balasannya.
 *
 * # Ia satu-satunya bagian layar ini yang MENULIS
 *
 * Dan yang menulis adalah pihak LUAR. Form balasannya karena itu tertutup sampai pengguna
 * menekan "Balas Pesan" pada satu baris — bukan tergambar di setiap baris sekaligus.
 * Preseden yang sama sudah ada pada modul Inbox Komunikasi Cabang.
 */
function PanelKomunikasi({
  kunciKlaim,
  kolom,
  baris,
}: {
  kunciKlaim: string
  kolom: Kolom[]
  baris: BarisKomunikasi[]
}) {
  const [membalas, setMembalas] = useState<string | null>(null)
  const [isian, setIsian] = useState('')
  const balas = useBalasKomunikasi(kunciKlaim)

  function mulaiMembalas(id: string) {
    setMembalas(id)
    setIsian('')
    balas.reset()
  }

  function kirim() {
    if (membalas === null) return
    balas.mutate(
      { percakapan: membalas, balasan: isian },
      {
        onSuccess: () => {
          setMembalas(null)
          setIsian('')
        },
      },
    )
  }

  const kolomTabel: Column<BarisKomunikasi>[] = [
    ...kolom.map((item) => ({
      key: item.kunci,
      title: item.judul,
      value: (row: BarisKomunikasi) => selKomunikasi(row, item.kunci),
      render: (row: BarisKomunikasi) =>
        item.tanggal
          ? formatTanggal(selKomunikasi(row, item.kunci))
          : isiAtauStrip(selKomunikasi(row, item.kunci)),
    })),
    {
      key: 'aksi',
      title: 'Aksi',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) =>
        // Tombolnya digambar menurut jawaban SERVER, bukan menurut kesimpulan layar.
        // Layar yang menyimpulkannya sendiri akan menggambar tombol pada percakapan
        // yang permintaannya justru ditolak.
        row.boleh_dibalas ? (
          <Button
            type="button"
            tone="kedua"
            onClick={() => mulaiMembalas(row.id)}
          >
            Balas Pesan
          </Button>
        ) : (
          <span className="text-xs text-slate-500">Sudah dijawab</span>
        ),
    },
  ]

  return (
    <div className="space-y-3">
      <DataTable<BarisKomunikasi>
        columns={kolomTabel}
        rows={baris}
        rowKey={(row) => row.id}
        title="Komunikasi"
        label="Riwayat komunikasi klaim"
        emptyMessage="Belum ada komunikasi pada klaim ini yang menyangkut Anda."
      />

      {membalas !== null && (
        <form
          className="space-y-3 rounded-lg border border-slate-200 bg-white p-4"
          onSubmit={(event) => {
            event.preventDefault()
            kirim()
          }}
        >
          <TextAreaField
            id="balasan-komunikasi-reas"
            label="Balasan Anda"
            value={isian}
            onChange={(event) => setIsian(event.target.value)}
            rows={4}
            hint={
              'Balasan hanya dapat dikirim SATU KALI per percakapan, dan langsung ' +
              'terbaca petugas Asuransi Sinar Mas.'
            }
          />

          <div className="flex flex-wrap gap-2">
            <Button type="submit" disabled={balas.isPending}>
              {balas.isPending ? 'Mengirim…' : 'Kirim Balasan'}
            </Button>
            <Button
              type="button"
              tone="kedua"
              onClick={() => setMembalas(null)}
              disabled={balas.isPending}
            >
              Batal
            </Button>
          </div>

          {balas.error != null && (
            <ErrorMessage
              title="Balasan tidak tersimpan"
              description={pesanGalat(balas.error)}
              tone="gangguan"
            />
          )}
        </form>
      )}
    </div>
  )
}

/** selPemberitahuan mengambil isi satu sel grid pemberitahuan sebagai teks. */
function selPemberitahuan(row: BarisPemberitahuan, kunci: string): string {
  const sel = (row as unknown as Record<string, unknown>)[kunci]
  return typeof sel === 'string' ? sel : ''
}

/** selDokumen mengambil isi satu sel grid dokumen sebagai teks. */
function selDokumen(row: BarisDokumen, kunci: string): string {
  const sel = (row as unknown as Record<string, unknown>)[kunci]
  return typeof sel === 'string' ? sel : ''
}

/** selKomunikasi mengambil isi satu sel grid komunikasi sebagai teks. */
function selKomunikasi(row: BarisKomunikasi, kunci: string): string {
  const sel = (row as unknown as Record<string, unknown>)[kunci]
  return typeof sel === 'string' ? sel : ''
}

/** isiAtauStrip menggambar tanda pisah untuk sel yang kosong. */
function isiAtauStrip(isi: string): ReactNode {
  if (isi.trim() === '') return <span className="text-slate-400">—</span>
  return isi
}
