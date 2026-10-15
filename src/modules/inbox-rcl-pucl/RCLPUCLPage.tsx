import { useState, type ReactNode } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { APIError } from "@/api/client";
import { useSelectedPortal } from "@/app/portal";
import { Button } from "@/components/Button";
import { DataTable, type Column } from "@/components/DataTable";
import { ErrorMessage } from "@/components/ErrorMessage";
import { FormField } from "@/components/FormField";
import { formatPegaDateTime, formatPegaElapsed } from "@/components/format";

import { RCLPUCLTabs } from "./RCLPUCLTabs";
import {
  useExportRCLPUCL,
  useKirimReminder,
  useRCLPUCLList,
  useRCLPUCLMetadata,
} from "./api";
import type { HasilReminder, JawabanReminder } from "./api";
import type {
  DateRange,
  ReportColumn,
  Tab,
  TabColumn,
  WorkItem,
} from "./types";

/**
 * Inbox RCL/PUCL — menu `MENU_ID 61`, pengganti harness `RCLPUCL_Harness`.
 *
 * Isinya antrean klaim yang **ditolak (RCL)** atau **diproses ulang (PUCL)**, dipartisi
 * menurut perjalanan surat PUCL-nya:
 *
 *   Cetak Surat          surat BELUM dicetak — menunggu tindakan paling awal
 *   Kelengkapan Dokumen  surat sudah dicetak, menunggu kelengkapan dari tertanggung
 *   Klaim MSIG           sama, tetapi jalur MSIG
 *
 * Per `D-79` ia benar-benar Inbox: barisnya diambil dari tabel penugasan Pega dan hilang
 * begitu penugasannya selesai. Karena itu modul ini milik `U-3`, bukan `U-6`.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari ketiga section-nya: bilah tab, dua isian tanggal pada tab pertama, lalu
 * grid berhalaman 50 baris. Judul kolom TIDAK diterjemahkan — `D-13` menetapkan tampilan
 * meniru Pega, dan itulah teks yang selama ini dibaca pengguna.
 *
 * # Hal yang paling mudah disalahpahami di layar ini
 *
 * KETIGA tab punya kolom yang IDENTIK. Tidak ada satu pun petunjuk visual yang membedakan
 * tab satu dari yang lain selain judulnya — sehingga keterangan tab dan catatan per tab
 * bukan hiasan di sini, melainkan satu-satunya hal yang menjelaskan mengapa sebuah klaim
 * ada di tab ini dan tidak di tab sebelah.
 */
export function RCLPUCLPage() {
  // Tab dan nomor halaman hidup di ALAMAT, bukan di state komponen.
  //
  // Alasannya satu dan nyata: mengklik Nomor Case membuka layar kerja di rute lain, dan
  // komponen ini dilepas. State yang hanya ada di memori akan hilang, sehingga petugas yang
  // kembali dari sebuah klaim mendarat di tab pertama halaman pertama — padahal ia sedang
  // mengerjakan halaman ketiga tab kedua. Pada antrean yang dibuka berpuluh kali sehari,
  // itu bukan ketidaknyamanan kecil.
  //
  // Rentang tanggal TIDAK ikut ke alamat: ia hanya dipakai tombol ekspor, tidak menyaring
  // tabel, dan menaruhnya di alamat akan menyiratkan ia bagian dari apa yang sedang dilihat.
  const [params, setParams] = useSearchParams();
  const [range, setRange] = useState<DateRange>({ dari: "", sampai: "" });

  const navigate = useNavigate();

  const tabCode = params.get("tab") ?? "";
  const page = Math.max(1, Number(params.get("halaman") ?? "1") || 1);

  const portal = useSelectedPortal((state) => state.alias);
  const meta = useRCLPUCLMetadata();

  const tabs: Tab[] = meta.data?.tab ?? [];
  const active = tabCode || meta.data?.tab_bawaan || "";
  const tab = tabs.find((candidate) => candidate.kode === active);

  // Tab terhalang TIDAK meminta isinya. Permintaannya pasti ditolak server dengan 422, dan
  // galat yang sudah diketahui pasti terjadi bukan galat yang layak ditampilkan — yang
  // layak ditampilkan adalah alasan terhalangnya.
  const blocked = tab?.terhalang === true;
  const list = useRCLPUCLList(active, page, meta.isSuccess && !blocked);

  /**
   * Berpindah tab mengembalikan ke halaman pertama.
   *
   * Tanpa itu, berpindah dari halaman 4 sebuah tab ke tab yang hanya punya 2 halaman akan
   * menampilkan tabel kosong yang terbaca seperti antrean yang memang kosong — dan pada
   * layar yang ketiga tabnya tampak sama, itu sangat mudah disalahartikan.
   */
  function selectTab(code: string) {
    setParams({ tab: code, halaman: "1" }, { replace: true });
  }

  /** Berpindah halaman, menjaga tab yang sedang terbuka. */
  function setPage(next: number) {
    setParams({ tab: active, halaman: String(next) }, { replace: true });
  }

  /**
   * Membuka layar kerja klaim — yang di Pega dijalankan Open Assignment.
   *
   * Tab dan halaman ikut ke alamat tujuan supaya tombol kembali mendarat di tempat yang
   * sama. Kuncinya dikodekan: `pzInsKey` memuat SPASI (`ASM-FW-GCNMFW-WORK PNC-1234`), dan
   * spasi mentah di dalam alamat bukan alamat yang sah.
   */
  function openCase(row: WorkItem) {
    const back = new URLSearchParams({ tab: active, halaman: String(page) });
    navigate(
      `/inbox-rcl-pucl/klaim/${encodeURIComponent(row.referensi)}?${back}`,
    );
  }

  if (portal === null) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            "Antrean RCL/PUCL milik satu badan hukum, dan aplikasi ini melayani empat. " +
            "Pilih portal di bilah atas untuk membukanya."
          }
          tone="gangguan"
        />
      </PageFrame>
    );
  }

  if (meta.isError) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Layar tidak dapat dibuka"
          description={messageOf(meta.error)}
          tone="gangguan"
        />
      </PageFrame>
    );
  }

  return (
    <PageFrame>
      <div className="mt-4">
        <RCLPUCLTabs tabs={tabs} active={active} onSelect={selectTab} />
      </div>

      {tab && (
        <>
          <p className="mt-3 text-sm text-slate-600">{tab.keterangan}</p>

          {tab.catatan && <TabNotice text={tab.catatan} />}

          {/*
            Bilah tindakan tetap digambar pada tab TERHALANG, hanya tombolnya mati. Tab yang
            belum dapat diisi pun belum dapat diekspor, dan tombol yang hilang sama sekali
            terbaca seperti layar yang berbeda — bukan seperti layar yang sama dengan satu
            tindakan yang sedang tidak tersedia.
          */}
          <ActionBar
            tab={tab}
            range={range}
            onRangeChange={setRange}
            // Laporan harian dapat diunduh meski tabelnya kosong: isinya memang bukan isi
            // tabel, dan rentang tanggalnya dapat memuat baris yang tidak satu pun tab
            // tampilkan.
            exportable={
              !blocked &&
              (tab.punya_laporan_rentang_tanggal ||
                (list.data?.paginasi.total ?? 0) > 0)
            }
            reportColumns={meta.data?.kolom_laporan ?? []}
          />

          {tab.terhalang ? (
            <BlockedNotice tab={tab} />
          ) : (
            <div className="mt-4">
              <DataTable<WorkItem>
                columns={columnsFor(tab, list.data?.baris ?? [], (row) => (
                  <CaseLink item={row} onOpen={openCase} />
                ))}
                rows={list.data?.baris ?? []}
                rowKey={rowKeyOf}
                label={`Antrean ${tab.nama}`}
                // Garis antarkolom, supaya gridnya terbaca BERKOTAK seperti grid Pega —
                // bukan berbaris. Judul kartu sengaja TIDAK diisi: grid lama duduk di dalam
                // kontainer `pyHeaderType = COLLAPSIBLE` yang tidak memuat caption apa pun,
                // dan nama tabnya sudah terbaca tepat di atasnya.
                gridLines
                // Kotak cari bawaan disembunyikan: hasilnya akan menyaring HANYA halaman
                // yang sedang terbuka, sehingga pengguna dapat diberi tahu "tidak ada"
                // untuk baris yang sebenarnya ada di halaman berikutnya.
                //
                // KOREKSI 2026-10-12. Di sini sebelumnya tertulis "layar lama pun tidak
                // punya kotak cari: ketiga Report Definition-nya tidak menyaring menurut
                // kata kunci sama sekali". Pernyataan itu benar tentang Report Definition
                // dan MENYIMPULKAN HAL YANG SALAH tentang layarnya.
                //
                // Penyaringan di grid Pega bukan pekerjaan Report Definition melainkan
                // setelan GRID-nya, dan ketiga section inbox menyalakannya:
                // `pyGridFiltering = true` pada ketiganya. Itulah corong penyaring di
                // setiap judul kolom yang terlihat di layar Pega yang berjalan.
                //
                // Jadi yang hilang di sini BUKAN kotak cari — layar lama memang tidak
                // punya — melainkan penyaring per kolom. Alasan pertama di atas tetap
                // berlaku dan tidak menggantikannya. Lihat PlannedDifferences.
                hideSearch
                isLoading={list.isPending}
                error={
                  list.isError ? (
                    <ErrorMessage
                      title="Antrean tidak dapat dimuat"
                      description={messageOf(list.error)}
                      tone="gangguan"
                    />
                  ) : undefined
                }
                emptyMessage={emptyMessageFor(tab)}
                pagination={{
                  page: list.data?.paginasi.halaman ?? 1,
                  size: list.data?.paginasi.ukuran ?? 50,
                  total: list.data?.paginasi.total ?? 0,
                  totalPage: list.data?.paginasi.total_halaman ?? 1,
                  onPageChange: setPage,
                  isLoading: list.isFetching,
                }}
              />
            </div>
          )}
        </>
      )}
    </PageFrame>
  );
}

function PageFrame({ children }: { children: ReactNode }) {
  return (
    <div className="halaman px-4 py-8">
      {/*
        Kepala layar dijaga SEPENDEK layar lama: judul, satu kalimat, lalu bilah tab.

        Sampai 2026-10-10 di sini berdiri pula tombol ekspor dan satu paragraf tentang sifat
        antrean bersama, sehingga grid-nya terdorong jauh ke bawah dan tata letaknya tidak
        lagi terbaca seperti layar yang ditirunya (`D-13`). Keduanya TIDAK dibuang: tombolnya
        pindah ke bilah tindakan tepat di atas grid — persis letaknya di Pega, yang
        menggambarnya sesudah kedua isian tanggal — dan keterangan antrean bersama pindah ke
        kaki layar bersama keterangan lain.
      */}
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Inbox RCL/PUCL</h1>
        <p className="mt-1 text-sm text-slate-600">
          Klaim yang ditolak (RCL) atau diproses ulang (PUCL), dikelompokkan
          menurut perjalanan surat PUCL-nya.
        </p>
      </header>

      {children}

      <SharedQueueNotice />
      <LetterActionNotice />
    </div>
  );
}

/**
 * Bilah tindakan di atas grid: kedua isian tanggal, lalu tombol ekspor.
 *
 * # Kenapa ketiganya sebaris, dan kenapa tombolnya di sini
 *
 * Karena begitulah urutannya di layar lama. Ketiga section RCL/PUCL menggambar
 * `FROM RCL/PUCL`, lalu `TO RCL/PUCL`, lalu tombol ber-`pyLabel = "Export to Excel"` —
 * berurutan di dalam satu tata letak yang sama, di atas grid
 * (`Section/InboxCetakSuratPUCLRCL_Section-Section.xml`). Sebelumnya tombol itu berdiri di
 * pojok kanan atas layar, terpisah dari isian yang mengisinya, dan itu letak yang dikarang.
 *
 * # Kenapa ia tetap ada di tab yang TIDAK punya rentang tanggal
 *
 * Karena tombolnya ada di ketiga section — terverifikasi pada ketiganya. Yang hanya dimiliki
 * tab "Cetak Surat" adalah kedua isian tanggalnya, bukan tombolnya.
 *
 * # Kenapa peringatan tanggalnya sekeras itu
 *
 * Karena isian ini TIDAK menyaring tabel di bawahnya — hanya berkas ekspornya. Itu perilaku
 * layar lama: grid-nya dipasok Report Definition yang tidak menyaring tanggal sama sekali,
 * sementara tombol ekspornya menjalankan kueri yang berbeda. Work Owner memutuskan
 * mereplikasinya (`P-5`).
 *
 * Tanpa peringatan, pengguna akan mengisi tanggal, melihat tabel tidak berubah, lalu
 * melaporkannya sebagai kerusakan.
 */
function ActionBar({
  tab,
  range,
  onRangeChange,
  exportable,
  reportColumns,
}: {
  tab: Tab;
  range: DateRange;
  onRangeChange: (next: DateRange) => void;
  /** Tombol ekspor hanya berguna bila ada yang dapat diekspor. */
  exportable: boolean;
  reportColumns: ReportColumn[];
}) {
  const berentang = tab.punya_laporan_rentang_tanggal;

  return (
    <section className="mt-4 rounded-kartu border border-slate-200 bg-white px-4 py-3 shadow-lembut">
      {/*
        `items-end` meratakan tombol dengan DASAR kotak isian, bukan dengan judulnya — tanpa
        itu tombolnya melayang sejajar tulisan "FROM RCL/PUCL".
      */}
      <div className="flex flex-wrap items-end gap-4">
        {berentang && (
          <>
            <FormField
              id="rcl-pucl-dari"
              label="FROM RCL/PUCL"
              type="date"
              value={range.dari}
              onChange={(event) =>
                onRangeChange({ ...range, dari: event.target.value })
              }
            />
            <FormField
              id="rcl-pucl-sampai"
              label="TO RCL/PUCL"
              type="date"
              value={range.sampai}
              onChange={(event) =>
                onRangeChange({ ...range, sampai: event.target.value })
              }
            />
          </>
        )}

        <ExportButton
          tab={tab}
          range={range}
          enabled={exportable}
          columns={reportColumns}
        />

        {/*
          Hanya tab "Kelengkapan Dokumen" yang punya tombol ini, dan yang menentukannya
          SERVER — bukan perbandingan kode tab di sini (`D-15`).

          Letaknya di sebelah "Export to Excel", persis seperti di Pega: keduanya
          berdampingan di bilah tindakan tepat di atas grid.
        */}
        {tab.punya_reminder && <ReminderButton tab={tab} />}
      </div>

      {berentang && (
        <p className="mt-3 text-xs text-slate-600">
          <span className="font-medium">
            Kedua tanggal ini hanya dipakai tombol unduh.
          </span>{" "}
          Tabel di bawah tidak ikut tersaring — sama seperti di layar lama.
          Isinya pun berbeda: berkas laporan memuat klaim yang suratnya sudah
          dicetak, klaim yang sudah selesai, dan klaim Personal Accident di luar
          antrean ini.
        </p>
      )}
    </section>
  );
}

/**
 * Keterangan bahwa layar ini antrean BERSAMA, bukan pekerjaan pemanggil.
 *
 * Tidak satu pun tab di sini menyaring menurut pengguna yang login — penyaringnya akun
 * antrean, bukan orang. Tanpa keterangan ini, petugas yang terbiasa dengan layar inbox lain
 * akan mengira daftarnya keliru karena memuat pekerjaan orang lain.
 *
 * Letaknya di KAKI layar, bukan di kepalanya: ia keterangan yang dibaca sekali lalu
 * diketahui seterusnya, sedangkan kepala layar harus terbaca seperti layar yang ditirunya.
 */
function SharedQueueNotice() {
  return (
    <p className="mt-6 text-xs text-slate-500">
      Ini <span className="font-medium">antrean bersama</span>: seluruh petugas
      RCL/PUCL pada portal yang sedang dipilih melihat daftar yang sama, bukan
      hanya pekerjaan Anda. Setiap pembukaannya tercatat.
    </p>
  );
}

/**
 * Catatan yang berlaku pada satu tab saja.
 *
 * Isinya datang dari SERVER, bukan ditulis tetap di sini, supaya ia hilang dengan
 * sendirinya begitu keadaannya berubah — khususnya catatan tab "Klaim MSIG", yang berlaku
 * hanya sampai DBA memastikan kolom penandanya.
 *
 * Itu bukan kelebihan teoretis. Tab "Cetak Surat" sempat membawa catatan yang menjelaskan
 * kolom "Tanggal Cetak Surat" yang selalu kosong; ketika kolomnya dibuang pada 2026-10-11,
 * catatannya ikut hilang tanpa satu baris pun disunting di layar ini.
 */
function TabNotice({ text }: { text: string }) {
  return (
    <div className="mt-3 rounded-kartu border border-sky-200 bg-sky-50 px-4 py-3">
      <p className="text-xs text-slate-700">{text}</p>
    </div>
  );
}

/**
 * Tombol ekspor.
 *
 * # Namanya "Export to Excel" di ketiga tab, persis seperti di Pega
 *
 * Terverifikasi pada ketiga section sebagai `pyLabel = "Export to Excel"` — dengan "to"
 * huruf kecil — ber-`pyStyleName = "Strong"`, yakni tombol tindakan utama.
 *
 * Sampai 2026-10-10 tab "Cetak Surat" menamainya "Unduh Laporan Harian". Maksudnya baik —
 * tab itu memang mengunduh LAPORAN HARIAN berbasis rentang tanggal, sedangkan dua tab lain
 * mengunduh salinan tabelnya — tetapi nama yang tidak pernah ada di layar lama melanggar
 * `D-13` dan membuat petugas mencari tombol yang ia kenal. Keterangan isi berkas di bawah
 * tombol tetap dipertahankan, dan itulah yang sebenarnya menjawab "apa yang saya unduh".
 *
 * # Nadanya `utama`, bukan oranye
 *
 * `pyStyleName = "Strong"` di Pega berwarna oranye. Nada tombol di aplikasi ini ditetapkan
 * baseline-nya — `utama`, `kedua`, `halus` — dan tidak ada nada oranye. Menambahkannya demi
 * satu layar akan memecah sistem warna yang dipakai 74 layar lain; yang ditiru karena itu
 * DERAJATNYA, bukan warnanya.
 */
function ExportButton({
  tab,
  range,
  enabled,
  columns,
}: {
  tab: Tab | undefined;
  range: DateRange;
  enabled: boolean;
  columns: ReportColumn[];
}) {
  const ekspor = useExportRCLPUCL();
  const isReport = tab?.punya_laporan_rentang_tanggal === true;

  return (
    <div className="flex max-w-md flex-col items-start gap-1">
      <Button
        tone="utama"
        disabled={!enabled || ekspor.isPending || tab === undefined}
        onClick={() =>
          ekspor.mutate({
            tab: tab?.kode ?? "",
            range: isReport ? range : undefined,
          })
        }
      >
        {ekspor.isPending ? "Menyiapkan berkas…" : "Export to Excel"}
      </Button>

      {isReport && columns.length > 0 && (
        <p className="text-xs text-slate-500">
          Berisi: {columns.map((column) => column.judul).join(" · ")}
        </p>
      )}

      {ekspor.isError && (
        <p className="text-xs text-red-700" role="alert">
          {messageOf(ekspor.error)}
        </p>
      )}
    </div>
  );
}
/**
 * Tombol "Reminder PUCL" beserta modalnya.
 *
 * # Ia ADA di Pega, dan modul ini sempat menyatakan sebaliknya
 *
 * Catatan lama menyatakan "Reminder PUCL tidak pernah digambar, bahkan di Pega", dengan
 * dasar syarat `1==2` pada tombol bernama sama. Itu benar untuk **layar kerja**
 * (`SendtoRCLPUCL`) dan SALAH untuk **inbox**: keduanya tombol yang berbeda. Yang di inbox
 * tergambar nyata di Pega yang berjalan — dilaporkan Work Owner 2026-10-12 beserta
 * tangkapan layarnya, dan terverifikasi ulang di
 * `Section/InboxKelengkapanDocPUCLRCL_Section-Section.xml:1156`.
 *
 * # Isinya baris tab ini juga, bukan kumpulan lain
 *
 * Penyaring `RDB List/ReminderPUCL-SQL.xml` IDENTIK dengan penyaring tab "Kelengkapan
 * Dokumen" — kelima syaratnya sama persis. Karena itu modalnya memakai daftar yang sama,
 * bukan permintaan baru yang kebetulan mirip: dua kueri yang seharusnya sama tetapi ditulis
 * terpisah adalah dua kueri yang dapat bergeser diam-diam.
 *
 * # Kenapa tombol kirimnya MATI, dan kenapa tetap digambar
 *
 * Di Pega, modal ini mengirim email ke `.ClaimData.EmailLOD` lewat `ASMSendsEmailAttachments`
 * memakai akun `"Admin-PNC"`. Kolom sumbernya `POOLDATA.T_CLAIM_PNC.EMAIL_LOD` — dan ia
 * **kosong pada seluruh 2.206 baris**, dihitung langsung 2026-10-01.
 *
 * Jadi tombol yang dapat ditekan di sini akan mengirim ke alamat kosong: gagal diam-diam,
 * atau lebih buruk, terkirim ke alamat yang salah. Ia digambar supaya tindakannya terlihat
 * ada, dan alasannya ditulis di sebelahnya — pola yang sama sudah dipakai layar kerja modul
 * ini untuk tindakan yang belum dapat dijalankan.
 */
function ReminderButton({ tab }: { tab: Tab }) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button tone="kedua" onClick={() => setOpen(true)}>
        Reminder PUCL
      </Button>

      {open && <ReminderDialog tab={tab} onClose={() => setOpen(false)} />}
    </>
  );
}

/**
 * Modal "Reminder PUCL" — daftar klaim yang dapat dipilih.
 *
 * Bentuknya mengikuti Pega: daftar berkotak centang per baris ditambah "Pilih semua" di
 * atasnya, lalu tombol kirim.
 *
 * Paginasinya dipertahankan, dan itu disengaja: daftar ini dapat memuat ribuan baris, dan
 * memuat seluruhnya sekaligus adalah persis beban yang `D-10` memperingatkannya. Pilihan
 * disimpan menurut NOMOR CASE, bukan nomor baris, supaya ia selamat melewati pergantian
 * halaman.
 */
function ReminderDialog({ tab, onClose }: { tab: Tab; onClose: () => void }) {
  const [page, setPage] = useState(1);
  const [chosen, setChosen] = useState<Set<string>>(new Set());

  const kirim = useKirimReminder();

  const list = useRCLPUCLList(tab.kode, page, true);
  const rows = list.data?.baris ?? [];

  // Pilihan disimpan menurut NOMOR KLAIM, bukan kunci baris.
  //
  // Kunci baris adalah `referensi|no_case` — bentuk yang hanya berarti bagi DataTable.
  // Menyimpannya di sini dan mengirimkannya sebagai nomor klaim adalah cacat yang
  // benar-benar terjadi 2026-10-12: peladen menerima "PNC-2067|PNC-2067", tidak
  // menemukannya, lalu melaporkan "klaim tidak lagi di antrean ini" — kalimat yang
  // menuduh DATA padahal yang salah adalah kuncinya.
  //
  // Yang disimpan kini SAMA PERSIS dengan yang dikirim. Itu bukan penghematan satu
  // pemetaan melainkan penghapusan seluruh kelas cacat ini: tidak ada lagi dua bentuk
  // identitas yang dapat tertukar.
  //
  // Baris tanpa nomor klaim TIDAK dapat dipilih — lihat reminderColumns. Pengingat
  // ditujukan ke sebuah klaim, dan baris tanpa nomor tidak menunjuk klaim mana pun.
  const onPage = rows
    .map((row) => row.no_case)
    .filter((number) => number !== "");
  const allOnPage =
    onPage.length > 0 && onPage.every((number) => chosen.has(number));

  function toggle(key: string) {
    setChosen((lama) => {
      const next = new Set(lama);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }

  /** "Pilih semua" berlaku untuk HALAMAN INI — itulah yang terlihat pengguna. */
  function toggleAll() {
    setChosen((lama) => {
      const next = new Set(lama);
      for (const key of onPage) {
        if (allOnPage) next.delete(key);
        else next.add(key);
      }
      return next;
    });
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-900/40 p-4 sm:p-8"
      role="dialog"
      aria-modal="true"
      aria-label="Reminder PUCL"
    >
      <div className="w-full max-w-5xl rounded-kartu border border-slate-200 bg-white shadow-lembut">
        <div className="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4">
          <div>
            <h2 className="text-base font-semibold text-slate-900">
              Reminder PUCL
            </h2>
            <p className="mt-1 text-sm text-slate-600">
              Pilih klaim yang akan dikirimi pengingat kelengkapan dokumen.
              Daftarnya sama dengan antrean &ldquo;Kelengkapan Dokumen&rdquo; —
              penyaringnya memang sama persis.
            </p>
          </div>
          <Button tone="halus" onClick={onClose}>
            Tutup
          </Button>
        </div>

        <div className="px-5 py-4">
          <label className="inline-flex items-center gap-2 text-sm font-medium text-slate-800">
            <input
              type="checkbox"
              checked={allOnPage}
              onChange={toggleAll}
              disabled={rows.length === 0}
              className="h-4 w-4 rounded border-slate-300"
            />
            Pilih semua
          </label>

          <div className="mt-3">
            <DataTable<WorkItem>
              columns={reminderColumns(rows, chosen, toggle)}
              rows={rows}
              rowKey={rowKeyOf}
              label="Klaim untuk Reminder PUCL"
              gridLines
              hideSearch
              isLoading={list.isPending}
              error={
                list.isError ? (
                  <ErrorMessage
                    title="Daftar tidak dapat dimuat"
                    description={messageOf(list.error)}
                    tone="gangguan"
                  />
                ) : undefined
              }
              emptyMessage="Tidak ada klaim yang menunggu kelengkapan dokumen."
              pagination={{
                page: list.data?.paginasi.halaman ?? 1,
                size: list.data?.paginasi.ukuran ?? 50,
                total: list.data?.paginasi.total ?? 0,
                totalPage: list.data?.paginasi.total_halaman ?? 1,
                onPageChange: setPage,
                isLoading: list.isFetching,
              }}
            />
          </div>
        </div>

        {kirim.data && <ReminderResult hasil={kirim.data} />}

        {kirim.isError && (
          <div className="px-5 pb-2">
            <ErrorMessage
              title="Pengingat tidak dapat dikirim"
              description={messageOf(kirim.error)}
              tone="gangguan"
            />
          </div>
        )}

        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-slate-200 px-5 py-4">
          <p className="text-xs text-slate-600">
            {chosen.size === 0
              ? "Belum ada klaim yang dipilih."
              : `${chosen.size} klaim dipilih.`}
          </p>

          {/*
            Tombolnya HIDUP, dan keadaannya ditentukan DATA — bukan kalimat yang ditulis
            tetap di layar.

            Sampai 2026-10-12 di sini berdiri tombol mati beserta kalimat "Email
            Tertanggung kosong pada seluruh baris, dihitung 2026-10-01". Kalimat itu benar
            saat ditulis dan menjadi salah begitu kolomnya diisi — persis kegagalan yang
            sama dengan "Kesembilannya" di kaki layar kerja. Keadaan basis data TIDAK boleh
            dibekukan menjadi kalimat.

            Penggantinya: tombolnya hidup selama ada yang dicentang, dan PELADEN yang
            melaporkan kenyataan per klaim — terkirim, belum punya alamat, tidak memenuhi
            syarat, atau gagal.
          */}
          <Button
            tone="utama"
            disabled={chosen.size === 0 || kirim.isPending}
            onClick={() => kirim.mutate([...chosen])}
          >
            {kirim.isPending ? "Mengirim…" : "Kirim Reminder"}
          </Button>
        </div>
      </div>
    </div>
  );
}

/**
 * Hasil pengiriman Reminder PUCL, dilaporkan PER KLAIM.
 *
 * # Kenapa per klaim, bukan satu kalimat
 *
 * Karena pengiriman sebagian adalah hasil yang LAZIM di sini: sebagian klaim punya alamat
 * surel tertanggung, sebagian belum. Satu kalimat "berhasil" menyembunyikan klaim yang
 * justru perlu ditindak, dan satu kalimat "gagal" menyembunyikan yang sudah terkirim —
 * lalu petugas mengirim ulang ke orang yang sudah menerimanya.
 *
 * Cacahnya datang dari PELADEN, tidak dihitung ulang di sini. Dua tempat yang menghitung
 * hal yang sama dapat berselisih, dan selisihnya muncul sebagai "3 terkirim" di atas
 * daftar yang memuat empat baris terkirim.
 */
function ReminderResult({ hasil }: { hasil: JawabanReminder }) {
  const { ringkasan } = hasil;

  const gagal = hasil.hasil.filter((baris) => baris.status !== "terkirim");

  return (
    <div className="border-t border-slate-200 px-5 py-4">
      <p className="text-sm font-medium text-slate-900">
        {ringkasan.terkirim > 0
          ? `${ringkasan.terkirim} pengingat terkirim.`
          : "Tidak ada pengingat yang terkirim."}
      </p>

      {gagal.length > 0 && (
        <>
          <p className="mt-1 text-xs text-slate-600">
            {/*
              Ketiga sebabnya disebut TERPISAH karena tindakannya berbeda: yang tanpa
              alamat menunggu Email Tertanggung diisi, yang tidak memenuhi syarat memang
              bukan urusan antrean ini, dan yang gagal dapat dicoba lagi.
            */}
            Sisanya tidak terkirim
            {ringkasan.tanpa_email > 0 &&
              ` · ${ringkasan.tanpa_email} belum punya Email Tertanggung`}
            {ringkasan.tidak_memenuhi_syarat > 0 &&
              ` · ${ringkasan.tidak_memenuhi_syarat} tidak lagi di antrean ini`}
            {ringkasan.gagal > 0 &&
              ` · ${ringkasan.gagal} ditolak peladen surel`}
            .
          </p>

          <ul className="mt-2 space-y-1">
            {gagal.map((baris) => (
              <li key={baris.nomor_klaim} className="text-xs text-slate-700">
                <span className="font-mono">{baris.nomor_klaim}</span>
                {" — "}
                {alasanReminder(baris)}
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  );
}

/**
 * alasanReminder menyusun kalimat satu baris hasil.
 *
 * Kalimatnya disusun di LAYAR dari kode status, bukan dikirim peladen sebagai kalimat jadi
 * — kecuali `gagal`, yang alasannya memang hanya diketahui peladen. Kode status adalah
 * kontrak yang stabil; kalimat yang dikirim peladen akan membuat perubahan susunan kata
 * menuntut penempelan ulang binary.
 */
function alasanReminder(baris: HasilReminder): string {
  switch (baris.status) {
    case "tanpa-email":
      return "Email Tertanggung belum diisi pada klaim ini.";
    case "tidak-memenuhi-syarat":
      return "Klaim ini tidak lagi berada di antrean Kelengkapan Dokumen.";
    case "gagal":
      return baris.alasan ?? "Pengiriman ditolak peladen surel.";
    default:
      return baris.status;
  }
}

/**
 * Kolom modal Reminder — kotak centang, lalu kolom yang dipakai memilih.
 *
 * Sengaja LEBIH SEDIKIT daripada grid tab-nya: yang dibutuhkan untuk memutuskan "klaim ini
 * diingatkan atau tidak" hanya identitas klaim, siapa tertanggungnya, dan sejak kapan
 * suratnya dicetak. Menyalin kesepuluh kolom grid ke dalam modal akan membuat modalnya
 * selebar layar dan kotak centangnya terdorong keluar pandangan.
 */
function reminderColumns(
  rows: WorkItem[],
  chosen: Set<string>,
  toggle: (key: string) => void,
): Column<WorkItem>[] {
  const nomor = new Map(rows.map((row, index) => [rowKeyOf(row), index + 1]));

  return [
    {
      key: "pilih",
      title: "",
      width: "3rem",
      noSort: true,
      // Dicentang menurut NOMOR KLAIM — nilai yang sama persis dengan yang dikirim ke
      // peladen. Lihat catatan pada `chosen` di ReminderDialog.
      value: (row) => (chosen.has(row.no_case) ? "dipilih" : ""),
      render: (row) => (
        <input
          type="checkbox"
          checked={chosen.has(row.no_case)}
          onChange={() => toggle(row.no_case)}
          // Baris tanpa nomor klaim tidak dapat dipilih: pengingat ditujukan ke sebuah
          // klaim, dan baris tanpa nomor tidak menunjuk klaim mana pun. Mematikannya
          // lebih jujur daripada membiarkannya dicentang lalu ditolak peladen.
          disabled={row.no_case === ""}
          aria-label={`Pilih ${row.no_case}`}
          className="h-4 w-4 rounded border-slate-300 disabled:cursor-not-allowed"
        />
      ),
    },
    {
      key: "nomor",
      title: "",
      width: "3rem",
      noSort: true,
      value: (row) => String(nomor.get(rowKeyOf(row)) ?? ""),
    },
    { key: "no_case", title: "Nomor Case", value: (row) => row.no_case },
    { key: "no_polis", title: "No Polis", value: (row) => row.no_polis },
    {
      key: "nama_tertanggung",
      title: "Nama Tertanggung",
      value: (row) => row.nama_tertanggung,
    },
    {
      key: "tanggal_cetak_surat",
      title: "Tanggal Cetak Surat",
      value: (row) => formatPegaDateTime(row.tanggal_cetak_surat) || "—",
    },
  ];
}

/**
 * Keterangan tab yang digambar tetapi belum dapat diisi.
 *
 * Alasan dan pemiliknya datang dari SERVER, bukan ditulis tetap di sini, supaya keduanya
 * hilang dengan sendirinya begitu penghalangnya hilang. Menyebut pemiliknya penting:
 * penghalang tanpa alamat tidak pernah hilang.
 */
function BlockedNotice({ tab }: { tab: Tab }) {
  return (
    <div className="mt-4 rounded-kartu border border-amber-200 bg-amber-50 px-4 py-4">
      <h2 className="text-sm font-semibold text-amber-900">
        {tab.nama} belum tersedia
      </h2>
      <p className="mt-2 text-sm text-slate-700">{tab.alasan_terhalang}</p>
      {tab.pemilik_penghalang && (
        <p className="mt-2 text-xs text-slate-600">
          <span className="font-medium">Menunggu:</span>{" "}
          {tab.pemilik_penghalang}
        </p>
      )}
    </div>
  );
}

/**
 * Keterangan DI MANA surat RCL/PUCL dicetak.
 *
 * # Kenapa keterangannya tetap ada, padahal tidak lagi menyebut Pega
 *
 * Karena pertanyaan penggunanya tidak berubah: "mencetak suratnya di mana?". Yang berubah
 * jawabannya. Sampai 2026-10-06 di sini berdiri kotak amber berjudul "Yang dikerjakan lewat
 * Pega" yang menyatakan Cetak Surat dan Reminder PUCL hanya dapat dikerjakan di Pega. Dua
 * hal membuatnya TIDAK LAGI BENAR, dan keduanya terbukti dari kode:
 *
 *   - **Cetak Surat sudah dikerjakan sistem ini.** Tombol "Download Dokumen" di layar kerja
 *     klaim menerbitkan PDF suratnya, melampirkannya, lalu menandai `TGL_CETAK_DOKUMEN_PUCL`
 *     — penanda yang memindahkan klaimnya ke tab "Kelengkapan Dokumen". Seluruhnya menulis
 *     ke `POOLDATA.TC_PNC_PUCL`, tabel milik aplikasi ini; tidak satu pun menempuh Pega.
 *   - **Reminder PUCL tidak pernah ada di Pega.** Syarat tampil tombolnya di layar lama
 *     ditulis `1==2` — syarat yang tidak pernah benar. Menyebutnya "dikerjakan lewat Pega"
 *     menyuruh petugas mencari tombol yang tidak pernah digambar sistem mana pun.
 *
 * # Kenapa nadanya netral, bukan amber
 *
 * Amber di aplikasi ini berarti terhalang. Tidak ada yang terhalang di layar ini: tindakannya
 * ada, hanya letaknya bukan di antrean melainkan di dalam klaimnya — persis seperti di Pega,
 * yang juga menuntut baris dibuka lebih dulu. Mempertahankan kotak amber akan membuat layar
 * yang berfungsi penuh terbaca seperti layar yang separuh jadi.
 */
function LetterActionNotice() {
  return (
    <p className="mt-2 text-xs text-slate-500">
      <span className="font-medium">Mencetak surat RCL/PUCL</span> dikerjakan
      dari dalam klaimnya: buka Nomor Case, lalu tekan &ldquo;Download
      Dokumen&rdquo;. Suratnya terbit, melampir ke klaim, dan klaimnya berpindah
      sendiri dari tab &ldquo;Cetak Surat&rdquo; ke &ldquo;Kelengkapan
      Dokumen&rdquo;.
    </p>
  );
}

/**
 * Tautan pada kolom "Nomor Case".
 *
 * # Kenapa TAUTAN pada nomor case, bukan tombol di ujung baris
 *
 * Karena begitulah layar lama. Ketiga section RCL/PUCL menggambar sel "Nomor Case" sebagai
 * `pyUIElement = link` ber-`pyLabel = .pyID`, dan `D-13` menetapkan tampilan mengikuti Pega.
 *
 * Versi pertama modul ini keliru di sini: ia menambahkan kolom tombol "Lihat Detail" yang
 * **tidak ada di Pega sama sekali**. Itu bukan sekadar beda tampilan — kolom yang tidak
 * pernah ada membuat pengguna mengira ada dua cara berbeda membuka baris, dan menggeser
 * lebar seluruh kolom lain.
 *
 * # Apa yang sebenarnya terjadi saat diklik di Pega
 *
 * `pyEvent = click` menjalankan `pyAction = runActivity` atas
 * `SetAssignmentInboxPUCL_act`, dengan satu parameter: `inskey = .pzInsKey`.
 *
 * Activity itu sendiri **nyaris kosong** — `pyUsage = FLOW`, satu `Property-Set` ke
 * `TempIns.pyNote`. Ia kait pra-proses; yang bekerja adalah **Open Assignment** bawaan
 * Pega, yang membuka klaim pada tahap alur kerjanya saat itu supaya petugas dapat
 * MENGERJAKANNYA — bukan membacanya.
 *
 * Itu jalur TULIS: Open Assignment mengunci objek kerja, dan flow action yang menunggu di
 * sana (`SendtoRCLPUCL` → `SetDataLampiranSuratRCLPUCL_Act`) menyiapkan lampiran surat
 * RCL/PUCL — persis tindakan "Cetak Surat". Selama masa paralel, objek kerja dan
 * penugasannya masih dimiliki Pega (`P-1`).
 *
 * # Apa yang dibuka di sini
 *
 * Layar kerja `SendtoRCLPUCL` — kedua bagiannya, "Lampiran Surat" dan "Penerimaan
 * Dokumen". Ia **BACA SAJA**: bagian yang menulis masih dimiliki Pega (`P-1`).
 *
 * Section-nya sempat tidak ada di export dan modul ini sempat hanya menampilkan isi baris
 * grid. Work Owner menambahkan ketiga rule yang hilang pada 2026-09-24 — induknya beserta
 * kedua sub-section-nya — sehingga isinya kini terbaca dari bukti, bukan ditebak.
 *
 * # Kenapa HALAMAN TERSENDIRI, bukan panel di bawah tabel
 *
 * Karena di Pega yang terbuka bukan pratinjau baris melainkan klaim pada tahap alur
 * kerjanya, untuk dikerjakan. Panel di bawah tabel menyiratkan yang pertama.
 *
 * Konsekuensinya ditangani bersamaan: berpindah halaman melepas komponen antrean, sehingga
 * tab dan nomor halaman pindah ke ALAMAT — bukan hidup di `useState`. Tanpa itu, petugas
 * yang membuka satu klaim dari halaman ketiga tab kedua mendarat kembali di tab pertama
 * halaman pertama.
 *
 * Rentang tanggal sengaja TIDAK ikut ke alamat: ia hanya dipakai tombol ekspor dan tidak
 * menyaring tabel, sehingga membawanya akan menyiratkan ia bagian dari apa yang dilihat.
 *
 * Mengarahkannya ke layar View Claim akan keliru: `ViewTempDetailClaim` memang ada, tetapi
 * dibuka `PNCInboxAdmin`, `PNCSearchKlaim`, `InboxManagerReopen1_Sec`, dan
 * `InputProgressClaim` — **tidak satu pun dari RCL/PUCL**.
 */
function CaseLink({
  item,
  onOpen,
}: {
  item: WorkItem;
  onOpen: (row: WorkItem) => void;
}) {
  if (item.no_case === "") return <span className="text-slate-400">—</span>;

  return (
    <button
      type="button"
      onClick={() => onOpen(item)}
      title={`Buka klaim ${item.no_case}`}
      className={[
        "rounded-kontrol text-left font-medium text-blue-700 underline-offset-2",
        "transition-colors duration-150 ease-halus hover:underline",
        "focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50",
        "disabled:cursor-not-allowed disabled:text-slate-400 disabled:no-underline",
      ].join(" ")}
    >
      {item.no_case}
    </button>
  );
}

/**
 * columnsFor menyusun kolom tabel dari bentuk yang ditetapkan server.
 *
 * # Kolom pertama adalah NOMOR URUT
 *
 * Grid lama menomori barisnya 1, 2, 3, … di paling kiri — `pyGridNumbering = true` pada
 * ketiga section RCL/PUCL. Judulnya KOSONG, begitu pula di sana, dan nomor urut tidak
 * menuntut penjelasan bagi pembaca layar.
 *
 * Penomorannya dimulai dari satu pada SETIAP halaman, bukan diteruskan dari halaman
 * sebelumnya. Itu yang dilakukan grid Pega, dan nomor ini memang hanya alat menunjuk baris —
 * "yang nomor dua" — bukan posisi di dalam seluruh antrean.
 *
 * Petanya disusun sekali. Mencari posisi sebuah baris saat menggambar setiap sel berarti
 * menelusuri seluruh daftar sebanyak jumlah barisnya.
 *
 * # TIDAK ada kolom aksi tambahan
 *
 * Layar lama tidak punya satu pun, dan versi pertama modul ini keliru menambahkannya.
 * Yang membuka baris adalah **tautan pada sel "Nomor Case"** — itulah bentuknya di ketiga
 * section RCL/PUCL (`D-13`).
 *
 * `value` tetap mengembalikan TEKS polos meski selnya digambar sebagai tautan. Keduanya
 * dipisah dengan sengaja oleh `DataTable`: yang dicari dan diurutkan adalah teksnya, yang
 * dilihat pengguna adalah gambarnya. Menyatukannya akan membuat pengurutan menelusuri
 * markup alih-alih nomor case.
 */
function columnsFor(
  tab: Tab,
  rows: WorkItem[],
  openCase: (row: WorkItem) => ReactNode,
): Column<WorkItem>[] {
  const nomor = new Map(rows.map((row, index) => [rowKeyOf(row), index + 1]));

  const kolomNomor: Column<WorkItem> = {
    key: "nomor",
    title: "",
    width: "3rem",
    noSort: true,
    value: (row) => String(nomor.get(rowKeyOf(row)) ?? ""),
  };

  const kolom = tab.kolom.map((column) => {
    const base: Column<WorkItem> = {
      key: column.kunci,
      title: column.judul,
      value: (row) => cellText(row, column),
    };

    if (column.kunci === "no_case") {
      return { ...base, render: openCase };
    }
    return base;
  });

  return [kolomNomor, ...kolom];
}

/**
 * rowKeyOf menyusun kunci satu baris.
 *
 * Ia dipakai DUA kali — oleh `DataTable` sebagai `rowKey`, dan oleh peta nomor urut di atas
 * — sehingga bentuknya ditulis sekali di sini. Dua tempat yang merakit kunci yang sama
 * dengan tangan adalah dua tempat yang dapat bergeser, dan pergeserannya muncul sebagai
 * kolom nomor urut yang kosong tanpa satu pun galat.
 */
function rowKeyOf(row: WorkItem): string {
  return `${row.referensi}|${row.no_case}`;
}

/**
 * cellText menyusun teks satu sel.
 *
 * Tiga perlakuan, dan masing-masing punya alasannya:
 *
 *  - **"Lama Klaim" digambar sebagai waktu RELATIF** — "21 hours ago", "1 day 1 hour ago".
 *    Judulnya menyebut durasi dan isinya tanggal, dan yang menjembatani keduanya adalah
 *    kontrolnya: sel itu satu-satunya di section ini yang ber-`pyDateTimeFormat =
 *    DateTime-Frame`, format waktu relatif bawaan Pega. Sampai 2026-10-10 layar ini
 *    menggambarnya sebagai tanggal penuh, dan itu nilai yang benar dengan bentuk yang salah.
 *  - **Kolom tanggal lain digambar `dd/MM/yy H:mm`** — bentuk sel tanggal pada grid Pega,
 *    mis. `09/10/26 17:05`. Sebelumnya hanya nilai ber-bentuk `YYYY-MM-DD` yang diformat,
 *    sehingga nilai yang SELALU membawa jam tidak pernah cocok dan tampil mentah
 *    (`2026-09-10 09:30:00`). Teks yang bukan tanggal dikembalikan apa adanya oleh
 *    `formatPegaDateTime`, sehingga kolom non-tanggal tidak tersentuh.
 *  - **Teks kosong menjadi tanda pisah**, bukan sel kosong yang tidak dapat dibedakan dari
 *    kolom yang gagal dimuat. Tanda pisah menyatakan "tidak ada isinya" alih-alih "gagal".
 *
 *    Perlakuan ini untuk sel yang KEBETULAN kosong. Kolom yang TIDAK MUNGKIN terisi bukan
 *    urusannya: "Tanggal Cetak Surat" di tab pertama sudah tidak dikirim server sama
 *    sekali sejak 2026-10-11, karena kolom itu justru penyaring tabnya. Satu kolom penuh
 *    tanda pisah bukan keterangan — ia ruang yang terpakai tanpa memberi tahu apa pun.
 */
function cellText(row: WorkItem, column: TabColumn): string {
  const value = row[column.kunci];

  if (value == null || value === "") return "—";

  const text = String(value);

  if (column.kunci === "lama_klaim") {
    // Kosong berarti tanggalnya di masa depan — baris seperti itu memang ada di data
    // warisan. Tanda pisah, bukan "0 minutes ago" yang menyatakan hal yang tidak benar.
    return formatPegaElapsed(text) || "—";
  }

  return formatPegaDateTime(text);
}

/**
 * emptyMessageFor menjelaskan antrean kosong menurut sebabnya.
 *
 * "Tidak ada data" tidak cukup di layar ini, dan alasannya lebih tajam daripada di layar
 * lain: ketiga tab punya kolom yang IDENTIK, sehingga pengguna yang melihat tabel kosong
 * tidak punya satu pun petunjuk apakah ia berada di tab yang salah, antreannya memang sepi,
 * atau ada yang keliru di konfigurasinya.
 *
 * Dua dari tiga pesan di bawah menunjuk KEKELIRUAN KONFIGURASI yang tidak menghasilkan satu
 * pun galat — dan keterangan inilah satu-satunya yang mengarahkan pengguna melapor alih-alih
 * menganggapnya wajar.
 */
function emptyMessageFor(tab: Tab): string {
  if (tab.punya_laporan_rentang_tanggal) {
    return (
      "Tidak ada klaim RCL/PUCL yang suratnya menunggu dicetak. Bila Anda yakin " +
      'seharusnya ada, periksa tab "Kelengkapan Dokumen" lebih dulu — klaim yang suratnya ' +
      "sudah dicetak pindah ke sana dengan sendirinya."
    );
  }

  if (tab.catatan) {
    // Tab "Klaim MSIG". Catatannya sudah menjelaskan kosongnya di atas grid, jadi pesan di
    // sini cukup mengarahkan ke sana alih-alih mengulangnya.
    return (
      "Tidak ada klaim jalur MSIG. Lihat catatan di atas tabel — tab ini memang " +
      "diperkirakan kosong, dan penyebabnya sedang dipastikan ke DBA."
    );
  }

  return (
    "Tidak ada klaim yang menunggu kelengkapan dokumen. Bila Anda yakin seharusnya ada, " +
    "laporkan ke tim teknis: klaim yang penanda persetujuannya belum pernah diisi tidak " +
    "muncul di sini, dan itu perilaku yang dibawa dari sistem lama."
  );
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message;
  if (error instanceof Error && error.message !== "") return error.message;
  return "Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.";
}
