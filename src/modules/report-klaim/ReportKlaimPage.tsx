import { useState } from "react";

import { APIError } from "@/api/client";
import { useSelectedPortal } from "@/app/portal";
import { Button } from "@/components/Button";
import { ErrorMessage } from "@/components/ErrorMessage";
import { Field } from "@/components/Field";
import { SelectField } from "@/components/SelectField";

import { useExportReport, useReportCatalog } from "./api";
import {
  EMPTY_FILTER,
  type BusinessLine,
  type ComplianceStatus,
  type ExportRequest,
  type Report,
  type ReportFilter,
} from "./types";

/**
 * Report Klaim — menu `MENU_ID 85`, pengganti harness `PNCTATReport`.
 *
 * Judul yang dibaca pengguna di sistem lama adalah **"Report Claim"**, dan isinya bukan
 * satu laporan melainkan **28 panel** yang berdiri sendiri-sendiri. Setiap panel hanya
 * berisi judul dan tombol Export; tidak ada tabel hasil di layar sama sekali — yang
 * keluar adalah berkas CSV.
 *
 * # Susunan layar dibaca dari harness, bukan dikarang
 *
 * Ralat Work Owner 2026-10-08 menolak susunan sebelumnya. Harness `PNCTATReport` dibaca
 * ulang, dan kontrolnya — menurut urutan tata letaknya sendiri — hanya empat:
 *
 *	posisi   properti                            kontrol            label
 *	167.588  TempLaporan.AnalystTransferDate     pxDateTime         Dari
 *	179.041  TempLaporan.DateOfLoss              pxDateTime         Sampai
 *	185.181  TempLaporan.StatusReceiver          pxDropdown         Bisnis
 *	193.497  ComplianceStatus                    pxRadioButtons     Status Compliance
 *
 * Dua kontrol lain ada di harness, tetapi JAUH di bawah — bukan di bilah penyaring:
 *
 *	874.168  TempLaporan.Country                 pxAutoComplete     tepat setelah REPORT KLAIM PER BISNIS
 *	961.442  TempLaporan.Remark                  pxCheckbox         tepat setelah REPORT AKSEPTASI, berlabel "Treaty"
 *
 * Empat hal karena itu diperbaiki di sini:
 *
 *  1. **Dropdown lini bisnis berlabel "Bisnis"**, bukan "Treaty". Yang berlabel "Treaty"
 *     adalah kotak centang pada panel Akseptasi — kontrol yang sama sekali berbeda.
 *  2. **Status Compliance menjadi tiga radio**, bukan dropdown (`pxRadioButtons`).
 *  3. **Autocomplete bisnis dan kotak centang pindah ke baris panelnya masing-masing**,
 *     seperti di harness.
 *  4. **Daftar panel menjadi satu kolom datar berurutan**, tanpa pengelompokan. Harness
 *     tidak punya kelompok; pengelompokan sebelumnya memindahkan panel dari tempat yang
 *     sudah dihafal pengguna.
 *
 * Satu isian yang sempat ada — kotak centang bersama "Tampilkan kolom rincian" — dihapus
 * seluruhnya. Ia tidak ada di harness; yang ada adalah kotak centang "Treaty" pada satu
 * panel, dan itulah yang sekarang digambar.
 */
export function ReportKlaimPage() {
  const portal = useSelectedPortal((state) => state.alias);
  const [filter, setFilter] = useState<ReportFilter>(EMPTY_FILTER);
  const [sedangDiunduh, setSedangDiunduh] = useState<string | null>(null);

  /*
   * Keadaan kotak centang "Treaty", DIKUNCI PER PANEL.
   *
   * Sebelumnya ia satu nilai di dalam `filter`, dan akibatnya dua: mencentang di satu
   * panel ikut mencentang di dua panel lain, dan ketiganya mulai kosong padahal di Pega
   * dua di antaranya mulai tercentang.
   *
   * Yang kedua lebih berat daripada kelihatannya — kotak ini memilih SUSUNAN KOLOM
   * berkas, bukan menyaring baris. Keadaan awal yang keliru berarti berkas yang diunduh
   * tanpa disentuh siapa pun punya kolom berbeda dari Pega.
   *
   * Nilai awalnya datang dari peladen (`penyaring.rincian_awal`) karena ia pengetahuan
   * tentang sistem lama. Yang di sini hanya MENYIMPAN perubahan pengguna.
   */
  const [rincian, setRincian] = useState<Record<string, boolean>>({});

  const rincianPanel = (laporan: Report) =>
    rincian[laporan.kode] ?? laporan.penyaring.rincian_awal;

  const catalog = useReportCatalog();
  const exportReport = useExportReport();

  // Daftar bisnis beserta hook-nya dicabut 2026-10-13 bersama panel REPORT KLAIM PER
  // BISNIS — satu-satunya yang memakainya.

  function unduh(laporan: Report, aksi: string) {
    setSedangDiunduh(laporan.kode);
    const request: ExportRequest = {
      kode: laporan.kode,
      aksi,
      filter: { ...filter, rincian: rincianPanel(laporan) },
      penyaring: laporan.penyaring,
    };
    exportReport.mutate(request, { onSettled: () => setSedangDiunduh(null) });
  }

  if (portal === null) {
    return (
      <div className="mx-auto max-w-3xl p-6">
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            "Berkas laporan memuat data klaim milik satu badan hukum, dan aplikasi ini " +
            "melayani empat. Pilih portal di bilah atas untuk membuka layar ini."
          }
          tone="penolakan"
        />
      </div>
    );
  }

  const ubah = (bagian: Partial<ReportFilter>) =>
    setFilter({ ...filter, ...bagian });

  return (
    <div className="mx-auto max-w-5xl space-y-5 p-4 sm:p-6">
      <header className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">
            {catalog.data?.judul ?? "Report Claim"}
          </h1>
          <p className="mt-1 text-sm text-slate-600">
            Pilih periode dan penyaringnya, lalu tekan Export pada laporan yang
            dituju. Berkasnya terunduh dalam bentuk CSV.
          </p>
        </div>
        {/*
          Tombol ini ada di Pega, di sudut yang sama, dan selama ini terlewat.

          Yang dimuat ulang DAFTAR panelnya — bukan isi laporan. Ketersediaan sebuah panel
          dapat berubah tanpa layar di-reload (koneksi kedua terpasang, master terisi),
          dan tanpa tombol ini satu-satunya cara menyegarkannya adalah memuat ulang
          seluruh halaman.
        */}
        <Button
          tone="kedua"
          disabled={catalog.isFetching}
          onClick={() => void catalog.refetch()}
        >
          {catalog.isFetching ? "Memuat…" : "Refresh List"}
        </Button>
      </header>

      <FilterBar
        filter={filter}
        ubah={ubah}
        liniBisnis={catalog.data?.lini_bisnis ?? []}
        statusCompliance={catalog.data?.status_compliance ?? []}
        pelanggaran={pelanggaranOf(exportReport.error)}
      />

      {exportReport.isError && (
        <ErrorMessage
          title="Berkas tidak dapat diunduh"
          description={messageOf(exportReport.error)}
          tone={toneOf(exportReport.error)}
        />
      )}

      {catalog.isPending && (
        <p className="text-sm text-slate-500">Memuat daftar laporan…</p>
      )}

      {catalog.isError && (
        <ErrorMessage
          title="Daftar laporan tidak dapat dimuat"
          description={messageOf(catalog.error)}
          tone="gangguan"
        />
      )}

      {/*
        DUA KOLOM, mengalir kiri-kanan lalu turun — persis layar Pega.

        Sebelum 2026-10-13 di sini satu kolom. Kesimpulannya ditarik dari ketiadaan
        penanda kelompok di harness, dan itu kekeliruan yang sama bentuknya dengan yang
        sudah dua kali terjadi di layar ini: menyimpulkan tampilan dari apa yang TIDAK
        ada di XML. Yang menjawabnya akhirnya tangkapan layar produksi.

        Urutan kartunya sendiri tidak berubah — ia memang urutan baca kiri-ke-kanan, dan
        itu sebabnya `CatalogInPegaOrder` sudah cocok sejak awal.

        Satu kolom di lebar telepon: dua kolom kartu selebar ini tidak terbaca di sana,
        dan layar Pega pun tidak pernah dibuka dari telepon.
      */}
      <div className="grid gap-3 md:grid-cols-2">
        {catalog.data?.laporan.map((laporan) => (
          <ReportRow
            key={laporan.kode}
            laporan={laporan}
            rincian={rincianPanel(laporan)}
            ubahRincian={(nilai) =>
              setRincian({ ...rincian, [laporan.kode]: nilai })
            }
            sedangDiunduh={sedangDiunduh === laporan.kode}
            onExport={(aksi) => unduh(laporan, aksi)}
          />
        ))}
      </div>
    </div>
  );
}

type FilterBarProps = {
  filter: ReportFilter;
  ubah: (bagian: Partial<ReportFilter>) => void;
  liniBisnis: BusinessLine[];
  statusCompliance: ComplianceStatus[];
  /** Pesan per isian dari penolakan terakhir, dikunci nama isiannya. */
  pelanggaran: Record<string, string>;
};

/**
 * Bilah penyaring bersama — EMPAT kontrol, persis seperti harness.
 *
 * # Kenapa tidak ada isian yang dinonaktifkan
 *
 * Sebelumnya isian dinonaktifkan bila kartu yang sedang tersorot tidak memakainya.
 * Niatnya menuntun; akibatnya menjebak, dan laporannya masuk 2026-10-01: kursor yang
 * bergerak ke arah dropdown melintasi kartu lain, dan dropdown itu mati tepat sebelum
 * disentuh.
 *
 * Layar Pega menampilkan keempat isiannya hidup setiap saat, dan isian yang tidak dipakai
 * sebuah laporan diabaikan peladen. Itu yang ditiru.
 */
function FilterBar({
  filter,
  ubah,
  liniBisnis,
  statusCompliance,
  pelanggaran,
}: FilterBarProps) {
  return (
    <div className="rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Field
          id="dari"
          label="Dari"
          type="date"
          value={filter.dari}
          error={pelanggaran["dari"]}
          onChange={(e) => ubah({ dari: e.target.value })}
        />
        <Field
          id="sampai"
          label="Sampai"
          type="date"
          value={filter.sampai}
          error={pelanggaran["sampai"]}
          onChange={(e) => ubah({ sampai: e.target.value })}
        />
        {/*
          Label "Bisnis" disalin dari harness; isinya lini bisnis (`TempLaporan.StatusReceiver`),
          yaitu nilai yang dibandingkan prakondisi activity-nya: "002", "005", "346", "003".
        */}
        <SelectField
          id="lini"
          label="Bisnis"
          value={filter.lini}
          options={liniBisnis
            .filter((l) => l.nilai !== "")
            .map((l) => ({ value: l.nilai, label: l.nama }))}
          emptyText="----- Pilih -----"
          onChange={(e) => ubah({ lini: e.target.value })}
        />

        {/*
          Tiga radio, bukan dropdown — `pyFormat = pxRadioButtons` pada harness.
          Nilainya "0", "1", "2" dari rule Property ComplianceStatus; labelnya dibaca
          pengguna. Mengirim labelnya akan membuat penyaringnya tidak pernah cocok.
        */}
        <fieldset>
          <legend className="mb-1 block text-sm font-medium text-slate-700">
            Status Compliance
          </legend>
          <div className="space-y-1">
            {statusCompliance.map((pilihan) => (
              <label
                key={pilihan.nilai}
                className="flex items-center gap-2 text-sm text-slate-700"
              >
                <input
                  type="radio"
                  name="status_compliance"
                  value={pilihan.nilai}
                  checked={filter.status_compliance === pilihan.nilai}
                  onChange={(e) => ubah({ status_compliance: e.target.value })}
                  className="h-4 w-4 border-slate-300"
                />
                {pilihan.nama}
              </label>
            ))}
          </div>
        </fieldset>
      </div>
    </div>
  );
}

type RowProps = {
  laporan: Report;
  /** Keadaan kotak centang "Treaty" MILIK PANEL INI, bukan keadaan bersama. */
  rincian: boolean;
  ubahRincian: (nilai: boolean) => void;
  sedangDiunduh: boolean;
  onExport: (aksi: string) => void;
};

/**
 * Satu baris laporan: judul, tombol Export di sampingnya, dan isian khusus panel itu.
 *
 * Panel yang TIDAK tersedia tetap digambar, bertanda sebabnya. Menghilangkannya membuat
 * pengguna melaporkan laporan yang "hilang", dan membuat kemajuan migrasi tidak terbaca
 * dari layar — perlakuan yang sama dengan butir menu yang belum punya layar.
 */
function ReportRow({
  laporan,
  rincian,
  ubahRincian,
  sedangDiunduh,
  onExport,
}: RowProps) {
  const mati = !laporan.tersedia;

  return (
    <article
      className={`rounded-kartu border p-4 shadow-lembut ${
        mati ? "border-slate-200 bg-slate-50" : "border-slate-200 bg-white"
      }`}
    >
      <div className="flex flex-wrap items-center gap-3">
        <h2
          className={`text-sm font-semibold ${mati ? "text-slate-500" : "text-slate-900"}`}
        >
          {laporan.judul}
        </h2>

        {mati ? (
          <span className="text-xs font-medium uppercase tracking-wide text-slate-400">
            Belum tersedia
          </span>
        ) : (
          laporan.tombol.map((tombol) => (
            <Button
              key={tombol.kode === "" ? laporan.kode : tombol.kode}
              tone={
                laporan.tombol.length > 1 && tombol.kode === "rejected"
                  ? "kedua"
                  : "utama"
              }
              disabled={sedangDiunduh}
              onClick={() => onExport(tombol.kode)}
            >
              {sedangDiunduh ? "Menyiapkan…" : tombol.label}
            </Button>
          ))
        )}
      </div>

      {/*
        Isian khusus panel ini — kotak centang "Treaty". Letaknya mengikuti layar Pega:
        ia berada DI BARIS panelnya sendiri, bukan di bilah penyaring atas.

        Autocomplete "Bisnis" berdiri di sini sampai 2026-10-13, lalu dicabut bersama
        panel REPORT KLAIM PER BISNIS yang tidak ada di Pega produksi.
      */}

      {!mati && laporan.penyaring.rincian && (
        <label className="mt-3 flex items-center gap-2 text-sm text-slate-700">
          <input
            type="checkbox"
            checked={rincian}
            onChange={(e) => ubahRincian(e.target.checked)}
            className="h-4 w-4 rounded border-slate-300"
          />
          {/*
            Labelnya "Treaty" dan isinya bukan treaty: ia memilih SUSUNAN KOLOM berkas,
            bukan menyaring baris. Label itu ditiru apa adanya (`D-13`) — penggunanya sudah
            mengenalnya dengan nama itu, dan memperbaikinya di layar berarti melatih ulang
            tanpa ada yang meminta.
          */}
          Treaty
        </label>
      )}

      {mati && (
        <p className="mt-2 text-xs leading-relaxed text-slate-500">
          {laporan.alasan}
          {laporan.penghalang !== undefined && laporan.penghalang !== "" && (
            <span className="ml-1 rounded bg-slate-200 px-1.5 py-0.5 font-medium text-slate-600">
              {laporan.penghalang}
            </span>
          )}
        </p>
      )}
    </article>
  );
}

function messageOf(failure: unknown): string {
  if (failure instanceof APIError) {
    // Pesan ringkasnya DIGANTI daftar isian yang kurang, bukan ditambahi: ringkasan
    // "Ada isian yang belum benar." tidak dapat ditindaklanjuti pada layar yang punya
    // empat isian dan 28 tombol.
    const perIsian = Object.values(failure.violations());
    if (perIsian.length > 0) return perIsian.join(" ");
    return failure.message;
  }
  if (failure instanceof Error) return failure.message;
  return "Terjadi kesalahan pada sistem.";
}

/**
 * Penolakan dan gangguan dibedakan nadanya.
 *
 * Isian yang belum benar dapat diperbaiki pengguna; kegagalan sistem tidak. Menyamakan
 * tampilannya membuat pengguna mencoba berulang kali pada hal yang tidak akan berubah.
 */
function toneOf(failure: unknown): "penolakan" | "gangguan" {
  if (failure instanceof APIError && failure.status < 500) return "penolakan";
  return "gangguan";
}

/** pelanggaranOf mengambil pesan PER ISIAN dari sebuah penolakan. */
function pelanggaranOf(failure: unknown): Record<string, string> {
  if (failure instanceof APIError) return failure.violations();
  return {};
}
