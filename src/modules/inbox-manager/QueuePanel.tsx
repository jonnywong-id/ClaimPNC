import { useEffect, useState } from "react";

import { Button } from "@/components/Button";
import { DataTable, type Column } from "@/components/DataTable";
import { ErrorMessage } from "@/components/ErrorMessage";
import { DownloadIcon, ReloadIcon } from "@/components/Icon";
import { useSession } from "@/app/session";

import { PaymentDetailPanel } from "./PaymentDetailPanel";
import { Chip, inputClass } from "./parts";
import type { Pagination, QueueRow, Tab, Verdict } from "./types";

type Props = {
  tab: Tab;
  rows: QueueRow[];
  pagination?: Pagination | undefined;
  isLoading: boolean;

  page: number;
  onPageChange: (page: number) => void;

  onDecide: (verdict: Verdict, keys: string[], reason: string) => void;
  isDeciding: boolean;

  onExport: () => void;
  isExporting: boolean;

  /** Tombol Refresh antrean ber-DETAILS — `Akseptasi_PaymentLeader1`. */
  onRefresh?: (() => void) | undefined;
  isRefreshing?: boolean | undefined;

  /** Pesan hasil keputusan terakhir; kosong berarti belum ada. */
  resultMessage?: string | undefined;
  decideError?: string | undefined;
};

/**
 * Satu antrean persetujuan — kartu tabel dengan bilah keputusan yang muncul saat ada baris
 * dipilih, mengikuti rancangan layar Work Owner 2026-10-14.
 *
 * Isinya tetap Pega: kolom, label tombol keputusan, label alasan, dan jalur massal datang dari
 * server (keputusan Work Owner 2026-10-14 — "tetap kolom Pega", "teks Pega").
 *
 * # Kenapa pemilihan baris ada di sini, bukan di DataTable
 *
 * `DataTable` dipakai lebih dari enam puluh layar yang hampir seluruhnya tidak memutuskan apa
 * pun (`U-2`). Kotak pilihnya karena itu kolom biasa lewat `render`, dan "pilih semua" berada
 * di kepala kartu.
 */
export function QueuePanel({
  tab,
  rows,
  pagination,
  isLoading,
  page,
  onPageChange,
  onDecide,
  isDeciding,
  onExport,
  isExporting,
  onRefresh,
  isRefreshing = false,
  resultMessage,
  decideError,
}: Props) {
  const [selected, setSelected] = useState<string[]>([]);
  const [reason, setReason] = useState("");
  // Panel detail yang dibuka tombol DETAILS. Ia terikat pada tab dan halamannya: berpindah
  // tab atau halaman menutupnya, tanpa efek yang mengosongkan state.
  const [openedAt, setOpenedAt] = useState({ tab: "", page: 0, no: "" });
  const opened =
    openedAt.tab === tab.kode && openedAt.page === page ? openedAt.no : "";
  const setOpened = (no: string) => setOpenedAt({ tab: tab.kode, page, no });
  const userName = useSession((state) => state.user?.nama ?? "");

  // Pilihan DIBUANG setiap kali antrean atau halamannya berganti, dan setelah keputusan
  // berhasil. Tanpa itu kunci yang dipilih di halaman 1 ikut terkirim saat penyelia menekan
  // tombol di halaman 2 — dan ia tidak akan melihat baris yang ikut terputuskan.
  useEffect(() => {
    setSelected([]);
    setReason("");
  }, [tab.kode, page, resultMessage]);

  const rule = tab.keputusan;
  const approveBlocked = Boolean(rule.alasan_setuju_ditahan);
  const allSelected = rows.length > 0 && selected.length === rows.length;
  const total = pagination?.total ?? rows.length;

  /*
    Kata tombol diambil dari Pega bila Pega punya katanya, dan hanya jatuh ke kata aplikasi
    bila tidak — dua antrean memang tidak punya tombol bernama di sana.
  */
  const approveLabel = rule.label_setujui ?? "Setujui";
  const rejectLabel = rule.label_tolak ?? "Tolak";
  const reasonMissing = rule.alasan_wajib_saat_menolak && reason.trim() === "";

  const columns: Column<QueueRow>[] = [
    ...(rule.dapat_diputuskan
      ? [
          {
            key: "pilih",
            title: "Pilih",
            width: "3.5rem",
            noSort: true,
            value: (row: QueueRow) =>
              selected.includes(row.kunci) ? "dipilih" : "",
            render: (row: QueueRow) => (
              <input
                type="checkbox"
                aria-label={`Pilih baris ${row.kunci}`}
                checked={selected.includes(row.kunci)}
                onChange={() => toggle(row.kunci)}
                className="size-4 cursor-pointer rounded border-slate-300 accent-blue-600"
              />
            ),
          },
        ]
      : []),
    ...(tab.kolom ?? []).map((column) => ({
      key: column.kunci,
      title: column.judul,
      value: (row: QueueRow) => row.sel[column.kunci] ?? "",
    })),
    // Tombol DETAILS per baris — `Akseptasi_PaymentLeader1`; keputusannya di panel detail.
    ...(tab.punya_detail_akseptasi
      ? [
          {
            key: "details",
            title: "",
            width: "6rem",
            noSort: true,
            value: () => "",
            render: (row: QueueRow) => (
              <Button
                type="button"
                tone={opened === row.kunci ? "utama" : "kedua"}
                aria-pressed={opened === row.kunci}
                aria-label={`DETAILS ${row.kunci}`}
                onClick={() => setOpened(row.kunci)}
              >
                DETAILS
              </Button>
            ),
          },
        ]
      : []),
  ];

  function toggle(key: string) {
    setSelected((current) => {
      if (current.includes(key)) return current.filter((item) => item !== key);

      // Pada antrean tanpa jalur massal, memilih baris lain MENGGANTI pilihan sebelumnya —
      // menumpuknya hanya menghasilkan permintaan yang ditolak server.
      return rule.dapat_massal ? [...current, key] : [key];
    });
  }

  function toggleAll() {
    setSelected(allSelected ? [] : rows.map((row) => row.kunci));
  }

  return (
    <div className="space-y-3">
      {/*
        Bilah keputusan muncul HANYA saat ada baris dipilih, tepat di atas tabelnya — bentuk
        rancangan layar. Tidak ada tombol yang tampak dapat ditekan tanpa pilihan.
      */}
      {rule.dapat_diputuskan && selected.length > 0 ? (
        <div className="space-y-2 rounded-kartu border border-blue-200 bg-blue-50/70 px-5 py-3">
          <div className="flex flex-wrap items-end gap-3">
            <p
              className="self-center text-sm font-medium text-blue-900"
              aria-live="polite"
            >
              {rule.dapat_massal
                ? `${selected.length} baris dipilih`
                : "1 baris dipilih — antrean ini diputuskan satu baris setiap kali."}
            </p>

            {/*
              Isian alasan hanya ada bila tabelnya PUNYA kolom alasan. Pada antrean yang
              tidak punya, ia tidak digambar sama sekali.
            */}
            {rule.label_alasan ? (
              <label className="min-w-[16rem] flex-1 text-sm">
                <span className="mb-1 block text-xs font-medium text-slate-700">
                  {rule.label_alasan}
                  {rule.alasan_wajib_saat_menolak ? (
                    <span className="font-normal text-slate-500">
                      {" "}
                      — wajib bila menolak
                    </span>
                  ) : null}
                </span>
                <input
                  type="text"
                  value={reason}
                  onChange={(event) => setReason(event.target.value)}
                  className={`w-full ${inputClass}`}
                />
              </label>
            ) : null}

            <div className="flex items-center gap-2">
              <Button
                type="button"
                disabled={isDeciding || reasonMissing}
                onClick={() => onDecide("tolak", selected, reason)}
              >
                {rejectLabel}
              </Button>
              <Button
                type="button"
                tone="utama"
                disabled={isDeciding || approveBlocked}
                title={rule.alasan_setuju_ditahan}
                onClick={() => onDecide("setujui", selected, reason)}
              >
                {approveLabel}
              </Button>
            </div>
          </div>

          {/* Teks dari rancangan layar Work Owner 2026-10-14. */}
          {userName ? (
            <p className="text-xs text-slate-600">
              Keputusan dikirim atas nama{" "}
              <span className="font-medium text-slate-800">{userName}</span> dan
              tidak dapat dibatalkan dari layar ini.
            </p>
          ) : null}
        </div>
      ) : null}

      {/*
        Alasan penahanan ditampilkan DI TEMPAT tombolnya bekerja, bukan disembunyikan di
        tooltip saja — tombol nonaktif tanpa keterangan akan dilaporkan sebagai kerusakan.
      */}
      {/* Pada antrean ber-DETAILS alasannya ditampilkan di form, tempat status dipilih. */}
      {approveBlocked && !tab.punya_detail_akseptasi ? (
        <p className="rounded-kontrol border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
          <span className="font-medium">Persetujuan sedang ditahan. </span>
          {rule.alasan_setuju_ditahan}
        </p>
      ) : null}

      {decideError ? (
        <ErrorMessage
          tone="penolakan"
          title="Keputusan tidak dapat disimpan"
          description={decideError}
        />
      ) : null}

      {resultMessage ? (
        <p className="text-sm text-slate-700" role="status">
          {resultMessage}
        </p>
      ) : null}

      {/*
        Sub-tab, pada tab yang di Pega memang punya: `Sec_PaymentAkseptasiKlaimCase1` memuat
        dua, tetapi yang kedua bersyarat `1==2` dan tidak pernah tampil.
      */}
      {tab.judul_sub_tab ? (
        <div
          className="border-b border-slate-200"
          role="tablist"
          aria-label={tab.nama}
        >
          <span
            role="tab"
            aria-selected
            className="inline-block border-b-2 border-blue-500 px-1 pb-2 text-sm font-semibold text-slate-900"
          >
            {tab.judul_sub_tab}
          </span>
        </div>
      ) : null}

      <DataTable<QueueRow>
        title={tab.nama}
        description={tab.keterangan}
        label={tab.nama}
        actions={
          <>
            <Chip tone={total > 0 ? "netral" : "redup"}>{total} pengajuan</Chip>
            {/*
              "Pilih semua" HANYA pada antrean yang punya `Select All` / `Deselect All` di
              Pega. Pada enam antrean lain keputusannya memang satu baris setiap kali.
            */}
            {rule.dapat_diputuskan && rule.dapat_massal ? (
              <label className="flex items-center gap-2 text-sm text-slate-700">
                <input
                  type="checkbox"
                  aria-label="Pilih semua baris di halaman ini"
                  checked={allSelected}
                  onChange={toggleAll}
                  disabled={rows.length === 0}
                  className="size-4 cursor-pointer rounded border-slate-300 accent-blue-600"
                />
                Pilih semua di halaman ini
              </label>
            ) : null}
            {tab.punya_detail_akseptasi && onRefresh ? (
              <Button type="button" onClick={onRefresh} disabled={isRefreshing}>
                <ReloadIcon className="h-4 w-4" />
                Refresh
              </Button>
            ) : null}
            <Button type="button" onClick={onExport} disabled={isExporting}>
              <DownloadIcon className="h-4 w-4" />
              {isExporting ? "Menyiapkan…" : "Export"}
            </Button>
          </>
        }
        columns={columns}
        rows={rows}
        rowKey={(row) => row.kunci}
        isLoading={isLoading}
        hideSearch
        emptyMessage="Tidak ada pengajuan yang menunggu persetujuan di antrean ini."
        // Kepala kolom tetap digambar pada antrean kosong: antrean yang kosong hari ini tetap
        // perlu menunjukkan data apa yang akan muncul di sana.
        showHeaderWhenEmpty
        {...(pagination
          ? {
              pagination: {
                page: pagination.halaman,
                size: pagination.ukuran,
                total: pagination.total,
                totalPage: pagination.total_halaman,
                onPageChange,
                isLoading,
              },
            }
          : {})}
      />

      {tab.punya_detail_akseptasi && opened !== "" ? (
        <PaymentDetailPanel
          key={opened}
          acceptedNo={opened}
          rule={rule}
          // Selain "Sedang Proses Komite", nomornya keluar dari antrean — panelnya ditutup.
          onSaved={(status) => {
            if (status !== PAYMENT_IN_COMMITTEE) setOpened("");
          }}
        />
      ) : null}
    </div>
  );
}

/** Kode "Sedang Proses Komite" — barisnya tetap menunggu setelah SIMPAN. */
const PAYMENT_IN_COMMITTEE = "0";
