import { useState } from "react";

import type { AutoClaimTab, AutoClaimUploadResponse } from "@/api/types";
import { Button } from "@/components/Button";
import { ErrorMessage } from "@/components/ErrorMessage";
import { ReloadIcon, UploadIcon } from "@/components/Icon";
import { useSelectedPortal } from "@/app/portal";

import { useAutoClaimTabList, useRefreshAutoClaim } from "./api";
import { CompanyBrowser } from "./CompanyBrowser";
import { PremiumCheckPanel } from "./PremiumCheckPanel";
import { PANEL_ID, SourceTab, tabId } from "./SourceTab";
import { UploadForm } from "./UploadForm";

/**
 * Layar Inbox Auto Claim.
 *
 * Pengganti `InboxAutoClaim/InboxAutoClaim-Harness.xml`. Susunannya dari atas:
 *
 *   - kepala halaman: judul, Refresh, dan Upload Data Klaim;
 *   - satu kartu berisi empat tab (Asuransi Kredit · ANEKA · Travel · Cek Premi) dan panel
 *     tab aktif: daftar perusahaan di kiri, grid batch perusahaan terpilih di kanan
 *     (`CompanyBrowser`).
 *
 * # Letak tombol (`D-82`, 2026-10-14 — mensupersede permintaan Work Owner 2026-09-29)
 *
 *   - Upload Data Klaim NAIK ke kepala halaman, sebaris dengan Refresh. Berkasnya tetap
 *     masuk ke tab yang sedang terbuka; di tab Cek Premi tombolnya tidak ada. Di layar
 *     sempit ia menjadi tombol melayang di pojok kanan bawah.
 *   - Teks "Portal entitas · Tab" dibuang: portal sudah tampil di bilah atas, tab sudah
 *     ditandai.
 *   - Proses Klaim dan Generate DLA tetap per BARIS batch (permintaan 2026-09-29). Mesin
 *     keduanya belum dibangun, sehingga keduanya nonaktif dengan alasan di `title`.
 *
 * # Tab keempat: Cek Premi
 *
 * Tab Cek Premi layar lama (`InboxAutoClaim-Harness.xml` :50640) tidak membaca tabel batch
 * apa pun, sehingga ia tidak datang dari daftar tab server (yang memetakan tab ke tabel).
 * Ia ditambahkan di sini dan menampilkan `PremiumCheckPanel` — tanpa tombol Upload.
 */

/** Kode tab Cek Premi. Tidak bertabrakan dengan kode tab server (aneka, kredit, travel). */
const PREMIUM_TAB = "cek-premi";

const premiumTab: AutoClaimTab = {
  kode: PREMIUM_TAB,
  label: "Cek Premi",
  tabel: "",
};
export function AutoClaimInboxPage() {
  const portal = useSelectedPortal((state) => state.alias);

  // Tab yang sedang terbuka. Bawaannya datang dari server supaya layar tidak memuat
  // daftar tab-nya sendiri.
  const [source, setSource] = useState("");
  const [uploadOpen, setUploadOpen] = useState(false);
  const [uploadNote, setUploadNote] = useState<AutoClaimUploadResponse | null>(
    null,
  );

  const tabList = useAutoClaimTabList();
  const activeSource = source === "" ? (tabList.data?.bawaan ?? "") : source;
  const allTab = tabList.data ? [...tabList.data.tab, premiumTab] : [];
  const activeTab = allTab.find((t) => t.kode === activeSource);
  const premiumActive = activeSource === PREMIUM_TAB;
  const activeLabel = activeTab?.label ?? "";

  const reload = useRefreshAutoClaim();

  return (
    <main className="halaman flex flex-col gap-3 px-4 py-6 max-sm:pb-24 sm:py-8">
      <header className="flex flex-wrap items-center gap-3 rounded-kartu border border-slate-200 bg-white px-4 py-3.5 shadow-lembut sm:px-5">
        <div className="min-w-0 flex-1">
          <h1 className="text-xl font-semibold text-slate-900">
            Inbox Auto Claim
          </h1>
          <p className="mt-0.5 text-sm text-slate-600">
            Batch klaim borongan dari perusahaan rekanan beserta hasil
            pemrosesannya.
          </p>
        </div>
        <Button
          tone="kedua"
          onClick={reload.refresh}
          disabled={reload.isFetching}
        >
          <ReloadIcon className="h-4 w-4" />
          {reload.isFetching ? "Memuat…" : "Refresh"}
        </Button>
        {/* Satu tombol untuk kedua lebar layar, bukan dua tombol yang saling
            disembunyikan: di layar sempit kelasnya sendiri yang memindahkannya ke pojok
            kanan bawah. Dua tombol berlabel sama membuat pembaca layar menyebutnya dua
            kali. Tombolnya tidak ada di tab Cek Premi — tab itu tidak menerima berkas. */}
        {activeSource !== "" && !premiumActive && (
          <Button
            tone="utama"
            onClick={() => setUploadOpen(true)}
            disabled={uploadOpen || portal === null}
            className="max-sm:fixed max-sm:right-4 max-sm:bottom-4 max-sm:z-40 max-sm:!rounded-full max-sm:!px-5 max-sm:!py-3 max-sm:shadow-terbang"
          >
            <UploadIcon className="h-4 w-4" />
            Upload Data Klaim
          </Button>
        )}
      </header>

      <section className="rounded-kartu border border-slate-200 bg-white shadow-lembut">
        {/* Tab duduk di atas segalanya: ia memilih TABEL yang dibaca seluruh layar, bukan
            menyaring isi tabel yang sama. */}
        <SourceTab tab={allTab} selected={activeSource} onSelect={setSource} />

        {activeSource !== "" && (
          <div
            role="tabpanel"
            id={PANEL_ID}
            aria-labelledby={tabId(activeSource)}
            className="p-3 sm:p-4"
          >
            {!premiumActive && (
              <UploadResult
                note={uploadNote}
                onClose={() => setUploadNote(null)}
              />
            )}

            {portal === null ? (
              <ErrorMessage
                title="Portal entitas belum dipilih"
                description="Data klaim dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
                tone="penolakan"
              />
            ) : premiumActive ? (
              <PremiumCheckPanel />
            ) : (
              // `key` pada tab membuat berpindah tab MEMASANG ULANG isinya: kata kunci,
              // perusahaan terpilih, dan halaman ikut hilang. Bukan kerapian — ketiga tab
              // membaca tabel berbeda, dan kode perusahaan di satu tab belum tentu ada di
              // tab lain.
              <CompanyBrowser key={activeSource} source={activeSource} />
            )}
          </div>
        )}
      </section>

      {uploadOpen && (
        <UploadForm
          source={activeSource}
          sourceLabel={activeLabel}
          onClose={() => setUploadOpen(false)}
          onUploaded={(result) => {
            setUploadNote(result);
            setUploadOpen(false);
          }}
        />
      )}
    </main>
  );
}

/**
 * Panel hasil unggahan, di dalam panel tab.
 *
 * Ia membedakan TIGA keadaan, dan pembedaannya bukan kerapian tampilan: yang "bertanda"
 * tersimpan dan terlihat di grid, yang "ditolak" tidak ada di mana pun. Pengguna yang
 * mengira keduanya sama akan mencari baris yang tidak pernah tersimpan.
 */
function UploadResult({
  note,
  onClose,
}: {
  note: AutoClaimUploadResponse | null;
  onClose: () => void;
}) {
  if (note === null) return null;

  return (
    <div
      role="status"
      className={
        note.ditolak.length > 0
          ? "mb-4 rounded-kartu border border-amber-200 bg-amber-50/80 p-4 text-sm shadow-lembut"
          : "mb-4 rounded-kartu border border-emerald-200 bg-emerald-50/80 p-4 text-sm shadow-lembut"
      }
    >
      <p className="font-medium text-slate-900">
        {note.jumlah_baris === 0
          ? "Tidak ada baris yang tersimpan"
          : `${note.jumlah_baris} baris tersimpan`}
      </p>

      {note.batch.length > 0 && (
        <ul className="mt-1 space-y-0.5 text-slate-800">
          {note.batch.map((b) => (
            <li key={`${b.kode_perusahaan}-${b.batch}`}>
              {b.nama_perusahaan === "" ? b.kode_perusahaan : b.nama_perusahaan}{" "}
              — batch <span className="font-medium">{b.batch}</span>,{" "}
              {b.jumlah_baris} baris
              {b.jumlah_bertanda > 0 && (
                <span className="text-amber-800">
                  {" "}
                  ({b.jumlah_lolos} menunggu proses, {b.jumlah_bertanda}{" "}
                  bertanda gagal)
                </span>
              )}
            </li>
          ))}
        </ul>
      )}

      {note.ditolak.length > 0 && (
        <div className="mt-3 border-t border-amber-200 pt-3">
          <p className="font-medium text-amber-900">
            {note.ditolak.length} baris TIDAK tersimpan
          </p>
          <p className="mt-0.5 text-xs text-amber-800">
            Perusahaan rekanannya tidak dapat ditemukan dari nomor polis,
            sehingga barisnya tidak punya tempat di grid mana pun. Perbaiki
            nomor polisnya lalu unggah ulang baris ini saja.
          </p>
          <ul className="mt-2 space-y-0.5 text-xs text-amber-900">
            {note.ditolak.map((row) => (
              <li key={`${row.baris}-${row.nomor_polis}`}>
                Baris{" "}
                <span className="font-medium tabular-nums">{row.baris}</span> —{" "}
                {row.nomor_polis === ""
                  ? "(nomor polis kosong)"
                  : row.nomor_polis}{" "}
                — {row.pesan}
              </li>
            ))}
          </ul>
        </div>
      )}

      {note.batch.length > 0 && (
        <p className="mt-2 text-xs text-slate-700">
          Baris yang menunggu proses belum menjadi klaim sampai batch-nya
          diproses.
        </p>
      )}

      <div className="mt-3">
        <Button tone="halus" onClick={onClose}>
          Tutup
        </Button>
      </div>
    </div>
  );
}
