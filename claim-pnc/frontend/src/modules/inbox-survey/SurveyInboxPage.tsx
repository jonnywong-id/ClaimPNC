import { useEffect, useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { ReloadIcon } from '@/components/Icon'
import { TabBar, type TabItem } from '@/components/TabBar'

import {
  PAGE_SIZE,
  useDaftarSurvei,
  useJumlahTabSurvei,
  useKPISurvei,
  useKeteranganSurvei,
} from './api'
import type {
  BarisKPI,
  IdentitasSurveyor,
  KolomLayar,
  TabLayar,
  TugasSurvei,
} from './types'

/** Dua tab besar pada harness: INBOX dan KPI. */
type Bagian = 'inbox' | 'kpi'

/**
 * My Work — menu `MENU_ID 50`, pengganti harness `InboxSurvey_Harness`.
 *
 * Isinya antrean kerja **Surveyor dan Loss Adjuster**: janji survei yang menunggu
 * dikonfirmasi, dikerjakan, ditagihkan, atau ditutup. Per `D-79` ia benar-benar Inbox —
 * barisnya pekerjaan milik pemanggil, hilang begitu tugasnya tuntas, punya tenggat (kolom
 * Aging), dan "hanya milik saya" adalah aturan kewenangan.
 *
 * JANGAN tertukar dengan menu `MENU_ID 80` "Lost Adjuster" — butir menu berbeda yang
 * harness-nya tidak ada di export sama sekali (`K-33`).
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Harness memuat DUA tab besar, dan keduanya dibangun:
 *
 *   INBOX  ketiga belas kolom dan ketujuh tab status
 *   KPI    ringkasan penilaian adjuster, dari `POOLDATA.DETAIL_KPI_ADJUSTER`
 *
 * Judul kolom dan judul tab datang dari SERVER, bukan diketik di sini: keduanya hasil
 * pembacaan `Section/InboxSurvey_section-Section.xml` yang tercatat di backend.
 *
 * # Layar ini melayani DUA populasi sekaligus
 *
 * Surveyor internal dan Loss Adjuster eksternal melihat layar yang sama; yang membedakan
 * isinya adalah identitas yang masuk, bukan penyaring yang dipilih pengguna. Keputusan Work
 * Owner 2026-09-28.
 *
 * # Layar ini hanya MEMBACA
 *
 * Menerima penugasan, menjadwal ulang survei, dan mengunggah laporan seluruhnya menempuh
 * `Surveyor_Flow` — sebuah flow yang TIDAK ADA di export. Selama masa paralel penugasan tetap
 * dikerjakan di Pega (`P-1`). Tidak ada tombol yang mengubah apa pun di sini, dan
 * ketiadaannya disengaja.
 */
export function SurveyInboxPage() {
  const navigate = useNavigate()
  const portal = useSelectedPortal((state) => state.alias)

  const [bagian, setBagian] = useState<Bagian>('inbox')
  const [tab, setTab] = useState('')
  const [cari, setCari] = useState('')
  const [lewati, setLewati] = useState(0)

  const [jenisKPI, setJenisKPI] = useState('outstanding')
  const [tahunKPI, setTahunKPI] = useState('')

  const keterangan = useKeteranganSurvei()
  const jumlahTab = useJumlahTabSurvei()
  const daftar = useDaftarSurvei(tab, cari, lewati)
  const kpi = useKPISurvei(jenisKPI, '', tahunKPI, bagian === 'kpi')

  /**
   * Tab bawaan datang dari SERVER, bukan ditulis tetap di sini.
   *
   * Alasannya bukan kerapian: yang menentukan tab mana yang terbuka pertama kali adalah
   * keputusan yang tercatat di domain (`inboxsurvey.DefaultTab`), dan menuliskannya juga di
   * sini akan membuat keduanya dapat berbeda tanpa ada yang menyadarinya.
   *
   * Ia hanya dipasang SEKALI, saat tab masih kosong. Tanpa penjagaan itu, setiap pemuatan
   * ulang keterangan akan melempar pengguna kembali ke tab Outstanding di tengah pekerjaan.
   */
  const tabBawaan = keterangan.data?.tab_bawaan ?? ''
  useEffect(() => {
    if (tab === '' && tabBawaan !== '') setTab(tabBawaan)
  }, [tab, tabBawaan])

  /**
   * Berpindah tab SEKALIGUS kembali ke halaman pertama dan mengosongkan pencarian.
   *
   * Pencarian ikut dikosongkan karena kata kunci yang cocok di satu tab hampir selalu tidak
   * cocok di tab lain — dan hasil kosong sesudah berpindah tab terbaca sebagai "tab itu
   * memang kosong", bukan sebagai "pencarian Anda masih menyala".
   */
  function pindahTab(kunci: string) {
    setTab(kunci)
    setCari('')
    setLewati(0)
  }

  /**
   * Mengubah kata kunci SEKALIGUS kembali ke halaman pertama.
   *
   * Tanpa penyetelan ulang itu, mengetik kata kunci saat berada di halaman lima akan
   * menampilkan halaman lima dari hasil yang baru — yang hampir selalu kosong, dan terbaca
   * sebagai "tidak ditemukan".
   */
  function ubahPencarian(nilai: string) {
    setCari(nilai)
    setLewati(0)
  }

  /**
   * Tujuan tautan baris.
   *
   * Di sistem lama, tautannya membuka penugasan `Surveyor_Flow` lewat kunci
   * `ASSIGN-WORKLIST <pyID>!Surveyor_Flow`. Flow itu TIDAK ADA di export, sehingga yang
   * dibuka di sini adalah klaimnya — bukan penugasan surveinya.
   *
   * Yang dikirim nomor klaimnya saja: `D-22` menetapkan sistem baru tidak pernah menuliskan
   * awalan kelas Pega lagi.
   */
  function bukaKlaim(nomorKlaim: string) {
    navigate(`/view-claim/${encodeURIComponent(nomorKlaim)}`)
  }

  if (portal === null) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Antrean survei milik satu badan hukum, dan aplikasi ini melayani empat. Pilih ' +
            'portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  if (keterangan.isError) {
    // Kegagalan keterangan layar HARUS ditampilkan, dan itu bukan kelengkapan.
    //
    // Tanpa keterangan, `tab` tidak pernah terisi; tanpa `tab`, permintaan daftar tidak
    // pernah dijalankan; dan permintaan yang tidak pernah dijalankan berstatus "menunggu"
    // selamanya. Akibatnya tabel menampilkan kerangka pemuatan yang tidak pernah selesai,
    // TANPA satu pun pesan — dan pengguna hanya melihat layar yang menggantung.
    //
    // Ini ditemukan lewat uji yang sengaja menggagalkan `/keterangan`, bukan lewat
    // pembacaan kode.
    return (
      <PageFrame>
        <ErrorMessage
          title="Layar tidak dapat disiapkan"
          description={pesanGalat(keterangan.error)}
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  const identitas = daftar.data?.identitas ?? jumlahTab.data?.identitas ?? null

  return (
    <PageFrame identitas={identitas}>
      <div className="mt-6">
        <TabBar
          tabs={TAB_BAGIAN}
          active={bagian}
          onSelect={(kode) => setBagian(kode as Bagian)}
          label="Bagian layar My Work"
        />
      </div>

      {bagian === 'inbox' ? (
        <BagianInbox
          tabTersedia={keterangan.data?.tab ?? []}
          jumlah={jumlahTab.data?.tab ?? []}
          jumlahGagal={jumlahTab.isError}
          tabAktif={tab}
          onPindahTab={pindahTab}
          kolom={keterangan.data?.kolom ?? []}
          baris={daftar.data?.data ?? []}
          total={daftar.data?.total ?? 0}
          lewati={lewati}
          onPindahHalaman={(nomor) => setLewati((nomor - 1) * PAGE_SIZE)}
          cari={cari}
          onCari={ubahPencarian}
          sedangMemuat={daftar.isPending}
          sedangMengambil={daftar.isFetching}
          galat={daftar.isError ? pesanGalat(daftar.error) : null}
          onMuatUlang={() => {
            daftar.refetch()
            jumlahTab.refetch()
          }}
          bukaKlaim={bukaKlaim}
        />
      ) : (
        <BagianKPI
          kolom={keterangan.data?.kolom_kpi ?? []}
          jenisTersedia={keterangan.data?.jenis_kpi ?? []}
          jenis={jenisKPI}
          onPilihJenis={setJenisKPI}
          tahun={tahunKPI}
          onUbahTahun={setTahunKPI}
          baris={kpi.data?.data ?? []}
          sedangMemuat={kpi.isPending}
          sedangMengambil={kpi.isFetching}
          galat={kpi.isError ? pesanGalat(kpi.error) : null}
          onMuatUlang={() => { kpi.refetch() }}
        />
      )}

      <Catatan
        judul="Perbedaan yang disengaja terhadap layar lama"
        baris={keterangan.data?.selisih_terencana ?? []}
      />
      <Catatan judul="Yang perlu diketahui" baris={keterangan.data?.keterbatasan ?? []} />
    </PageFrame>
  )
}

/**
 * Bilah dua tab besar: INBOX dan KPI, sesuai harness.
 *
 * Memakai `TabBar` bersama, bukan tombol buatan sendiri. Bilah buatan sendiri di sini
 * sempat ada dan MIRIP tapi tidak sama — `rounded-t-md` alih-alih `rounded-t-kontrol`,
 * tanpa `ease-halus` — yaitu jarak yang cukup dekat untuk tampak seperti cacat dan cukup
 * jauh untuk terlihat berbeda dari layar sebelahnya di menu.
 */
const TAB_BAGIAN: TabItem[] = [
  { kode: 'inbox', nama: 'INBOX', keterangan: 'Antrean pekerjaan survei Anda.' },
  { kode: 'kpi', nama: 'KPI', keterangan: 'Ringkasan penilaian adjuster.' },
]

/** Bagian INBOX — bilah tujuh tab status, tabel, dan paginasinya. */
function BagianInbox({
  tabTersedia,
  jumlah,
  jumlahGagal,
  tabAktif,
  onPindahTab,
  kolom,
  baris,
  total,
  lewati,
  onPindahHalaman,
  cari,
  onCari,
  sedangMemuat,
  sedangMengambil,
  galat,
  onMuatUlang,
  bukaKlaim,
}: {
  tabTersedia: TabLayar[]
  jumlah: { kunci: string; total: number }[]
  /** Jumlah baris per tab gagal diambil — angkanya tidak dapat ditampilkan. */
  jumlahGagal: boolean
  tabAktif: string
  onPindahTab: (kunci: string) => void
  kolom: KolomLayar[]
  baris: TugasSurvei[]
  total: number
  lewati: number
  onPindahHalaman: (nomor: number) => void
  cari: string
  onCari: (nilai: string) => void
  sedangMemuat: boolean
  sedangMengambil: boolean
  galat: string | null
  onMuatUlang: () => void
  bukaKlaim: (nomorKlaim: string) => void
}) {
  const halaman = Math.floor(lewati / PAGE_SIZE) + 1
  const totalHalaman = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // Angka pada bilah tab menyatakan isi SELURUH tab, bukan isi hasil pencarian. Itu perilaku
  // Pega; menyamakan keduanya akan membuat angka tab berkedip setiap huruf yang diketik.
  const angka = new Map(jumlah.map((j) => [j.kunci, j.total]))

  // Angka dititipkan ke NAMA tabnya, bukan digambar sebagai lencana tersendiri.
  //
  // `TabBar` bersama tidak menyediakan slot lencana, dan menambahkannya berarti mengubah
  // komponen yang sudah dipakai layar lain. Tanda kurung terbaca sama jelasnya, dan
  // konsistensi bilah tab antarlayar lebih berharga daripada bentuk lencananya.
  //
  // Tab yang BELUM TERSEDIA memakai slot yang sama untuk menyatakan keadaannya, bukan angka.
  // Membiarkannya tampil polos akan membuatnya terbaca sebagai tab kosong — dan tab kosong
  // tidak pernah dilaporkan siapa pun sebagai kerusakan.
  const tabs: TabItem[] = tabTersedia.map((t) => ({
    kode: t.kunci,
    nama: !t.tersedia
      ? `${t.judul} (belum tersedia)`
      : angka.has(t.kunci)
        ? `${t.judul} (${angka.get(t.kunci)})`
        : t.judul,
    ...(t.alasan_tak_tersedia ?? t.keterangan
      ? { keterangan: t.alasan_tak_tersedia ?? t.keterangan }
      : {}),
  }))

  // Tab yang sedang dibuka tetapi belum dapat dihitung.
  //
  // Ia TETAP dapat dipilih, dan itu disengaja. Tab yang dimatikan tanpa penjelasan sama
  // membingungkannya dengan tab kosong; yang dibutuhkan pengguna adalah SEBABNYA, dan sebab
  // itu baru muat ditampilkan setelah tabnya dibuka.
  const tabDibuka = tabTersedia.find((t) => t.kunci === tabAktif)
  const tabBelumTersedia = tabDibuka && !tabDibuka.tersedia ? tabDibuka : null

  return (
    <>
      <div className="mt-4">
        <TabBar
          tabs={tabs}
          active={tabAktif}
          onSelect={onPindahTab}
          label="Tab status antrean survei"
        />
      </div>

      {jumlahGagal && (
        // Kegagalan penghitung tab HARUS terlihat, dan ini bukan kelengkapan.
        //
        // Tanpa pesan ini, tab tampil tanpa angka — dan pengguna tidak dapat membedakan
        // "tab ini memang kosong" dari "angkanya gagal diambil". Yang pertama tidak pernah
        // dilaporkan siapa pun sebagai kerusakan.
        //
        // Ia pemberitahuan di dalam layar, BUKAN pengganti seluruh layar: daftarnya sendiri
        // tetap dapat dibaca, dan menutup layar karena penghitungnya gagal akan mengambil
        // lebih banyak daripada yang hilang.
        <p className="mt-2 text-xs text-slate-600" role="status">
          Jumlah pekerjaan per tab tidak dapat diambil, sehingga angkanya tidak ditampilkan.
          Daftarnya sendiri tetap dapat dibuka. Tekan Muat ulang untuk mencoba lagi.
        </p>
      )}

      {tabBelumTersedia ? (
        // Tab yang belum dapat dihitung MENGGANTI tabelnya, bukan menampilkannya kosong.
        //
        // Daftar kosong terbaca sebagai "tidak ada pekerjaan untuk saya", dan pembacaan itu
        // salah: yang benar adalah kolom penggeraknya belum ada. Perbedaan keduanya menentukan
        // — yang pertama tidak pernah dilaporkan siapa pun, yang kedua akan.
        <div className="mt-4">
          <ErrorMessage
            title={`Tab ${tabBelumTersedia.judul} belum dapat ditampilkan`}
            description={
              (tabBelumTersedia.alasan_tak_tersedia ??
                'Kolom penggeraknya belum tersedia di basis data.') +
              ' Tab lain pada bilah di atas tetap berisi.'
            }
            tone="gangguan"
          />
        </div>
      ) : (
        <div className="mt-4">
          <DataTable<TugasSurvei>
            columns={buildColumns(kolom, bukaKlaim)}
            rows={baris}
            // Kunci baris memakai survei_id DAN index_survei sekaligus.
            //
            // Satu klaim dapat punya beberapa janji survei, dan masing-masing adalah barisnya
            // sendiri. Memakai klaim_id saja akan membuat React menemukan kunci ganda pada
            // baris yang memang seharusnya berbeda.
            rowKey={(row) => `${row.survei_id}|${row.index_survei}`}
            title="Antrean pekerjaan survei"
            label="Antrean My Work"
            {...(total > 0 ? { description: `${total} pekerjaan pada tab ini.` } : {})}
            isLoading={sedangMemuat}
            // HANYA Claim No. Layar lama juga mencari pada Reference No, dan kolom itu belum
            // tersedia — menyebutnya di sini akan menjanjikan pencarian yang tidak terjadi.
            searchLabel="Cari Claim No"
            emptyMessage="Tidak ada pekerjaan pada tab ini untuk Anda saat ini."
            serverSearch={{ value: cari, onChange: onCari, matchCount: total }}
            actions={
              <Button tone="halus" onClick={onMuatUlang} disabled={sedangMengambil}>
                <ReloadIcon className="h-4 w-4" />
                {sedangMengambil ? 'Memuat…' : 'Muat ulang'}
              </Button>
            }
            error={
              galat ? (
                <ErrorMessage
                  title="Antrean tidak dapat dimuat"
                  description={galat}
                  tone="gangguan"
                />
              ) : undefined
            }
            pagination={{
              page: halaman,
              size: PAGE_SIZE,
              total,
              totalPage: totalHalaman,
              onPageChange: onPindahHalaman,
              isLoading: sedangMengambil,
            }}
          />
        </div>
      )}
    </>
  )
}

/** Judul jenis ringkasan KPI, dipasangkan dengan nilai yang dikirim server. */
const JUDUL_JENIS_KPI: Record<string, string> = {
  outstanding: 'Outstanding KPI Adjuster',
  final: 'Data Final KPI Adjuster',
  kuartal: 'Data Final per Tahun',
}

/** Bagian KPI — ringkasan penilaian adjuster. */
function BagianKPI({
  kolom,
  jenisTersedia,
  jenis,
  onPilihJenis,
  tahun,
  onUbahTahun,
  baris,
  sedangMemuat,
  sedangMengambil,
  galat,
  onMuatUlang,
}: {
  kolom: KolomLayar[]
  jenisTersedia: string[]
  jenis: string
  onPilihJenis: (nilai: string) => void
  tahun: string
  onUbahTahun: (nilai: string) => void
  baris: BarisKPI[]
  sedangMemuat: boolean
  sedangMengambil: boolean
  galat: string | null
  onMuatUlang: () => void
}) {
  // Kolom pertama berganti arti menurut jenis ringkasannya: nama adjuster pada dua yang
  // pertama, TAHUN pada yang ketiga. Judulnya ikut berganti supaya kolomnya tidak terbaca
  // sebagai nama orang yang kebetulan berupa angka.
  const judulKelompok = jenis === 'kuartal' ? 'TAHUN' : 'ADJUSTER'

  const tabs: TabItem[] = jenisTersedia.map((j) => ({
    kode: j,
    nama: JUDUL_JENIS_KPI[j] ?? j,
  }))

  return (
    <>
      <div className="mt-4">
        <TabBar
          tabs={tabs}
          active={jenis}
          onSelect={onPilihJenis}
          label="Jenis ringkasan KPI adjuster"
        />
      </div>

      <div className="mt-4 w-32">
        {/*
          `Field`, bukan `<input>` buatan sendiri.

          Alasannya bukan kerapian. `src/styles.css:281` menempelkan permukaan gelap lewat
          selector `input[class~="bg-white"]`, dan isian tanpa kelas itu KELUAR dari tema —
          ia tampil transparan di tema gelap. `Field` membawanya, berikut `htmlFor` yang
          benar dan cincin fokus yang sama dengan seluruh isian lain di aplikasi ini.
        */}
        <Field
          id="kpi-tahun"
          label="Tahun"
          type="text"
          inputMode="numeric"
          value={tahun}
          onChange={(event) => onUbahTahun(event.target.value)}
          placeholder="semua"
          hint="Kosongkan untuk seluruh tahun."
        />
      </div>

      <div className="mt-4">
        <DataTable<BarisKPI>
          columns={buildKPIColumns(kolom, judulKelompok)}
          rows={baris}
          rowKey={(row) => row.kelompok}
          title={JUDUL_JENIS_KPI[jenis] ?? 'Ringkasan KPI Adjuster'}
          label="Ringkasan KPI adjuster"
          description="Rata-rata penilaian, dibulatkan dua desimal — sama dengan layar lama."
          isLoading={sedangMemuat}
          emptyMessage="Belum ada penilaian KPI untuk cakupan Anda."
          actions={
            <Button tone="halus" onClick={onMuatUlang} disabled={sedangMengambil}>
              <ReloadIcon className="h-4 w-4" />
              {sedangMengambil ? 'Memuat…' : 'Muat ulang'}
            </Button>
          }
          error={
            galat ? (
              <ErrorMessage
                title="Ringkasan KPI tidak dapat dimuat"
                description={galat}
                tone="gangguan"
              />
            ) : undefined
          }
        />
      </div>
    </>
  )
}

/**
 * Menyusun kolom tabel antrean dari judul yang dikirim server.
 *
 * Yang datang dari server hanyalah KUNCI dan JUDULNYA; cara menggambarnya tetap milik layar.
 * Kunci yang tidak dikenal dilewati, bukan digambar kosong: kolom baru menuntut keputusan
 * tampilan yang belum diambil.
 */
function buildColumns(
  kolom: KolomLayar[],
  bukaKlaim: (nomorKlaim: string) => void,
): Column<TugasSurvei>[] {
  const hasil: Column<TugasSurvei>[] = []

  for (const k of kolom) {
    const dibangun = renderer[k.kunci]?.(k, bukaKlaim)
    if (dibangun) hasil.push(dibangun)
  }
  return hasil
}

/** Menyusun kesepuluh kolom tabel KPI — satu kelompok dan sembilan angka. */
function buildKPIColumns(kolom: KolomLayar[], judulKelompok: string): Column<BarisKPI>[] {
  const hasil: Column<BarisKPI>[] = [
    {
      key: 'kelompok',
      title: judulKelompok,
      width: '14rem',
      value: (row) => row.kelompok,
      render: (row) => <Teks nilai={row.kelompok} />,
    },
  ]

  for (const k of kolom) {
    const ambil = angkaKPI[k.kunci]
    if (!ambil) continue

    hasil.push({
      key: k.kunci,
      title: k.judul,
      width: '10rem',
      value: (row) => String(ambil(row)),
      render: (row) => (
        <span className="tabular-nums text-slate-700">{ambil(row).toFixed(2)}</span>
      ),
    })
  }
  return hasil
}

/** Pengambil kesembilan angka KPI, dikunci dengan `kunci` yang dikirim server. */
const angkaKPI: Record<string, ((row: BarisKPI) => number) | undefined> = {
  penjadwalan_survey: (row) => row.penjadwalan_survey,
  immediate_advice: (row) => row.immediate_advice,
  preliminary_advice: (row) => row.preliminary_advice,
  interim_report: (row) => row.interim_report,
  update_progress: (row) => row.update_progress,
  tanggapan_komunikasi: (row) => row.tanggapan_komunikasi,
  propose_adjustment: (row) => row.propose_adjustment,
  final_report: (row) => row.final_report,
  nilai: (row) => row.nilai,
}

/** Teks sel yang kosong digambar sebagai em dash, bukan dibiarkan hampa. */
function Teks({ nilai }: { nilai: string }) {
  if (!nilai) return <span className="text-slate-400">—</span>
  return <span className="truncate">{nilai}</span>
}

/**
 * Judul kolom yang belum tersedia diberi penanda, bukan dibiarkan tampak biasa.
 *
 * Tanpa penanda, kolom yang SELALU kosong tidak dapat dibedakan dari kolom yang kebetulan
 * kosong pada halaman ini — dan yang kedua terbaca sebagai "datanya belum diisi petugas".
 */
function judulKolom(k: KolomLayar): string {
  return k.tersedia ? k.judul : `${k.judul} · belum tersedia`
}

/**
 * Sel kolom yang belum tersedia.
 *
 * Ia menyebut keadaannya, bukan menggambar em dash. Em dash pada kolom yang SELALU kosong
 * tidak dapat dibedakan dari em dash pada baris yang kebetulan kosong, dan sebab keduanya
 * berbeda jauh: yang satu menunggu Tim Pega, yang lain menunggu petugas mengisi.
 */
function SelBelumTersedia({ keterangan }: { keterangan: string | undefined }) {
  return (
    <span className="text-xs text-slate-400" title={keterangan ?? ''}>
      belum tersedia
    </span>
  )
}

type Renderer = (
  k: KolomLayar,
  bukaKlaim: (nomorKlaim: string) => void,
) => Column<TugasSurvei>

/**
 * Cara menggambar tiap kolom antrean, dikunci dengan `kunci` yang dikirim server.
 *
 * Judulnya TIDAK diketik di sini — ia diambil dari `k.judul`, supaya satu-satunya sumber
 * judul tetap backend.
 */
const renderer: Record<string, Renderer | undefined> = {
  // Appointment No, Reference No, dan Status ASM ber-`tersedia: false` hari ini.
  //
  // Ketiganya TETAP digambar — `D-13` menetapkan bentuk layar mengikuti Pega, dan menghapus
  // tiga dari tiga belas kolom akan membuat pengguna yang hafal layarnya mengira isinya
  // hilang. Yang berubah: judul dan selnya menyatakan sebabnya.
  appointment_no: (k) => ({
    key: k.kunci,
    title: judulKolom(k),
    width: '10rem',
    value: (row) => row.appointment_no,
    render: (row) =>
      row.appointment_no ? (
        <Teks nilai={row.appointment_no} />
      ) : (
        <SelBelumTersedia keterangan={k.keterangan} />
      ),
  }),

  reference_no: (k) => ({
    key: k.kunci,
    title: judulKolom(k),
    width: '10rem',
    value: (row) => row.reference_no,
    render: (row) =>
      row.reference_no ? (
        <Teks nilai={row.reference_no} />
      ) : (
        <SelBelumTersedia keterangan={k.keterangan} />
      ),
  }),

  claim_no: (k, bukaKlaim) => ({
    key: k.kunci,
    title: k.judul,
    width: '11rem',
    value: (row) => row.claim_no,
    render: (row) =>
      row.claim_no ? (
        <button
          type="button"
          className="rounded-md font-mono text-xs font-medium text-blue-700 underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
          onClick={() => bukaKlaim(row.claim_no)}
        >
          {row.claim_no}
        </button>
      ) : (
        <span className="text-slate-400">belum bernomor</span>
      ),
  }),

  policy_no: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '11rem',
    value: (row) => row.policy_no,
    render: (row) => <Teks nilai={row.policy_no} />,
  }),

  insured_name: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '14rem',
    value: (row) => row.insured_name,
    render: (row) => <Teks nilai={row.insured_name} />,
  }),

  cob: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '10rem',
    value: (row) => row.cob,
    render: (row) => <Teks nilai={row.cob} />,
  }),

  cause_of_loss: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '12rem',
    value: (row) => row.cause_of_loss,
    render: (row) =>
      row.cause_of_loss ? (
        <Teks nilai={row.cause_of_loss} />
      ) : (
        <span className="text-xs text-slate-400" title={k.keterangan ?? ''}>
          —
        </span>
      ),
  }),

  location: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '12rem',
    value: (row) => row.location,
    render: (row) => <Teks nilai={row.location} />,
  }),

  pic_asm: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '10rem',
    value: (row) => row.pic_asm,
    render: (row) => <Teks nilai={row.pic_asm} />,
  }),

  pic_loss_adjuster: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '12rem',
    value: (row) => row.pic_loss_adjuster,
    render: (row) => <Teks nilai={row.pic_loss_adjuster} />,
  }),

  date_of_loss: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '9rem',
    value: (row) => row.date_of_loss,
    render: (row) => <Teks nilai={row.date_of_loss} />,
  }),

  aging: (k) => ({
    key: k.kunci,
    title: k.judul,
    width: '8rem',
    value: (row) => (row.aging === null ? '' : String(row.aging)),
    render: (row) =>
      // `null` dan `0` digambar BERBEDA, dan itu inti kolomnya.
      //
      // `null` berarti tanggal janji surveinya tidak ada, sehingga umurnya tidak dapat
      // dihitung. Menggambar keduanya sama akan menampilkan "0 hari" pada baris yang
      // sebenarnya tidak punya angka — angka yang terlihat sah dan salah.
      row.aging === null ? (
        <span className="text-xs text-slate-400" title={k.keterangan ?? ''}>
          tidak dapat dihitung
        </span>
      ) : (
        <span className="tabular-nums text-slate-700">{row.aging} hari</span>
      ),
  }),

  status_asm: (k) => ({
    key: k.kunci,
    title: judulKolom(k),
    width: '11rem',
    value: (row) => row.status_asm,
    render: (row) =>
      row.status_asm ? (
        <Teks nilai={row.status_asm} />
      ) : (
        <SelBelumTersedia keterangan={k.keterangan} />
      ),
  }),
}

function PageFrame({
  children,
  identitas,
}: {
  children: ReactNode
  identitas?: IdentitasSurveyor | null
}) {
  return (
    <div className="mx-auto max-w-[110rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">My Work</h1>
        <p className="mt-1 text-sm text-slate-600">
          Pekerjaan survei yang menunggu Anda. Barisnya hilang begitu janji surveinya
          diselesaikan.
        </p>

        {identitas && (
          // Cakupan ditampilkan HANYA bila pemanggil seorang leader.
          //
          // Tanpa keterangan ini, seorang leader yang melihat baris atas nama orang lain
          // tidak punya cara menjelaskan kenapa — dan yang pertama kali terpikir adalah
          // "layarnya bocor".
          <p className="mt-2 text-xs text-slate-500">
            Ditampilkan sebagai <span className="font-medium">{identitas.nama}</span>
            {identitas.leader && identitas.jumlah_tim > 0 && (
              <>
                {' '}
                — termasuk pekerjaan {identitas.jumlah_tim} anggota tim Anda:{' '}
                {identitas.cakupan.filter((nama) => nama !== identitas.nama).join(', ')}
              </>
            )}
          </p>
        )}
      </header>
      {children}
    </div>
  )
}

/**
 * Catatan di bawah layar.
 *
 * Datang dari SERVER, bukan ditulis tetap di sini, supaya hilang dengan sendirinya begitu
 * penghalangnya hilang. Pada layar ini daftarnya panjang — empat kolom adjuster belum ada di
 * basis data, empat kueri tab layar lama hilang dari export, dan satu pemetaan kolom masih
 * menunggu DBA — sehingga menuliskannya di sini berarti menyunting layar setiap kali satu
 * penghalang selesai.
 */
function Catatan({ judul, baris }: { judul: string; baris: string[] }) {
  if (baris.length === 0) return null

  return (
    <section className="mt-6 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <h2 className="text-sm font-medium text-slate-800">{judul}</h2>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-xs text-slate-600">
        {baris.map((line) => (
          <li key={line}>{line}</li>
        ))}
      </ul>
    </section>
  )
}

/** pesanGalat mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function pesanGalat(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
