import { useEffect, useState } from "react";

import { APIError, NetworkError } from "@/api/client";
import type { AutoClaimBatch, AutoClaimBatchOrder } from "@/api/types";
import { Button } from "@/components/Button";
import { DataTable, type Column } from "@/components/DataTable";
import { ErrorMessage } from "@/components/ErrorMessage";
import { DownloadIcon, SearchIcon } from "@/components/Icon";

import { useAutoClaimBatchList, useExportAutoClaim } from "./api";
import { BatchDetail } from "./BatchDetail";
import { loadMessage } from "./messages";

type Props = {
  source: string;
  company: string;
  companyName: string;
  /** Jumlah batch perusahaan ini menurut ringkasan, untuk lencana di kepala panel. */
  batchCount: number;
};

/** Jeda ketik sebelum kata kunci dikirim ke server. */
const SEARCH_DELAY_MS = 350;

/**
 * Daftar batch SATU perusahaan — panel kanan pada CompanyBrowser.
 *
 * # Muat satu layar (permintaan Work Owner 2026-09-29)
 *
 * Grid 9 kolom layar lama (`Section/Inbox_AS_KREDIT_Sect-Section.xml`) ditambah tiga
 * tombol tidak muat di samping daftar perusahaan, dan kolom User Upload terpotong.
 * Penyesuaiannya, tanpa membuang satu angka pun:
 *
 *   - KODE dan Nama Perusahaan DIBUANG dari grid: keduanya sudah tertulis di kepala panel,
 *     dan setiap baris grid ini milik perusahaan yang sama. Peringatan "tidak terdaftar di
 *     Master Auto Claim" pindah menjadi satu pita di atas grid.
 *   - Di Upload dan Diproses DIGABUNG menjadi "diproses/diunggah" dengan bilah kemajuan;
 *     selisihnya disebut sebagai "+n menunggu".
 *
 * # Penataan ulang (`D-82`, 2026-10-14)
 *
 *   - Tombol unduh Berhasil/Gagal pindah dari kolom Aksi menjadi ikon DI SAMPING
 *     angkanya. Angka 0 tidak punya tombol — tidak ada yang dapat diunduh. Nama
 *     aksesibilitasnya tetap lengkap ("Export berhasil batch …").
 *   - Kolom Aksi tinggal tiga tombol sebaris: Detail, Generate DLA, Proses Klaim (utama).
 *   - Kepala panel memuat pencarian no batch / user upload dan pilihan urutan. Keduanya
 *     dikerjakan SERVER: grid ini dipaginasi server, sehingga menyaring di peramban hanya
 *     menyaring halaman yang sedang tampil.
 *   - Keterangan "Sumber: <tabel>" dibuang dari layar.
 *
 * Jarak sel memakai mode rapat `DataTable` (`dense`). Di layar sempit grid berubah menjadi
 * kartu — perilaku bawaan `DataTable`.
 */
export function CompanyBatches({
  source,
  company,
  companyName,
  batchCount,
}: Props) {
  const [page, setPage] = useState(1);
  const [opened, setOpened] = useState<string | null>(null);
  const [text, setText] = useState("");
  const [search, setSearch] = useState("");
  const [order, setOrder] = useState<AutoClaimBatchOrder>("terbaru");

  // Kata kunci dikirim setelah pengguna berhenti mengetik, bukan pada setiap huruf.
  // Halaman kembali ke 1: halaman 3 dari hasil lama belum tentu ada pada hasil baru.
  useEffect(() => {
    const timer = window.setTimeout(() => {
      setSearch(text.trim());
      setPage(1);
    }, SEARCH_DELAY_MS);
    return () => window.clearTimeout(timer);
  }, [text]);

  const list = useAutoClaimBatchList(source, company, page, search, order);
  const exportFile = useExportAutoClaim();

  const label = companyName === "" ? company : companyName;

  const compact = "!px-2.5 !py-1 text-xs";

  function exportButton(row: AutoClaimBatch, hasil: "berhasil" | "gagal") {
    const count = hasil === "berhasil" ? row.jumlah_berhasil : row.jumlah_gagal;
    // Angka 0 tanpa tombol, tetapi tempatnya tetap terisi supaya angka di kolom yang
    // sama tetap lurus.
    if (count === 0) {
      return <span aria-hidden="true" className="inline-block h-7 w-7" />;
    }
    return (
      <button
        type="button"
        disabled={exportFile.isPending}
        aria-label={`Export ${hasil} batch ${row.batch} ${row.kode_perusahaan}`}
        title={
          hasil === "berhasil" ? "Export baris berhasil" : "Export baris gagal"
        }
        onClick={() =>
          exportFile.mutate({ source, company, batch: row.batch, hasil })
        }
        className={[
          "inline-flex h-7 w-7 items-center justify-center rounded-lg border border-slate-300 bg-white",
          "transition-colors duration-150 ease-halus hover:bg-slate-50 disabled:opacity-50",
          "focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/25",
          hasil === "berhasil" ? "text-blue-600" : "text-red-600",
        ].join(" ")}
      >
        <DownloadIcon className="h-3.5 w-3.5" />
      </button>
    );
  }

  const columns: Column<AutoClaimBatch>[] = [
    {
      key: "batch",
      title: "Batch",
      width: "4.5rem",
      value: (row) => row.batch,
      render: (row) => (
        <span className="text-[0.9375rem] font-bold text-slate-900 tabular-nums">
          {row.batch}
        </span>
      ),
    },
    {
      // Bagian KUNCI PENGELOMPOKAN: satu nomor batch yang diunggah pada dua tanggal tampil
      // sebagai dua baris, dan tanpa kolom ini keduanya tampak kembar.
      key: "tanggal",
      title: "Tgl proses",
      width: "6.5rem",
      value: (row) => row.tanggal_proses,
      render: (row) => (
        <span className="tabular-nums">{row.tanggal_proses}</span>
      ),
    },
    {
      key: "proses",
      title: "Diproses",
      width: "8.5rem",
      value: (row) => `${row.jumlah_proses}/${row.jumlah_upload}`,
      render: (row) => {
        const percent =
          row.jumlah_upload === 0
            ? 0
            : Math.round((row.jumlah_proses / row.jumlah_upload) * 100);
        return (
          <span className="inline-flex flex-col gap-0.5">
            <span className="inline-flex items-center gap-2">
              <span
                aria-hidden="true"
                className="h-1.5 w-12 overflow-hidden rounded-full bg-slate-200"
              >
                <span
                  className="block h-full rounded-full bg-blue-600"
                  style={{ width: `${percent}%` }}
                />
              </span>
              <span className="font-bold text-slate-900 tabular-nums">
                {row.jumlah_proses}/{row.jumlah_upload}
              </span>
            </span>
            {row.jumlah_belum_proses > 0 && (
              <span className="text-xs font-normal text-amber-700">
                +{row.jumlah_belum_proses} menunggu
              </span>
            )}
          </span>
        );
      },
    },
    {
      key: "berhasil",
      title: "Berhasil",
      width: "5.5rem",
      value: (row) => String(row.jumlah_berhasil),
      render: (row) => (
        <span className="inline-flex items-center gap-1.5">
          <span
            className={[
              "min-w-[1ch] text-right text-sm font-bold tabular-nums",
              row.jumlah_berhasil > 0 ? "text-emerald-700" : "text-slate-400",
            ].join(" ")}
          >
            {row.jumlah_berhasil}
          </span>
          {exportButton(row, "berhasil")}
        </span>
      ),
    },
    {
      key: "gagal",
      title: "Gagal",
      width: "5.5rem",
      value: (row) => String(row.jumlah_gagal),
      render: (row) => (
        <span className="inline-flex items-center gap-1.5">
          <span
            className={[
              "min-w-[1ch] text-right text-sm font-bold tabular-nums",
              row.jumlah_gagal > 0 ? "text-red-700" : "text-slate-400",
            ].join(" ")}
          >
            {row.jumlah_gagal}
          </span>
          {exportButton(row, "gagal")}
        </span>
      ),
    },
    {
      key: "user",
      title: "User upload",
      value: (row) => row.user_upload,
      render: (row) => (
        <span
          className="block max-w-[10rem] truncate text-xs font-semibold text-slate-900"
          title={row.user_upload}
        >
          {row.user_upload || "—"}
        </span>
      ),
    },
    {
      key: "aksi",
      title: "Aksi",
      width: "17rem",
      noSort: true,
      alignRight: true,
      value: () => "",
      render: (row) => (
        <span className="inline-flex flex-wrap justify-end gap-1.5 md:flex-nowrap md:whitespace-nowrap">
          <Button
            tone="kedua"
            className={compact}
            onClick={() => {
              exportFile.reset();
              setOpened(row.batch);
            }}
            aria-label={`Detail batch ${row.batch} ${row.kode_perusahaan}`}
          >
            Detail
          </Button>
          {/* Proses Klaim kini PER BATCH (permintaan Work Owner 2026-09-29) — layar lama
              menaruhnya sekali per tab dan memproses semua batch sekaligus
              (`CreateCasePNCAgent_*`). Generate DLA memang per baris di layar lama
              (`GenerateDLA_Askredit`). Mesin keduanya belum dibangun, jadi tombolnya
              nonaktif; alasannya ada di `title`, bukan catatan di layar. */}
          <Button
            tone="kedua"
            className={compact}
            disabled
            aria-label={`Generate DLA batch ${row.batch} ${row.kode_perusahaan}`}
            title="Generate DLA batch ini — belum tersedia di aplikasi ini"
          >
            Generate DLA
          </Button>
          <Button
            tone="utama"
            className={compact}
            disabled
            aria-label={`Proses klaim batch ${row.batch} ${row.kode_perusahaan}`}
            title="Proses Klaim batch ini — belum tersedia di aplikasi ini"
          >
            Proses Klaim
          </Button>
        </span>
      ),
    },
  ];

  const searchId = `cari-batch-${company}`;
  const orderId = `urut-batch-${company}`;

  return (
    <div className="flex flex-col">
      <header className="flex flex-wrap items-center gap-2.5 pb-3">
        <h2 className="text-[1.0625rem] font-bold text-slate-900">{label}</h2>
        {companyName !== "" && (
          <span className="inline-flex h-[22px] items-center rounded-md bg-slate-100 px-2 text-xs font-bold text-slate-700 tabular-nums">
            {company}
          </span>
        )}
        <span className="inline-flex h-[22px] items-center rounded-full bg-slate-100 px-2 text-xs font-bold text-slate-700 tabular-nums">
          {batchCount} batch
        </span>
        <div className="flex w-full flex-wrap items-center gap-2 lg:ml-auto lg:w-auto lg:flex-nowrap">
          <div className="relative w-full sm:w-64">
            <label htmlFor={searchId} className="sr-only">
              Cari no batch / user upload
            </label>
            <span
              aria-hidden="true"
              className="pointer-events-none absolute inset-y-0 left-0 flex w-9 items-center justify-center text-slate-400"
            >
              <SearchIcon className="h-4 w-4" />
            </span>
            <input
              id={searchId}
              type="search"
              value={text}
              onChange={(event) => setText(event.target.value)}
              placeholder="Cari no batch / user upload…"
              className={[
                "w-full rounded-kontrol border border-slate-400 bg-white py-2 pl-9 pr-3",
                "text-sm text-slate-900 placeholder:text-slate-500",
                "transition-[border-color,box-shadow] duration-150 ease-halus hover:border-slate-500",
                "focus:border-blue-500 focus:outline-none focus:ring-4 focus:ring-blue-500/15",
              ].join(" ")}
            />
          </div>
          <label
            htmlFor={orderId}
            className="text-xs text-slate-500 sm:ml-auto lg:ml-2"
          >
            Urutkan
          </label>
          <select
            id={orderId}
            value={order}
            onChange={(event) => {
              setOrder(event.target.value as AutoClaimBatchOrder);
              setPage(1);
            }}
            className="rounded-kontrol border border-slate-300 bg-white py-1.5 pr-8 pl-2.5 text-xs font-semibold text-slate-800 focus:border-blue-500 focus:outline-none focus:ring-4 focus:ring-blue-500/15"
          >
            <option value="terbaru">Batch terbaru</option>
            <option value="terlama">Batch terlama</option>
          </select>
        </div>
      </header>

      <div className="flex flex-col gap-3">
        {companyName === "" && (
          // Baris perusahaan seperti ini TIDAK AKAN PERNAH berhasil diproses — pencarian
          // penerima klaim membaca master yang sama — jadi petugas perlu melihatnya.
          <p
            role="note"
            className="rounded-kartu border border-amber-200 bg-amber-50 px-4 py-2.5 text-sm text-amber-900"
          >
            Tidak terdaftar di Master Auto Claim — kode {company} perlu
            didaftarkan. Batch perusahaan ini tidak akan berhasil diproses
            sebelum itu.
          </p>
        )}

        {exportFile.isError && (
          <ErrorMessage
            title="Berkas gagal diunduh"
            description={
              exportFile.error instanceof APIError
                ? exportFile.error.message
                : "Periksa koneksi jaringan Anda, lalu coba lagi."
            }
            tone={
              exportFile.error instanceof NetworkError
                ? "gangguan"
                : "penolakan"
            }
          />
        )}

        {list.isError ? (
          <ErrorMessage {...loadMessage(list.error)} />
        ) : (
          <DataTable
            columns={columns}
            rows={list.data?.batch ?? []}
            rowKey={(row) =>
              `${row.kode_perusahaan}-${row.batch}-${row.tanggal_proses}`
            }
            isLoading={list.isPending}
            // Nama tabelnya menyebut perusahaannya: layar ini memuat daftar perusahaan DAN
            // grid batch, dan pembaca layar perlu membedakan keduanya.
            label={`Batch ${label}`}
            hideSearch
            dense
            emptyMessage={
              search === ""
                ? "Perusahaan ini belum punya batch klaim."
                : `Tidak ada batch yang cocok dengan “${search}”.`
            }
            pagination={{
              page: list.data?.paginasi.halaman ?? page,
              size: list.data?.paginasi.ukuran ?? 15,
              total: list.data?.paginasi.total ?? 0,
              totalPage: list.data?.paginasi.total_halaman ?? 0,
              onPageChange: setPage,
              isLoading: list.isFetching,
            }}
          />
        )}
      </div>

      {opened !== null && (
        <DetailDialog batch={opened} onClose={() => setOpened(null)}>
          <BatchDetail
            company={company}
            companyName={companyName}
            source={source}
            batch={opened}
            onClose={() => setOpened(null)}
          />
        </DetailDialog>
      )}
    </div>
  );
}

/**
 * Pop-up rincian batch (permintaan Work Owner 2026-09-29).
 *
 * Sebelumnya rincian dibuka DI BAWAH grid, sehingga petugas harus menggulir untuk
 * melihatnya dan grid terdorong keluar layar. Pop-up menjaga grid tetap di tempat dan
 * menutup kembali dengan Escape, tombol "Tutup rincian", atau klik di luar kotaknya —
 * pola yang sama dengan pop-up Upload Data Klaim.
 */
function DetailDialog({
  batch,
  onClose,
  children,
}: {
  batch: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") onClose();
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={`Rincian batch ${batch}`}
      className="fixed inset-0 z-50 flex items-start justify-center bg-slate-900/40 px-3 py-6 sm:items-center sm:px-6"
      onClick={(event) => {
        // Hanya klik pada latar yang menutup; klik di dalam kotak tidak.
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div className="max-h-full w-full max-w-5xl overflow-y-auto rounded-kartu bg-white p-4 shadow-terbang sm:p-6">
        {children}
      </div>
    </div>
  );
}
