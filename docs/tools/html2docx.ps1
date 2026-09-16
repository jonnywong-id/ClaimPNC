param(
    [Parameter(Mandatory=$true)][string]$HtmlPath,
    [Parameter(Mandatory=$true)][string]$DocxPath,
    [string]$Title   = 'Migrasi Aplikasi Claim PNC ke Golang',
    [string]$Subject = 'Dokumen proyek migrasi Claim PNC',
    [string]$Company = 'PT Asuransi Sinar Mas'
)

$ErrorActionPreference = 'Stop'

if (Test-Path $DocxPath) { Remove-Item $DocxPath -Force }

$word = New-Object -ComObject Word.Application
$word.Visible = $false
$word.DisplayAlerts = 0

try {
    $doc = $word.Documents.Open($HtmlPath, $false, $true, $false)


    # ---- normalisasi tabel -------------------------------------------------
    # Word mengimpor tabel HTML dengan lebar kolom TETAP dan autofit MATI, sehingga
    # tabel yang lebih lebar dari area cetak terpotong. Terukur pada ADR.docx sebelum
    # perbaikan: 11 dari 64 tabel melebihi 415pt, terparah 896pt.
    #
    # wdAutoFitWindow (2) memaksa setiap tabel muat di lebar halaman. Baris judul
    # diulang di tiap halaman supaya tabel panjang (mis. inventaris 74 harness) tetap
    # terbaca, dan baris boleh terpenggal antar halaman supaya tidak ada isi yang hilang.
    try {
        $ps0    = $doc.Sections.Item(1).PageSetup
        $usable = $ps0.PageWidth - $ps0.LeftMargin - $ps0.RightMargin
        $n = 0; $kecilkan = 0
        foreach ($t in $doc.Tables) {
            $n++
            # Lebar yang menyesatkan datang dari SEL, bukan dari tabel: Word mengimpor
            # tiap sel dengan lebar tetap sendiri, dan autofit tingkat tabel tidak
            # mengaturnya ulang. Buktinya sebuah tabel 2 kolom berisi kata 'Accepted'
            # memakai kolom selebar 946pt. Jadi lebar sel dinolkan dulu ke Auto.
            try { $t.Range.Cells.PreferredWidthType = 1 } catch {}   # wdPreferredWidthAuto
            $t.AllowAutoFit       = $true
            $t.AutoFitBehavior(1) | Out-Null   # wdAutoFitContent - buang lebar impor
            $t.AutoFitBehavior(2) | Out-Null   # wdAutoFitWindow  - kunci ke lebar halaman
            $t.PreferredWidthType = 2      # wdPreferredWidthPercent
            $t.PreferredWidth     = 100
            try { $t.Rows.AllowBreakAcrossPages = $true } catch {}
            try { $t.Rows.Item(1).HeadingFormat = $true } catch {}
            # tabel berkolom banyak tetap sesak walau sudah autofit - kecilkan hurufnya
            if ($t.Columns.Count -ge 6) { $t.Range.Font.Size = 8;   $kecilkan++ }
            elseif ($t.Columns.Count -eq 5) { $t.Range.Font.Size = 8.5; $kecilkan++ }
        }
        Write-Output ("  tabel dinormalisasi: " + $n + " (huruf dikecilkan pada " + $kecilkan + " tabel berkolom banyak)")
    } catch { Write-Output ("  (normalisasi tabel dilewati: " + $_.Exception.Message + ")") }

    # properti dokumen.
    # BuiltInDocumentProperties adalah koleksi COM late-bound: pemanggilan .Item(...) langsung
    # dari PowerShell gagal ("Object reference not set"), jadi dipakai InvokeMember.
    try {
        $props = $doc.BuiltInDocumentProperties
        $isi = @{ 'Title' = $Title; 'Subject' = $Subject; 'Company' = $Company }
        foreach ($k in $isi.Keys) {
            $p = [System.__ComObject].InvokeMember('Item', 'GetProperty', $null, $props, @($k))
            [System.__ComObject].InvokeMember('Value', 'SetProperty', $null, $p, @($isi[$k]))
        }
        Write-Output '  properti dokumen diisi'
    } catch { Write-Output ('  (properti dokumen dilewati: ' + $_.Exception.Message + ')') }

    # nomor halaman di footer
    try {
        foreach ($sec in $doc.Sections) {
            $footer = $sec.Footers.Item(1)   # wdHeaderFooterPrimary
            $rng = $footer.Range
            $rng.Text = ''
            $rng.ParagraphFormat.Alignment = 1   # center
            $rng.Font.Size = 8
            $rng.Font.Name = 'Segoe UI'
            $rng.Fields.Add($rng, 33) | Out-Null   # wdFieldPage
        }
    } catch { Write-Output ('  (footer dilewati: ' + $_.Exception.Message + ')') }

    $saved = $false
    foreach ($fmt in 16, 12) {
        try {
            $doc.SaveAs([ref]$DocxPath, [ref]$fmt)
            $saved = $true
            Write-Output ('  disimpan dengan format ' + $fmt)
            break
        } catch {
            Write-Output ('  format ' + $fmt + ' gagal: ' + $_.Exception.Message)
        }
    }
    if (-not $saved) { throw 'Tidak dapat menyimpan sebagai .docx' }

    try { $doc.Close([ref]$false) } catch { Write-Output ('  (Close: ' + $_.Exception.Message + ')') }
} finally {
    # Word kadang gagal menutup dirinya sendiri (RPC failed / 0x800706BE) SETELAH berkas
    # tersimpan utuh. Kegagalan itu tidak boleh menggagalkan skrip — hasilnya diperiksa di bawah.
    try { $word.Quit() } catch { Write-Output ('  (Quit: ' + $_.Exception.Message + ' - berkas tetap diperiksa)') }
    try { [System.Runtime.InteropServices.Marshal]::ReleaseComObject($word) | Out-Null } catch { }
    Get-Process WINWORD -ErrorAction SilentlyContinue | Where-Object { $_.MainWindowTitle -eq '' } | Stop-Process -Force -ErrorAction SilentlyContinue
}

if (Test-Path $DocxPath) {
    $f = Get-Item $DocxPath
    Write-Output ('OK: ' + $f.FullName + '  (' + [math]::Round($f.Length/1KB,1) + ' KB)')
} else {
    throw 'Berkas .docx tidak terbentuk'
}
