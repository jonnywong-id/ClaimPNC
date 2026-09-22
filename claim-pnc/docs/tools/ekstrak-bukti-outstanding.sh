#!/usr/bin/env bash
#
# Mengumpulkan seluruh bukti Pega untuk modul Inbox Outstanding ke satu berkas.
#
# KENAPA SCRIPT INI ADA
# ---------------------
# Artefak Inbox Outstanding tersebar di enam berkas XML berukuran total ~1,8 MB.
# Menggalinya ulang setiap kali percakapan dimulai memakan waktu dan — lebih buruk —
# membuka peluang salah baca yang berbeda-beda tiap kali. Script ini menghasilkan satu
# berkas teks yang dapat dibaca utuh, sehingga setiap klaim di dokumen perancangan
# dapat diperiksa terhadap sumbernya tanpa menebak.
#
# YANG TIDAK DILAKUKAN SCRIPT INI
# -------------------------------
# Ia TIDAK menafsirkan. Ia hanya memotong dan menempelkan apa adanya beserta nomor
# barisnya. Seluruh tafsir hidup di dokumen perancangan, bukan di sini.
#
# CARA MENJALANKAN
#   bash ekstrak-bukti-outstanding.sh
#   bash ekstrak-bukti-outstanding.sh /path/ke/keluaran.txt
#
# Dijalankan dari akar folder export Pega (tempat CLAUDE.md berada).

set -euo pipefail

KELUARAN="${1:-bukti-inbox-outstanding.txt}"

# Dijalankan dari mana pun, selalu bekerja terhadap folder script ini berada.
cd "$(dirname "$0")"

if [[ ! -d "Harness" || ! -d "RDB List" ]]; then
  echo "GAGAL: jalankan dari akar folder export Pega (yang memuat Harness/ dan 'RDB List/')." >&2
  exit 1
fi

# Menulis judul bagian supaya keluarannya dapat dinavigasi dengan pencarian.
bagian() {
  printf '\n\n%s\n== %s\n%s\n\n' \
    "================================================================================" \
    "$1" \
    "================================================================================"
}

# Menulis sub-judul.
#
# Lewat `printf '%s\n'`, bukan `printf '--- ... ---\n'`: printf menafsirkan argumen
# pertama yang diawali tanda hubung sebagai OPSI, dan menolak jalan. Bentuk ini membuat
# teks apa pun aman, termasuk yang diawali tanda hubung.
sub() {
  printf '%s\n' "--- $1 ---"
}

# Memotong satu elemen XML beserta isinya, dengan nomor baris dipertahankan.
# Nomor baris penting: dokumen perancangan merujuk dengan `berkas:baris`.
#
# Dipakai awk, bukan sed: tag penutup XML memuat garis miring (`</pyBrowseSQL>`) yang
# menutup pola sed lebih awal dan menghasilkan galat parse. awk memakai pencocokan
# teks biasa lewat index(), sehingga tidak ada karakter yang perlu di-escape.
potong() {
  local berkas="$1" mulai="$2" akhir="$3"
  awk -v a="$mulai" -v b="$akhir" '
    index($0, a) { di_dalam = 1 }
    di_dalam     { printf "%6d\t%s\n", NR, $0 }
    di_dalam && index($0, b) { exit }
  ' "$berkas"
}

{
  printf 'BUKTI INBOX OUTSTANDING — diekstrak dari export Pega\n'
  printf 'Dibuat: %s\n' "$(date '+%Y-%m-%d %H:%M:%S')"
  printf 'Folder: %s\n' "$(pwd)"
  printf '\nSeluruh isi di bawah adalah SALINAN APA ADANYA dari export, bukan tafsir.\n'

  # ---------------------------------------------------------------- 1. ketersediaan
  bagian "1. KETERSEDIAAN ARTEFAK"
  printf 'Harness InboxOutstanding_Harness (dirujuk menu):\n'
  if ls Harness/ | grep -qi "^InboxOutstanding"; then
    ls -la Harness/ | grep -i "InboxOutstanding"
  else
    printf '  TIDAK ADA di export — inilah sebabnya InboxRegister_Harness dipakai\n'
    printf '  sebagai rujukan BENTUK layar. (K-33)\n'
  fi

  printf '\nArtefak pendukung yang ADA:\n'
  for f in \
    "RDB List/BrowseInboxOutstanding1-SQL.xml" \
    "Activity/InboxOutstanding_Act-Act.xml" \
    "Activity/ExportOutstanding_Act-Act.xml" \
    "Activity/GCNMTransferDataKlaim_act-Act.xml" \
    "Section/InboxOutstandingClaim_Section-Section.xml" \
    "Section/InboxOutstandingperCabang_Section-Section.xml"
  do
    if [[ -f "$f" ]]; then
      printf '  [ADA]   %-58s %s byte\n' "$f" "$(wc -c < "$f" | tr -d ' ')"
    else
      printf '  [HILANG] %s\n' "$f"
    fi
  done

  # ---------------------------------------------------------------- 2. SQL inti
  bagian "2. KUERI INTI — RDB List/BrowseInboxOutstanding1-SQL.xml"
  printf 'Nomor baris di kiri. Inilah satu-satunya definisi "Outstanding" yang mengikat.\n\n'
  potong "RDB List/BrowseInboxOutstanding1-SQL.xml" '<pyBrowseSQL>' '</pyBrowseSQL>'

  # ---------------------------------------------------------------- 3. batas data
  bagian "3. BATAS DATA PER JABATAN — Activity/InboxOutstanding_Act-Act.xml"
  printf 'Potongan WHERE yang disisipkan ke kueri lewat {ASIS:TempView.pyNote}.\n'
  printf 'Baca berpasangan: precondition menentukan potongan mana yang dipakai.\n\n'
  sub 'precondition (jabatan pemanggil)'
  grep -n 'pyStepsPreCondParamsWhen>OperatorID.pyPosition' \
    "Activity/InboxOutstanding_Act-Act.xml" || true
  printf '\n--- potongan WHERE yang dipasang ---\n'
  grep -n '<PropertiesValue>"AND' "Activity/InboxOutstanding_Act-Act.xml" || true
  printf '\n--- urutan langkah beserta keterangannya ---\n'
  grep -n '<pyStepsDescription>\|<pyStepsActivityName>' \
    "Activity/InboxOutstanding_Act-Act.xml" \
    | sed 's/<[^>]*>//g' | head -40 || true

  # ---------------------------------------------------------------- 4. derivasi status
  bagian "4. DERIVASI NILAI TAMPILAN — Activity/InboxOutstanding_Act-Act.xml"
  printf 'Nilai yang DIHITUNG activity setelah kueri, bukan diambil dari kolom.\n\n'
  grep -n '<pyExpression>' "Activity/InboxOutstanding_Act-Act.xml" \
    | sed 's/<[^>]*>//g' | head -30 || true
  printf '\n--- nilai yang di-set ---\n'
  grep -n '<PropertiesValue>' "Activity/InboxOutstanding_Act-Act.xml" \
    | sed 's/<[^>]*>//g' | sort -u -t: -k2 | head -20 || true

  # ---------------------------------------------------------------- 5. kolom grid
  bagian "5. CAPTION — Section/InboxOutstandingClaim_Section-Section.xml"
  printf 'SELURUH pyCaption beserta nomor barisnya, tanpa disaring.\n\n'
  printf 'PERINGATAN: daftar ini BUKAN semuanya kolom grid. Ia mencampur tiga hal —\n'
  printf '  (a) judul kolom grid        : No Klaim, No Polis, PIC Teknik, Admin PNC, ...\n'
  printf '  (b) isian form Transfer     : User ID Lama, User ID Baru, Type User, ...\n'
  printf '  (c) judul layar             : Inbox Outstanding\n'
  printf 'Memilahnya menuntut pembacaan section, bukan penyaringan kata kunci. Menyaring\n'
  printf 'dengan kata kunci JUSTRU MELEWATKAN kolom — "Report Date", "PIC Teknik", dan\n'
  printf '"Admin PNC" terlewat dengan cara itu pada sesi 2026-09-19.\n\n'
  grep -n 'pyRuleName>pyCaption ' "Section/InboxOutstandingClaim_Section-Section.xml" \
    | sed 's/<[^>]*>//g' | sed 's/pyCaption //' | sort -u -t: -k2 || true

  # ---------------------------------------------------------------- 6. aksi
  bagian "6. AKSI DI LAYAR — Section/InboxOutstandingClaim_Section-Section.xml"
  printf 'Tombol beserta nomor barisnya.\n\n'
  grep -n '<pyLabel>Transfer<\|<pyLabel>Change New User<\|<pyLabel>Export Excel<\|<pyLabel>Search Data<\|<pyLabel>Clear Filter<\|<pyLabel>Select All<' \
    "Section/InboxOutstandingClaim_Section-Section.xml" | sed 's/<[^>]*>//g' || true

  printf '\n--- activity yang dipanggil tombol ---\n'
  grep -n '<pyActivity>GCNMTransferDataKlaim_act<\|<pyActivity>ExportOutstanding_Act<' \
    "Section/InboxOutstandingClaim_Section-Section.xml" | sed 's/<[^>]*>//g' || true

  # ---------------------------------------------------------------- 7. transfer
  bagian "7. APA YANG DILAKUKAN TRANSFER — Activity/GCNMTransferDataKlaim_act-Act.xml"
  printf 'Dipanggil tombol "Change New User". Langkah-langkahnya:\n\n'
  grep -n '<pyStepsDescription>\|<pyStepsActivityName>' \
    "Activity/GCNMTransferDataKlaim_act-Act.xml" \
    | sed 's/<[^>]*>//g' | head -40 || true

  # ---------------------------------------------------------------- 8. export
  bagian "8. FORMAT EXPORT — Activity/ExportOutstanding_Act-Act.xml"
  printf 'Tombolnya berbunyi "Export Excel"; yang dipanggil sebenarnya:\n\n'
  grep -c 'pxConvertResultsToCSV' "Activity/ExportOutstanding_Act-Act.xml" \
    | sed 's/^/  pxConvertResultsToCSV dipanggil: /' || true
  printf '  -> keluarannya CSV, bukan XLSX. Padanan Go cukup encoding/csv (pustaka standar).\n'

  # ---------------------------------------------------------------- 9. pemanggil
  bagian "9. SIAPA MEMANGGIL APA"
  printf 'Pemakai Section/InboxOutstandingClaim_Section:\n'
  grep -rl "InboxOutstandingClaim_Section" --include="*.xml" . | sed 's|^\./|  |' || true
  printf '\nPemakai Activity/GCNMTransferDataKlaim_act:\n'
  grep -rl "GCNMTransferDataKlaim" --include="*.xml" . | sed 's|^\./|  |' || true
  printf '\nApakah ada di InboxRegister_Harness? '
  if grep -q "GCNMTransferDataKlaim" "Harness/InboxRegister_Harness-Harness.xml" 2>/dev/null; then
    printf 'YA\n'
  else
    printf 'TIDAK — nol kemunculan.\n'
  fi

  # ---------------------------------------------------------------- 10. rujukan bentuk
  bagian "10. RUJUKAN BENTUK LAYAR — Harness/InboxRegister_Harness-Harness.xml"
  printf 'Section yang dimuat harness ini:\n'
  grep -oE '<pyRuleName>InboxRegister_Section</pyRuleName>' \
    "Harness/InboxRegister_Harness-Harness.xml" | sort -u | sed 's/<[^>]*>//g;s/^/  /' || true
  printf '\nJudul kolom & kendali pada InboxRegister_Section (pembanding):\n'
  grep -n 'pyRuleName>pyCaption ' "Section/InboxRegister_Section-Section.xml" 2>/dev/null \
    | sed 's/<[^>]*>//g' | sed 's/pyCaption //' | sort -u -t: -k2 | head -25 || true

  # ---------------------------------------------------------------- 11. sumber lini
  bagian "11. SUMBER LINI BISNIS — keputusan Work Owner"
  printf 'Keputusan: M_LOGIN_PNC + kolom BARU linebusiness.\n'
  printf 'Alasan Work Owner: MST_USER_TEKNIK hanya berisi PIC Teknik, sedangkan\n'
  printf 'M_LOGIN_PNC akan dipakai untuk karyawan juga (group akses / akses menu).\n\n'
  printf 'Kolom M_LOGIN_PNC hari ini (dari Database/m_login_pnc.csv):\n'
  head -1 Database/m_login_pnc.csv 2>/dev/null | sed 's/^/  /' || printf '  (csv tidak ditemukan)\n'
  printf '  -> LINEBUSINESS BELUM ADA. Menambahkannya menempuh D-63:\n'
  printf '     permintaan tertulis -> persetujuan Work Owner -> dijalankan DBA.\n\n'
  printf 'Nilai yang harus ditampung, dari precondition Pega:\n'
  printf '  NONMBU, BONDING, PA, TRAVEL\n'
  printf 'Catatan: nilai lain terlihat di rule lain (mis. TRAVELOKA) — daftar pastinya\n'
  printf 'BELUM DIPUTUSKAN.\n'

} > "$KELUARAN"

printf 'Selesai. Bukti ditulis ke: %s (%s baris)\n' \
  "$KELUARAN" "$(wc -l < "$KELUARAN" | tr -d ' ')"
