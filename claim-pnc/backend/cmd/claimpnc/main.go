// Command claimpnc adalah satu-satunya titik masuk aplikasi Claim PNC.
//
// Aplikasi ini modular monolith (ADR-0001): satu binary yang memuat seluruh modul,
// dikompilasi tanpa dependensi runtime eksternal dan menyajikan API sekaligus berkas
// statis antarmuka.
//
// Berkas ini sengaja tipis. Tugasnya hanya tiga: membaca konfigurasi, merakit adapter
// di balik setiap seam, dan menyalakan server. **Tidak ada aturan bisnis di sini.**
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
	"claim-pnc/internal/auth/repo/memory"
	"claim-pnc/internal/auth/repo/sqlstore"
	"claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/daftardetaildokumentravel"
	"claim-pnc/internal/daftardetailtipedokumen"
	"claim-pnc/internal/daftarobjekdokumen"
	"claim-pnc/internal/daftartipedokumen"
	"claim-pnc/internal/daftartipedokumenbisnis"
	"claim-pnc/internal/detailpenyebab"
	"claim-pnc/internal/inboxacceptopenprotection"
	"claim-pnc/internal/inboxanalystdoctor"
	"claim-pnc/internal/inboxautoclaim"
	"claim-pnc/internal/inboxclaimtreatynonprop"
	"claim-pnc/internal/inboxclaimtreatyprop"
	"claim-pnc/internal/inboxcloseclaim"
	"claim-pnc/internal/inboxinvestigator"
	"claim-pnc/internal/inboxkomunikasicabang"
	"claim-pnc/internal/inboxlaporanklaim"
	"claim-pnc/internal/inboxmanagerreceivepucl"
	"claim-pnc/internal/inboxoutstanding"
	"claim-pnc/internal/inboxprogressclaim"
	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxreceivetka"
	"claim-pnc/internal/inboxsalvage"
	"claim-pnc/internal/inboxxol"
	"claim-pnc/internal/komite"
	"claim-pnc/internal/masterautoclaim"
	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/mastercolsimasonline"
	"claim-pnc/internal/masterdokumentravel"
	"claim-pnc/internal/masterdominanfactor"
	"claim-pnc/internal/mastergroupingsparepart"
	"claim-pnc/internal/masterkategorisparepart"
	"claim-pnc/internal/laporanhasilai"
	"claim-pnc/internal/masterlogin"
	"claim-pnc/internal/mastermasking"
	"claim-pnc/internal/masterpanel"
	"claim-pnc/internal/masterpasal"
	"claim-pnc/internal/masterpasalai"
	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/masterpenyebabkerugian"
	"claim-pnc/internal/masterpicteknik"
	"claim-pnc/internal/masterreas"
	"claim-pnc/internal/masterrecovery"
	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/masterstatusprogres"
	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/mastersurveyors"
	"claim-pnc/internal/mastertipesparepart"
	"claim-pnc/internal/mastertipesurveyors"
	"claim-pnc/internal/masterxol"
	"claim-pnc/internal/menu"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/db"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/platform/random"
	"claim-pnc/internal/portal"
	"claim-pnc/internal/reportklaim"
	"claim-pnc/internal/reportkpi"
	riwayatklaimhttp "claim-pnc/internal/riwayatklaim/http"
	riwayatklaimmemory "claim-pnc/internal/riwayatklaim/repo/memory"
	riwayatklaimsql "claim-pnc/internal/riwayatklaim/repo/sqlstore"
	riwayatklaimusecase "claim-pnc/internal/riwayatklaim/usecase"
	"claim-pnc/spa"

	"claim-pnc/internal/archivedokumenklaim"
	archivedokumenklaimgateway "claim-pnc/internal/archivedokumenklaim/gateway"
	archivedokumenklaimhttp "claim-pnc/internal/archivedokumenklaim/http"
	archivedokumenklaimmemory "claim-pnc/internal/archivedokumenklaim/repo/memory"
	archivedokumenklaimsql "claim-pnc/internal/archivedokumenklaim/repo/sqlstore"
	archivedokumenklaimusecase "claim-pnc/internal/archivedokumenklaim/usecase"
	authhttp "claim-pnc/internal/auth/http"
	daftardetaildokumentravelhttp "claim-pnc/internal/daftardetaildokumentravel/http"
	daftardetaildokumentravelmemory "claim-pnc/internal/daftardetaildokumentravel/repo/memory"
	daftardetaildokumentravelsql "claim-pnc/internal/daftardetaildokumentravel/repo/sqlstore"
	daftardetaildokumentravelusecase "claim-pnc/internal/daftardetaildokumentravel/usecase"
	daftardetailtipedokumenhttp "claim-pnc/internal/daftardetailtipedokumen/http"
	daftardetailtipedokumenmemory "claim-pnc/internal/daftardetailtipedokumen/repo/memory"
	daftardetailtipedokumensql "claim-pnc/internal/daftardetailtipedokumen/repo/sqlstore"
	daftardetailtipedokumenusecase "claim-pnc/internal/daftardetailtipedokumen/usecase"
	daftarobjekdokumenhttp "claim-pnc/internal/daftarobjekdokumen/http"
	daftarobjekdokumenmemory "claim-pnc/internal/daftarobjekdokumen/repo/memory"
	daftarobjekdokumensql "claim-pnc/internal/daftarobjekdokumen/repo/sqlstore"
	daftarobjekdokumenusecase "claim-pnc/internal/daftarobjekdokumen/usecase"
	daftartipedokumenhttp "claim-pnc/internal/daftartipedokumen/http"
	daftartipedokumenmemory "claim-pnc/internal/daftartipedokumen/repo/memory"
	daftartipedokumensql "claim-pnc/internal/daftartipedokumen/repo/sqlstore"
	daftartipedokumenusecase "claim-pnc/internal/daftartipedokumen/usecase"
	daftartipedokumenbisnishttp "claim-pnc/internal/daftartipedokumenbisnis/http"
	daftartipedokumenbisnismemory "claim-pnc/internal/daftartipedokumenbisnis/repo/memory"
	daftartipedokumenbisnissql "claim-pnc/internal/daftartipedokumenbisnis/repo/sqlstore"
	daftartipedokumenbisnisusecase "claim-pnc/internal/daftartipedokumenbisnis/usecase"
	detailpenyebabhttp "claim-pnc/internal/detailpenyebab/http"
	detailpenyebabmemory "claim-pnc/internal/detailpenyebab/repo/memory"
	detailpenyebabsql "claim-pnc/internal/detailpenyebab/repo/sqlstore"
	detailpenyebabusecase "claim-pnc/internal/detailpenyebab/usecase"
	inboxacceptopenprotectionhttp "claim-pnc/internal/inboxacceptopenprotection/http"
	inboxacceptopenprotectionmemory "claim-pnc/internal/inboxacceptopenprotection/repo/memory"
	inboxacceptopenprotectionsql "claim-pnc/internal/inboxacceptopenprotection/repo/sqlstore"
	inboxacceptopenprotectionusecase "claim-pnc/internal/inboxacceptopenprotection/usecase"
	inboxanalystdoctorhttp "claim-pnc/internal/inboxanalystdoctor/http"
	inboxanalystdoctormemory "claim-pnc/internal/inboxanalystdoctor/repo/memory"
	inboxanalystdoctorsql "claim-pnc/internal/inboxanalystdoctor/repo/sqlstore"
	inboxanalystdoctorusecase "claim-pnc/internal/inboxanalystdoctor/usecase"
	inboxautoclaimhttp "claim-pnc/internal/inboxautoclaim/http"
	inboxautoclaimmemory "claim-pnc/internal/inboxautoclaim/repo/memory"
	inboxautoclaimsql "claim-pnc/internal/inboxautoclaim/repo/sqlstore"
	inboxautoclaimusecase "claim-pnc/internal/inboxautoclaim/usecase"
	inboxclaimtreatynonprophttp "claim-pnc/internal/inboxclaimtreatynonprop/http"
	inboxclaimtreatynonpropmemory "claim-pnc/internal/inboxclaimtreatynonprop/repo/memory"
	inboxclaimtreatynonpropsql "claim-pnc/internal/inboxclaimtreatynonprop/repo/sqlstore"
	inboxclaimtreatynonpropusecase "claim-pnc/internal/inboxclaimtreatynonprop/usecase"
	inboxclaimtreatypropthttp "claim-pnc/internal/inboxclaimtreatyprop/http"
	inboxclaimtreatypropmemory "claim-pnc/internal/inboxclaimtreatyprop/repo/memory"
	inboxclaimtreatypropsql "claim-pnc/internal/inboxclaimtreatyprop/repo/sqlstore"
	inboxclaimtreatypropusecase "claim-pnc/internal/inboxclaimtreatyprop/usecase"
	inboxcloseclaimhttp "claim-pnc/internal/inboxcloseclaim/http"
	inboxcloseclaimmemory "claim-pnc/internal/inboxcloseclaim/repo/memory"
	inboxcloseclaimsql "claim-pnc/internal/inboxcloseclaim/repo/sqlstore"
	inboxcloseclaimusecase "claim-pnc/internal/inboxcloseclaim/usecase"
	inboxinvestigatorhttp "claim-pnc/internal/inboxinvestigator/http"
	inboxinvestigatormemory "claim-pnc/internal/inboxinvestigator/repo/memory"
	inboxinvestigatorsql "claim-pnc/internal/inboxinvestigator/repo/sqlstore"
	inboxinvestigatorusecase "claim-pnc/internal/inboxinvestigator/usecase"
	inboxkomunikasicabanghttp "claim-pnc/internal/inboxkomunikasicabang/http"
	inboxkomunikasicabangmemory "claim-pnc/internal/inboxkomunikasicabang/repo/memory"
	inboxkomunikasicabangsql "claim-pnc/internal/inboxkomunikasicabang/repo/sqlstore"
	inboxkomunikasicabangusecase "claim-pnc/internal/inboxkomunikasicabang/usecase"
	inboxlaporanklaimhttp "claim-pnc/internal/inboxlaporanklaim/http"
	inboxlaporanklaimmemory "claim-pnc/internal/inboxlaporanklaim/repo/memory"
	inboxlaporanklaimsql "claim-pnc/internal/inboxlaporanklaim/repo/sqlstore"
	inboxlaporanklaimusecase "claim-pnc/internal/inboxlaporanklaim/usecase"
	inboxmanagerreceivepuclhttp "claim-pnc/internal/inboxmanagerreceivepucl/http"
	inboxmanagerreceivepuclmemory "claim-pnc/internal/inboxmanagerreceivepucl/repo/memory"
	inboxmanagerreceivepuclsql "claim-pnc/internal/inboxmanagerreceivepucl/repo/sqlstore"
	inboxmanagerreceivepuclusecase "claim-pnc/internal/inboxmanagerreceivepucl/usecase"
	inboxoutstandinghttp "claim-pnc/internal/inboxoutstanding/http"
	inboxoutstandingmemory "claim-pnc/internal/inboxoutstanding/repo/memory"
	inboxoutstandingsql "claim-pnc/internal/inboxoutstanding/repo/sqlstore"
	inboxoutstandingusecase "claim-pnc/internal/inboxoutstanding/usecase"
	inboxprogressclaimhttp "claim-pnc/internal/inboxprogressclaim/http"
	inboxprogressclaimmemory "claim-pnc/internal/inboxprogressclaim/repo/memory"
	inboxprogressclaimsql "claim-pnc/internal/inboxprogressclaim/repo/sqlstore"
	inboxprogressclaimusecase "claim-pnc/internal/inboxprogressclaim/usecase"
	inboxrclpuclhttp "claim-pnc/internal/inboxrclpucl/http"
	inboxrclpuclmemory "claim-pnc/internal/inboxrclpucl/repo/memory"
	inboxrclpuclsql "claim-pnc/internal/inboxrclpucl/repo/sqlstore"
	inboxrclpuclusecase "claim-pnc/internal/inboxrclpucl/usecase"
	inboxreceivetkahttp "claim-pnc/internal/inboxreceivetka/http"
	inboxreceivetkanotif "claim-pnc/internal/inboxreceivetka/notification"
	inboxreceivetkamemory "claim-pnc/internal/inboxreceivetka/repo/memory"
	inboxreceivetkasql "claim-pnc/internal/inboxreceivetka/repo/sqlstore"
	inboxreceivetkausecase "claim-pnc/internal/inboxreceivetka/usecase"
	inboxsalvagehttp "claim-pnc/internal/inboxsalvage/http"
	inboxsalvagememory "claim-pnc/internal/inboxsalvage/repo/memory"
	inboxsalvagesql "claim-pnc/internal/inboxsalvage/repo/sqlstore"
	inboxsalvageusecase "claim-pnc/internal/inboxsalvage/usecase"
	inboxxolhttp "claim-pnc/internal/inboxxol/http"
	inboxxolmemory "claim-pnc/internal/inboxxol/repo/memory"
	inboxxolsql "claim-pnc/internal/inboxxol/repo/sqlstore"
	inboxxolusecase "claim-pnc/internal/inboxxol/usecase"
	"claim-pnc/internal/inputreqprotection"
	inputreqprotectionhttp "claim-pnc/internal/inputreqprotection/http"
	inputreqprotectionmemory "claim-pnc/internal/inputreqprotection/repo/memory"
	inputreqprotectionsql "claim-pnc/internal/inputreqprotection/repo/sqlstore"
	inputreqprotectionusecase "claim-pnc/internal/inputreqprotection/usecase"
	komitehttp "claim-pnc/internal/komite/http"
	komitememory "claim-pnc/internal/komite/repo/memory"
	komitesql "claim-pnc/internal/komite/repo/sqlstore"
	komiteusecase "claim-pnc/internal/komite/usecase"
	masterautoclaimhttp "claim-pnc/internal/masterautoclaim/http"
	masterautoclaimmemory "claim-pnc/internal/masterautoclaim/repo/memory"
	masterautoclaimsql "claim-pnc/internal/masterautoclaim/repo/sqlstore"
	masterautoclaimusecase "claim-pnc/internal/masterautoclaim/usecase"
	masterbengkelhttp "claim-pnc/internal/masterbengkel/http"
	masterbengkelmemory "claim-pnc/internal/masterbengkel/repo/memory"
	masterbengkelsql "claim-pnc/internal/masterbengkel/repo/sqlstore"
	masterbengkelusecase "claim-pnc/internal/masterbengkel/usecase"
	mastercolhttp "claim-pnc/internal/mastercolsimasonline/http"
	mastercolmemory "claim-pnc/internal/mastercolsimasonline/repo/memory"
	mastercolsql "claim-pnc/internal/mastercolsimasonline/repo/sqlstore"
	mastercolusecase "claim-pnc/internal/mastercolsimasonline/usecase"
	masterdokumentravelhttp "claim-pnc/internal/masterdokumentravel/http"
	masterdokumentravelmemory "claim-pnc/internal/masterdokumentravel/repo/memory"
	masterdokumentravelsql "claim-pnc/internal/masterdokumentravel/repo/sqlstore"
	masterdokumentravelusecase "claim-pnc/internal/masterdokumentravel/usecase"
	masterdominanfactorhttp "claim-pnc/internal/masterdominanfactor/http"
	masterdominanfactormemory "claim-pnc/internal/masterdominanfactor/repo/memory"
	masterdominanfactorsql "claim-pnc/internal/masterdominanfactor/repo/sqlstore"
	masterdominanfactorusecase "claim-pnc/internal/masterdominanfactor/usecase"
	mastergroupingspareparthttp "claim-pnc/internal/mastergroupingsparepart/http"
	mastergroupingsparepartmemory "claim-pnc/internal/mastergroupingsparepart/repo/memory"
	mastergroupingsparepartsql "claim-pnc/internal/mastergroupingsparepart/repo/sqlstore"
	mastergroupingsparepartusecase "claim-pnc/internal/mastergroupingsparepart/usecase"
	masterkategorispareparthttp "claim-pnc/internal/masterkategorisparepart/http"
	masterkategorisparepartmemory "claim-pnc/internal/masterkategorisparepart/repo/memory"
	masterkategorisparepartsql "claim-pnc/internal/masterkategorisparepart/repo/sqlstore"
	masterkategorisparepartusecase "claim-pnc/internal/masterkategorisparepart/usecase"
	masterloginhttp "claim-pnc/internal/masterlogin/http"
	masterloginmemory "claim-pnc/internal/masterlogin/repo/memory"
	masterloginsql "claim-pnc/internal/masterlogin/repo/sqlstore"
	masterloginusecase "claim-pnc/internal/masterlogin/usecase"
	mastermaskinghttp "claim-pnc/internal/mastermasking/http"
	mastermaskingmemory "claim-pnc/internal/mastermasking/repo/memory"
	mastermaskingsql "claim-pnc/internal/mastermasking/repo/sqlstore"
	mastermaskingusecase "claim-pnc/internal/mastermasking/usecase"
	masterpanelhttp "claim-pnc/internal/masterpanel/http"
	masterpanelmemory "claim-pnc/internal/masterpanel/repo/memory"
	masterpanelsql "claim-pnc/internal/masterpanel/repo/sqlstore"
	masterpanelusecase "claim-pnc/internal/masterpanel/usecase"
	masterpasalhttp "claim-pnc/internal/masterpasal/http"
	masterpasalmemory "claim-pnc/internal/masterpasal/repo/memory"
	masterpasalsql "claim-pnc/internal/masterpasal/repo/sqlstore"
	masterpasalusecase "claim-pnc/internal/masterpasal/usecase"
	laporanhasilaihttp "claim-pnc/internal/laporanhasilai/http"
	laporanhasilaimemory "claim-pnc/internal/laporanhasilai/repo/memory"
	laporanhasilaisql "claim-pnc/internal/laporanhasilai/repo/sqlstore"
	laporanhasilaiusecase "claim-pnc/internal/laporanhasilai/usecase"
	masterpasalaihttp "claim-pnc/internal/masterpasalai/http"
	masterpasalaimemory "claim-pnc/internal/masterpasalai/repo/memory"
	masterpasalaisql "claim-pnc/internal/masterpasalai/repo/sqlstore"
	masterpasalaiusecase "claim-pnc/internal/masterpasalai/usecase"
	masterpenolakanhttp "claim-pnc/internal/masterpenolakan/http"
	masterpenolakanmemory "claim-pnc/internal/masterpenolakan/repo/memory"
	masterpenolakansql "claim-pnc/internal/masterpenolakan/repo/sqlstore"
	masterpenolakanusecase "claim-pnc/internal/masterpenolakan/usecase"
	masterpenyebabkerugianhttp "claim-pnc/internal/masterpenyebabkerugian/http"
	masterpenyebabkerugianmemory "claim-pnc/internal/masterpenyebabkerugian/repo/memory"
	masterpenyebabkerugiansql "claim-pnc/internal/masterpenyebabkerugian/repo/sqlstore"
	masterpenyebabkerugianusecase "claim-pnc/internal/masterpenyebabkerugian/usecase"
	masterpicteknikdirectory "claim-pnc/internal/masterpicteknik/directory"
	masterpicteknikhttp "claim-pnc/internal/masterpicteknik/http"
	masterpicteknikmemory "claim-pnc/internal/masterpicteknik/repo/memory"
	masterpictekniksql "claim-pnc/internal/masterpicteknik/repo/sqlstore"
	masterpicteknikusecase "claim-pnc/internal/masterpicteknik/usecase"
	masterreashttp "claim-pnc/internal/masterreas/http"
	masterreasmemory "claim-pnc/internal/masterreas/repo/memory"
	masterreassql "claim-pnc/internal/masterreas/repo/sqlstore"
	masterreasusecase "claim-pnc/internal/masterreas/usecase"
	masterrecoveryhttp "claim-pnc/internal/masterrecovery/http"
	masterrecoverymemory "claim-pnc/internal/masterrecovery/repo/memory"
	masterrecoverysql "claim-pnc/internal/masterrecovery/repo/sqlstore"
	masterrecoveryusecase "claim-pnc/internal/masterrecovery/usecase"
	masterrecoveryva "claim-pnc/internal/masterrecovery/virtualaccount"
	masterrekeningcashier "claim-pnc/internal/masterrekening/cashier"
	masterrekeninghttp "claim-pnc/internal/masterrekening/http"
	masterrekeningnotif "claim-pnc/internal/masterrekening/notification"
	masterrekeningmemory "claim-pnc/internal/masterrekening/repo/memory"
	masterrekeningsql "claim-pnc/internal/masterrekening/repo/sqlstore"
	masterrekeningusecase "claim-pnc/internal/masterrekening/usecase"
	"claim-pnc/internal/mastersparepart"
	masterspareparthttp "claim-pnc/internal/mastersparepart/http"
	mastersparepartmemory "claim-pnc/internal/mastersparepart/repo/memory"
	mastersparepartsql "claim-pnc/internal/mastersparepart/repo/sqlstore"
	mastersparepartusecase "claim-pnc/internal/mastersparepart/usecase"
	masterstatushttp "claim-pnc/internal/masterstatus/http"
	masterstatusmemory "claim-pnc/internal/masterstatus/repo/memory"
	masterstatussql "claim-pnc/internal/masterstatus/repo/sqlstore"
	masterstatususecase "claim-pnc/internal/masterstatus/usecase"
	masterstatusprogreshttp "claim-pnc/internal/masterstatusprogres/http"
	masterstatusprogresmemory "claim-pnc/internal/masterstatusprogres/repo/memory"
	masterstatusprogressql "claim-pnc/internal/masterstatusprogres/repo/sqlstore"
	masterstatusprogresusecase "claim-pnc/internal/masterstatusprogres/usecase"
	mastersupplierhttp "claim-pnc/internal/mastersupplier/http"
	mastersuppliermemory "claim-pnc/internal/mastersupplier/repo/memory"
	mastersuppliersql "claim-pnc/internal/mastersupplier/repo/sqlstore"
	mastersupplierusecase "claim-pnc/internal/mastersupplier/usecase"
	mastersurveyorsaccount "claim-pnc/internal/mastersurveyors/account"
	mastersurveyorscommittee "claim-pnc/internal/mastersurveyors/committee"
	mastersurveyorshttp "claim-pnc/internal/mastersurveyors/http"
	mastersurveyorsmemory "claim-pnc/internal/mastersurveyors/repo/memory"
	mastersurveyorssql "claim-pnc/internal/mastersurveyors/repo/sqlstore"
	mastersurveyorsusecase "claim-pnc/internal/mastersurveyors/usecase"
	mastertipespareparthttp "claim-pnc/internal/mastertipesparepart/http"
	mastertipesparepartmemory "claim-pnc/internal/mastertipesparepart/repo/memory"
	mastertipesparepartsql "claim-pnc/internal/mastertipesparepart/repo/sqlstore"
	mastertipesparepartusecase "claim-pnc/internal/mastertipesparepart/usecase"
	mastertipesurveyorshttp "claim-pnc/internal/mastertipesurveyors/http"
	mastertipesurveyorsmemory "claim-pnc/internal/mastertipesurveyors/repo/memory"
	mastertipesurveyorssql "claim-pnc/internal/mastertipesurveyors/repo/sqlstore"
	mastertipesurveyorsusecase "claim-pnc/internal/mastertipesurveyors/usecase"
	masterxolhttp "claim-pnc/internal/masterxol/http"
	masterxolnotif "claim-pnc/internal/masterxol/notification"
	masterxolmemory "claim-pnc/internal/masterxol/repo/memory"
	masterxolsql "claim-pnc/internal/masterxol/repo/sqlstore"
	masterxolusecase "claim-pnc/internal/masterxol/usecase"
	menuhttp "claim-pnc/internal/menu/http"
	menumemory "claim-pnc/internal/menu/repo/memory"
	menusql "claim-pnc/internal/menu/repo/sqlstore"
	menuusecase "claim-pnc/internal/menu/usecase"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
	portalsql "claim-pnc/internal/portal/repo/sqlstore"
	reportklaimhttp "claim-pnc/internal/reportklaim/http"
	reportklaimmemory "claim-pnc/internal/reportklaim/repo/memory"
	reportklaimsql "claim-pnc/internal/reportklaim/repo/sqlstore"
	reportklaimusecase "claim-pnc/internal/reportklaim/usecase"
	reportkpihttp "claim-pnc/internal/reportkpi/http"
	reportkpimemory "claim-pnc/internal/reportkpi/repo/memory"
	reportkpisql "claim-pnc/internal/reportkpi/repo/sqlstore"
	reportkpiusecase "claim-pnc/internal/reportkpi/usecase"
	"claim-pnc/internal/riwayatklaim"
)

// defaultEnvFile dibaca bila ada. Nilai yang sudah ada di lingkungan proses menang atas
// isinya, sehingga satu perintah dapat menimpa satu nilai tanpa menyunting berkas.
const defaultEnvFile = ".env"

func main() {
	if err := run(); err != nil {
		// Kegagalan saat start ditulis ke stderr dan menghentikan proses. Aplikasi
		// yang setengah hidup lebih berbahaya daripada aplikasi yang tidak start.
		fmt.Fprintln(os.Stderr, "gagal menjalankan aplikasi:", err)
		os.Exit(1)
	}
}

func run() error {
	// Dua flag, keduanya untuk mode periksa. Aplikasi normal tidak memakai flag sama
	// sekali — seluruh konfigurasinya dari .env atau lingkungan (ADR-0025).
	checkMode := flag.Bool("periksa", false,
		"periksa integrasi basis data dan HCC/HCQ lalu berhenti; tidak menulis apa pun")
	testLogin := flag.String("login", "",
		"nama pengguna yang dicoba pada mode periksa; kata sandinya dibaca dari stdin")
	flag.Parse()

	if err := config.LoadEnvFile(defaultEnvFile); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if *checkMode {
		return check(cfg, *testLogin, os.Stdin, os.Stdout)
	}

	logger := logging.New(slog.LevelInfo)
	logger.Info("konfigurasi terbaca", slog.Any("konfigurasi", cfg.Summary()))

	assembly, err := build(cfg, logger)
	if err != nil {
		return err
	}
	defer assembly.close()

	spaFiles, err := spa.Files()
	if err != nil {
		logger.Warn("antarmuka tidak tersedia; aplikasi hanya melayani API",
			slog.String("sebab", err.Error()))
		spaFiles = nil
	} else {
		// Kapan antarmuka yang TERSEMAT dibangun — bukan kapan `npm run build` terakhir
		// dijalankan di folder frontend. Keduanya berbeda bila binary tidak ikut
		// dibangun ulang, dan perbedaan itu tidak meninggalkan jejak lain sama sekali:
		// aplikasi menyajikan layar versi lama tanpa satu pun galat, sehingga fitur yang
		// sudah diperbaiki tampak masih rusak.
		logger.Info("antarmuka tersemat", slog.String("dibangun", spaVersionText()))
	}

	// Satu penulis JSON dan satu penulis galat dipakai bersama seluruh modul, supaya
	// bentuk respons dan header Cache-Control-nya tidak berbeda antarmodul.
	writeJSON := func(w http.ResponseWriter, r *http.Request, status int, body any) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	writeAuthError := authhttp.WriteError(logger)

	handlerAuth := authhttp.NewHandler(assembly.auth, logger)
	handlerKomite := komitehttp.NewHandler(komitehttp.Options{
		Service:             assembly.komite,
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: komitehttp.ErrorWriter(writeAuthError),
	})
	handlerKomiteInbox := komitehttp.NewInboxHandler(komitehttp.InboxHandlerOptions{
		Service: assembly.komiteInbox,
		// Jembatan satu arah dari modul auth ke modul Komite. Ia dipasang di sini, bukan
		// di dalam salah satu modul, supaya keduanya tetap tidak saling mengimpor — yang
		// tahu keduanya hanyalah berkas perakitan ini.
		//
		// Yang dijembatani LOGIN, bukan NIK. Inbox disaring terhadap `PXASSIGNEDOPERATORID`
		// pada worklist Pega, yang berisi nama seperti `ELLENSUPRIYATI` — dan Work Owner
		// menetapkan kunci pencocokannya adalah login yang DIKETIK pengguna
		// (`docs/keputusan-implementasi.md` §16.5).
		Caller: func(ctx context.Context) (komitehttp.InboxCaller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return komitehttp.InboxCaller{}, false
			}
			return komitehttp.InboxCaller{
				Login: baseCtx.User.Login,
				Name:  baseCtx.User.Name,
			}, true
		},
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: komitehttp.ErrorWriter(writeAuthError),
	})

	handlerPortal := portalhttp.NewHandler(portalhttp.Options{
		Repo:         assembly.portal,
		ReadyAliases: assembly.readyAliases,
		PrimaryAlias: cfg.PrimaryPortal,
		Logger:       logger,
		WriteResponse: func(w http.ResponseWriter, r *http.Request, status int, body any) {
			authhttp.WriteJSON(w, r, status, body, logger)
		},
		WriteError: authhttp.WriteError(logger),
	})

	// Galat portal dipetakan modul portal, sisanya diteruskan ke pemeta modul auth.
	// Urutan pembungkusnya menentukan: yang lebih khusus memeriksa lebih dulu. Satu
	// rantai untuk setiap modul bisnis, bukan satu tafsiran per modul.
	writePortalAwareError := portalhttp.WithPortalError(writeAuthError, writeJSON)

	// Master Status Klaim dirakit SESUDAH penulis galat sadar-portal terbentuk: sejak
	// modul ini menjadi per portal, galat portalnya harus dipetakan modul portal — bukan
	// jatuh ke pemeta galat auth sebagai 500.
	handlerMasterStatus := masterstatushttp.NewHandler(masterstatushttp.Options{
		Service:             assembly.masterStatus,
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: masterstatushttp.ErrorWriter(writePortalAwareError),
	})

	// Master Dominan Factor dirakit dengan pola yang sama, dan dengan alasan yang sama:
	// seluruh rutenya menyentuh basis data entitas, sehingga galat portalnya harus
	// dipetakan modul portal — bukan jatuh ke pemeta galat auth sebagai 500.
	dominantFactorHandler := masterdominanfactorhttp.NewHandler(masterdominanfactorhttp.Options{
		Service:             assembly.masterDominanFactor,
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: masterdominanfactorhttp.ErrorWriter(writePortalAwareError),
	})

	// Master XOL dirakit dengan pola yang sama, ditambah satu bahan yang modul master
	// lain tidak punya: identitas pemanggil. Menyimpan di layar ini sekaligus MENGAJUKAN
	// struktur treaty ke komite — persis seperti sistem lama — dan pengajuan tanpa jejak
	// siapa yang mengajukan tidak punya arti (`D-59`).
	xolHandler, err := masterxolhttp.NewHandler(masterxolhttp.Options{
		Service: assembly.masterXOL,
		// Jembatan satu arah dari modul auth ke modul Master XOL. Ia dipasang di sini,
		// bukan di dalam salah satu modul, supaya kedua modul tetap tidak saling
		// mengimpor — yang tahu keduanya hanyalah berkas perakitan ini.
		Caller: func(ctx context.Context) (masterxolhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterxolhttp.Caller{}, false
			}
			return masterxolhttp.Caller{
				Identity: baseCtx.User.Identity,
				Name:     baseCtx.User.Name,
			}, true
		},
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: masterxolhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Penyebab Kerugian dirakit dengan pola yang sama. Satu hal yang
	// membedakannya dari master lain: keterangan di sini menjadi kolom PENGELOMPOKAN
	// pada laporan Pega yang memakai `GROUP BY COL_DESC`, sehingga portal yang keliru
	// mengubah pengelompokan laporan — bukan hanya isi satu layar.
	causeOfLossHandler := masterpenyebabkerugianhttp.NewHandler(masterpenyebabkerugianhttp.Options{
		Service:             assembly.masterPenyebabKerugian,
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: masterpenyebabkerugianhttp.ErrorWriter(writePortalAwareError),
	})

	maskingHandler, err := mastermaskinghttp.NewHandler(mastermaskinghttp.Options{
		Service: assembly.masterMasking,
		// Jembatan satu arah dari modul auth ke modul Master Masking. Ia dipasang di sini,
		// bukan di dalam salah satu modul, supaya kedua modul tetap tidak saling mengimpor —
		// yang tahu keduanya hanyalah berkas perakitan ini.
		//
		// Identitas pemanggil mengisi kolom USERINPUT. Di modul ini ia lebih dari sekadar
		// jejak: yang dicatat adalah siapa yang memberi atau mencabut kewenangan membuka
		// nomor KTP, surel, dan nomor telepon nasabah. `D-59` menetapkan tidak ada
		// pemisahan tugas formal, sehingga catatan ini satu-satunya kontrol pengimbang.
		Caller: func(ctx context.Context) (mastermaskinghttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return mastermaskinghttp.Caller{}, false
			}
			return mastermaskinghttp.Caller{
				Identity: baseCtx.User.Identity,
				Name:     baseCtx.User.Name,
			}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    mastermaskinghttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	progressStatusHandler, err := masterstatusprogreshttp.NewHandler(masterstatusprogreshttp.Options{
		Service:       assembly.masterStatusProgres,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterstatusprogreshttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	clauseAIHandler, err := masterpasalaihttp.NewHandler(masterpasalaihttp.Options{
		Service:       assembly.masterPasalAI,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterpasalaihttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Laporan Hasil AI (MENU_ID 82). Penulis galatnya dirantai, bukan diganti: modul ini
	// punya galat validasi sendiri — kedua isian tanggal yang wajib — dan meneruskan
	// selebihnya ke penulis galat portal dan sesi.
	aiReportHandler, err := laporanhasilaihttp.NewHandler(laporanhasilaihttp.Options{
		Service:             assembly.laporanHasilAI,
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: laporanhasilaihttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Status Progres 2 memakai penulis galat yang SAMA PERSIS dengan tingkat 1,
	// bukan rantai baru: keduanya memetakan galat lewat satu fungsi petakanGalat, sehingga
	// satu jenis galat tidak pernah dijawab dua bentuk yang berbeda.
	progressStatus2Handler, err := masterstatusprogreshttp.NewHandler2(masterstatusprogreshttp.Options2{
		Service:       assembly.masterStatusProgres2,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterstatusprogreshttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Penolakan Klaim memakai penulis galat yang SAMA dengan master status
	// progres: ia pun menyentuh basis data entitas, sehingga galat portal harus dijawab
	// dengan kode yang sudah dikenal frontend.
	rejectionHandler, err := masterpenolakanhttp.NewHandler(masterpenolakanhttp.Options{
		Service: assembly.masterPenolakan,
		Komite:  assembly.masterPenolakanKomite,
		// Jembatan satu arah dari modul auth. Ia dipasang di sini, bukan di dalam salah
		// satu modul, supaya kedua modul tetap tidak saling mengimpor — yang tahu
		// keduanya hanyalah berkas perakitan ini.
		//
		// Login yang DIKETIK pengguna, bukan NIK: kolom USER_INPUT pada
		// POOLDATA.MST_PENOLAKAN_KLAIM_2 sudah berisi login pada baris-baris lama.
		Caller: func(ctx context.Context) (masterpenolakanhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterpenolakanhttp.Caller{}, false
			}
			return masterpenolakanhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterpenolakanhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Auto Claim memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	masterAutoClaimHandler, err := masterautoclaimhttp.NewHandler(masterautoclaimhttp.Options{
		Service: assembly.masterAutoClaim,
		// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
		// tidak saling mengimpor.
		//
		// Login yang DIKETIK pengguna, bukan NIK — dan di modul ini pilihan itu
		// MENENTUKAN, bukan sekadar rapi: nilainya dibandingkan dengan kolom KOMITE,
		// yang berisi OPERATOR_ID dari POOLDATA.EMAILKOMITE. Memakai NIK akan membuat
		// tab Komite Approval selalu kosong, tanpa satu pun galat.
		Caller: func(ctx context.Context) (masterautoclaimhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterautoclaimhttp.Caller{}, false
			}
			return masterautoclaimhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterautoclaimhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Bengkel memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	workshopHandler, err := masterbengkelhttp.NewHandler(masterbengkelhttp.Options{
		Service: assembly.masterBengkel,
		// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
		// tidak saling mengimpor.
		//
		// Login yang DIKETIK pengguna, bukan NIK — sama seperti modul master lain. Di
		// modul ini ia TIDAK pernah tersimpan: POOLDATA.BENGKEL_HE tidak punya satu pun
		// kolom pencatat pelaku maupun waktu, sehingga identitas pemanggil hanya masuk
		// log. Itu keterbatasan tabelnya, dan sudah dicatat pada usecase.Actor.
		Caller: func(ctx context.Context) (masterbengkelhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterbengkelhttp.Caller{}, false
			}
			return masterbengkelhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterbengkelhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Panel memakai penulis galat yang SAMA dengan modul bisnis lain: ia menyentuh
	// basis data entitas, sehingga galat portal harus dijawab dengan kode yang sudah
	// dikenal frontend.
	panelHandler, err := masterpanelhttp.NewHandler(masterpanelhttp.Options{
		Service: assembly.masterPanel,
		// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
		// tidak saling mengimpor.
		//
		// Login yang DIKETIK pengguna, bukan NIK — sama seperti modul master lain. Di
		// modul ini ia TIDAK pernah tersimpan: POOLDATA.PANEL_HE tidak punya satu pun
		// kolom pencatat pelaku maupun waktu, sehingga identitas pemanggil hanya masuk
		// log. Itu keterbatasan tabelnya, dan sudah dicatat pada usecase.Actor.
		Caller: func(ctx context.Context) (masterpanelhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterpanelhttp.Caller{}, false
			}
			return masterpanelhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterpanelhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Sparepart memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	sparepartHandler, err := masterspareparthttp.NewHandler(masterspareparthttp.Options{
		Service: assembly.masterSparepart,
		// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
		// tidak saling mengimpor.
		//
		// Login yang DIKETIK pengguna, bukan NIK — sama seperti modul master lain. Di modul
		// ini ia BENAR-BENAR TERSIMPAN ke kolom USER_UPDATE, berbeda dari Master Panel yang
		// tabelnya tidak punya kolom pencatat pelaku sama sekali. Kolom itu sudah berisi
		// login pada baris-baris lama, dan menuliskan NIK ke kolom yang sama akan membuat dua
		// bentuk identitas hidup berdampingan tanpa cara membedakannya.
		Caller: func(ctx context.Context) (masterspareparthttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterspareparthttp.Caller{}, false
			}
			return masterspareparthttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterspareparthttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Grouping Sparepart memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	groupingHandler, err := mastergroupingspareparthttp.NewHandler(
		mastergroupingspareparthttp.Options{
			Service: assembly.masterGroupingSparepart,
			// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
			// tidak saling mengimpor.
			//
			// Login yang DIKETIK pengguna, bukan NIK — sama seperti modul master lain. Di
			// modul ini ia TIDAK tersimpan ke basis data: kedua tabelnya tidak punya satu pun
			// kolom pencatat pelaku. Ia dipakai untuk mencatat siapa yang mengubah apa di log,
			// satu-satunya tempat yang tersedia sampai S-5 Jejak Audit dibangun.
			Caller: func(ctx context.Context) (mastergroupingspareparthttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return mastergroupingspareparthttp.Caller{}, false
				}
				return mastergroupingspareparthttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeJSON,
			WriteError:    mastergroupingspareparthttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	// Master Kategori Sparepart memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	//
	// Ia MENERIMA Caller meski tabelnya tidak punya kolom pencatat pelaku. Login-nya tidak
	// tersimpan di basis data — ia hanya masuk ke log, dan log itulah satu-satunya tempat
	// siapa yang menyetujui sebuah kategori terekam. Lihat masterkategorisparepart/usecase.Actor.
	partCategoryHandler, err := masterkategorispareparthttp.NewHandler(
		masterkategorispareparthttp.Options{
			Service: assembly.masterKategoriSparepart,
			Caller: func(ctx context.Context) (masterkategorispareparthttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return masterkategorispareparthttp.Caller{}, false
				}
				return masterkategorispareparthttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeJSON,
			WriteError:    masterkategorispareparthttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	// Master Tipe Sparepart memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	//
	// Ia MENERIMA Caller meski tabelnya tidak punya kolom pencatat pelaku. Login-nya tidak
	// tersimpan di basis data — ia hanya masuk ke log, dan log itulah satu-satunya tempat
	// siapa yang menyetujui sebuah tipe terekam. Lihat mastertipesparepart/usecase.Actor.
	partTypeHandler, err := mastertipespareparthttp.NewHandler(
		mastertipespareparthttp.Options{
			Service: assembly.masterTipeSparepart,
			Caller: func(ctx context.Context) (mastertipespareparthttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return mastertipespareparthttp.Caller{}, false
				}
				return mastertipespareparthttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeJSON,
			WriteError:    mastertipespareparthttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	// Master Login memakai penulis galat yang SAMA dengan modul bisnis lain.
	//
	// Ia MENERIMA Caller, dan di sini Caller bukan sekadar pengisi log seperti pada kedua
	// modul di atas: login pemanggil dipakai MENURUNKAN kolom LOGINLEADER setiap baris baru
	// — `Activity/CNMInsertMstLoginSurveyor_act` mencari leader milik pengguna yang
	// menyimpan, lalu menuliskannya ke baris yang dibuat. Tanpa identitas itu, setiap baris
	// baru lahir tanpa tautan tim. Lihat masterlogin/usecase.Service.Create.
	// Detail Penyebab Kerugian. Caller-nya HANYA pengisi log — POOLDATA.D_CAUSE_OF_LOSS
	// hanya punya D_COL_ID dan JSONDATA, tanpa satu pun kolom pencatat pelaku maupun waktu.
	// Itu keterbatasan tabelnya, dan menambahkannya menempuh `D-63`; lihat
	// detailpenyebab/usecase.Actor.
	causeOfLossDetailHandler, err := detailpenyebabhttp.NewHandler(
		detailpenyebabhttp.Options{
			Service: assembly.detailPenyebab,
			Caller: func(ctx context.Context) (detailpenyebabhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return detailpenyebabhttp.Caller{}, false
				}
				return detailpenyebabhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeJSON,
			WriteError:    detailpenyebabhttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	surveyorLoginHandler, err := masterloginhttp.NewHandler(
		masterloginhttp.Options{
			Service: assembly.masterLogin,
			Caller: func(ctx context.Context) (masterloginhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return masterloginhttp.Caller{}, false
				}
				return masterloginhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeJSON,
			WriteError:    masterloginhttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	// Master Pasal Kerugian memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	//
	// Ia TIDAK menerima Caller, dan itu bukan kelalaian: POOLDATA.V_M_DATA_PASAL hanya
	// punya tiga kolom — IDDATA, IDPASAL, JSONPASAL — sehingga tidak ada tempat menuliskan
	// siapa dan kapan. Akibatnya perubahan dan penghapusan di layar itu tidak meninggalkan
	// jejak di basis data; keterbatasan itu dicatat pada masterpasalhttp.Handler.
	clauseHandler, err := masterpasalhttp.NewHandler(masterpasalhttp.Options{
		Service:       assembly.masterPasal,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterpasalhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Inbox Investigator (MENU_ID 48) — antrean pekerjaan di workbasket `InvestigatorPNC`.
	//
	// Ia TIDAK menerima Caller, dan itu akibat langsung dari bentuk layarnya: yang
	// ditampilkan adalah isi WORKBASKET, yaitu antrean bersama yang belum bertuan (`D-26`).
	// Tidak ada satu baris pun yang diturunkan dari identitas pemanggil.
	//
	// Itu berubah begitu layar kerjanya dibangun: mengambil pekerjaan dari antrean menuntut
	// identitas pengambilnya, dan modul itulah yang akan membutuhkannya.
	investigatorInboxHandler, err := inboxinvestigatorhttp.NewHandler(
		inboxinvestigatorhttp.Options{
			Service:       assembly.inboxInvestigator,
			Logger:        logger,
			WriteResponse: writeJSON,
			WriteError:    inboxinvestigatorhttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	// Inbox Receive TKA (MENU_ID 49) atas POOLDATA.T_CLAIM_TKA_H.
	//
	// Ia TIDAK menerima Caller, dan berbeda dari modul baca-saja, di sini ketiadaannya
	// adalah UTANG yang disadari — bukan akibat bentuk layarnya. Modul ini mengubah
	// `TGLDOKLENGKAP` pada data klaim, sehingga siapa pelakunya adalah keterangan yang
	// seharusnya tercatat sebagai jejak audit.
	//
	// Modul Jejak Audit (`S-5`) belum ada dan daftar peristiwa wajib auditnya masih
	// ditunggu dari Compliance (`ADR-0026`). Sampai itu tiba, pelakunya hanya tercatat di
	// log aplikasi lewat middleware permintaan. Itu tidak memadai: `D-59` menetapkan
	// satuan izin adalah MENU dan tidak ada pemisahan tugas, sehingga jejak audit adalah
	// satu-satunya kontrol pengimbang yang tersisa.
	receiveTKAHandler, err := inboxreceivetkahttp.NewHandler(
		inboxreceivetkahttp.Options{
			Service:       assembly.inboxReceiveTKA,
			Logger:        logger,
			WriteResponse: writeJSON,
			WriteError:    inboxreceivetkahttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	// Master Reas (MENU_ID 35) atas POOLDATA.T_REINSURER.
	//
	// Ia TIDAK menerima Caller, dan itu bukan kelalaian melainkan akibat langsung dari
	// modulnya yang hanya MEMBACA: tidak ada baris yang diturunkan dari identitas
	// pemanggil, dan tidak ada perubahan yang perlu dicatat pelakunya.
	//
	// Alasan modul ini hanya membaca ada pada banner paket masterreas — ringkasnya,
	// satu-satunya penulis tabel itu di sistem lama adalah alur PLA/DLA lewat
	// `Database/UPDATEREAS.prc`, dipanggil `UpdateDetailPLA2` dan `UpdateDetailDLA2`,
	// bukan layar master ini.
	reinsurerMemberHandler, err := masterreashttp.NewHandler(masterreashttp.Options{
		Service:       assembly.masterReas,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterreashttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master Supplier memakai penulis galat yang SAMA dengan modul bisnis lain: ia
	// menyentuh basis data entitas, sehingga galat portal harus dijawab dengan kode yang
	// sudah dikenal frontend.
	supplierHandler, err := mastersupplierhttp.NewHandler(mastersupplierhttp.Options{
		Service: assembly.masterSupplier,
		// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
		// tidak saling mengimpor.
		//
		// Login yang DIKETIK pengguna, bukan NIK — sama seperti modul master lain.
		// Berbeda dari Master Bengkel dan Master Panel, di modul ini ia BENAR-BENAR
		// TERSIMPAN: ia menjadi kunci USERKLAIMID di dalam dokumen supplier dan kolom
		// USER_REQ pada baris permintaan persetujuan. Keduanya ditulis sistem lama juga
		// (`CreateNewMasterSupplier_post` step 6), sehingga jejaknya bukan tambahan.
		Caller: func(ctx context.Context) (mastersupplierhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return mastersupplierhttp.Caller{}, false
			}
			return mastersupplierhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    mastersupplierhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Menu dirakit dari POOLDATA.M_MENU_APLIKASI_PNC dan M_OTORISASI_PNC. Jembatan
	// konteks pemanggilnya SATU ARAH dari modul auth, dipasang di sini supaya kedua
	// modul tetap tidak saling mengimpor.
	menuHandler, err := menuhttp.NewHandler(menuhttp.Options{
		Service: assembly.menu,
		Caller: func(ctx context.Context) (menuhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return menuhttp.Caller{}, false
			}
			// Login yang DIKETIK pengguna, bukan NIK: itulah yang dicocokkan ke
			// M_LOGIN_GROUP_PNC.LOGIN_ID dan M_OTORISASI_PNC.LOGIN_ID_GROUP.
			return menuhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    menuhttp.ErrorWriter(writeAuthError),
	})
	if err != nil {
		return err
	}

	// Bahan penentu portal aktif dipakai setiap modul bisnis yang menyentuh basis data
	// entitas. Ia dirakit sekali di sini supaya modul-modul berikutnya memakai
	// pemeriksaan yang sama persis, bukan masing-masing menafsirkannya sendiri.
	activePortalDeps := portalhttp.ActivePortalDeps{
		Repo:         assembly.portal,
		ReadyAliases: assembly.readyAliases,
		Logger:       logger,
		WriteError:   writePortalAwareError,
	}

	// Master Dokumen Travel. Seperti Master Status Progres, tabelnya ada di basis data
	// SETIAP entitas — rutenya karena itu memasang pemeriksaan portal sendiri di dalam
	// Mount.
	travelDocumentHandler, err := masterdokumentravelhttp.NewHandler(masterdokumentravelhttp.Options{
		Service:       assembly.masterDokumenTravel,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterdokumentravelhttp.ErrorWriter(writePortalAwareError),
	})

	// Inbox Auto Claim. Ia menyentuh basis data entitas, sehingga rutenya memakai
	// activePortalDeps yang sama dengan modul bisnis lain.
	autoClaimHandler, err := inboxautoclaimhttp.NewHandler(inboxautoclaimhttp.Options{
		Service: assembly.inboxAutoClaim,
		// Jembatan satu arah dari modul auth. Yang dibutuhkan hanya LOGIN pemanggil,
		// karena itulah yang tertulis di kolom USERINPUT dan tampil sebagai
		// "User Upload" di grid.
		Caller: func(ctx context.Context) (inboxautoclaimhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inboxautoclaimhttp.Caller{}, false
			}
			return inboxautoclaimhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    inboxautoclaimhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	surveyorTypeHandler, err := mastertipesurveyorshttp.NewHandler(mastertipesurveyorshttp.Options{
		Service:       assembly.masterTipeSurveyors,
		Logger:        logger,
		WriteResponse: writeJSON,
		// Penulis galat yang sudah sadar portal: galat portal dipetakan modul portal,
		// sisanya diteruskan ke pemeta modul auth.
		WriteError: mastertipesurveyorshttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Master COL Simas Online. Sama seperti dua modul di atas, tabelnya ada di basis data
	// SETIAP entitas — termasuk POOLDATA.BUSINESS yang hanya dibacanya.
	//
	// Namanya dibedakan dari causeOfLossHandler di atas dengan sengaja: keduanya memang
	// "penyebab kerugian", tetapi modulnya berbeda — yang ini Simas Online (M_CAUSE_OF_LOSS
	// milik COL), yang di atas tingkat golongan (MENU_ID 20).
	simasOnlineCauseOfLossHandler, err := mastercolhttp.NewHandler(mastercolhttp.Options{
		Service:       assembly.masterCOLSimasOnline,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    mastercolhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Daftar Tipe Dokumen. Sama seperti tiga modul di atas, tabelnya ada di basis data
	// SETIAP entitas. Ia satu-satunya di antara keempatnya yang juga menuliskan JEJAK
	// SIMPAN — kolom USER_EDIT dan TGL_EDIT — sehingga ia perlu tahu siapa pemanggilnya.
	documentTypeHandler, err := daftartipedokumenhttp.NewHandler(daftartipedokumenhttp.Options{
		Service: assembly.daftarTipeDokumen,
		Logger:  logger,
		// Jembatan satu arah dari modul auth, bentuknya sama dengan yang dipakai master
		// rekening di bawah. Ia dipasang di sini, bukan di dalam salah satu modul, supaya
		// kedua modul tetap tidak saling mengimpor — yang tahu keduanya hanyalah berkas
		// perakitan ini.
		Caller: func(ctx context.Context) (daftartipedokumenhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return daftartipedokumenhttp.Caller{}, false
			}
			return daftartipedokumenhttp.Caller{Identity: baseCtx.User.Identity}, true
		},
		WriteResponse: writeJSON,
		WriteError:    daftartipedokumenhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Daftar Objek Dokumen. Tabelnya ada di basis data SETIAP entitas, termasuk
	// POOLDATA.BUSINESS yang hanya dibacanya.
	//
	// Tidak memakai Caller: tabelnya tidak punya kolom jejak simpan, sehingga tidak ada
	// yang perlu diisi dengan identitas pemanggil. Menambahkannya "untuk berjaga-jaga"
	// akan menyiratkan ada jejak yang sebenarnya tidak tersimpan di mana pun.
	documentObjectHandler, err := daftarobjekdokumenhttp.NewHandler(daftarobjekdokumenhttp.Options{
		Service:       assembly.daftarObjekDokumen,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    daftarobjekdokumenhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Daftar Detail Dokumen Travel. Seperti keempat modul di atas, seluruh tabelnya ada
	// di basis data SETIAP entitas — termasuk POOLDATA.M_DOCTRAVEL dan
	// POOLDATA.M_PLANTRAVEL yang hanya dibacanya sebagai daftar pilihan.
	travelDocumentDetailHandler, err := daftardetaildokumentravelhttp.NewHandler(daftardetaildokumentravelhttp.Options{
		Service:       assembly.daftarDetailDokumenTravel,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    daftardetaildokumentravelhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Daftar Detail Tipe Dokumen (MENU_ID 41). Seluruh tabelnya ada di basis data SETIAP
	// entitas — termasuk keempat master yang hanya dibacanya sebagai daftar pilihan.
	//
	// Ia MEMBUTUHKAN identitas pemanggil: activity lamanya mengisi USER_EDIT dari
	// `OperatorID.pyUserIdentifier` sebelum halamannya diserialisasi
	// (`Activity/CNMInsertDetailTypeDocument_act-Act.xml` langkah 1), sehingga
	// meniadakannya akan mengosongkan kolom yang hari ini terisi. Jembatannya dipasang di
	// sini, bukan di dalam salah satu modul, supaya keduanya tetap tidak saling
	// mengimpor.
	detailDocumentTypeHandler, err := daftardetailtipedokumenhttp.NewHandler(daftardetailtipedokumenhttp.Options{
		Service: assembly.daftarDetailTipeDokumen,
		Logger:  logger,
		Caller: func(ctx context.Context) (daftardetailtipedokumenhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return daftardetailtipedokumenhttp.Caller{}, false
			}
			return daftardetailtipedokumenhttp.Caller{Identity: baseCtx.User.Identity}, true
		},
		WriteResponse: writeJSON,
		WriteError:    daftardetailtipedokumenhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Daftar Tipe Dokumen Bisnis. Seperti modul di atas, seluruh tabelnya ada di basis
	// data SETIAP entitas — termasuk keempat master yang hanya dibacanya sebagai daftar
	// pilihan.
	//
	// Berbeda dari modul di atas, ia MEMBUTUHKAN identitas pemanggil: procedure lamanya
	// mengisi USER_EDIT dari `OperatorID.pyUserIdentifier`
	// (`InsertDetailTypeDocumentBusiness_act:574`), sehingga meniadakannya akan
	// mengosongkan kolom yang hari ini terisi. Jembatannya dipasang di sini, bukan di
	// dalam salah satu modul, supaya keduanya tetap tidak saling mengimpor.
	businessDocumentRuleHandler, err := daftartipedokumenbisnishttp.NewHandler(daftartipedokumenbisnishttp.Options{
		Service: assembly.daftarTipeDokumenBisnis,
		Logger:  logger,
		Caller: func(ctx context.Context) (daftartipedokumenbisnishttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return daftartipedokumenbisnishttp.Caller{}, false
			}
			return daftartipedokumenbisnishttp.Caller{
				Identity: baseCtx.User.Identity,
				Position: baseCtx.User.Position,
			}, true
		},
		WriteResponse: writeJSON,
		WriteError:    daftartipedokumenbisnishttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	surveyorHandler, err := mastersurveyorshttp.NewHandler(mastersurveyorshttp.Options{
		Service: assembly.masterSurveyors,
		// Jembatan satu arah dari modul auth ke modul Master Surveyors. Ia dipasang di
		// sini, bukan di dalam salah satu modul, supaya kedua modul tetap tidak saling
		// mengimpor — yang tahu keduanya hanyalah berkas perakitan ini.
		//
		// Identity yang dipakai adalah yang SAMA dengan milik master rekening, karena
		// keduanya dibandingkan dengan kolom komite yang berisi Operator ID.
		Caller: func(ctx context.Context) (mastersurveyorshttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return mastersurveyorshttp.Caller{}, false
			}
			return mastersurveyorshttp.Caller{
				Identity: baseCtx.User.Identity,
				Name:     baseCtx.User.Name,
			}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    mastersurveyorshttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Inbox Laporan Klaim. Jembatan pemanggilnya membawa LOGIN yang DIKETIK pengguna,
	// bukan NIK: itulah yang dicocokkan ke pxcreateoperator pada tabel warisan dan ke
	// sender pada percakapan.
	claimReportHandler, err := inboxlaporanklaimhttp.NewHandler(inboxlaporanklaimhttp.Options{
		Service: assembly.inboxLaporanKlaim,
		Caller: func(ctx context.Context) (inboxlaporanklaim.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inboxlaporanklaim.Caller{}, false
			}
			// Cabang SENGAJA tidak dibawa dari sini. `baseCtx.User.BranchCode` adalah
			// kode cabang HCC/HCQ (`Placement.BranchCode`), dan layar itu membandingkan
			// terhadap POOLDATA.BRANCH.ID — sistem kode yang berbeda. Memakainya membuat
			// daftar tampil kosong tanpa satu pun pesan galat, dan itu benar-benar
			// terjadi. Penerjemahannya kini tugas BranchResolver.
			return inboxlaporanklaim.Caller{
				Login: baseCtx.User.Login,
				Name:  baseCtx.User.Name,
			}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    inboxlaporanklaimhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	picTeknikHandler, err := masterpicteknikhttp.NewHandler(masterpicteknikhttp.Options{
		Service:       assembly.masterPicTeknik,
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterpicteknikhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	recoveryHandler, err := masterrecoveryhttp.NewHandler(masterrecoveryhttp.Options{
		Service: assembly.masterRecovery,
		// Jembatan satu arah dari modul auth ke modul Master Recovery. Ia dipasang di
		// sini, bukan di dalam salah satu modul, supaya kedua modul tetap tidak saling
		// mengimpor — yang tahu keduanya hanyalah berkas perakitan ini.
		//
		// Identitas pemanggil mengisi kolom USERNAME pada batch dan INPUTOPERATOR pada
		// bukti bayar. Keduanya WAJIB: `D-59` menetapkan tidak ada pemisahan tugas formal,
		// sehingga jejak siapa-mengerjakan-apa adalah satu-satunya kontrol pengimbang yang
		// tersisa.
		Caller: func(ctx context.Context) (masterrecoveryhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterrecoveryhttp.Caller{}, false
			}
			return masterrecoveryhttp.Caller{
				Identity: baseCtx.User.Identity,
				Name:     baseCtx.User.Name,
			}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    masterrecoveryhttp.ErrorWriter(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Inbox XOL. Jembatan pemanggilnya membawa LOGIN, sama seperti View History Claim:
	// identitas yang dipakai sistem lama di layar ini adalah `OperatorID.pyUserIdentifier`,
	// bukan NIK.
	//
	// Modul ini MEMBACA SAJA (keputusan Work Owner 2026-09-20). Ketiga rute tulisnya ada
	// tetapi menolak dengan alasan — lihat inboxxolhttp.Mount.
	inboxXOLHandler := inboxxolhttp.NewHandler(inboxxolhttp.Options{
		Service: assembly.inboxXOL,
		GetCaller: func(ctx context.Context) (inboxxolhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inboxxolhttp.Caller{}, false
			}
			return inboxxolhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:    logger,
		WriteJSON: writeJSON,
		// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
		// pemeriksaan portal.
		FallbackErrorWriter: inboxxolhttp.ErrorWriterFrom(writePortalAwareError),
	})
	if err != nil {
		return err
	}

	// Inbox Claim Treaty Prop (`MENU_ID 54`). Jembatan pemanggilnya membawa LOGIN dengan
	// alasan yang sama seperti Inbox XOL: yang dicocokkan ke `PXASSIGNEDOPERATORID` pada
	// tabel penugasan Pega adalah `OperatorID.pyUserIdentifier`, bukan NIK.
	//
	// Modul ini MEMBACA SAJA (keputusan Work Owner 2026-09-21). Rute tulisnya ada tetapi
	// menolak dengan alasan — lihat inboxclaimtreatypropthttp.Mount.
	claimTreatyPropHandler := inboxclaimtreatypropthttp.NewHandler(
		inboxclaimtreatypropthttp.Options{
			Service: assembly.inboxClaimTreatyProp,
			GetCaller: func(ctx context.Context) (inboxclaimtreatypropthttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxclaimtreatypropthttp.Caller{}, false
				}
				return inboxclaimtreatypropthttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxclaimtreatypropthttp.ErrorWriter(writePortalAwareError),
		})

	// Inbox Claim Treaty Non Prop (`MENU_ID 55`). Layar SAUDARA dari yang di atas, dan
	// dirakit terpisah dengan sengaja: keduanya membaca tabel, kolom, dan penanda objek
	// kerja yang berbeda — lihat kepala `internal/inboxclaimtreatynonprop`.
	//
	// Jembatan pemanggilnya membawa LOGIN dengan alasan yang sama: yang dicocokkan ke
	// `PXASSIGNEDOPERATORID` adalah `OperatorID.pyUserIdentifier`, bukan NIK.
	claimTreatyNonPropHandler := inboxclaimtreatynonprophttp.NewHandler(
		inboxclaimtreatynonprophttp.Options{
			Service: assembly.inboxClaimTreatyNonProp,
			GetCaller: func(ctx context.Context) (inboxclaimtreatynonprophttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxclaimtreatynonprophttp.Caller{}, false
				}
				return inboxclaimtreatynonprophttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxclaimtreatynonprophttp.ErrorWriter(writePortalAwareError),
		})

	// Inbox Manager Receive / PUCL (`MENU_ID 56`).
	//
	// Jembatan pemanggilnya membawa LOGIN seperti modul inbox lain, tetapi ALASANNYA
	// berbeda dan perlu dibaca sebelum disamakan: di sini login TIDAK dipakai menyaring
	// satu pun kueri. Layar ini pandangan penyelia atas pekerjaan seluruh petugas, dan
	// identitasnya dipakai untuk JEJAK — setiap pembukaan dicatat, bukan hanya yang
	// mencurigakan (lihat `internal/inboxmanagerreceivepucl/usecase`).
	managerReceivePUCLHandler := inboxmanagerreceivepuclhttp.NewHandler(
		inboxmanagerreceivepuclhttp.Options{
			Service: assembly.inboxManagerReceivePUCL,
			GetCaller: func(ctx context.Context) (inboxmanagerreceivepuclhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxmanagerreceivepuclhttp.Caller{}, false
				}
				return inboxmanagerreceivepuclhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxmanagerreceivepuclhttp.ErrorWriter(writePortalAwareError),
		})

	// Inbox RCL/PUCL (`MENU_ID 61`).
	//
	// Jembatan pemanggilnya membawa LOGIN dengan alasan yang sama seperti Inbox Manager
	// Receive / PUCL, dan perlu dibaca sebelum disamakan dengan modul inbox lain: di sini
	// login TIDAK dipakai menyaring satu pun kueri. Antreannya BERSAMA — penyaringnya akun
	// `RCLPUCL`, bukan pengguna — sehingga setiap petugas melihat daftar yang sama.
	// Identitasnya dipakai untuk JEJAK, dan pada permintaan laporan harian rentang
	// tanggalnya ikut dicatat (lihat `internal/inboxrclpucl/usecase`).
	rclPUCLHandler := inboxrclpuclhttp.NewHandler(
		inboxrclpuclhttp.Options{
			Service: assembly.inboxRCLPUCL,
			GetCaller: func(ctx context.Context) (inboxrclpuclhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxrclpuclhttp.Caller{}, false
				}
				return inboxrclpuclhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxrclpuclhttp.ErrorWriter(writePortalAwareError),
		})

	// Report KPI PNC (`MENU_ID 84`), tab KPI Adjuster.
	//
	// Jembatan pemanggilnya membawa LOGIN, dan di sini login benar-benar TIDAK dipakai
	// menyaring apa pun — laporannya pandangan penyelia atas seluruh adjuster, sama seperti
	// di Pega. Identitasnya dipakai untuk JEJAK, beserta penyaring yang dipilihnya (lihat
	// `internal/reportkpi/usecase`).
	reportKPIHandler := reportkpihttp.NewHandler(
		reportkpihttp.Options{
			Service: assembly.reportKPI,
			GetCaller: func(ctx context.Context) (reportkpihttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return reportkpihttp.Caller{}, false
				}
				return reportkpihttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: reportkpihttp.ErrorWriter(writePortalAwareError),
		})

	// Report Klaim. Identitas pemanggilnya TIDAK menyaring satu baris pun — laporan ini
	// memang laporan lintas cabang, sama seperti di Pega. Ia dipakai untuk JEJAK: siapa
	// mengunduh laporan apa, kapan, dengan penyaring apa.
	reportKlaimHandler, err := reportklaimhttp.NewHandler(
		reportklaimhttp.Options{
			Service: assembly.reportKlaim,
			Caller: func(ctx context.Context) (reportklaim.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return reportklaim.Caller{}, false
				}
				return reportklaim.Caller{
					Login: baseCtx.User.Login,
					Name:  baseCtx.User.Name,
				}, true
			},
			Logger:        logger,
			WriteResponse: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			WriteError: reportklaimhttp.ErrorWriter(writePortalAwareError),
		})
	if err != nil {
		return err
	}

	// Inbox Salvage. Jembatan pemanggilnya membawa LOGIN, dan di modul ini ia MENYARING —
	// bukan sekadar mengisi jejak.
	//
	// Dua tempat memakainya. Daftar "Request Balai Lelang" menampilkan pengajuan yang
	// `PNC_SALVAGE.PIC`-nya pemanggil sendiri, dan pencacahnya disaring dengan cara yang
	// sama. Memakai NIK di sini akan membuat daftar itu kosong bagi setiap pengguna —
	// kolomnya menyimpan Operator ID, bukan NIK.
	//
	// Ia pula yang menjadi `PIC` pengajuan yang disimpan lewat tombol Submit. Sistem lama
	// mengambilnya dari PIC Teknik yang tercatat pada klaimnya; di sini yang tercatat
	// adalah orang yang menyimpannya. Perbedaannya nyata pada daftar "Request Balai
	// Lelang", dan ia dicatat di `keputusan-implementasi.md`.
	salvageHandler := inboxsalvagehttp.NewHandler(
		inboxsalvagehttp.Options{
			Service: assembly.inboxSalvage,
			GetCaller: func(ctx context.Context) (inboxsalvagehttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxsalvagehttp.Caller{}, false
				}
				return inboxsalvagehttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxsalvagehttp.ErrorWriter(writePortalAwareError),
		})

	// Inbox Progress Claim. Jembatan pemanggilnya juga membawa LOGIN: itulah yang
	// dicocokkan ke `PEGA_DASHBOARDPNC.PIC` dan `MST_USER_TEKNIK.OPERATOR_ID`, dan
	// memakai NIK di sini akan membuat rekap per PIC kosong bagi setiap pengguna.
	inboxProgressClaimHandler := inboxprogressclaimhttp.NewHandler(
		inboxprogressclaimhttp.Options{
			Service: assembly.inboxProgressClaim,
			GetCaller: func(ctx context.Context) (inboxprogressclaimhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxprogressclaimhttp.Caller{}, false
				}
				return inboxprogressclaimhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxprogressclaimhttp.ErrorWriter(writePortalAwareError),
		})

	// Inbox Analyst Doctor. Jembatan pemanggilnya membawa LOGIN, dan di modul ini ia bukan
	// kenyamanan melainkan syarat: Report Definition menyaring
	// `PC_ASSIGN_WORKLIST.PXASSIGNEDOPERATORID` dengan identitas pemanggil, sehingga memakai
	// NIK di sini akan membuat antrean tampak KOSONG bagi setiap pengguna — dan antrean
	// kosong tidak pernah dilaporkan siapa pun sebagai kerusakan.
	//
	// Clock disuntikkan karena kolom "Lama Waktu Klaim" dihitung darinya (`F-5`).
	inboxAnalystDoctorHandler := inboxanalystdoctorhttp.NewHandler(
		inboxanalystdoctorhttp.Options{
			Service: assembly.inboxAnalystDoctor,
			GetCaller: func(ctx context.Context) (inboxanalystdoctorhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxanalystdoctorhttp.Caller{}, false
				}
				return inboxanalystdoctorhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Clock:     clock.System{},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxanalystdoctorhttp.ErrorWriter(writePortalAwareError),
		})

	// Inbox Komunikasi Cabang. Jembatan pemanggilnya membawa LOGIN, dan di sini alasannya
	// paling keras di antara seluruh modul inbox: login BUKAN sekadar jejak, melainkan
	// bahan yang diterjemahkan menjadi KODE CABANG — dan kode cabang itulah batas datanya.
	//
	// Memakai NIK di sini akan membuat penerjemahan gagal pada setiap pengguna, karena yang
	// dicocokkan `GetIDCabang` adalah `V_HRD_MST.login_aplikasi`. Akibatnya bukan daftar
	// kosong melainkan yang lebih buruk: setiap petugas jatuh ke jalur kantor pusat dan
	// melihat percakapan yang bukan haknya (`P-5`, lihat inboxkomunikasicabang.BranchFilter).
	//
	// NAMA ikut dibawa sejak 2026-09-24, ketika modul ini mulai menulis. Ia tersimpan sebagai
	// `REPLYFROMNAME` bersama balasannya — bukan diambil lewat join saat dibaca, karena jejak
	// yang namanya diambil lewat join berubah ketika orangnya berganti nama.
	komunikasiCabangHandler := inboxkomunikasicabanghttp.NewHandler(
		inboxkomunikasicabanghttp.Options{
			Service: assembly.inboxKomunikasiCabang,
			GetCaller: func(ctx context.Context) (inboxkomunikasicabanghttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxkomunikasicabanghttp.Caller{}, false
				}
				return inboxkomunikasicabanghttp.Caller{
					Login: baseCtx.User.Login,
					Name:  baseCtx.User.Name,
				}, true
			},
			Logger:    logger,
			WriteJSON: writeJSON,
			// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
			// pemeriksaan portal.
			FallbackErrorWriter: inboxkomunikasicabanghttp.ErrorWriter(writePortalAwareError),
		})

	outstandingHandler := inboxoutstandinghttp.NewHandler(inboxoutstandinghttp.Options{
		Service: assembly.inboxOutstanding,
		// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
		// tidak saling mengimpor.
		//
		// Hanya Login yang diambil: dari sanalah lini bisnis pemanggil dibaca, dan
		// batas data TIDAK PERNAH berasal dari badan permintaan maupun query string.
		GetCaller: func(ctx context.Context) (inboxoutstandinghttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inboxoutstandinghttp.Caller{}, false
			}
			return inboxoutstandinghttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:    logger,
		WriteJSON: writeJSON,
		// Galat portal dipetakan modul portal lebih dulu, sisanya jatuh ke pemeta galat
		// auth. Rantai yang sama dipakai modul master status progres.
		FallbackErrorWriter: inboxoutstandinghttp.ErrorWriter(writePortalAwareError),
	})

	// Input Req Protection. Pembuat permintaan diambil dari SESI, bukan dari badan
	// permintaan — ia satu-satunya jejak siapa yang meminta pembukaan proteksi.
	protectionRequestHandler := inputreqprotectionhttp.NewHandler(inputreqprotectionhttp.Options{
		Service: assembly.inputReqProtection,
		GetCaller: func(ctx context.Context) (inputreqprotectionhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inputreqprotectionhttp.Caller{}, false
			}
			return inputreqprotectionhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: inputreqprotectionhttp.ErrorWriter(writePortalAwareError),
	})

	// Inbox Accept Open Protection. Pelaku akseptasi juga diambil dari sesi: `D-59`
	// menetapkan tidak ada pemisahan tugas formal, sehingga kolom DIAKSEP_OLEH adalah
	// satu-satunya kontrol pengimbang yang tersisa atas persetujuan ini.
	protectionAcceptHandler := inboxacceptopenprotectionhttp.NewHandler(
		inboxacceptopenprotectionhttp.Options{
			Service: assembly.inboxAcceptOpenProtection,
			GetCaller: func(ctx context.Context) (inboxacceptopenprotectionhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return inboxacceptopenprotectionhttp.Caller{}, false
				}
				return inboxacceptopenprotectionhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:              logger,
			WriteJSON:           writeJSON,
			FallbackErrorWriter: inboxacceptopenprotectionhttp.ErrorWriter(writePortalAwareError),
		})

	closeClaimHandler := inboxcloseclaimhttp.NewHandler(inboxcloseclaimhttp.Options{
		Service: assembly.inboxCloseClaim,
		// Jembatan satu arah dari modul auth, dipasang di sini supaya kedua modul tetap
		// tidak saling mengimpor.
		//
		// DUA field diambil, berbeda dari modul yang hanya membaca: jejak permintaan
		// menyimpan NAMA pemohon bersama login-nya, supaya jejak itu tetap terbaca utuh
		// tanpa join ke tabel pengguna. Jejak yang namanya diambil lewat join akan berubah
		// ketika orangnya berganti nama — dan jejak yang dapat berubah bukan jejak.
		GetCaller: func(ctx context.Context) (inboxcloseclaimhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inboxcloseclaimhttp.Caller{}, false
			}
			return inboxcloseclaimhttp.Caller{
				Login: baseCtx.User.Login,
				Name:  baseCtx.User.Name,
			}, true
		},
		Logger:    logger,
		WriteJSON: writeJSON,
		// Galat portal dipetakan modul portal lebih dulu, sisanya jatuh ke pemeta galat
		// auth — rantai yang sama dipakai modul Inbox Outstanding tepat di atasnya.
		FallbackErrorWriter: inboxcloseclaimhttp.ErrorWriter(writePortalAwareError),
	})

	// View History Claim. Jembatan pemanggilnya membawa LOGIN, bukan NIK: itulah yang
	// dicocokkan ke kolom LOGIN pada POOLDATA.MST_PROTEKSI_DATA_PNC, dan memakai NIK di
	// sini akan membuat setiap pengguna tampak belum terdaftar di gerbang proteksi.
	claimHistoryHandler := riwayatklaimhttp.NewHandler(riwayatklaimhttp.Options{
		Service: assembly.riwayatKlaim,
		GetCaller: func(ctx context.Context) (riwayatklaimhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return riwayatklaimhttp.Caller{}, false
			}
			return riwayatklaimhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:    logger,
		WriteJSON: writeJSON,
		// Galat portal ikut dikenali, karena seluruh rute modul ini berada di balik
		// pemeriksaan portal.
		FallbackErrorWriter: riwayatklaimhttp.ErrorWriter(writePortalAwareError),
	})

	// Archive Dokumen Klaim. Jembatan pemanggilnya membawa TIGA hal, dan ketiganya
	// dipakai untuk keperluan yang berbeda:
	//
	//	Login       mengisi kolom USERINPUT — siapa yang mengarsipkan berkasnya
	//	Position    menentukan lini bisnis yang tampak di daftar kirim ke cabang
	//	BranchCode  mengisi kolom KODECABANG
	//
	// Yang kedua itu KENDALI AKSES, bukan kenyamanan tampilan: penyaringannya ditegakkan
	// di server, sejalan dengan `D-59`.
	archiveDocumentHandler := archivedokumenklaimhttp.NewHandler(archivedokumenklaimhttp.Options{
		Service: assembly.archiveDokumenKlaim,
		GetCaller: func(ctx context.Context) (archivedokumenklaimhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return archivedokumenklaimhttp.Caller{}, false
			}
			return archivedokumenklaimhttp.Caller{
				Login:      baseCtx.User.Login,
				Position:   baseCtx.User.Position,
				BranchCode: baseCtx.User.BranchCode,
			}, true
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: archivedokumenklaimhttp.ErrorWriter(writePortalAwareError),
	})

	accountHandler := masterrekeninghttp.NewHandler(masterrekeninghttp.Options{
		// Adapter dari pemilih layanan bertipe konkret menjadi pemilih bertipe antarmuka.
		// Galatnya dikembalikan lebih dulu, bukan dibungkus: nil bertipe *Service yang
		// terlanjur masuk ke antarmuka akan terbaca sebagai layanan yang ada padahal
		// tidak.
		ServiceSelector: func(alias string) (masterrekeninghttp.Service, error) {
			service, err := assembly.masterRekening(alias)
			if err != nil {
				return nil, err
			}
			return service, nil
		},
		// Jembatan satu arah dari modul auth ke modul master rekening. Ia dipasang di
		// sini, bukan di dalam salah satu modul, supaya kedua modul tetap tidak saling
		// mengimpor — yang tahu keduanya hanyalah berkas perakitan ini.
		Caller: func(ctx context.Context) (masterrekeninghttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterrekeninghttp.Caller{}, false
			}
			return masterrekeninghttp.Caller{
				Identity: baseCtx.User.Identity,
				Name:     baseCtx.User.Name,
				Email:    baseCtx.User.Email,
			}, true
		},
		Logger: logger,
	})

	router := httpserver.Router(httpserver.Deps{
		Logger:   logger,
		SPAFiles: spaFiles,
		MountAPI: func(api chi.Router) {
			// Setiap modul memasang rutenya sendiri di sini. Modul berikutnya cukup
			// menambah satu baris; server tidak perlu tahu isinya.
			authhttp.Mount(api, handlerAuth, assembly.auth, logger)

			// List portal berada di balik sesi: pemilihnya ada di dalam aplikasi,
			// bukan di layar masuk (ADR-0030, berpindah portal tanpa login ulang).
			api.Group(func(protected chi.Router) {
				protected.Use(authhttp.Authenticate(assembly.auth, authhttp.WriteError(logger)))
				portalhttp.Mount(protected, handlerPortal)

				// Menu berada di balik sesi tetapi TIDAK di balik pemeriksaan portal:
				// peta menu dan kewenangannya hidup di basis data portal utama dan tidak
				// punya kolom entitas. Menuntut portal di sini akan membuat menunya gagal
				// justru saat pengguna belum memilih entitas.
				menuhttp.Mount(protected, menuHandler)

				// Master Status Progres 1. Rutenya memasang pemeriksaan portal sendiri
				// di dalam Mount — hanya pada rute yang benar-benar menyentuh basis
				// data entitas.
				masterstatusprogreshttp.Mount(protected, progressStatusHandler, activePortalDeps)
				// Master Status Progres 2. SELURUH rutenya dipasangi pemeriksaan portal —
				// termasuk daftar induknya, yang dibaca dari tabel tingkat 1 milik entitas
				// yang bersangkutan, bukan daftar tetap milik aplikasi.
				masterstatusprogreshttp.Mount2(protected, progressStatus2Handler, activePortalDeps)
				// Master Penolakan Klaim. SATU pemasangan untuk DUA tab — Penolakan
				// Klaim dan Penolakan Komite — karena keduanya satu layar dan satu
				// butir menu (MENU_ID 25). Seluruh rutenya dipasangi pemeriksaan
				// portal; tidak ada satu pun yang isinya milik aplikasi.
				masterpenolakanhttp.Mount(protected, rejectionHandler, activePortalDeps)
				// Master Auto Claim. SELURUH rutenya dipasangi pemeriksaan portal —
				// master maupun ketiga lookup-nya dibaca dari basis data entitas, dan
				// dua entitas punya sumber bisnis serta daftar client yang berbeda.
				masterautoclaimhttp.Mount(protected, masterAutoClaimHandler, activePortalDeps)
				// Master Bengkel. SELURUH rutenya dipasangi pemeriksaan portal —
				// POOLDATA.BENGKEL_HE dan ketiga tabel acuannya ada di basis data setiap
				// entitas, dan dua entitas punya daftar cabang yang berbeda.
				masterbengkelhttp.Mount(protected, workshopHandler, activePortalDeps)
				// Master Panel. Rutenya memasang pemeriksaan portal sendiri di dalam
				// Mount — SELURUHNYA kecuali daftar pilihan Lokasi dan Sisi, yang
				// isinya konstanta yang ditanam di activity Pega dan bukan dibaca dari
				// basis data entitas mana pun.
				masterpanelhttp.Mount(protected, panelHandler, activePortalDeps)
				// Master Sparepart. SELURUH rutenya dipasangi pemeriksaan portal —
				// termasuk daftar pilihannya, yang berbeda dari Master Panel: kategori
				// dan tipe suku cadang dibaca dari basis data entitas, bukan dari
				// konstanta yang ditanam di activity Pega.
				masterspareparthttp.Mount(protected, sparepartHandler, activePortalDeps)
				// Master Grouping Sparepart. SELURUH rutenya dipasangi pemeriksaan
				// portal — kedua tabel groupingnya DAN keempat sumber acuannya
				// (PANEL_HE, LOKASI_PANEL_HE, SPAREPART_HE, branddetail) ada di basis
				// data setiap entitas. Tidak ada satu pun rutenya yang isinya konstanta
				// milik aplikasi.
				mastergroupingspareparthttp.Mount(protected, groupingHandler, activePortalDeps)
				// Master Kategori Sparepart. SELURUH rutenya dipasangi pemeriksaan
				// portal — tabelnya ada di basis data setiap entitas, dan tidak ada
				// satu pun rutenya yang isinya milik aplikasi.
				//
				// Ia MENULIS tabel yang dibaca Master Sparepart di atas sebagai daftar
				// acuan. Satu tabel, satu penulis (P-1).
				masterkategorispareparthttp.Mount(protected, partCategoryHandler,
					activePortalDeps)
				// Master Tipe Sparepart. SELURUH rutenya dipasangi pemeriksaan portal —
				// tabelnya ada di basis data setiap entitas, dan `/pilihan` membaca
				// tabel kategori entitas itu juga.
				//
				// Ia MENULIS tabel yang dibaca Master Sparepart sebagai daftar acuan
				// Tipe, dan MEMBACA tabel yang ditulis Master Kategori Sparepart di
				// atas. Satu tabel, satu penulis (P-1).
				mastertipespareparthttp.Mount(protected, partTypeHandler,
					activePortalDeps)
				// Master Login. SELURUH rutenya dipasangi pemeriksaan portal —
				// POOLDATA.MST_LOGIN_SURVEYOR ada di basis data setiap entitas, dan
				// isinya menentukan siapa yang boleh bekerja sebagai surveyor pada
				// badan hukum itu. Tidak ada satu pun rutenya yang isinya milik
				// aplikasi: kelima isiannya diketik bebas, dan dua kolom sisanya
				// diturunkan server.
				masterloginhttp.Mount(protected, surveyorLoginHandler,
					activePortalDeps)
				// Detail Penyebab Kerugian. Rutenya memasang pemeriksaan portal sendiri
				// di dalam Mount — SELURUHNYA kecuali daftar Status Aktif, yang isinya
				// konstanta domain dan sama di keempat portal.
				//
				// Yang dipisahkan pemeriksaan itu bukan sekadar daftar acuan: sebab
				// kerugian yang dapat dipilih menentukan bagaimana klaim dinilai pada
				// badan hukum itu.
				detailpenyebabhttp.Mount(protected, causeOfLossDetailHandler,
					activePortalDeps)
				// Master Pasal Kerugian. Rutenya memasang pemeriksaan portal sendiri di
				// dalam Mount — SELURUHNYA kecuali daftar Kategori, yang isinya milik
				// aplikasi dan bukan dibaca dari basis data entitas mana pun.
				masterpasalhttp.Mount(protected, clauseHandler, activePortalDeps)

				// Master Pasal AI (MENU_ID 36). Baca-saja, satu rute.
				masterpasalaihttp.Mount(protected, clauseAIHandler, activePortalDeps)

				// Laporan Hasil AI (MENU_ID 82), kelompok menu REPORT. Baca-saja, dua
				// rute: satu menjawab JSON untuk layar, satu menjawab CSV untuk unduhan.
				//
				// Di Pega keduanya SATU activity — tombol "Export To Excel" membuka
				// `SearchDataLaporanAI(flagss=2)` yang sama di jendela baru. Dipisah di
				// sini karena bentuk keluarannya memang dua hal yang berbeda.
				laporanhasilaihttp.Mount(protected, aiReportHandler, activePortalDeps)
				// Inbox Investigator (MENU_ID 48). Modul INBOX pertama, dan modul
				// pertama yang berada di bawah awalan `/inbox/...` — kelompok menu
				// tersendiri di sistem lama (`MENU_ID 2`, induk dari 30 butir).
				//
				// SELURUH rutenya dipasangi pemeriksaan portal: antrean pekerjaan ada
				// di basis data setiap entitas, dan barisnya memuat nama tertanggung
				// serta nama peserta klaim badan hukum itu.
				inboxinvestigatorhttp.Mount(protected, investigatorInboxHandler,
					activePortalDeps)
				// Inbox Receive TKA (MENU_ID 49). Modul INBOX kedua, dan yang
				// PERTAMA yang menulis.
				//
				// SELURUH rutenya dipasangi pemeriksaan portal, dan pada rute
				// tulisnya itu lebih dari sekadar mencegah kebocoran baca:
				// tanpa pemeriksaan portal, satu permintaan dapat mengubah
				// `TGLDOKLENGKAP` pada klaim milik badan hukum lain (`R-20`).
				inboxreceivetkahttp.Mount(protected, receiveTKAHandler,
					activePortalDeps)
				// Master Reas. SELURUH rutenya dipasangi pemeriksaan portal —
				// POOLDATA.T_REINSURER ada di basis data SETIAP entitas, dan isinya
				// menentukan mitra reasuransi mana yang menerima pemberitahuan klaim
				// badan hukum itu beserta surel tujuannya. Hanya rute BACA yang
				// terdaftar; lihat banner paket masterreas.
				masterreashttp.Mount(protected, reinsurerMemberHandler,
					activePortalDeps)
				// Master Supplier. SELURUH rutenya dipasangi pemeriksaan portal —
				// M_SUPPLIER dan keempat tabel acuannya ada di basis data setiap
				// entitas, dan dua entitas punya daftar cabang serta supplier yang
				// berbeda. Tidak ada satu pun rutenya yang isinya milik aplikasi.
				mastersupplierhttp.Mount(protected, supplierHandler, activePortalDeps)
				// Master Dokumen Travel. Seluruh rutenya menyentuh basis data
				// entitas, sehingga pemeriksaan portal dipasang atas semuanya.
				masterdokumentravelhttp.Mount(protected, travelDocumentHandler, activePortalDeps)
				// Master COL Simas Online. Ia juga memasang /master/bisnis — daftar
				// acuan milik GISFW yang hanya dibaca. Bila kelak ada modul Master
				// Bisnis tersendiri, rute itu pindah ke sana; chi akan panik saat start
				// bila keduanya mendaftarkannya bersamaan, dan itu justru yang membuat
				// kekeliruan itu mustahil lolos diam-diam.
				mastercolhttp.Mount(protected, simasOnlineCauseOfLossHandler, activePortalDeps)
				// Daftar Tipe Dokumen. Seluruh rutenya menyentuh basis data entitas,
				// sehingga pemeriksaan portal dipasang atas semuanya.
				daftartipedokumenhttp.Mount(protected, documentTypeHandler, activePortalDeps)
				// Daftar Objek Dokumen. Seluruh rutenya menyentuh basis data entitas.
				// Ia TIDAK mendaftarkan /master/bisnis — rute itu sudah dimiliki Master
				// COL Simas Online di atas, dan layar modul ini memakainya bersama.
				daftarobjekdokumenhttp.Mount(protected, documentObjectHandler, activePortalDeps)
				// Daftar Detail Dokumen Travel. Ia juga memasang
				// /master/dokumen-travel-pilihan dan /master/plan-travel — dua daftar
				// acuan yang tabelnya dimiliki modul lain dan tim lain, dan hanya
				// dibacanya. Bila kelak ada modul Master Plan Travel tersendiri, rute
				// kedua pindah ke sana; chi akan panik saat start bila keduanya
				// mendaftarkannya bersamaan, dan itu justru yang membuat kekeliruan itu
				// mustahil lolos diam-diam.
				daftardetaildokumentravelhttp.Mount(protected, travelDocumentDetailHandler, activePortalDeps)
				// Daftar Detail Tipe Dokumen (MENU_ID 41). Keempat daftar acuannya
				// dikirim lewat SATU rute di bawah sub-rutenya sendiri —
				// /master/detail-tipe-dokumen/pilihan — bukan sebagai empat rute
				// sejajar seperti modul di bawah.
				//
				// Bedanya bukan selera: yang dikembalikan bukan salah satu master
				// melainkan GABUNGAN keempatnya dalam bentuk yang hanya berarti bagi
				// form itu, sehingga ia tidak menyiratkan kepemilikan tabel mana pun.
				// Form-nya pun cukup satu permintaan, bukan empat.
				daftardetailtipedokumenhttp.Mount(protected, detailDocumentTypeHandler, activePortalDeps)
				// Daftar Tipe Dokumen Bisnis. Ia juga memasang EMPAT daftar acuan —
				// /master/bisnis-pilihan, /master/tipe-dokumen-pilihan,
				// /master/detail-dokumen-pilihan, dan /master/objek-dokumen-pilihan —
				// yang keempat tabelnya dimiliki modul lain dan tim lain, dan hanya
				// dibacanya.
				//
				// Akhiran `-pilihan` membuat keempatnya tidak bertabrakan dengan jalur
				// CRUD milik pemiliknya: /master/bisnis sudah dipakai Master COL Simas
				// Online, /master/tipe-dokumen oleh Daftar Tipe Dokumen, dan
				// /master/objek-dokumen oleh Daftar Objek Dokumen. Bila kelak salah
				// satunya dipindahkan, chi akan panik saat start — dan itu justru yang
				// membuat kekeliruan itu mustahil lolos diam-diam.
				daftartipedokumenbisnishttp.Mount(protected, businessDocumentRuleHandler, activePortalDeps)

				// Inbox Auto Claim memuat nomor polis, nilai klaim, dan nama
				// perusahaan rekanan; tidak satu pun boleh terbaca tanpa sesi.
				inboxautoclaimhttp.Mount(protected, autoClaimHandler, activePortalDeps)

				// View History Claim memuat nama tertanggung, nomor polis, dan tanggal
				// lahir peserta — satu pencarian dapat mengembalikan seluruh riwayat
				// klaim seorang nasabah. Selain sesi, ia dijaga gerbang proteksi data
				// yang jatahnya berkurang tiap kali layar dibuka.
				riwayatklaimhttp.Mount(protected, claimHistoryHandler, activePortalDeps)

				// Archive Dokumen Klaim memuat nomor klaim, nomor polis, dan nama
				// tertanggung, dan salah satu rutenya MENGIRIM berkas ke sistem milik
				// tim lain. Seluruh rutenya memasang pemeriksaan portal di dalam Mount.
				archivedokumenklaimhttp.Mount(protected, archiveDocumentHandler, activePortalDeps)
				// Inbox Laporan Klaim. Seluruh rutenya memasang pemeriksaan portal di
				// dalam Mount — tidak satu pun yang boleh dilayani tanpa entitas yang
				// jelas, karena setiap rutenya menyentuh basis data entitas.
				inboxlaporanklaimhttp.Mount(protected, claimReportHandler, activePortalDeps)
				inboxoutstandinghttp.Mount(protected, outstandingHandler, activePortalDeps)
				inputreqprotectionhttp.Mount(protected, protectionRequestHandler, activePortalDeps)
				inboxacceptopenprotectionhttp.Mount(protected, protectionAcceptHandler, activePortalDeps)
				// Inbox Close Claim — klaim yang sudah tutup, beserta permintaan
				// membukanya kembali dan menyalinnya.
				//
				// Satu-satunya modul inbox yang MENULIS. Yang ditulisnya bukan klaim
				// melainkan POOLDATA.CPNC_PERMINTAAN_KLAIM, tabel milik aplikasi ini
				// sendiri — `P-1` menetapkan klaim masih ditulis Pega selama masa paralel.
				//
				// Kewenangannya belum diperiksa per peran (TKT-F3-005). Di Pega, butir
				// menunya dijaga When rule yang membukanya bagi empat access group
				// ditambah TIGA Operator ID perorangan yang tertanam di dalam rule —
				// persis jenis hardcode yang D-15 hapus.
				inboxcloseclaimhttp.Mount(protected, closeClaimHandler, activePortalDeps)
				// Master Tipe Surveyors. Sama seperti di atas: pemeriksaan portal
				// dipasang di dalam Mount, karena SELURUH rutenya menyentuh basis
				// data entitas.
				mastertipesurveyorshttp.Mount(protected, surveyorTypeHandler, activePortalDeps)
				// Master Surveyors — daftar ORANGNYA. Barisnya memuat nama, alamat,
				// telepon, surel, dan nama login aplikasi seseorang; tidak satu pun
				// boleh terbaca tanpa sesi. Keputusan komite di dalamnya diperiksa
				// terhadap kolom KOMITE, dan itu satu-satunya kontrol kewenangan yang
				// benar-benar ada selama TKT-F3-005 belum dikerjakan (D-59).
				mastersurveyorshttp.Mount(protected, surveyorHandler, activePortalDeps)
				// Master PIC Teknik. Sama seperti di atas — seluruh rutenya menyentuh
				// entitas, termasuk pencarian direktori yang alamat layanannya pun dibaca
				// per entitas.
				masterpicteknikhttp.Mount(protected, picTeknikHandler, activePortalDeps)
				// Master Recovery. Sama seperti di atas, dengan satu hal yang lebih
				// berat: rutenya MENERBITKAN REKENING VIRTUAL dan MENCATAT NILAI UANG,
				// sehingga portal yang keliru bukan sekadar menampilkan data yang salah
				// — ia dapat mengarahkan dana ke rekening badan hukum lain (R-20).
				masterrecoveryhttp.Mount(protected, recoveryHandler, activePortalDeps)
				// Master rekening memuat nama, NIK, nomor rekening, dan surel pihak
				// ketiga; tidak satu pun boleh terbaca tanpa sesi.
				masterrekeninghttp.Mount(protected, accountHandler, activePortalDeps)
				// Master data juga berada di balik sesi. Pemeriksaan peran — "apakah
				// pemanggil memiliki menu Master Data" (D-59) — belum ada di sini
				// karena TKT-F3-004 dan TKT-F3-005 belum dikerjakan; keadaannya sama
				// dengan seluruh rute lain hari ini.
				masterstatushttp.Mount(protected, handlerMasterStatus, activePortalDeps)
				// Master Dominan Factor. Sama seperti di atas — seluruh rutenya
				// menyentuh basis data entitas. Satu hal yang membedakannya: nama
				// faktor di sini ikut terbaca LAPORAN Outstanding per Cabang lewat
				// LISTAGG, sehingga portal yang keliru mengubah isi laporan, bukan
				// hanya tampilan satu layar.
				masterdominanfactorhttp.Mount(protected, dominantFactorHandler, activePortalDeps)
				// Master XOL. Seluruh rutenya menyentuh basis data entitas, dan di modul
				// ini akibat salah entitas menjalar jauh: struktur treaty menentukan
				// pembagian klaim ke para reasuradur, sehingga limit dan share satu badan
				// hukum yang terbaca — apalagi tersimpan — di badan hukum lain akan
				// mengubah hasil perhitungan PLA/DLA (R-20).
				masterxolhttp.Mount(protected, xolHandler, activePortalDeps)
				// Master Penyebab Kerugian — TINGKAT GOLONGAN saja (MENU_ID 20).
				// Rinciannya (MENU_ID 38) butir menu tersendiri dan belum dibangun.
				// Keterangan di sini dibaca 19 rule Pega dan menjadi kolom
				// pengelompokan pada laporan, sehingga portal yang keliru mengubah
				// pengelompokan laporan entitas lain (`R-20`).
				masterpenyebabkerugianhttp.Mount(protected, causeOfLossHandler, activePortalDeps)
				// Master Masking. Portal yang keliru di sini berakibat paling berat
				// di antara seluruh master yang sudah dibangun: yang tampil bukan
				// daftar kode, melainkan daftar siapa yang boleh membuka nomor KTP
				// dan nomor telepon nasabah badan hukum lain (`R-20`).
				mastermaskinghttp.Mount(protected, maskingHandler, activePortalDeps)

				// Sebelas modul sisanya dirakit di modules.go. Kegagalan di sini tidak
				// dapat dikembalikan sebagai galat — MountAPI tidak mengembalikan apa
				// pun — sehingga ia dinaikkan menjadi panic yang ditangkap saat start.
				// Aplikasi yang menyala dengan sebagian rute diam-diam hilang jauh lebih
				// berbahaya daripada aplikasi yang menolak menyala.
				if err := mountExtra(
					protected, assembly.extra, activePortalDeps,
					writeJSON, writePortalAwareError, logger,
				); err != nil {
					panic(fmt.Errorf("modul tambahan gagal dipasang: %w", err))
				}
				// Modul Komite memasang tiga kelompok rute sekaligus: master ambang di
				// bawah master/, perhitungan penjenjangan di bawah komite/, dan Inbox
				// Komite di bawah komite/inbox.
				//
				// Dua yang pertama DIBACA SAJA — tidak ada satu pun jalur yang menulis ke
				// POOLDATA.EMAILKOMITE selama masa paralel (P-1, keputusan Work Owner
				// 2026-09-17).
				//
				// Yang ketiga MENULIS, dan hanya ke satu tempat: tabel keputusan milik
				// aplikasi ini sendiri (migrasi 0004). Kasusnya tetap dibaca saja dari
				// tabel warisan.
				//
				// Isi layar ini memperlihatkan siapa yang berwenang menyetujui uang, dan
				// setiap barisnya memuat nilai klaim serta nama tertanggung. Tidak satu
				// pun boleh terbaca tanpa sesi.
				komitehttp.Mount(protected, handlerKomite, handlerKomiteInbox)

				// Inbox XOL memuat nilai klaim agregat satu perjanjian reasuransi,
				// nama reasuradur, dan alamat surelnya. Tidak satu pun boleh terbaca
				// tanpa sesi, dan seluruhnya dijaga pemeriksaan portal.
				//
				// Ia MEMBACA SAJA (keputusan Work Owner 2026-09-20): keempat tabel XOL
				// yang ditulis sistem lama tetap dimiliki Pega selama masa paralel
				// (`P-1`). Bedakan dari Master XOL di atas, yang MENULIS struktur
				// treaty-nya — keduanya menyentuh MST_XOL_PNC dan kerabatnya, dan hanya
				// satu di antaranya yang boleh menulis.
				inboxxolhttp.Mount(protected, inboxXOLHandler, activePortalDeps)

				// Inbox Claim Treaty Prop memuat nama tertanggung dan nama Ceding Co —
				// perusahaan asuransi yang mengalihkan risikonya kepada ASM. Keduanya
				// milik satu badan hukum, sehingga seluruh rutenya dijaga pemeriksaan
				// portal, termasuk rute keterangan layarnya.
				//
				// Ia MEMBACA SAJA (keputusan Work Owner 2026-09-21): pembuatan klaim
				// treaty menulis objek kerja di tabel yang masih dimiliki Pega selama
				// masa paralel (`P-1`).
				inboxclaimtreatypropthttp.Mount(
					protected, claimTreatyPropHandler, activePortalDeps)

				// Inbox Claim Treaty Non Prop memuat data yang sama sifatnya —
				// nama tertanggung dan nama Ceding Co milik satu badan hukum —
				// sehingga rutenya dijaga pemeriksaan portal yang sama.
				//
				// Ia punya satu rute yang tidak dimiliki layar Prop: ekspor berkas.
				// Berkas itu memuat data nasabah, dan justru karena ia terunduh ke
				// perangkat pengguna, pemeriksaan portalnya tidak boleh lebih longgar
				// daripada layarnya.
				inboxclaimtreatynonprophttp.Mount(
					protected, claimTreatyNonPropHandler, activePortalDeps)

				// Inbox Manager Receive / PUCL memuat nomor polis dan nama
				// tertanggung dari DUA antrean sekaligus, dan tidak satu pun
				// tabnya menyaring menurut pemanggil — ia memang pandangan
				// penyelia. Justru karena itu pemeriksaan portalnya tidak boleh
				// lebih longgar: yang terlihat di sini adalah seluruh berkas dan
				// seluruh klaim RCL/PUCL milik satu badan hukum.
				inboxmanagerreceivepuclhttp.Mount(
					protected, managerReceivePUCLHandler, activePortalDeps)

				// Inbox RCL/PUCL memuat nomor polis dan nama tertanggung dari
				// antrean BERSAMA — tidak satu pun tabnya menyaring menurut
				// pemanggil, karena penyaringnya akun antrean. Pemeriksaan
				// portalnya karena itu tidak boleh lebih longgar: yang terlihat
				// di sini adalah seluruh klaim RCL/PUCL milik satu badan hukum.
				//
				// Rute ekspornya menuntut hal yang sama dan sedikit lebih:
				// berkas laporan hariannya dapat diunduh dan dibawa keluar,
				// dengan rentang tanggal yang ditentukan penggunanya sendiri.
				inboxrclpuclhttp.Mount(
					protected, rclPUCLHandler, activePortalDeps)

				// Report KPI PNC memuat penilaian kinerja adjuster yang bekerja untuk
				// SATU badan hukum. Rutenya karena itu menuntut portal — termasuk rute
				// keterangan layarnya, supaya layar tidak tergambar separuh sebelum
				// penolakannya sampai.
				reportkpihttp.Mount(protected, reportKPIHandler, activePortalDeps)
				// Katalognya pun menuntut portal, supaya layar tidak tergambar lalu
				// tombolnya ditolak setelah pengguna mengisi rentang tanggal.
				reportklaimhttp.Mount(protected, reportKlaimHandler, activePortalDeps)
				// Inbox Komunikasi Cabang memuat percakapan antarpetugas tentang
				// klaim yang sedang berjalan — milik satu badan hukum, bukan milik
				// badan hukum lain. Rutenya menuntut portal DAN disaring cabang;
				// batas kedua itu diselesaikan di dalam modulnya, bukan di sini.
				inboxkomunikasicabanghttp.Mount(
					protected, komunikasiCabangHandler, activePortalDeps)

				// Inbox Salvage memuat nomor klaim DAN nilai uang — nilai pengajuan
				// PIC, nilai request balai lelang, nilai penawaran. Rutenya menuntut
				// portal karena alasan yang sama dengan modul inbox lain, dan satu
				// alasan tambahan yang khas modul ini: ia MENULIS. Permintaan simpan
				// yang jatuh ke koneksi bawaan tidak sekadar menampilkan data entitas
				// lain — ia menyisipkan baris ke dalamnya (`R-20`).
				inboxsalvagehttp.Mount(
					protected, salvageHandler, activePortalDeps)

				// Inbox Progress Claim memuat nama tertanggung, nomor polis, dan
				// catatan progres — seluruhnya milik satu badan hukum. Rutenya karena
				// itu menuntut portal, sama seperti Inbox Admin.
				inboxprogressclaimhttp.Mount(
					protected, inboxProgressClaimHandler, activePortalDeps)

				// Inbox Analyst Doctor memuat nama tertanggung dan klaim Personal
				// Accident. Rutenya menuntut portal karena alasan yang sama dengan
				// modul di atasnya, ditambah satu yang khas: `FR-R2` membatasi akses
				// data medis, dan pembatasan itu tidak bermakna bila datanya datang
				// dari entitas yang salah.
				inboxanalystdoctorhttp.Mount(
					protected, inboxAnalystDoctorHandler, activePortalDeps)
			})
		},
	})

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	logger.Info("server menyala", slog.String("alamat", cfg.Address))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server berhenti: %w", err)
	}
	return nil
}

// assembly memegang seluruh modul yang sudah terpasang beserta cara menutupnya.
type assembly struct {
	auth   *usecase.Service
	portal portal.Repo
	// masterRekening memilih layanan milik satu portal entitas; sejak 2026-09-19
	// POOLDATA.LST_ACCOUNT dibaca dan ditulis per entitas, bukan dari portal utama saja.
	masterRekening func(string) (*masterrekeningusecase.Service, error)
	masterStatus   *masterstatususecase.Service

	// komite membaca master ambang dan menghitung penjenjangan persetujuan (B-7).
	komite *komiteusecase.Service

	// komiteInbox melayani layar Inbox Komite: daftar pekerjaan anggota komite dan
	// pencatatan keputusannya (`TKT-B07-002`, MENU_ID 52).
	komiteInbox *komiteusecase.InboxService

	// masterStatusProgres memakai pemilih repo per portal, bukan repo tunggal:
	// tabelnya ada di basis data SETIAP entitas (ADR-0030).
	masterStatusProgres *masterstatusprogresusecase.Service

	// masterDokumenTravel memakai pemilih repo per portal dengan alasan yang sama:
	// POOLDATA.M_DOCTRAVEL adalah data acuan milik satu badan hukum, dan setiap
	// entitas punya basis datanya sendiri.
	masterDokumenTravel *masterdokumentravelusecase.Service

	// masterCOLSimasOnline memakai pemilih repo per portal dengan alasan yang sama —
	// POOLDATA.M_CAUSE_OF_LOSS ada di basis data setiap entitas — dan ditambah satu
	// pemilih lagi untuk master bisnis milik GISFW yang hanya dibacanya.
	masterCOLSimasOnline *mastercolusecase.Service

	// daftarTipeDokumen memakai pemilih repo per portal dengan alasan yang sama:
	// POOLDATA.LST_DOC_TYPE ada di basis data setiap entitas, dan ID-nya diterbitkan
	// dari kode situs milik basis data itu (`PEGA_LST_DOC_TYPE.prc:12`).
	daftarTipeDokumen *daftartipedokumenusecase.Service

	// daftarObjekDokumen memakai DUA pemilih per portal: satu untuk tabelnya sendiri,
	// satu untuk POOLDATA.BUSINESS milik GISFW yang hanya dibacanya — bentuk yang sama
	// persis dengan masterCOLSimasOnline di atas, karena grid "ID Bisnis" pada kedua layar
	// membaca master yang sama.
	daftarObjekDokumen *daftarobjekdokumenusecase.Service

	// daftarDetailDokumenTravel memakai TIGA pemilih per portal: satu untuk kedua
	// tabelnya sendiri, satu untuk POOLDATA.M_DOCTRAVEL milik modul Master Dokumen
	// Travel, dan satu untuk POOLDATA.M_PLANTRAVEL milik GISFW. Ketiganya hidup di basis
	// data entitas yang sama, tetapi mengisi seam yang berbeda.
	daftarDetailDokumenTravel *daftardetaildokumentravelusecase.Service

	// masterGroupingSparepart melayani layar Master Grouping Sparepart — penautan suku
	// cadang ke panel bodi pada sebuah kendaraan, dikelompokkan menurut nomor rangka. Ia
	// memakai pemilih repo per portal dengan alasan yang sama seperti master lain:
	// POOLDATA.SPAREPART_HE_VIN_KEY, POOLDATA.SPAREPART_HE_VIN_GROUP, beserta keempat sumber
	// acuannya ada di basis data SETIAP entitas (ADR-0030).
	masterGroupingSparepart *mastergroupingsparepartusecase.Service

	// masterKategoriSparepart melayani layar Master Kategori Sparepart — penggolongan suku
	// cadang yang menjadi acuan Master Sparepart dan Master Tipe Sparepart. Ia memakai
	// pemilih repo per portal dengan alasan yang sama seperti master lain:
	// POOLDATA.GCNM_M_SPAREPART_CATEGORY ada di basis data SETIAP entitas (ADR-0030).
	masterKategoriSparepart *masterkategorisparepartusecase.Service

	// masterTipeSparepart melayani layar Master Tipe Sparepart — penggolongan tingkat kedua
	// di bawah kategori, yang menjadi acuan Tipe pada Master Sparepart. Ia memakai pemilih
	// repo per portal dengan alasan yang sama seperti master lain:
	// POOLDATA.GCNM_M_SPAREPART_TYPE ada di basis data SETIAP entitas (ADR-0030).
	masterTipeSparepart *mastertipesparepartusecase.Service

	// masterPasal melayani layar Master Pasal Kerugian — daftar baku butir ketentuan polis
	// yang dirujuk saat klaim dinilai. Ia memakai pemilih repo per portal dengan alasan
	// yang sama seperti master lain: POOLDATA.V_M_DATA_PASAL dan POOLDATA.BUSINESS ada di
	// basis data SETIAP entitas (ADR-0030).
	masterPasal *masterpasalusecase.Service

	// masterPasalAI melayani layar Master Pasal AI — daftar wording polis yang dipakai
	// penilaian AI atas klaim. Ia BACA-SAJA: layar lamanya tidak punya jalur tulis sama
	// sekali.
	//
	// Pemilih repo per portal dengan alasan yang sama seperti master lain: tabelnya ada di
	// basis data SETIAP entitas (ADR-0030).
	masterPasalAI *masterpasalaiusecase.Service

	// laporanHasilAI melayani layar Laporan Hasil AI (MENU_ID 82) — perbandingan penilaian
	// AI dengan keputusan komite yang menyusul. Ia BACA-SAJA: kedua tombol layar lamanya
	// memanggil activity yang sama, dan activity itu tidak memuat satu pun langkah tulis.
	//
	// Pemilih repo per portal: kedua tabelnya memuat nama tertanggung dan keputusan uang,
	// dan keduanya ada di basis data SETIAP entitas (ADR-0030).
	laporanHasilAI *laporanhasilaiusecase.Service

	// masterSupplier melayani layar Master Supplier — daftar supplier rekanan beserta
	// syarat dagangnya. Ia memakai pemilih repo per portal dengan alasan yang sama
	// seperti master lain: M_SUPPLIER dan keempat tabel acuannya ada di basis data
	// SETIAP entitas (ADR-0030).
	masterSupplier *mastersupplierusecase.Service

	// masterLogin melayani layar Master Login (MENU_ID 37) atas
	// POOLDATA.MST_LOGIN_SURVEYOR. Ia memakai pemilih repo per portal dengan alasan yang
	// sama seperti master lainnya — tabelnya ada di basis data SETIAP entitas (ADR-0030) —
	// ditambah satu yang khas modul ini: isinya menentukan siapa yang boleh bekerja sebagai
	// surveyor pada badan hukum itu.
	masterLogin *masterloginusecase.Service

	// detailPenyebab melayani layar Detail Penyebab Kerugian (MENU_ID 38) atas
	// POOLDATA.D_CAUSE_OF_LOSS — rincian di bawah Master Penyebab Kerugian, yang menjadi
	// pilihan petugas saat klaim diregistrasi. Ia memakai pemilih repo per portal dengan
	// alasan yang sama seperti master lainnya: tabelnya ada di basis data SETIAP entitas
	// (ADR-0030).
	detailPenyebab *detailpenyebabusecase.Service

	// inboxInvestigator melayani layar Inbox Investigator (MENU_ID 48) — antrean pekerjaan
	// yang menunggu di workbasket `InvestigatorPNC`.
	//
	// Modul INBOX pertama di aplikasi ini; pembedaan inbox dari layar master dan layar
	// pencarian ditetapkan `D-79`. Ia memakai pemilih repo per portal dengan alasan yang
	// sama seperti modul lain, dan di sini taruhannya termasuk yang tertinggi: antrean satu
	// badan hukum memuat nama tertanggung dan nama peserta klaimnya (ADR-0030, R-20).
	inboxInvestigator *inboxinvestigatorusecase.Service

	// inboxReceiveTKA melayani layar Inbox Receive TKA (MENU_ID 49) — klaim TKA yang
	// tanggal kelengkapan dokumennya belum diisi.
	//
	// Modul INBOX kedua, dan yang PERTAMA yang menulis. Ia mengubah `TGLDOKLENGKAP` pada
	// data klaim, sehingga taruhan pemilih repo per portal di sini melampaui kebocoran
	// baca: portal yang keliru berarti mengubah tanggal pada klaim milik badan hukum lain
	// (ADR-0030, R-20).
	inboxReceiveTKA *inboxreceivetkausecase.Service

	// masterReas melayani layar Master Reas (MENU_ID 35) atas POOLDATA.T_REINSURER —
	// daftar mitra reasuransi penerima pemberitahuan PLA, Pre-DLA, dan DLA. Ia memakai
	// pemilih repo per portal dengan alasan yang sama seperti master lainnya: tabelnya ada
	// di basis data SETIAP entitas (ADR-0030).
	//
	// Satu-satunya layanan master yang HANYA MEMBACA; alasannya ada pada banner paket
	// masterreas.
	masterReas *masterreasusecase.Service

	// masterPenolakan melayani tab Penolakan Klaim pada layar Master Penolakan Klaim.
	// Ia memakai pemilih repo per portal dengan alasan yang sama seperti master status
	// progres: kedua tabelnya ada di basis data SETIAP entitas (ADR-0030).
	masterPenolakan *masterpenolakanusecase.Service
	// daftarDetailTipeDokumen memakai DUA pemilih per portal: satu untuk kedua tabelnya
	// sendiri, dan satu untuk KEEMPAT master yang hanya dibacanya.
	//
	// Keempat master itu berada di satu seam, bukan empat seperti pada modul di bawah,
	// karena keempatnya dibutuhkan bersamaan oleh SATU form dan gagal dengan cara yang
	// sama — daftar pilihannya kosong sementara isiannya tetap dapat diketik sendiri.
	// Memisahkannya akan menghasilkan empat selector yang selalu dipilih bersamaan dan
	// empat jalur galat yang ditangani dengan cara yang persis sama.
	daftarDetailTipeDokumen *daftardetailtipedokumenusecase.Service

	// daftarTipeDokumenBisnis memakai LIMA pemilih per portal: satu untuk kedua tabelnya
	// sendiri, dan empat untuk master yang hanya dibacanya — POOLDATA.BUSINESS milik
	// GISFW, V_LST_DOC_TYPE milik modul Daftar Tipe Dokumen, V_LST_DET_TYPE_DOC milik
	// MENU_ID 41 yang belum dibangun, dan V_LST_DOC_OBJ milik modul Daftar Objek Dokumen.
	// Kelimanya hidup di basis data entitas yang sama, tetapi mengisi seam yang berbeda —
	// dan pemisahannya itulah yang menegakkan `P-1` tanpa bergantung pada ingatan.
	daftarTipeDokumenBisnis *daftartipedokumenbisnisusecase.Service

	// masterTipeSurveyors juga per portal, dengan alasan yang sama: POOLDATA.M_SURVEYORS
	// ada di basis data setiap entitas, dan golongan surveyor satu badan hukum tidak
	// boleh terbaca dari badan hukum lain (R-20).
	masterTipeSurveyors *mastertipesurveyorsusecase.Service

	// masterSurveyors adalah daftar ORANGNYA, anak dari master tipe di atas. Per portal
	// dengan alasan yang sama, dan satu alasan tambahan yang lebih berat: barisnya memuat
	// nama, alamat, telepon, dan surel orang — juga nama login aplikasi mereka.
	masterSurveyors *mastersurveyorsusecase.Service

	// masterPicTeknik juga per portal: POOLDATA.MST_USER_TEKNIK ada di basis data setiap
	// entitas, dan daftar petugas satu badan hukum tidak boleh terbaca dari badan hukum
	// lain (R-20).
	masterPicTeknik *masterpicteknikusecase.Service

	// masterRecovery juga per portal: POOLDATA.MST_RECOVERY_ASM_PENJAMINAN dan master
	// virtual account-nya ada di basis data setiap entitas. Di modul ini R-20 menyentuh
	// hal yang paling berat akibatnya — nomor rekening virtual satu badan hukum yang
	// terbaca dari badan hukum lain berarti dana dapat diarahkan ke rekening yang keliru.
	masterRecovery *masterrecoveryusecase.Service

	// masterDominanFactor juga per portal: POOLDATA.M_DOMINAN_FACTOR ada di basis data
	// setiap entitas. Akibat portal yang keliru di sini halus tetapi luas — nama faktor
	// dominan ikut terbaca laporan Outstanding per Cabang lewat LISTAGG, sehingga yang
	// salah bukan satu layar melainkan isi laporan yang dibaca manajemen (`R-20`).
	masterDominanFactor *masterdominanfactorusecase.Service

	// masterXOL juga per portal: keempat tabel MST_XOL_* ada di basis data setiap
	// entitas. Akibat salah entitas di sini menjalar paling jauh di antara butir master —
	// struktur treaty menentukan pembagian klaim ke reasuradur, sehingga yang keliru
	// bukan satu layar melainkan nilai yang dihitung PLA/DLA sesudahnya (R-20).
	masterXOL *masterxolusecase.Service

	// masterPenyebabKerugian juga per portal: POOLDATA.M_CAUSE_OF_LOSS ada di basis data
	// setiap entitas. Akibat portal yang keliru di sini serupa dengan faktor dominan
	// tetapi jangkauannya lebih luas — keterangan penyebab kerugian dibaca 19 rule Pega
	// dan menjadi kolom PENGELOMPOKAN pada dasbor klaim per penyebab kerugian serta
	// laporan XOL per bisnis, sehingga yang keliru bukan satu layar melainkan
	// pengelompokan laporan yang dibaca manajemen (`R-20`).
	masterPenyebabKerugian *masterpenyebabkerugianusecase.Service

	// masterMasking juga per portal: POOLDATA.MST_PROTEKSI_DATA_PNC ada di basis data
	// setiap entitas. Di antara seluruh master yang sudah dibangun, inilah yang portal
	// kelirunya paling berat akibatnya — isinya adalah daftar SIAPA yang boleh melihat
	// nomor KTP, surel, dan nomor telepon nasabah tanpa disamarkan. Membacanya dari
	// entitas yang salah berarti membocorkan peta kewenangan data pribadi badan hukum
	// lain, dan menulisnya ke entitas yang salah berarti memberi orang kewenangan di
	// tempat yang bukan haknya — keduanya tanpa satu pun pesan galat (`R-20`).
	masterMasking *mastermaskingusecase.Service

	// menu menyusun peta menu beserta kewenangan pemakainya.
	menu *menuusecase.Service

	// extra memegang sepuluh modul yang perakitannya ada di modules.go. Ia satu field,
	// bukan sebelas, supaya berkas ini tidak ikut tumbuh setiap kali satu modul dirakit.
	extra extraServices
	// inboxAutoClaim memakai pemilih repo per portal, sama seperti masterStatusProgres:
	// POOLDATA.TMP_BATCH_AUTO_CLAIM ada di basis data SETIAP entitas (ADR-0030).
	inboxAutoClaim *inboxautoclaimusecase.Service

	// inboxXOL melayani layar Inbox XOL (`MENU_ID 53`).
	inboxXOL *inboxxolusecase.Service

	// inboxClaimTreatyProp melayani layar Inbox Claim Treaty Prop (`MENU_ID 54`).
	//
	// Kedua tabel penugasan yang dibacanya ada di basis data SETIAP entitas (`ADR-0030`),
	// sama seperti modul inbox lain.
	inboxClaimTreatyProp *inboxclaimtreatypropusecase.Service

	// inboxClaimTreatyNonProp melayani layar Inbox Claim Treaty Non Prop (`MENU_ID 55`).
	//
	// Ia layar SAUDARA dari yang di atas dan sengaja berdiri sendiri: ketiga tabel yang
	// dibacanya, penanda objek kerjanya, dan kolom gridnya berbeda.
	inboxClaimTreatyNonProp *inboxclaimtreatynonpropusecase.Service

	// inboxManagerReceivePUCL melayani layar Inbox Manager Receive / PUCL (`MENU_ID 56`).
	//
	// Ia menyatukan DUA antrean yang kelas objek kerjanya berbeda — berkas penerimaan
	// dokumen dan klaim RCL/PUCL — karena begitulah harness `ReceiveDoucument_Harness`
	// menyusunnya.
	inboxManagerReceivePUCL *inboxmanagerreceivepuclusecase.Service
	inboxRCLPUCL            *inboxrclpuclusecase.Service
	reportKPI               *reportkpiusecase.Service
	reportKlaim             *reportklaimusecase.Service

	// inboxSalvage melayani layar Inbox Salvage (`MENU_ID 71`).
	//
	// Ia satu-satunya modul inbox yang MENULIS. Dua tabelnya —
	// `POOLDATA.PNC_SALVAGE` dan `POOLDATA.DETAIL_PNC_SALVAGE` — dimiliki modul ini selama
	// masa paralel, karena seluruh penulisnya di Pega adalah layar yang digantikannya
	// (`P-1`).
	inboxSalvage *inboxsalvageusecase.Service

	// inboxKomunikasiCabang melayani layar Inbox Komunikasi Cabang (`MENU_ID 70`).
	//
	// Berbeda dari modul inbox di atasnya, layar ini DISARING menurut cabang pemanggilnya —
	// bukan antrean bersama. Batas itu diturunkan dari login lewat BranchResolver.
	inboxKomunikasiCabang *inboxkomunikasicabangusecase.Service

	// inboxProgressClaim melayani layar Inbox Progress Claim (`MENU_ID 65`).
	inboxProgressClaim *inboxprogressclaimusecase.Service

	// inboxAnalystDoctor melayani layar Inbox Analyst Doctor (`MENU_ID 60`) — antrean
	// penilaian medis milik satu petugas.
	inboxAnalystDoctor *inboxanalystdoctorusecase.Service

	// inboxLaporanKlaim melayani layar Inbox Laporan Klaim. Sama seperti master status
	// progres, ia memakai pemilih repo per portal: berkas laporan adalah data bisnis
	// milik satu badan hukum (ADR-0030).
	inboxLaporanKlaim *inboxlaporanklaimusecase.Service

	// inboxOutstanding melayani layar Inbox Outstanding — klaim yang masih berjalan.
	inboxOutstanding *inboxoutstandingusecase.Service

	// inboxCloseClaim melayani layar Inbox Close Claim — klaim yang sudah TUTUP.
	//
	// Ia kebalikan tepat dari inboxOutstanding tepat di atasnya: keduanya menyaring dua
	// nilai PYSTATUSWORK yang sama dengan arah yang berlawanan. Satu-satunya modul inbox
	// yang MENULIS, dan yang ditulisnya bukan klaim melainkan permintaan atas klaim.
	inboxCloseClaim *inboxcloseclaimusecase.Service
	// inputReqProtection melayani layar Input Req Protection — permintaan pembukaan
	// proteksi beserta form inputnya.
	inputReqProtection *inputreqprotectionusecase.Service

	// inboxAcceptOpenProtection melayani layar Inbox Accept Open Protection — antrean
	// akseptasi atas permintaan yang sama.
	//
	// Kedua modul menyentuh SATU tabel, tetapi menulis kolom yang berbeda: yang pertama
	// kolom pembuatan, yang kedua kolom akseptasi. Pembagian itu yang menjaga P-1 tetap
	// berlaku tanpa menggabungkan keduanya menjadi satu modul.
	inboxAcceptOpenProtection *inboxacceptopenprotectionusecase.Service

	readyAliases func() []string
	close        func()

	// masterStatusProgres2 memakai pemilih repo per portal dengan alasan yang sama, dan
	// menerima pemilih tingkat 1 sebagai bahan kedua: penambahan tingkat 2 membaca baris
	// induknya untuk memastikan induk itu ada dan menyalin namanya.
	masterStatusProgres2 *masterstatusprogresusecase.Service2

	// masterAutoClaim melayani layar Master Auto Claim — daftar Sumber Bisnis yang
	// klaimnya boleh dibuat otomatis. Ia memakai pemilih repo per portal dengan alasan
	// yang sama seperti master lain: POOLDATA.M_AUTO_CLAIM_PNC beserta ketiga tabel
	// acuannya ada di basis data SETIAP entitas (ADR-0030).
	masterAutoClaim *masterautoclaimusecase.Service

	// masterBengkel melayani layar Master Bengkel — daftar bengkel rekanan beserta
	// syarat kerja samanya. Ia memakai pemilih repo per portal dengan alasan yang sama
	// seperti master lain: POOLDATA.BENGKEL_HE beserta ketiga tabel acuannya ada di basis
	// data SETIAP entitas (ADR-0030).
	masterBengkel *masterbengkelusecase.Service

	// masterPanel melayani layar Master Panel — daftar panel bodi kendaraan berat
	// beserta perlakuan klaim yang berlaku atasnya. Ia memakai pemilih repo per portal
	// dengan alasan yang sama seperti master lain: POOLDATA.PANEL_HE dan tabel anaknya
	// POOLDATA.LOKASI_PANEL_HE ada di basis data SETIAP entitas (ADR-0030).
	masterPanel *masterpanelusecase.Service

	// masterSparepart melayani layar Master Sparepart — daftar suku cadang alat berat
	// beserta harga, dimensi, dan batas stoknya. Ia memakai pemilih repo per portal dengan
	// alasan yang sama seperti master lain: POOLDATA.SPAREPART_HE beserta kedua tabel
	// acuannya ada di basis data SETIAP entitas (ADR-0030).
	masterSparepart *mastersparepartusecase.Service

	// masterPenolakanKomite melayani tab Penolakan Komite pada layar yang SAMA. Ia
	// layanan tersendiri karena tabelnya tidak sekerabat dan tidak punya satu pun kolom
	// yang menghubungkannya dengan kedua tabel di atas.
	masterPenolakanKomite *masterpenolakanusecase.ServiceKomite

	// riwayatKlaim melayani layar View History Claim (`MENU_ID 76`).
	riwayatKlaim *riwayatklaimusecase.Service

	// archiveDokumenKlaim melayani layar Archive Dokumen Klaim (`MENU_ID 77`).
	archiveDokumenKlaim *archivedokumenklaimusecase.Service
}

// storage memegang seluruh repo yang sudah terpasang di atas sumbernya.
type storage struct {
	user    auth.UserRepo
	session auth.SessionRepo
	portal  portal.Repo
	// claimStatusSelector memilih penyimpanan master status klaim milik satu portal.
	// Sejak 2026-09-19 tabelnya dibaca per entitas, bukan dari portal utama saja.
	claimStatusSelector masterstatus.RepoSelector

	// komite adalah master ambang POOLDATA.EMAILKOMITE — DIBACA SAJA.
	komite komite.Repo

	// komiteInbox membaca kasus komite dari tabel warisan; komiteDecision menulis
	// keputusannya ke tabel milik aplikasi ini.
	//
	// Keduanya dinyatakan TERPISAH meski satu objek yang sama dapat mengisi keduanya
	// (adapter memori memang demikian). Pembelahannya mengikuti kepemilikan tabel:
	// yang satu tidak boleh menulis apa pun, yang lain menulis.
	komiteInbox    komite.InboxRepo
	komiteDecision komite.DecisionRepo

	// accountSelector memilih penyimpanan master rekening milik satu portal entitas.
	// Repo dan BankRepo dipilih bersamaan karena keduanya hidup di basis data yang sama.
	accountSelector func(alias string) (masterrekening.Repo, masterrekening.BankRepo, error)

	// rekeningDiOracle menyatakan master rekening dipasang di atas POOLDATA.LST_ACCOUNT
	// yang sungguhan, bukan di memori. Adapter tiruan yang menulis jejak karangan
	// dilarang di atasnya — lihat rakitMasterRekening.
	accountInOracle bool

	// warisan bernilai nil bila koneksi Oracle tidak dibuka. Ia memberi akses baca ke
	// tiga tabel milik sistem lama: M_PORTAL_PNC, M_LOGIN_PNC, dan GCNM_CONNECT_REST.
	legacy *sqlstore.Legacy

	// inboxXOLSelector memilih penyimpanan Inbox XOL milik satu portal.
	//
	// Fungsi, bukan repo tunggal, dengan alasan yang sama seperti selector di atasnya:
	// perjanjian XOL dan nilai klaimnya adalah data bisnis milik satu badan hukum
	// (`ADR-0030`). Satu repo bersama akan membaca perjanjian satu entitas dari basis
	// data entitas lain — kebocoran lintas badan hukum yang justru dicegah `R-20`.
	inboxXOLSelector inboxxol.RepoSelector

	// claimTreatyPropSelector memilih penyimpanan Inbox Claim Treaty Prop milik satu
	// portal, dengan alasan yang sama persis: barisnya memuat nama tertanggung dan nama
	// Ceding Co, dan keduanya milik satu badan hukum.
	claimTreatyPropSelector inboxclaimtreatyprop.RepoSelector

	// claimTreatyNonPropSelector memilih penyimpanan Inbox Claim Treaty Non Prop milik
	// satu portal, dengan alasan yang sama persis dengan selector di atasnya.
	claimTreatyNonPropSelector inboxclaimtreatynonprop.RepoSelector

	// managerReceivePUCLSelector memilih penyimpanan Inbox Manager Receive / PUCL milik
	// satu portal.
	//
	// Alasannya sama dengan selector di atasnya, dan di modul ini taruhannya paling besar:
	// tidak satu pun tabnya menyaring menurut pemanggil, sehingga jatuh ke koneksi bawaan
	// berarti memperlihatkan SELURUH antrean satu badan hukum kepada petugas badan hukum
	// lain (`R-20`).
	managerReceivePUCLSelector inboxmanagerreceivepucl.RepoSelector
	rclPUCLSelector            inboxrclpucl.RepoSelector
	reportKPISelector          reportkpi.RepoSelector
	reportKlaimSelector        reportklaim.RepoSelector

	// salvageSelector memilih penyimpanan salvage milik satu portal.
	//
	// Taruhannya lebih besar daripada selector di atasnya, dan alasannya satu: modul ini
	// menulis. Portal yang salah di sini bukan sekadar menampilkan data entitas lain — ia
	// menyisipkan baris ke dalamnya.
	salvageSelector inboxsalvage.RepoSelector

	// komunikasiCabangSelector memilih penyimpanan percakapan milik satu portal.
	komunikasiCabangSelector inboxkomunikasicabang.RepoSelector

	// komunikasiCabangBranch menerjemahkan login petugas menjadi kode cabang klaimnya.
	//
	// Ia SALINAN seam yang sama dengan claimReportBranch, bukan pemakaian ulangnya, dan itu
	// disengaja: seam dideklarasikan di paket yang MEMAKAINYA (`08-TECHNICAL-STRATEGY.md`
	// §2 aturan 2), sehingga kedua modul dapat berpindah ke API pengganti DB Link (`D-25`,
	// `R-03`) pada waktu yang berbeda tanpa saling menunggu.
	//
	// Ia hidup di basis data PORTAL UTAMA, bukan per entitas: HRD dan master pengguna
	// asuransi adalah data lingkup identitas, sama seperti M_LOGIN_PNC dan M_PORTAL_PNC.
	komunikasiCabangBranch inboxkomunikasicabang.BranchResolver

	// inboxProgressClaimSelector memilih penyimpanan progres klaim milik satu portal.
	//
	// Ia fungsi dengan alasan yang sama: progres klaim satu badan hukum bukan progres
	// badan hukum lain, dan barisnya memuat nama tertanggung (`ADR-0030`, `R-20`).
	inboxProgressClaimSelector inboxprogressclaim.RepoSelector

	// inboxAnalystDoctorSelector memilih penyimpanan antrean penilaian medis milik satu
	// portal.
	//
	// Alasannya sama dengan selector di atasnya, ditambah satu yang lebih berat: barisnya
	// adalah klaim Personal Accident, dan `FR-R2` memperlakukan data medis secara khusus.
	// Jatuh ke koneksi bawaan di sini bukan sekadar menampilkan entitas yang salah — ia
	// menampilkan data medis entitas yang salah.
	inboxAnalystDoctorSelector inboxanalystdoctor.RepoSelector

	// claimReportSelector memilih penyimpanan berkas laporan klaim milik satu portal.
	//
	// Alasannya sama dengan progressStatusSelector di bawah, ditambah satu yang khas
	// modul ini: ia membaca DUA tabel sekaligus — tabel warisan Pega dan tabel milik
	// aplikasi ini — dan keduanya hidup di basis data entitas yang sama.
	claimReportSelector inboxlaporanklaim.RepoSelector

	// outstandingSelector memilih penyimpanan klaim milik satu portal.
	outstandingSelector inboxoutstanding.RepoSelector

	// protectionRequestSelector memilih penyimpanan permintaan proteksi milik satu portal.
	//
	// Ia fungsi, bukan repo tunggal, karena POOLDATA.T_CLAIM_OPENPROTECTION ada di basis
	// data SETIAP entitas (ADR-0030).
	protectionRequestSelector inputreqprotection.RepoSelector

	// protectionAcceptSelector memilih penyimpanan antrean akseptasi milik satu portal.
	//
	// Ia menunjuk tabel yang SAMA dengan protectionRequestSelector; yang berbeda adalah
	// kolom yang ditulisnya.
	protectionAcceptSelector inboxacceptopenprotection.RepoSelector
	// claimReportBranch menerjemahkan login petugas menjadi kode cabang klaimnya.
	//
	// Ia TIDAK diambil dari profil HCC/HCQ: kode cabang yang dipakai layar itu adalah
	// POOLDATA.BRANCH.ID, diturunkan lewat HRD dan master pengguna asuransi — sistem kode
	// yang berbeda dari Placement.BranchCode. Lihat inboxlaporanklaim.BranchResolver.
	//
	// Ia hidup di basis data PORTAL UTAMA, bukan per entitas: HRD dan master pengguna
	// adalah data lingkup identitas, sama seperti M_LOGIN_PNC dan M_PORTAL_PNC.
	claimReportBranch inboxlaporanklaim.BranchResolver

	// closeClaimSelector memilih penyimpanan klaim TUTUP milik satu portal.
	closeClaimSelector inboxcloseclaim.RepoSelector

	// closeClaimRequests memilih penyimpanan PERMINTAAN ReOpen dan Copy Klaim milik satu
	// portal.
	//
	// Ia terpisah dari closeClaimSelector meski keduanya melayani satu layar, dan
	// pembelahannya mengikuti kepemilikan tabel: yang pertama membaca tabel milik Pega,
	// yang kedua menulis tabel milik aplikasi ini sendiri (`P-1`).
	closeClaimRequests inboxcloseclaim.RequestRepoSelector

	// progressStatusSelector memilih penyimpanan master status progres milik satu portal.
	//
	// Ia fungsi, bukan repo tunggal, karena tabelnya ada di basis data SETIAP entitas
	// (ADR-0030). Satu repo bersama akan menulis data seluruh entitas ke satu tempat,
	// kebocoran lintas badan hukum yang justru dicegah R-20.
	progressStatusSelector masterstatusprogres.RepoSelector

	// travelDocumentSelector memilih penyimpanan master dokumen travel milik satu
	// portal, dengan alasan yang sama seperti progressStatusSelector di atas.
	travelDocumentSelector masterdokumentravel.RepoSelector

	// clauseAISelector memilih penyimpanan Master Pasal AI milik satu portal.
	//
	// POOLDATA.MST_PASAL_AI ada di basis data setiap entitas (ADR-0030), sehingga satu repo
	// bersama akan menampilkan wording polis satu badan hukum kepada pengguna badan hukum
	// lain — kebocoran yang justru dicegah R-20.
	clauseAISelector masterpasalai.RepoSelector

	// aiReportSelector memilih penyimpanan Laporan Hasil AI milik satu portal.
	//
	// POOLDATA.T_CLAIM_DATA_RESULTS_AI dan POOLDATA.T_CLAIM_KOMITE_LIST ada di basis data
	// setiap entitas (ADR-0030). Satu repo bersama akan menampilkan penilaian AI atas
	// klaim satu badan hukum kepada pengguna badan hukum lain — kebocoran yang justru
	// dicegah R-20, dan di modul ini barisnya memuat keputusan uang.
	aiReportSelector laporanhasilai.RepoSelector

	// masterAutoClaimSelector memilih penyimpanan Master Auto Claim milik satu portal.
	// POOLDATA.M_AUTO_CLAIM_PNC dan ketiga tabel acuannya ada di basis data setiap
	// entitas.
	//
	// Namanya dibedakan dari autoClaimSelector milik Inbox Auto Claim di bawah: keduanya
	// "auto claim", tetapi tabel dan modulnya berbeda.
	masterAutoClaimSelector masterautoclaim.RepoSelector
	// simasOnlineCauseOfLossSelector memilih penyimpanan master COL Simas Online milik
	// satu portal, dengan alasan yang sama seperti kedua pemilih di atas.
	//
	// Namanya dibedakan dari causeOfLossSelector di bawah dengan sengaja: keduanya memang
	// "penyebab kerugian", tetapi tabel dan modulnya berbeda. Memakai satu nama untuk
	// keduanya akan membuat salah satunya diam-diam tertimpa saat dirakit.
	simasOnlineCauseOfLossSelector mastercolsimasonline.RepoSelector

	// documentTypeSelector memilih penyimpanan daftar tipe dokumen milik satu portal,
	// dengan alasan yang sama seperti ketiga pemilih di atas.
	documentTypeSelector daftartipedokumen.RepoSelector

	// documentObjectSelector memilih penyimpanan daftar objek dokumen milik satu portal,
	// dengan alasan yang sama seperti pemilih di atas.
	documentObjectSelector daftarobjekdokumen.RepoSelector

	// documentObjectBusinessSelector memilih pembaca POOLDATA.BUSINESS untuk modul Daftar
	// Objek Dokumen.
	//
	// Terpisah dari businessSelector di bawah meski keduanya membaca tabel yang SAMA,
	// karena keduanya mengisi seam milik modul yang berbeda. Menyatukannya akan membuat
	// modul Daftar Objek Dokumen mengimpor tipe modul Master COL Simas Online — dan sejak
	// itu, perubahan di salah satunya merambat ke yang lain tanpa alasan. Perlakuan yang
	// sama sudah dipakai travelChoiceSelector terhadap travelDocumentSelector di atas.
	documentObjectBusinessSelector daftarobjekdokumen.BusinessRepoSelector

	// groupingSelector memilih penyimpanan Master Grouping Sparepart milik satu portal.
	// Kedua tabel groupingnya dan keempat sumber acuannya ada di basis data setiap entitas.
	groupingSelector mastergroupingsparepart.RepoSelector

	// partCategorySelector memilih penyimpanan Master Kategori Sparepart milik satu portal.
	// POOLDATA.GCNM_M_SPAREPART_CATEGORY ada di basis data setiap entitas — tabel yang sama
	// yang dibaca sparepartSelector sebagai daftar acuan, dan ditulis oleh yang ini.
	partCategorySelector masterkategorisparepart.RepoSelector

	// partTypeSelector memilih penyimpanan Master Tipe Sparepart milik satu portal.
	// POOLDATA.GCNM_M_SPAREPART_TYPE ada di basis data setiap entitas — tabel yang sama yang
	// dibaca sparepartSelector sebagai daftar acuan Tipe, dan ditulis oleh yang ini.
	//
	// Repo yang sama juga MEMBACA GCNM_M_SPAREPART_CATEGORY untuk dropdown Kategorinya;
	// yang menulis tabel itu tetap partCategorySelector (P-1).
	partTypeSelector mastertipesparepart.RepoSelector

	// surveyorLoginSelector memilih penyimpanan Master Login milik satu portal.
	//
	// POOLDATA.MST_LOGIN_SURVEYOR ada di basis data setiap entitas, dan pemisahannya di
	// sini lebih berarti daripada pada master penggolongan: isinya menentukan siapa yang
	// boleh bekerja sebagai surveyor pada satu badan hukum. Baris yang bocor ke portal lain
	// bukan sekadar data yang salah tempat — ia orang yang muncul di daftar entitas yang
	// bukan haknya (R-20).
	surveyorLoginSelector masterlogin.RepoSelector

	// causeOfLossDetailSelector memilih penyimpanan Detail Penyebab Kerugian milik satu
	// portal. POOLDATA.D_CAUSE_OF_LOSS ada di basis data setiap entitas, dan isinya
	// menentukan sebab kerugian apa saja yang dapat dipilih pada badan hukum itu — sehingga
	// baris yang bocor ke portal lain mengubah cara klaim entitas itu dinilai, bukan
	// sekadar menampilkan baris yang salah tempat (R-20).
	causeOfLossDetailSelector detailpenyebab.RepoSelector

	// investigatorInboxSelector memilih penyimpanan Inbox Investigator milik satu portal.
	// Antrean pekerjaan ada di basis data setiap entitas; baris yang bocor ke portal lain
	// menampakkan nama tertanggung dan nama peserta klaim badan hukum yang bukan haknya
	// (R-20).
	investigatorInboxSelector inboxinvestigator.RepoSelector

	// receiveTKASelector memilih penyimpanan Inbox Receive TKA milik satu portal.
	// POOLDATA.T_CLAIM_TKA_H ada di basis data setiap entitas, dan modul ini MENULIS —
	// pemilih yang keliru bukan hanya menampakkan pekerjaan badan hukum lain, melainkan
	// mengubah tanggal pada klaimnya (R-20).
	receiveTKASelector inboxreceivetka.RepoSelector

	// reinsurerMemberSelector memilih penyimpanan Master Reas milik satu portal.
	// POOLDATA.T_REINSURER ada di basis data setiap entitas, dan isinya menentukan ke mana
	// pemberitahuan klaim satu badan hukum dikirim. Baris yang bocor ke portal lain
	// menampakkan daftar mitra reasuransi beserta surel tujuannya kepada entitas yang bukan
	// haknya (R-20).
	reinsurerMemberSelector masterreas.RepoSelector

	// clauseSelector memilih penyimpanan Master Pasal Kerugian milik satu portal.
	// POOLDATA.V_M_DATA_PASAL dan master lini bisnisnya ada di basis data setiap entitas.
	clauseSelector masterpasal.RepoSelector
	// detailTravelSelector memilih penyimpanan detail dokumen travel milik satu portal,
	// dengan alasan yang sama seperti keempat pemilih di atas.
	detailTravelSelector daftardetaildokumentravel.RepoSelector

	// travelChoiceSelector memilih pembaca POOLDATA.M_DOCTRAVEL sebagai daftar pilihan.
	//
	// Terpisah dari travelDocumentSelector meski keduanya membaca tabel yang SAMA,
	// karena keduanya mengisi seam milik modul yang berbeda. Menyatukannya akan membuat
	// modul Daftar Detail mengimpor tipe modul Master Dokumen Travel — dan sejak itu,
	// perubahan di salah satunya merambat ke yang lain tanpa alasan.
	travelChoiceSelector daftardetaildokumentravel.DocumentRepoSelector

	// travelPlanSelector memilih pembaca POOLDATA.M_PLANTRAVEL milik GISFW yang HANYA
	// DIBACA (`D-03`), sejajar dengan businessSelector di bawah.
	travelPlanSelector daftardetaildokumentravel.PlanRepoSelector

	// Kedua pemilih modul Daftar Detail Tipe Dokumen (MENU_ID 41).
	//
	// Yang kedua melayani KEEMPAT master rujukannya sekaligus, berbeda dari modul di
	// bawah yang memisahkannya menjadi empat. Alasannya ada pada komentar
	// `assembly.daftarDetailTipeDokumen`.
	detailDocumentTypeSelector          daftardetailtipedokumen.RepoSelector
	detailDocumentTypeReferenceSelector daftardetailtipedokumen.ReferenceRepoSelector

	// Kelima pemilih modul Daftar Tipe Dokumen Bisnis.
	//
	// Keempat yang terakhir membaca tabel yang SAMA dengan pemilih modul lain di berkas
	// ini — BUSINESS dibaca tiga modul, V_LST_DOC_OBJ dua modul — dan tetap dibiarkan
	// terpisah, dengan alasan yang sama seperti travelChoiceSelector di atas: keduanya
	// mengisi seam milik modul yang berbeda, dan menyatukannya akan membuat satu modul
	// mengimpor tipe modul lain.
	businessDocumentRuleSelector  daftartipedokumenbisnis.RepoSelector
	businessDocumentRuleBusiness  daftartipedokumenbisnis.BusinessRepoSelector
	businessDocumentRuleDocType   daftartipedokumenbisnis.DocumentTypeRepoSelector
	businessDocumentRuleDetailDoc daftartipedokumenbisnis.DetailTypeDocRepoSelector
	businessDocumentRuleObjectDoc daftartipedokumenbisnis.ObjectDocRepoSelector

	// businessSelector memilih master bisnis milik satu portal.
	//
	// Terpisah dari causeOfLossSelector meski keduanya selalu dipilih bersamaan, karena
	// keduanya mengisi seam yang berbeda: yang satu tabel milik modul COL, yang lain
	// POOLDATA.BUSINESS milik GISFW yang HANYA DIBACA (D-03).
	businessSelector mastercolsimasonline.BusinessRepoSelector

	// surveyorTypeSelector memilih penyimpanan master tipe surveyor milik satu portal,
	// dengan alasan yang sama persis.
	surveyorTypeSelector mastertipesurveyors.RepoSelector

	// surveyorSelector memilih penyimpanan Master Surveyors — daftar ORANGNYA, bukan
	// tipenya — milik satu portal, dengan alasan yang sama persis.
	surveyorSelector mastersurveyors.RepoSelector

	// picTeknikSelector memilih penyimpanan master PIC teknik milik satu portal, dengan
	// alasan yang sama persis.
	picTeknikSelector masterpicteknik.RepoSelector

	// dominantFactorSelector memilih penyimpanan master faktor dominan milik satu portal,
	// dengan alasan yang sama persis.
	dominantFactorSelector masterdominanfactor.RepoSelector

	// xolSelector memilih penyimpanan Master XOL milik satu portal, dengan alasan yang
	// sama persis.
	xolSelector masterxol.RepoSelector

	// causeOfLossSelector memilih penyimpanan master penyebab kerugian milik satu portal,
	// dengan alasan yang sama persis.
	causeOfLossSelector masterpenyebabkerugian.RepoSelector

	// recoverySelector memilih penyimpanan Master Recovery milik satu portal, dengan
	// alasan yang sama persis.
	recoverySelector masterrecovery.RepoSelector

	// maskingSelector memilih penyimpanan Master Masking milik satu portal, dengan alasan
	// yang sama persis — dan di modul ini akibat kelalaiannya yang paling berat.
	maskingSelector mastermasking.RepoSelector

	// menu dibaca dari basis data portal UTAMA, sama seperti M_LOGIN_PNC dan
	// M_PORTAL_PNC: peta menu dan kewenangan pemakainya adalah data lingkup
	// identitas, bukan data bisnis milik satu badan hukum.
	menu menu.Repo

	// extra memegang pemilih penyimpanan sepuluh modul yang dirakit di modules.go.
	extra extraSelectors
	// autoClaimSelector memilih penyimpanan Inbox Auto Claim milik satu portal.
	//
	// Alasannya sama dengan progressStatusSelector: tabelnya ada di basis data SETIAP
	// entitas, dan satu repo bersama akan menulis data seluruh entitas ke satu tempat —
	// kebocoran lintas badan hukum yang justru dicegah R-20.
	autoClaimSelector inboxautoclaim.RepoSelector

	readyAliases func() []string
	close        func()

	account masterrekening.Repo

	accountBank masterrekening.BankRepo

	masterStatus masterstatus.Repo

	// progressStatus2Selector memilih penyimpanan tingkat 2 milik satu portal, dengan
	// alasan yang sama persis: POOLDATA.GCNM_MST_PROGRESS ada di basis data setiap entitas.
	progressStatus2Selector masterstatusprogres.RepoSelector2

	// workshopSelector memilih penyimpanan Master Bengkel milik satu portal.
	// POOLDATA.BENGKEL_HE dan ketiga tabel acuannya ada di basis data setiap entitas.
	workshopSelector masterbengkel.RepoSelector

	// panelSelector memilih penyimpanan Master Panel milik satu portal.
	// POOLDATA.PANEL_HE dan tabel anaknya POOLDATA.LOKASI_PANEL_HE ada di basis data
	// setiap entitas.
	panelSelector masterpanel.RepoSelector

	// sparepartSelector memilih penyimpanan Master Sparepart milik satu portal.
	sparepartSelector mastersparepart.RepoSelector

	// supplierSelector memilih penyimpanan Master Supplier milik satu portal.
	// M_SUPPLIER, antrean POOLDATA.PROTEKSI_KLAIMMBU, dan keempat tabel acuannya ada di
	// basis data setiap entitas.
	supplierSelector mastersupplier.RepoSelector

	// rejectionSelector memilih penyimpanan Master Penolakan Klaim milik satu portal.
	// POOLDATA.MST_PENOLAKAN_KLAIM_1 dan _2 ada di basis data setiap entitas.
	rejectionSelector masterpenolakan.RepoSelector

	// rejectionKomiteSelector memilih penyimpanan Master Penolakan Komite milik satu
	// portal — POOLDATA.MST_REJECTED_KOMITE, juga per entitas.
	rejectionKomiteSelector masterpenolakan.RepoSelectorKomite

	// claimHistorySelector dan claimProtectionSelector memilih penyimpanan View History
	// Claim milik satu portal.
	//
	// Keduanya fungsi, bukan repo tunggal, karena riwayat klaim DAN jatah proteksi
	// seorang pengguna adalah data bisnis milik satu badan hukum (`ADR-0030`). Satu repo
	// bersama akan membaca riwayat satu entitas dari basis data entitas lain — kebocoran
	// lintas badan hukum yang justru dicegah `R-20`.
	claimHistorySelector riwayatklaim.RepoSelector

	claimProtectionSelector riwayatklaim.ProtectionRepoSelector

	// archiveSelector memilih penyimpanan Archive Dokumen Klaim milik satu portal.
	//
	// Ia fungsi, bukan repo tunggal, karena berkas arsip adalah data bisnis milik satu
	// badan hukum (`ADR-0030`). Satu repo bersama akan mengarsipkan berkas satu entitas
	// ke basis data entitas lain — kebocoran lintas badan hukum yang dicegah `R-20`.
	archiveSelector archivedokumenklaim.RepoSelector
}

// build menyusun seluruh modul di balik seam-nya masing-masing.
func build(cfg config.Config, logger *slog.Logger) (assembly, error) {
	production := cfg.Environment == config.Production

	store, err := buildStorage(cfg, production, logger)
	if err != nil {
		return assembly{}, err
	}

	identitySystem, err := buildIdentity(cfg, production, store.legacy)
	if err != nil {
		store.close()
		return assembly{}, err
	}

	service, err := usecase.NewService(usecase.Options{
		Identity:        identitySystem,
		UserRepo:        store.user,
		SessionRepo:     store.session,
		Clock:           clock.System{},
		SessionLifetime: cfg.Session.Lifetime,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	progressStatusService, err := masterstatusprogresusecase.NewService(masterstatusprogresusecase.Options{
		RepoSelector: store.progressStatusSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	clauseAIService, err := masterpasalaiusecase.NewService(masterpasalaiusecase.Options{
		RepoSelector: store.clauseAISelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	aiReportService, err := laporanhasilaiusecase.NewService(laporanhasilaiusecase.Options{
		RepoSelector: store.aiReportSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	progressStatus2Service, err := masterstatusprogresusecase.NewService2(masterstatusprogresusecase.Options2{
		RepoSelector:   store.progressStatus2Selector,
		ParentSelector: store.progressStatusSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	travelDocumentService, err := masterdokumentravelusecase.NewService(masterdokumentravelusecase.Options{
		RepoSelector: store.travelDocumentSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	simasOnlineCauseOfLossService, err := mastercolusecase.NewService(mastercolusecase.Options{
		RepoSelector:     store.simasOnlineCauseOfLossSelector,
		BusinessSelector: store.businessSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Clock disuntikkan, bukan dipanggil di dalam repo: modul ini mengisi TGL_EDIT, dan
	// `F-5` menetapkan waktu dibaca lewat satu seam supaya penyimpanan dapat diuji
	// deterministik dan konversi zona waktu tidak tersebar.
	documentTypeService, err := daftartipedokumenusecase.NewService(daftartipedokumenusecase.Options{
		RepoSelector: store.documentTypeSelector,
		Clock:        clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Daftar Objek Dokumen. Tidak memakai Clock: tabelnya tidak punya kolom jejak simpan
	// (USER_EDIT / TGL_EDIT) — Report Definition-nya hanya memuat ID, KET_DOC_OBJ, dan
	// OLD_ID — sehingga tidak ada waktu yang perlu dibaca.
	documentObjectService, err := daftarobjekdokumenusecase.NewService(daftarobjekdokumenusecase.Options{
		RepoSelector:     store.documentObjectSelector,
		BusinessSelector: store.documentObjectBusinessSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	travelDocumentDetailService, err := daftardetaildokumentravelusecase.NewService(daftardetaildokumentravelusecase.Options{
		RepoSelector:     store.detailTravelSelector,
		DocumentSelector: store.travelChoiceSelector,
		PlanSelector:     store.travelPlanSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Alasan yang sama seperti di bawah: jam sistem lewat seam Clock (`F-5`), supaya
	// TGL_EDIT tidak mengikuti zona waktu server basis data (`R-12`).
	detailDocumentTypeService, err := daftardetailtipedokumenusecase.NewService(daftardetailtipedokumenusecase.Options{
		RepoSelector:      store.detailDocumentTypeSelector,
		ReferenceSelector: store.detailDocumentTypeReferenceSelector,
		Clock:             clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Grouping Sparepart TIDAK menerima Clock, berbeda dari Master Sparepart: kedua
	// tabelnya tidak punya satu pun kolom waktu, dan `Activity/UpdateGroupingSparepartHE_act`
	// tidak memanggil @DateTime.CurrentDateTime() sama sekali. Menyuntikkan jam yang tidak
	// akan pernah dipakai hanya akan menyesatkan pembaca berikutnya.
	groupingService, err := mastergroupingsparepartusecase.NewService(
		mastergroupingsparepartusecase.Options{
			RepoSelector: store.groupingSelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Kategori Sparepart TIDAK menerima Clock, berbeda dari Master Sparepart yang
	// bertetangga dengannya: POOLDATA.GCNM_M_SPAREPART_CATEGORY tidak punya satu pun kolom
	// waktu, sehingga tidak ada yang perlu distempel. Menyerahkan jam yang tidak pernah
	// dipakai hanya akan menyesatkan pembaca berikutnya.
	partCategoryService, err := masterkategorisparepartusecase.NewService(
		masterkategorisparepartusecase.Options{
			RepoSelector: store.partCategorySelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Tipe Sparepart juga TIDAK menerima Clock, dengan alasan yang sama seperti
	// tetangganya di atas: POOLDATA.GCNM_M_SPAREPART_TYPE tidak punya satu pun kolom waktu
	// maupun kolom pelaku.
	partTypeService, err := mastertipesparepartusecase.NewService(
		mastertipesparepartusecase.Options{
			RepoSelector: store.partTypeSelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Login juga TIDAK menerima Clock: POOLDATA.MST_LOGIN_SURVEYOR tidak punya satu
	// pun kolom waktu maupun kolom pelaku. Ketujuh kolomnya terbaca lengkap dari ketiga
	// rule SQL yang menyentuhnya, dan tidak satu pun menampung siapa atau kapan.
	surveyorLoginService, err := masterloginusecase.NewService(
		masterloginusecase.Options{
			RepoSelector: store.surveyorLoginSelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Detail Penyebab Kerugian juga TIDAK menerima Clock, dan alasannya sama:
	// POOLDATA.D_CAUSE_OF_LOSS hanya punya D_COL_ID dan JSONDATA — tidak ada satu pun kolom
	// waktu maupun kolom pelaku untuk distempel.
	causeOfLossDetailService, err := detailpenyebabusecase.NewService(
		detailpenyebabusecase.Options{
			RepoSelector: store.causeOfLossDetailSelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Inbox Investigator TIDAK menerima Clock maupun Logger: ia hanya membaca, dan tidak
	// ada satu pun nilainya yang bergantung pada jam dinding. Kolom "Lama Masuk Inbox"
	// menampilkan tanggal survei apa adanya — bukan durasi yang dihitung. Lihat
	// inboxinvestigator.Task.SurveyDate.
	investigatorInboxService, err := inboxinvestigatorusecase.NewService(
		inboxinvestigatorusecase.Options{
			RepoSelector: store.investigatorInboxSelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Inbox Receive TKA MENERIMA Logger — berbeda dari Inbox Investigator — karena ia
	// menulis: pengisian tanggal adalah peristiwa bisnis, dan sampai modul Jejak Audit
	// (`S-5`) ada, log aplikasi adalah satu-satunya tempat peristiwa itu tercatat.
	//
	// Ia TIDAK menerima Clock. Satu-satunya tanggal yang ditulisnya datang dari pengguna,
	// persis seperti `Param.Tanggal` pada activity lama; tidak ada nilai yang bergantung
	// pada jam dinding.
	receiveTKAService, err := inboxreceivetkausecase.NewService(
		inboxreceivetkausecase.Options{
			RepoSelector: store.receiveTKASelector,
			Notifier:     buildReceiveTKANotifier(cfg, logger),
			Logger:       logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Reas TIDAK menerima Clock maupun Logger di layanannya: ia hanya membaca,
	// sehingga tidak ada peristiwa yang perlu distempel maupun dicatat. Lihat
	// masterreasusecase.NewService.
	reinsurerMemberService, err := masterreasusecase.NewService(
		masterreasusecase.Options{
			RepoSelector: store.reinsurerMemberSelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Penolakan Klaim menerima Clock karena ia menulis kolom TANGGALKIRIM.
	// Jam yang sama dipakai modul auth, sehingga waktu di seluruh aplikasi berasal dari
	// satu sumber dan tidak ada satu pun penambahan 7 jam manual yang menyelinap masuk.
	rejectionService, err := masterpenolakanusecase.NewService(masterpenolakanusecase.Options{
		RepoSelector: store.rejectionSelector,
		Clock:        clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Tab Penolakan Komite TIDAK menerima Clock: tabelnya tidak punya kolom waktu maupun
	// kolom pelaku sama sekali.
	rejectionKomiteService, err := masterpenolakanusecase.NewServiceKomite(masterpenolakanusecase.OptionsKomite{
		RepoSelector: store.rejectionKomiteSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Jam sistem disuntikkan lewat seam Clock (`F-5`), bukan dipanggil di dalam repo dan
	// bukan diambil dari jam basis data lewat SQL. Tanpa itu, EDIT_DATE mengikuti zona
	// waktu server basis data (`R-12`) dan penyimpanan tidak dapat diuji deterministik.
	businessDocumentRuleService, err := daftartipedokumenbisnisusecase.NewService(daftartipedokumenbisnisusecase.Options{
		RepoSelector:          store.businessDocumentRuleSelector,
		BusinessSelector:      store.businessDocumentRuleBusiness,
		DocumentTypeSelector:  store.businessDocumentRuleDocType,
		DetailTypeDocSelector: store.businessDocumentRuleDetailDoc,
		ObjectDocSelector:     store.businessDocumentRuleObjectDoc,
		Clock:                 clock.System{},
		// Kelima kode lini MBU yang dilewati tombol "Pilih semua". Nilainya dari
		// konfigurasi, bukan konstanta di dalam kode (`D-15`) — bawaannya sudah sama
		// dengan `Activity/SetAllBusiness-Act.xml:984`.
		BulkSelectExcludedBusinesses: cfg.BulkSelectExcludedBusinesses,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	menuService, err := menuusecase.NewService(menuusecase.Options{Repo: store.menu})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	claimStatusService, err := masterstatususecase.NewService(masterstatususecase.Options{
		RepoSelector: store.claimStatusSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	claimReportService, err := inboxlaporanklaimusecase.NewService(inboxlaporanklaimusecase.Options{
		RepoSelector:   store.claimReportSelector,
		BranchResolver: store.claimReportBranch,
		Clock:          clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	maskingService, err := mastermaskingusecase.NewService(mastermaskingusecase.Options{
		RepoSelector: store.maskingSelector,
		// Waktu datang dari jam yang sama dengan modul lain, bukan dari SYSDATE basis data
		// seperti procedure lama. `docs/Steering/07` §4.4 menetapkan konversi dan sumber
		// waktu berada di satu tempat; itulah yang menutup `R-12`.
		Now: clock.System{}.Now,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	xolService, err := masterxolusecase.NewService(masterxolusecase.Options{
		RepoSelector: store.xolSelector,
		Notifier:     buildXOLNotifier(cfg, logger),
		Logger:       logger,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	dominantFactorService, err := masterdominanfactorusecase.NewService(masterdominanfactorusecase.Options{
		RepoSelector: store.dominantFactorSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	causeOfLossService, err := masterpenyebabkerugianusecase.NewService(masterpenyebabkerugianusecase.Options{
		RepoSelector: store.causeOfLossSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	surveyorTypeService, err := mastertipesurveyorsusecase.NewService(mastertipesurveyorsusecase.Options{
		RepoSelector: store.surveyorTypeSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Master Surveyors — daftar ORANGNYA, anak dari master tipe di atas.
	//
	// Dua seam-nya dirakit di sini karena keduanya menyangkut keputusan yang bukan milik
	// modul master:
	//
	//   - Accounts: pembuatan akun aplikasi surveyor. Sistem lama membuat operator Pega
	//     lewat GCNMCreateOperator; `P-1` melarang Go menulis tabel operator milik Pega
	//     selama masa paralel, dan `F-3` yang memiliki identitas di sistem baru masih
	//     terhalang kontrak HCC/HCQ (`R-14`). Pengisi seam di tahap ini MENCATAT
	//     permintaannya tanpa membuat akun — dan itu keputusan yang dicatat, bukan
	//     kelalaian. Lihat paket mastersurveyors/account.
	//   - Committee: penetapan komite penentu, yang di sistem lama dibaca dari
	//     POOLDATA.EMAILKOMITE — master milik `B-7`, bukan milik modul ini. Kueri mana
	//     persisnya yang dipakai jalur surveyor tidak dapat dibaca dari export (`R-16`),
	//     sehingga yang dipasang sekarang adalah penetapan tetap.
	surveyorService, err := mastersurveyorsusecase.NewService(mastersurveyorsusecase.Options{
		RepoSelector: store.surveyorSelector,
		Committee:    mastersurveyorscommittee.Fixed{},
		Accounts:     mastersurveyorsaccount.NewRecorder(logger),
		Clock:        clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	employeeDirectory, err := buildEmployeeDirectory(cfg, store.legacy, logger)
	if err != nil {
		store.close()
		return assembly{}, err
	}

	picTeknikService, err := masterpicteknikusecase.NewService(masterpicteknikusecase.Options{
		RepoSelector: store.picTeknikSelector,
		Directory:    employeeDirectory,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	recoveryIssuer, err := buildVirtualAccountIssuer(cfg, store.legacy, logger)
	if err != nil {
		store.close()
		return assembly{}, err
	}

	recoveryService, err := masterrecoveryusecase.NewService(masterrecoveryusecase.Options{
		RepoSelector: store.recoverySelector,
		Issuer:       recoveryIssuer,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	extra, err := buildExtraServices(store, logger)
	workshopService, err := masterbengkelusecase.NewService(masterbengkelusecase.Options{
		RepoSelector: store.workshopSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	panelService, err := masterpanelusecase.NewService(masterpanelusecase.Options{
		RepoSelector: store.panelSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	sparepartService, err := mastersparepartusecase.NewService(mastersparepartusecase.Options{
		RepoSelector: store.sparepartSelector,
		Clock:        clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	clauseService, err := masterpasalusecase.NewService(masterpasalusecase.Options{
		RepoSelector: store.clauseSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	supplierService, err := mastersupplierusecase.NewService(mastersupplierusecase.Options{
		RepoSelector: store.supplierSelector,
		Clock:        clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	claimHistoryService, err := riwayatklaimusecase.NewService(riwayatklaimusecase.Options{
		RepoSelector:       store.claimHistorySelector,
		ProtectionSelector: store.claimProtectionSelector,
		Clock:              clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	archiveDokumenKlaimService, err := archivedokumenklaimusecase.NewService(
		archivedokumenklaimusecase.Options{
			RepoSelector: store.archiveSelector,
			Gateway:      buildArchiveGateway(store.legacy, logger),
			Clock:        clock.System{},
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	masterAutoClaimService, err := masterautoclaimusecase.NewService(masterautoclaimusecase.Options{
		RepoSelector: store.masterAutoClaimSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	autoClaimService, err := inboxautoclaimusecase.NewService(inboxautoclaimusecase.Options{
		RepoSelector: store.autoClaimSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	inboxXOLService, err := inboxxolusecase.NewService(inboxxolusecase.Options{
		RepoSelector: store.inboxXOLSelector,
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	claimTreatyPropService, err := inboxclaimtreatypropusecase.NewService(
		inboxclaimtreatypropusecase.Options{
			RepoSelector: store.claimTreatyPropSelector,

			// Logger diberikan supaya pembukaan antrean tanpa penyaring kepemilikan
			// ("See All Claim") tercatat. Sampai pemeriksaan peran ada (`TKT-F3-005`),
			// jejak di log adalah satu-satunya hal yang menyatakan siapa memakainya.
			Logger: logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	claimTreatyNonPropService, err := inboxclaimtreatynonpropusecase.NewService(
		inboxclaimtreatynonpropusecase.Options{
			RepoSelector: store.claimTreatyNonPropSelector,

			// Alasan yang sama dengan layar Prop, ditambah satu yang khas modul ini:
			// ekspor berkas memakai penyaring yang sama, sehingga satu unduhan dengan
			// "See All Claim" mengeluarkan nama tertanggung seluruh petugas ke berkas
			// yang tersimpan di perangkat pengguna. Jejaknya di log adalah satu-satunya
			// hal yang menyatakan itu terjadi.
			Logger: logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	managerReceivePUCLService, err := inboxmanagerreceivepuclusecase.NewService(
		inboxmanagerreceivepuclusecase.Options{
			RepoSelector: store.managerReceivePUCLSelector,

			// Logger di sini WAJIB, bukan pelengkap. Modul lain mencatat hanya saat
			// penyaring kepemilikan dilepas; di modul ini penyaring itu memang tidak
			// pernah ada — layarnya pandangan penyelia, dan SETIAP pembukaannya dicatat.
			//
			// Sampai pemeriksaan peran ada (`TKT-F3-004`), jejak itulah satu-satunya
			// kontrol yang menyatakan siapa membuka antrean seluruh petugas (`D-59`).
			Logger: logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	rclPUCLService, err := inboxrclpuclusecase.NewService(
		inboxrclpuclusecase.Options{
			RepoSelector: store.rclPUCLSelector,

			// Logger WAJIB, dengan alasan yang sama seperti modul di atasnya DITAMBAH
			// satu: antrean layar ini bersama, sehingga tidak ada penyaring kepemilikan
			// sama sekali — dan berkas laporan hariannya dapat diunduh dengan rentang
			// tanggal yang ditentukan penggunanya sendiri. Rentang yang lebar adalah hal
			// yang harus dapat ditelusuri setelahnya (`D-59`).
			Logger: logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Report KPI PNC (`MENU_ID 84`), tab KPI Adjuster.
	//
	// Logger WAJIB, dan alasannya berbeda dari modul inbox di atasnya: yang dibaca layar
	// ini bukan pekerjaan melainkan PENILAIAN KINERJA adjuster yang dapat dinamai, dan
	// laporannya tidak disaring per pengguna sama sekali. Selama pemeriksaan peran belum
	// ada (`TKT-F3-004`), jejak yang menyebut siapa membukanya — beserta adjuster dan
	// periode yang dipilihnya — adalah satu-satunya kontrol pengimbang (`D-59`).
	reportKPIService, err := reportkpiusecase.NewService(
		reportkpiusecase.Options{
			RepoSelector: store.reportKPISelector,
			Logger:       logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Report Klaim (`MENU_ID 85`), 28 panel ekspor.
	//
	// Audit sengaja BELUM diisi, dan itu bukan kelalaian melainkan keadaan yang dicatat:
	// `S-5` adalah modul tersendiri yang belum ada, dan seam-nya dibuat justru supaya
	// pemasangannya kelak tidak menyentuh satu baris pun aturan di dalam modul ini.
	//
	// Akibatnya harus disadari. Berkas dari modul ini memuat data nasabah LINTAS CABANG,
	// dan `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang karena tidak ada
	// pemisahan tugas. Sampai `S-5` ada, yang tersisa hanyalah baris log biasa dari
	// lapisan HTTP — cukup untuk menelusuri, tidak cukup untuk dipertanggungjawabkan.
	reportKlaimService, err := reportklaimusecase.NewService(
		reportklaimusecase.Options{
			RepoSelector: store.reportKlaimSelector,
			Clock:        clock.System{},
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Logger disuntikkan dengan alasan yang mirip, tetapi ambangnya berbeda: yang diawasi
	// di sini adalah rekap per PIC, satu-satunya bagian layar ini yang TIDAK dipaginasi —
	// mengikuti sistem lama yang juga tidak memaginasinya.
	inboxProgressClaimService, err := inboxprogressclaimusecase.NewService(
		inboxprogressclaimusecase.Options{
			RepoSelector: store.inboxProgressClaimSelector,
			Clock:        clock.System{},
			Logger:       logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Inbox Analyst Doctor tidak menerima Clock di sini: waktu hanya dibutuhkan saat
	// menyusun jawaban — kolom "Lama Waktu Klaim" — bukan saat mengambil antreannya.
	// Menaruhnya di usecase akan menambah ketergantungan yang tidak dipakai satu baris pun.
	inboxAnalystDoctorService, err := inboxanalystdoctorusecase.NewService(
		inboxanalystdoctorusecase.Options{
			RepoSelector: store.inboxAnalystDoctorSelector,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	outstandingService, err := inboxoutstandingusecase.NewService(store.outstandingSelector)
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Pembangkit pengenal dan jam dipasok di sini, bukan dibaca modul dari jam sistem.
	//
	// Keduanya seam supaya waktu permintaan dapat diuji secara deterministik — dan supaya
	// tidak ada satu pun tempat di dalam modul yang memanggil time.Now() sendiri, yang
	// persis pola `Set7Hours` sistem lama yang menambah tujuh jam manual di 118 titik.
	closeClaimService, err := inboxcloseclaimusecase.NewService(inboxcloseclaimusecase.Options{
		Claims:   store.closeClaimSelector,
		Requests: store.closeClaimRequests,
		IDs:      inboxcloseclaimmemory.IDGenerator{},
		Clock:    clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	salvageService, err := inboxsalvageusecase.NewService(
		inboxsalvageusecase.Options{
			RepoSelector: store.salvageSelector,

			// Logger WAJIB, dan di modul ini alasannya paling kuat di antara seluruh
			// modul inbox: ia MENULIS nilai uang.
			//
			// `D-59` menetapkan satuan izin adalah menu dan TIDAK ada pemisahan tugas
			// formal — orang yang sama dapat membuat pengajuan salvage dan, bila perannya
			// memiliki menunya, menyetujuinya. Tidak ada kontrol teknis yang
			// mencegahnya, sehingga jejak inilah satu-satunya kontrol pengimbang yang
			// tersisa.
			Logger: logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	komunikasiCabangService, err := inboxkomunikasicabangusecase.NewService(
		inboxkomunikasicabangusecase.Options{
			RepoSelector: store.komunikasiCabangSelector,

			// BranchResolver WAJIB — dan modul ini menolak dibentuk tanpanya.
			//
			// Tanpa penerjemah, batas data layar ini tidak dapat ditentukan sama sekali,
			// dan satu-satunya jalan yang tersisa adalah menampilkan percakapan siapa saja.
			// Membiarkannya nil lalu "menanganinya nanti" adalah persis cara batas data
			// menghilang tanpa ada yang menyadarinya.
			BranchResolver: store.komunikasiCabangBranch,

			// Clock WAJIB sejak modul ini menulis (2026-09-24). Ia mengisi
			// `CREATEDATEREPLY` — tanggal balasan, yang menjadi DASAR PENGURUTAN tab
			// "Sudah Dijawab".
			//
			// Waktunya datang dari aplikasi, bukan dari basis data. Sistem lama memakai
			// `@CurrentDateTime()` pada `PNCReplyMessageCabang`, yaitu jam server aplikasi —
			// bukan `SYSDATE`. Itu direplikasi, dan sekaligus memenuhi `F-5`: satu-satunya
			// tempat waktu dibaca adalah seam ini, sehingga tidak ada penambahan 7 jam manual
			// yang dapat terselip.
			Clock: clock.System{},

			// Logger WAJIB, dengan satu alasan tambahan yang khas layar ini: petugas yang
			// kode cabangnya TIDAK terbaca dilayani sebagai kantor pusat (`P-5`), dan itu
			// pelebaran batas data yang tidak menghasilkan satu pun galat. Jejaknya adalah
			// satu-satunya hal yang dapat menjawab "siapa saja yang terkena" bila keputusan
			// itu kelak ditinjau ulang (`D-59`).
			//
			// Sejak modul ini menulis, jejaknya menjawab lebih daripada itu: siapa membalas
			// percakapan mana, dan siapa menutup percakapan yang tidak dapat dibuka kembali.
			Logger: logger,
		})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Policy tidak dipasok: DefaultPolicy dipakai — mode kumulatif, dan hanya
	// Non-MBU yang memakai pita dengan batas Rp 100.000.000 (`D-52`, `D-70`).
	//
	// Ia BELUM bergantung pada portal yang sedang melayani, dan itu batas yang disadari:
	// entitas Simasnet memakai mode satu-penyetuju, dan entitas SMI memakai batas pita
	// USD 7.000 — keduanya ada di rule yang sama (`Activity/SetEmailKomite-Act.xml`).
	// Menyambungkannya ke portal aktif adalah `TKT-F6-002`, yang menuntut portal melekat
	// pada permintaan alih-alih pada keadaan global (`R-20`).
	//
	// Randomizer dipasok sekarang meski portal ASM tidak memakainya: bila kelak portal
	// diganti ke mode satu-penyetuju, pemilihannya langsung acak — bukan diam-diam
	// selalu jatuh ke orang yang sama.
	komiteService, err := komiteusecase.NewService(komiteusecase.Options{
		Repo:       store.komite,
		Randomizer: random.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Inbox Komite memakai jam sistem dalam UTC, sama dengan modul lain. Ia dipasok
	// eksplisit — bukan dibiarkan memakai bawaan — supaya jelas terbaca bahwa Aging
	// dihitung dari jam SERVER, bukan jam peramban. Satu kenyataan tidak boleh punya dua
	// umur.
	komiteInboxService, err := komiteusecase.NewInboxService(komiteusecase.InboxOptions{
		Cases:     store.komiteInbox,
		Decisions: store.komiteDecision,
		IDs:       komitememory.IDGenerator{},
		Clock:     clock.System{},
	})
	if err != nil {
		store.close()
		return assembly{}, err
	}

	// Jam dan zona keduanya diserahkan, bukan dibaca di dalam modul: aturan "proteksi ganda
	// di hari yang sama" bergantung pada TANGGAL WIB, dan aturan yang membaca jam sendiri
	// tidak dapat diuji tanpa menunggu pergantian hari.
	protectionRequestService, err := inputreqprotectionusecase.NewService(
		inputreqprotectionusecase.Options{Protections: store.protectionRequestSelector},
	)
	if err != nil {
		store.close()
		return assembly{}, err
	}

	protectionAcceptService, err := inboxacceptopenprotectionusecase.NewService(
		inboxacceptopenprotectionusecase.Options{Protections: store.protectionAcceptSelector},
	)
	if err != nil {
		store.close()
		return assembly{}, err
	}

	return assembly{
		auth:                      service,
		portal:                    store.portal,
		masterRekening:            buildMasterRekening(cfg, store, logger),
		masterStatus:              claimStatusService,
		masterStatusProgres:       progressStatusService,
		masterStatusProgres2:      progressStatus2Service,
		masterPasalAI:             clauseAIService,
		laporanHasilAI:            aiReportService,
		masterAutoClaim:           masterAutoClaimService,
		masterBengkel:             workshopService,
		masterPanel:               panelService,
		masterSparepart:           sparepartService,
		masterGroupingSparepart:   groupingService,
		masterKategoriSparepart:   partCategoryService,
		masterTipeSparepart:       partTypeService,
		masterLogin:               surveyorLoginService,
		detailPenyebab:            causeOfLossDetailService,
		inboxInvestigator:         investigatorInboxService,
		inboxReceiveTKA:           receiveTKAService,
		masterReas:                reinsurerMemberService,
		masterPasal:               clauseService,
		masterSupplier:            supplierService,
		masterPenolakan:           rejectionService,
		masterPenolakanKomite:     rejectionKomiteService,
		riwayatKlaim:              claimHistoryService,
		archiveDokumenKlaim:       archiveDokumenKlaimService,
		masterDokumenTravel:       travelDocumentService,
		masterCOLSimasOnline:      simasOnlineCauseOfLossService,
		daftarTipeDokumen:         documentTypeService,
		daftarObjekDokumen:        documentObjectService,
		daftarDetailDokumenTravel: travelDocumentDetailService,
		daftarDetailTipeDokumen:   detailDocumentTypeService,
		daftarTipeDokumenBisnis:   businessDocumentRuleService,
		masterTipeSurveyors:       surveyorTypeService,
		masterSurveyors:           surveyorService,
		masterPicTeknik:           picTeknikService,
		masterRecovery:            recoveryService,
		masterDominanFactor:       dominantFactorService,
		masterXOL:                 xolService,
		masterPenyebabKerugian:    causeOfLossService,
		masterMasking:             maskingService,
		menu:                      menuService,
		extra:                     extra,
		readyAliases:              store.readyAliases,
		close:                     store.close,
		komite:                    komiteService,
		komiteInbox:               komiteInboxService,
		inboxAutoClaim:            autoClaimService,
		inboxXOL:                  inboxXOLService,
		inboxClaimTreatyProp:      claimTreatyPropService,
		inboxClaimTreatyNonProp:   claimTreatyNonPropService,
		inboxManagerReceivePUCL:   managerReceivePUCLService,
		inboxRCLPUCL:              rclPUCLService,
		inboxSalvage:              salvageService,
		reportKPI:                 reportKPIService,
		reportKlaim:               reportKlaimService,
		inboxKomunikasiCabang:     komunikasiCabangService,
		inboxProgressClaim:        inboxProgressClaimService,
		inboxAnalystDoctor:        inboxAnalystDoctorService,
		inboxLaporanKlaim:         claimReportService,
		inboxOutstanding:          outstandingService,
		inputReqProtection:        protectionRequestService,
		inboxAcceptOpenProtection: protectionAcceptService,
		inboxCloseClaim:           closeClaimService,
	}, nil
}

// buildMasterRekening menyusun modul Master Rekening di balik seam-nya, SATU LAYANAN PER
// PORTAL.
//
// Seam Kasir diisi klien HTTP nyata bila alamatnya sudah dikonfigurasi, dan tiruan bila
// belum. Perbedaannya diumumkan di log: layar yang tampak bekerja padahal pendaftaran
// ke Kasir tidak pernah terjadi adalah kegagalan yang tidak terlihat siapa pun sampai
// pembayaran pertama tertahan.
// buildReceiveTKANotifier menyiapkan pengirim pemberitahuan kelengkapan dokumen TKA.
//
// # nil BUKAN kegagalan, dan itu sengaja
//
// Modul lain memakai pengirim TIRUAN ketika SMTP belum dikonfigurasi. Di sini yang
// dikembalikan justru nil, dan bedanya terlihat oleh pengguna: layar membedakan
// "pemberitahuan belum dipasang" dari "pemberitahuan gagal dikirim", dan pengirim tiruan
// yang selalu berhasil akan melaporkan keadaan KETIGA yang tidak benar — bahwa surelnya
// terkirim.
//
// Pengisian tanggalnya sendiri tetap berjalan penuh. Pemberitahuan adalah akibat samping,
// bukan syarat; lihat usecase.Complete.
//
// # Penerimanya TIDAK pernah diturunkan dari siapa yang menekan tombol
//
// `Activity/SubmitTanggalLengkapTKA-Act.xml` memilih penerimanya dengan bercabang pada tiga
// Operator ID yang tertanam di dalam rule, salah satunya menunjuk akun surel pribadi di
// jalur produksi. Percabangan itu dicabut (`D-15`, `D-67`); yang berlaku adalah satu daftar
// dari `SMTP_PENERIMA_TKA`.
func buildReceiveTKANotifier(
	cfg config.Config,
	logger *slog.Logger,
) inboxreceivetka.Notifier {
	if !cfg.SMTP.TKAActive() {
		logger.Warn("pemberitahuan kelengkapan dokumen TKA tidak aktif",
			slog.String("akibat",
				"tanggal tetap tersimpan, tetapi tidak ada yang diberi tahu lewat surel"),
			slog.String("perbaikan",
				"isi SMTP_HOST, SMTP_PORT, SMTP_DARI, dan SMTP_PENERIMA_TKA"))
		return nil
	}

	return inboxreceivetkanotif.NewSender(inboxreceivetkanotif.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		User:     cfg.SMTP.User,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
		To:       cfg.SMTP.TKARecipients,
		Timeout:  cfg.SMTP.Timeout,
	})
}

// buildMasterRekening menyusun modul Master Rekening di balik seam-nya.
//
// # Kenapa satu layanan per portal (2026-09-19)
//
// Seam Kasir, seam Notifier, dan jam sama untuk seluruh entitas — ketiganya konfigurasi
// tingkat aplikasi. Yang BERBEDA per entitas ada dua, dan keduanya menentukan hasil:
//
//   - penyimpanannya, karena POOLDATA.LST_ACCOUNT ada di basis data setiap entitas;
//   - `PortalAlias`, yang menentukan apakah rekening yang disetujui DIDAFTARKAN KE KASIR
//     (`portalsRegisteredWithCashier` di usecase/decide.go).
//
// Sebelum penyelarasan ini, alias yang dipakai selalu portal UTAMA — sehingga keputusan
// "daftarkan ke Kasir atau tidak" dijawab dengan entitas yang salah bagi setiap pengguna
// yang sedang melihat entitas lain. Sekarang ia dijawab dengan entitas yang benar-benar
// dipilih.
//
// Layanannya dibuat saat pertama diminta lalu dipakai kembali; membuatnya ulang setiap
// permintaan berarti membuang seluruh keadaan seam-nya tanpa alasan.
func buildMasterRekening(cfg config.Config, store storage, logger *slog.Logger) func(string) (*masterrekeningusecase.Service, error) {
	var (
		cashierSystem masterrekening.Cashier
		notifier      masterrekening.Notifier
	)

	if cfg.SMTP.Active() {
		notifier = masterrekeningnotif.NewSender(masterrekeningnotif.Config{
			Host:     cfg.SMTP.Host,
			Port:     cfg.SMTP.Port,
			User:     cfg.SMTP.User,
			Password: cfg.SMTP.Password,
			From:     cfg.SMTP.From,
			To:       cfg.SMTP.AlertRecipients,
			Timeout:  cfg.SMTP.Timeout,
		})
	} else {
		notifier = &masterrekeningnotif.Fake{}
		logger.Warn("pengirim surel tiruan dipakai",
			slog.String("akibat", "Tim IT TIDAK diberi tahu lewat surel bila pendaftaran ke Cashier gagal"),
			slog.String("perbaikan", "isi SMTP_HOST, SMTP_PORT, SMTP_DARI, dan SMTP_PENERIMA_PERINGATAN"))
	}

	switch {
	case cfg.Cashier.Active():
		cashierSystem = masterrekeningcashier.NewClient(masterrekeningcashier.Config{
			RegisterURL: cfg.Cashier.RegisterURL,
			UpdateURL:   cfg.Cashier.UpdateURL,
			User:        cfg.Cashier.User,
			Password:    cfg.Cashier.Password,
			Timeout:     cfg.Cashier.Timeout,
		})

	case store.accountInOracle:
		// KASIR TIRUAN DILARANG DI ATAS BASIS DATA SUNGGUHAN.
		//
		// Fake menjawab "berhasil" beserta nomor rekening Kasir karangan. Bila
		// jawaban itu ditulis ke POOLDATA.LST_ACCOUNT yang asli, kolom STS_SERVICE
		// dan ID_REKASIR akan memuat jejak pendaftaran yang tidak pernah terjadi —
		// dan tidak ada apa pun sesudahnya yang dapat membedakannya dari pendaftaran
		// yang sungguhan. Data palsu di master rekening lebih berbahaya daripada
		// langkah yang hilang.
		//
		// Seam dibiarkan nil. Usecase sudah menanganinya: rekening tetap dapat
		// disetujui komite, dan langkah pendaftaran ke Kasir dilewati tanpa
		// meninggalkan jejak apa pun.
		cashierSystem = nil
		logger.Warn("pendaftaran ke Cashier DILEWATI",
			slog.String("sebab", "alamat sistem Cashier belum dikonfigurasi, sedangkan master rekening membaca basis data sungguhan"),
			slog.String("akibat", "rekening yang disetujui komite TIDAK didaftarkan ke Cashier, dan tidak ada jejak Cashier yang ditulis"),
			slog.String("perbaikan", "isi KASIR_URL_DAFTAR_REKENING dan KASIR_URL_PERBARUI_REKENING"))

	default:
		// Penyimpanan di memori: tiruan aman dipakai dan memang berguna, karena ia
		// membuat seluruh alur dapat dicoba tanpa basis data dan tanpa jaringan.
		cashierSystem = masterrekeningcashier.NewFake()
		logger.Warn("sistem Cashier tiruan dipakai",
			slog.String("akibat", "rekening yang disetujui komite TIDAK didaftarkan ke Cashier yang sesungguhnya"),
			slog.String("perbaikan", "isi KASIR_URL_DAFTAR_REKENING dan KASIR_URL_PERBARUI_REKENING"))
	}

	var lock sync.Mutex
	cache := map[string]*masterrekeningusecase.Service{}

	return func(alias string) (*masterrekeningusecase.Service, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))

		lock.Lock()
		defer lock.Unlock()
		if existing, already := cache[clean]; already {
			return existing, nil
		}

		accountRepo, bankRepo, err := store.accountSelector(clean)
		if err != nil {
			return nil, err
		}

		fresh := masterrekeningusecase.NewService(masterrekeningusecase.Options{
			Repo:     accountRepo,
			Bank:     bankRepo,
			Cashier:  cashierSystem,
			Notifier: notifier,
			Clock:    clock.System{},
			// Alias entitas YANG DIMINTA, bukan portal utama. Inilah yang membuat
			// keputusan pendaftaran ke Kasir dijawab dengan entitas yang benar.
			PortalAlias: clean,
		})
		cache[clean] = fresh
		return fresh, nil
	}
}

// needsOracle menyatakan apakah koneksi basis data harus dibuka.
//
// Dua sebab yang BERBEDA, dan memisahkannya penting:
//
//   - PENYIMPANAN=oracle  → tabel CPNC_PENGGUNA dan CPNC_SESI_AKTIF hidup di sana.
//   - IDENTITAS_ADAPTER=hcq → alamat layanan HCQ (GCNM_CONNECT_REST) dan daftar login
//     non-karyawan (M_LOGIN_PNC) dibaca dari sana, tanpa menyentuh tabel CPNC_ sama
//     sekali.
//
// Menyatukan keduanya — seperti yang saya lakukan mula-mula — memaksa migrasi 0001
// selesai sebelum integrasi HCC/HCQ dapat dicoba lewat layar, padahal keduanya tidak
// saling bergantung.
func needsOracle(cfg config.Config) bool {
	return cfg.Storage == config.StorageOracle ||
		cfg.IdentityAdapter == config.IdentityAdapterHCQ
}

// buildStorage membuka koneksi portal bila diperlukan dan memasang repo di atasnya.
func buildStorage(cfg config.Config, production bool, logger *slog.Logger) (storage, error) {
	if cfg.Storage == config.StorageMemory && production {
		// Session di memori satu instans melanggar tuntutan stateless (D-27): instans
		// kedua di belakang load balancer tidak akan mengenali sesi yang diterbitkan
		// instans pertama. Penolakannya ada di kode, bukan di nilai konfigurasi.
		return storage{}, errors.New(
			"penyimpanan memori menolak berjalan di lingkungan produksi: sesi wajib dikenali seluruh instans (D-27)")
	}

	store := storage{
		close:        func() {},
		readyAliases: func() []string { return nil },
	}

	if needsOracle(cfg) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		pool, err := db.NewPool(ctx, cfg.PrimaryPortal, portalParameters(cfg), func(alias string, err error) {
			// Portal yang gagal dibuka dicatat tetapi tidak menghentikan aplikasi:
			// pengisian kredensial tiap entitas berjalan bertahap, dan satu entitas
			// yang belum siap tidak boleh menghalangi entitas yang sudah siap.
			logger.Warn("portal tidak tersedia",
				slog.String("portal", alias),
				slog.String("sebab", err.Error()))
		})
		if err != nil {
			return storage{}, err
		}
		logger.Info("koneksi portal terbuka", slog.Any("portal", pool.Available()))

		// Koneksi KEDUA tiap portal — pengganti DB Link `@ASMD` yang `R-03` belum
		// sediakan API-nya (keputusan Work Owner 2026-09-24).
		//
		// Kegagalannya TIDAK PERNAH menghentikan start, dan ketiadaannya bukan galat:
		// aplikasi berjalan penuh, dan yang hilang hanyalah kolom laporan yang
		// membutuhkannya — kolom yang lalu dikosongkan dan ditandai di layar.
		//
		// Keadaan itu tetap DICATAT di log saat start supaya tidak lolos tanpa disadari,
		// persis perlakuan yang sama pada blok Kasir dan SMTP.
		anekaPool := db.NewOptionalPool(ctx, anekaParameters(cfg), func(alias string, err error) {
			logger.Warn("koneksi kedua portal tidak tersedia",
				slog.String("portal", alias),
				slog.String("sebab", err.Error()))
		})
		if tersedia := anekaPool.Available(); len(tersedia) > 0 {
			logger.Info("koneksi kedua portal terbuka", slog.Any("portal", tersedia))
		} else {
			logger.Warn("tidak ada koneksi kedua portal yang terbuka; " +
				"kolom laporan yang bersumber dari sana akan dikosongkan (R-03)")
		}
		primary := pool.Primary()
		store.legacy = sqlstore.NewLegacy(primary)
		store.portal = portalsql.NewRepo(primary)
		store.accountSelector = func(alias string) (masterrekening.Repo, masterrekening.BankRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, nil, err
			}
			return masterrekeningsql.NewRepo(conn), masterrekeningsql.NewBankRepo(conn), nil
		}
		store.accountInOracle = true
		store.readyAliases = pool.Available
		store.menu = menusql.NewRepo(primary)

		// Master ambang komite dipasang pada koneksi UTAMA, dan itu CACAT YANG DISADARI —
		// bukan sekadar sementara.
		//
		// Work Owner menegaskan 2026-09-19 bahwa setiap server punya POOLDATA-nya sendiri,
		// dan isi EMAILKOMITE BERBEDA antar server: baris SIMASNET hanya ada di POOLDATA
		// server Simasnet. Selama repo ini terpasang pada koneksi utama, portal mana pun
		// yang dipilih pengguna akan membaca tangga ambang milik portal UTAMA.
		//
		// Akibatnya bukan galat melainkan angka yang salah tanpa tanda: layar menampilkan
		// jenjang persetujuan entitas lain, dan tidak ada yang terlihat keliru. Itu kelas
		// kegagalan yang sama dengan `R-20`.
		//
		// Hari ini belum menimbulkan kerugian karena hanya portal utama yang dilayani.
		// Memperbaikinya adalah `TKT-F6-002` — koneksi diambil dari portal AKTIF, yang
		// menuntut portal melekat pada permintaan alih-alih pada keadaan global.
		store.komite = komitesql.NewRepo(primary)

		// Inbox Komite membaca tabel WARISAN — `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`,
		// `PC_ASSIGN_WORKLIST`, `T_CLAIM_KOMITE_LIST`, `T_CLAIM_DATA_RESULTS_AI` — dan
		// menulis keputusannya ke tabel MILIK APLIKASI INI (`CPNC_KOMITE_KEPUTUSAN`,
		// migrasi 0004). Keduanya dipasang pada koneksi yang sama.
		//
		// Cacat portalnya SAMA PERSIS dengan master ambang di atas, dan di sini akibatnya
		// lebih berat: yang salah portal bukan angka acuan melainkan DAFTAR PEKERJAAN
		// beserta nilai klaim dan nama tertanggung milik badan hukum lain. Itu `R-20`
		// secara harfiah, dan penutupannya `TKT-F6-002`.
		store.komiteInbox = komitesql.NewInboxRepo(primary)
		store.komiteDecision = komitesql.NewDecisionRepo(primary)

		// KEDUA kumpulan koneksi ditutup bersamaan. Menutup yang pertama saja akan
		// meninggalkan koneksi kedua tetap terbuka saat aplikasi berhenti — kebocoran
		// yang tidak terlihat sebagai galat, dan baru terbaca sebagai sesi menggantung
		// di sisi basis data.
		store.close = func() {
			pool.Close()
			anekaPool.Close()
		}

		// Setiap permintaan memilih koneksi entitasnya sendiri. Portal yang tidak
		// dikenal atau koneksinya belum hidup menghasilkan galat dari For(), TIDAK
		// pernah dialihkan ke koneksi utama sebagai cadangan.
		store.progressStatusSelector = func(alias string) (masterstatusprogres.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterstatusprogressql.NewRepo(conn), nil
		}

		// Master dokumen travel memakai pemilih yang sama bentuknya, dan dengan
		// jaminan yang sama: portal yang tidak dikenal atau belum hidup menghasilkan
		// galat dari For(), TIDAK pernah dialihkan ke koneksi utama sebagai cadangan.
		store.travelDocumentSelector = func(alias string) (masterdokumentravel.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterdokumentravelsql.NewRepo(conn), nil
		}
		store.progressStatus2Selector = func(alias string) (masterstatusprogres.Repo2, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterstatusprogressql.NewRepo2(conn), nil
		}
		store.masterAutoClaimSelector = func(alias string) (masterautoclaim.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterautoclaimsql.NewRepo(conn), nil
		}
		store.workshopSelector = func(alias string) (masterbengkel.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterbengkelsql.NewRepo(conn), nil
		}
		store.panelSelector = func(alias string) (masterpanel.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterpanelsql.NewRepo(conn), nil
		}
		store.sparepartSelector = func(alias string) (mastersparepart.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastersparepartsql.NewRepo(conn), nil
		}
		store.groupingSelector = func(alias string) (mastergroupingsparepart.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastergroupingsparepartsql.NewRepo(conn), nil
		}
		store.partCategorySelector = func(alias string) (masterkategorisparepart.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterkategorisparepartsql.NewRepo(conn), nil
		}
		store.partTypeSelector = func(alias string) (mastertipesparepart.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastertipesparepartsql.NewRepo(conn), nil
		}
		store.surveyorLoginSelector = func(alias string) (masterlogin.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterloginsql.NewRepo(conn), nil
		}
		store.clauseAISelector = func(alias string) (masterpasalai.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterpasalaisql.NewRepo(conn), nil
		}
		store.aiReportSelector = func(alias string) (laporanhasilai.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return laporanhasilaisql.NewRepo(conn), nil
		}
		store.causeOfLossDetailSelector = func(alias string) (detailpenyebab.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return detailpenyebabsql.NewRepo(conn), nil
		}
		store.investigatorInboxSelector = func(alias string) (inboxinvestigator.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxinvestigatorsql.NewRepo(conn), nil
		}
		store.receiveTKASelector = func(alias string) (inboxreceivetka.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxreceivetkasql.NewRepo(conn), nil
		}
		store.reinsurerMemberSelector = func(alias string) (masterreas.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterreassql.NewRepo(conn), nil
		}
		store.clauseSelector = func(alias string) (masterpasal.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterpasalsql.NewRepo(conn), nil
		}
		store.supplierSelector = func(alias string) (mastersupplier.Store, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastersuppliersql.NewRepo(conn), nil
		}
		store.rejectionSelector = func(alias string) (masterpenolakan.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterpenolakansql.NewRepo(conn), nil
		}
		store.rejectionKomiteSelector = func(alias string) (masterpenolakan.RepoKomite, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterpenolakansql.NewRepoKomite(conn), nil
		}

		store.claimHistorySelector = func(alias string) (riwayatklaim.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return riwayatklaimsql.NewRepo(conn), nil
		}

		store.claimProtectionSelector = func(alias string) (riwayatklaim.ProtectionRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return riwayatklaimsql.NewProtectionRepo(conn), nil
		}

		store.archiveSelector = func(alias string) (archivedokumenklaim.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return archivedokumenklaimsql.NewRepo(conn), nil
		}

		// Master COL Simas Online memakai DUA pemilih di atas koneksi yang sama: satu
		// untuk tabelnya sendiri, satu untuk POOLDATA.BUSINESS yang hanya dibacanya.
		// Keduanya memakai pool.For yang sama, sehingga daftar bisnis yang tampil pasti
		// berasal dari entitas yang sedang dipilih pengguna — bukan dari entitas lain.
		store.simasOnlineCauseOfLossSelector = func(alias string) (mastercolsimasonline.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastercolsql.NewRepo(conn), nil
		}
		store.businessSelector = func(alias string) (mastercolsimasonline.BusinessRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastercolsql.NewBusinessRepo(conn), nil
		}

		// Daftar tipe dokumen memakai pemilih yang sama bentuknya, dan dengan jaminan
		// yang sama: portal yang tidak dikenal atau belum hidup menghasilkan galat dari
		// For(), TIDAK pernah dialihkan ke koneksi utama sebagai cadangan.
		store.documentTypeSelector = func(alias string) (daftartipedokumen.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return daftartipedokumensql.NewRepo(conn), nil
		}

		// Daftar Objek Dokumen memakai DUA pemilih di atas koneksi yang sama, sama seperti
		// Master COL Simas Online: satu untuk tabelnya sendiri, satu untuk
		// POOLDATA.BUSINESS yang hanya dibacanya. Keduanya lewat pool.For yang sama,
		// sehingga daftar bisnis yang tampil pasti berasal dari entitas yang sedang
		// dipilih pengguna — bukan dari entitas lain.
		store.documentObjectSelector = func(alias string) (daftarobjekdokumen.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return daftarobjekdokumensql.NewRepo(conn), nil
		}
		store.documentObjectBusinessSelector = func(alias string) (daftarobjekdokumen.BusinessRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return daftarobjekdokumensql.NewBusinessRepo(conn), nil
		}

		// Daftar Detail Dokumen Travel memakai TIGA pemilih di atas koneksi yang sama:
		// satu untuk kedua tabelnya sendiri, satu untuk M_DOCTRAVEL, satu untuk
		// M_PLANTRAVEL. Ketiganya lewat pool.For yang sama, sehingga daftar pilihan yang
		// tampil pasti berasal dari entitas yang sedang dipilih pengguna — bukan dari
		// entitas lain (`R-20`).
		store.detailTravelSelector = func(alias string) (daftardetaildokumentravel.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return daftardetaildokumentravelsql.NewRepo(conn), nil
		}
		store.travelChoiceSelector = func(alias string) (daftardetaildokumentravel.DocumentRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return daftardetaildokumentravelsql.NewDocumentRepo(conn), nil
		}
		store.travelPlanSelector = func(alias string) (daftardetaildokumentravel.PlanRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return daftardetaildokumentravelsql.NewPlanRepo(conn), nil
		}

		// Daftar Detail Tipe Dokumen memakai DUA pemilih di atas koneksi yang sama.
		// Keempat master rujukannya dilayani satu repo — bukan empat — karena keempatnya
		// hidup di basis data entitas yang sama dan selalu dibaca bersamaan.
		store.detailDocumentTypeSelector = func(alias string) (daftardetailtipedokumen.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return daftardetailtipedokumensql.NewRepo(conn), nil
		}
		store.detailDocumentTypeReferenceSelector = func(alias string) (daftardetailtipedokumen.ReferenceRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return daftardetailtipedokumensql.NewReferenceRepo(conn), nil
		}

		// Daftar Tipe Dokumen Bisnis memakai LIMA pemilih di atas koneksi yang sama, dengan
		// alasan yang sama seperti ketiga pemilih di atas: seluruh daftar pilihan yang
		// tampil pasti berasal dari entitas yang sedang dipilih pengguna (`R-20`).
		store.businessDocumentRuleSelector = func(alias string) (daftartipedokumenbisnis.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return daftartipedokumenbisnissql.NewRepo(conn), nil
		}
		store.businessDocumentRuleBusiness = func(alias string) (daftartipedokumenbisnis.BusinessRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return daftartipedokumenbisnissql.NewBusinessRepo(conn), nil
		}
		store.businessDocumentRuleDocType = func(alias string) (daftartipedokumenbisnis.DocumentTypeRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return daftartipedokumenbisnissql.NewDocumentTypeRepo(conn), nil
		}
		store.businessDocumentRuleDetailDoc = func(alias string) (daftartipedokumenbisnis.DetailTypeDocRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return daftartipedokumenbisnissql.NewDetailTypeDocRepo(conn), nil
		}
		store.businessDocumentRuleObjectDoc = func(alias string) (daftartipedokumenbisnis.ObjectDocRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return daftartipedokumenbisnissql.NewObjectDocRepo(conn), nil
		}

		store.autoClaimSelector = func(alias string) (inboxautoclaim.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxautoclaimsql.NewRepo(conn), nil
		}

		store.claimReportSelector = func(alias string) (inboxlaporanklaim.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxlaporanklaimsql.NewRepo(conn, clock.System{}), nil
		}

		// Klaim dibaca dari basis data entitasnya sendiri, dengan aturan yang sama:
		// portal yang tidak dikenal atau belum siap menghasilkan galat dari For(),
		// TIDAK pernah dialihkan ke koneksi utama.
		store.outstandingSelector = func(alias string) (inboxoutstanding.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxoutstandingsql.NewRepo(conn), nil
		}

		// Repo proteksi dan repo master tipe dibentuk dari SATU koneksi yang sama, dan
		// dikembalikan bersamaan. Memilih keduanya lewat dua pemanggilan terpisah membuka
		// kemungkinan proteksi dibaca dari portal yang satu dan nama tipenya dari portal
		// yang lain — kelas cacat yang tidak menghasilkan galat apa pun.
		store.protectionRequestSelector = func(alias string) (inputreqprotection.Stores, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return inputreqprotection.Stores{}, err
			}
			return inputreqprotection.Stores{
				Protections: inputreqprotectionsql.NewRepo(conn),
				Types:       inputreqprotectionsql.NewTypeRepo(conn),
				Claims:      inputreqprotectionsql.NewClaimRepo(conn),
			}, nil
		}
		store.protectionAcceptSelector = func(alias string) (inboxacceptopenprotection.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxacceptopenprotectionsql.NewRepo(conn), nil
		}

		// Klaim TUTUP dibaca dari basis data entitasnya sendiri, dengan aturan yang sama
		// seperti klaim berjalan di atasnya.
		store.closeClaimSelector = func(alias string) (inboxcloseclaim.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxcloseclaimsql.NewRepo(conn), nil
		}

		// Permintaan ReOpen dan Copy Klaim ditulis ke basis data entitas yang SAMA dengan
		// klaimnya — bukan ke portal utama.
		//
		// Ini bukan pilihan kerapian: permintaan atas klaim milik satu badan hukum yang
		// tercatat di basis data badan hukum lain adalah kebocoran yang persis `R-20`
		// larang, dan pada modul ini akibatnya melampaui tampilan.
		//
		// Tabelnya dibuat migrasi `0006`, yang BELUM dijalankan DBA di lingkungan mana pun.
		// Sampai itu terjadi, pembacaannya gagal dan usecase menanganinya sebagai
		// "permintaan tidak dapat dibaca" — daftarnya tetap tampil, penandanya tidak muncul,
		// dan kegagalannya dicatat di log.
		store.closeClaimRequests = func(alias string) (inboxcloseclaim.RequestRepo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxcloseclaimsql.NewRequestRepo(conn), nil
		}

		store.surveyorTypeSelector = func(alias string) (mastertipesurveyors.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastertipesurveyorssql.NewRepo(conn), nil
		}
		store.surveyorSelector = func(alias string) (mastersurveyors.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastersurveyorssql.NewRepo(conn), nil
		}
		store.claimStatusSelector = func(alias string) (masterstatus.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterstatussql.NewRepo(conn), nil
		}
		store.maskingSelector = func(alias string) (mastermasking.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return mastermaskingsql.NewRepo(conn), nil
		}
		store.picTeknikSelector = func(alias string) (masterpicteknik.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterpictekniksql.NewRepo(conn), nil
		}
		store.dominantFactorSelector = func(alias string) (masterdominanfactor.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterdominanfactorsql.NewRepo(conn), nil
		}
		store.xolSelector = func(alias string) (masterxol.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterxolsql.NewRepo(conn), nil
		}
		store.causeOfLossSelector = func(alias string) (masterpenyebabkerugian.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterpenyebabkerugiansql.NewRepo(conn), nil
		}
		store.recoverySelector = func(alias string) (masterrecovery.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return masterrecoverysql.NewRepo(conn), nil
		}

		// Kesepuluh modul yang perakitannya ada di modules.go memakai kolam koneksi yang
		// sama, dengan jaminan yang sama pula.
		setExtraOracleSelectors(pool, &store)
		store.inboxXOLSelector = func(alias string) (inboxxol.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxxolsql.NewRepo(conn), nil
		}

		store.claimTreatyPropSelector = func(alias string) (inboxclaimtreatyprop.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxclaimtreatypropsql.NewRepo(conn), nil
		}

		store.claimTreatyNonPropSelector = func(
			alias string,
		) (inboxclaimtreatynonprop.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxclaimtreatynonpropsql.NewRepo(conn), nil
		}

		store.managerReceivePUCLSelector = func(
			alias string,
		) (inboxmanagerreceivepucl.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxmanagerreceivepuclsql.NewRepo(conn), nil
		}

		store.reportKPISelector = func(alias string) (reportkpi.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return reportkpisql.NewRepo(conn), nil
		}

		// Report Klaim adalah satu-satunya modul yang menerima DUA koneksi: basis data
		// portalnya, dan koneksi KEDUA portal yang sama (`ANEKA_<PORTAL_ALIAS>_*`) —
		// pengganti DB Link `@ASMD` yang `R-03` belum sediakan API-nya.
		//
		// Koneksi keduanya BOLEH tidak ada, dan ketiadaannya tidak menggagalkan apa pun:
		// yang hilang hanyalah kolom laporan yang bersumber dari sana, dan kolom itu
		// dikosongkan serta ditandai. Karena itu kegagalan For() di sini diterjemahkan
		// menjadi nil — bukan dilewatkan diam-diam, melainkan dinyatakan sebagai "tidak
		// ada koneksi kedua", keadaan yang memang sudah ditangani repo.
		//
		// Yang TIDAK boleh gagal diam-diam adalah koneksi portalnya sendiri; galatnya
		// dikembalikan apa adanya, tidak pernah dialihkan ke koneksi utama.
		store.reportKlaimSelector = func(alias string) (reportklaim.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			if !anekaPool.Has(alias) {
				return reportklaimsql.NewRepo(conn, nil), nil
			}
			second, err := anekaPool.For(alias)
			if err != nil {
				return reportklaimsql.NewRepo(conn, nil), nil
			}
			return reportklaimsql.NewRepo(conn, second), nil
		}

		store.rclPUCLSelector = func(alias string) (inboxrclpucl.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxrclpuclsql.NewRepo(conn), nil
		}

		store.salvageSelector = func(alias string) (inboxsalvage.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxsalvagesql.NewRepo(conn), nil
		}

		store.komunikasiCabangSelector = func(
			alias string,
		) (inboxkomunikasicabang.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxkomunikasicabangsql.NewRepo(conn), nil
		}

		store.inboxProgressClaimSelector = func(alias string) (inboxprogressclaim.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxprogressclaimsql.NewRepo(conn), nil
		}

		store.inboxAnalystDoctorSelector = func(alias string) (inboxanalystdoctor.Repo, error) {
			conn, err := pool.For(alias)
			if err != nil {
				return nil, err
			}
			return inboxanalystdoctorsql.NewRepo(conn), nil
		}

		store.claimReportBranch = inboxlaporanklaimsql.NewBranchResolver(primary)

		// Penerjemah cabang KEDUA, milik modul Inbox Komunikasi Cabang.
		//
		// Ia membaca objek yang sama lewat kueri yang sama, dan itu BUKAN pengulangan yang
		// terlewat: seam-nya dideklarasikan di paket yang memakainya, sehingga kedua modul
		// dapat berpindah ke API pengganti DB Link (`D-25`) pada waktu yang berbeda.
		//
		// Keduanya menunjuk `primary` dengan alasan yang sama — HRD dan master pengguna
		// asuransi adalah data lingkup identitas, bukan data per entitas.
		store.komunikasiCabangBranch = inboxkomunikasicabangsql.NewBranchResolver(primary)
	} else {
		store.portal = portalmemory.NewRepo(portalmemory.SampleList()...)
		store.accountSelector = accountSelectorMemory(cfg.PrimaryPortal)
		// Ke-33 status nyata ikut dimuat, sehingga layar Master Status Klaim dapat
		// dicoba lengkap tanpa Oracle dan tanpa menunggu migrasi 0002.
		store.readyAliases = func() []string { return []string{cfg.PrimaryPortal} }
		// Isi master ambang yang sebenarnya ikut dimuat, sehingga layar Ambang Komite
		// dan simulasi penjenjangan dapat dicoba lengkap tanpa Oracle — termasuk
		// ketujuh kasus pada spec B-7.
		store.komite = komitememory.NewSampleRepo()
		// Kasus komite contoh ikut dimuat, sehingga layar Inbox Komite dapat dicoba
		// LENGKAP — termasuk alur keputusannya — tanpa Oracle dan tanpa menunggu migrasi
		// 0004. Seluruh isinya KARANGAN, berbeda dari master ambang di atas: tidak ada
		// satu pun ekstrak antrean komite yang pernah diserahkan kepada kami. Lihat
		// repo/memory/inbox_sample.go.
		inboxStore := komitememory.NewSampleInboxStore()
		store.komiteInbox = inboxStore
		store.komiteDecision = inboxStore
		store.progressStatusSelector = progressStatusSelectorMemory(cfg.PrimaryPortal)
		store.travelDocumentSelector = travelDocumentSelectorMemory(cfg.PrimaryPortal)

		// Satu master bisnis dipakai bersama seluruh pemilih COL di memori, meniru
		// kenyataannya: bisnis dan COL hidup di basis data yang SAMA pada satu entitas.
		simasOnlineBusiness := mastercolmemory.NewBusinessRepo(mastercolmemory.SampleBusinessList()...)
		store.simasOnlineCauseOfLossSelector = simasOnlineCauseOfLossSelectorMemory(cfg.PrimaryPortal, simasOnlineBusiness)
		store.businessSelector = businessSelectorMemory(cfg.PrimaryPortal, simasOnlineBusiness)
		store.documentTypeSelector = documentTypeSelectorMemory(cfg.PrimaryPortal)

		// Kedua pemilih Daftar Objek Dokumen dirakit bersamaan dengan alasan yang sama
		// seperti pasangan COL di atas: objek dokumen dan bisnis hidup di basis data yang
		// SAMA pada satu entitas.
		//
		// Master bisnisnya instans TERSENDIRI, bukan simasOnlineBusiness di atas, karena
		// tipenya memang berbeda — setiap modul mendeklarasikan seam-nya sendiri. Isinya
		// sengaja sama persis, sehingga kedua layar tetap menampilkan master yang sama.
		documentObjectBusiness := daftarobjekdokumenmemory.NewBusinessRepo(daftarobjekdokumenmemory.SampleBusinessList()...)
		store.documentObjectSelector = documentObjectSelectorMemory(cfg.PrimaryPortal)
		store.documentObjectBusinessSelector = documentObjectBusinessSelectorMemory(cfg.PrimaryPortal, documentObjectBusiness)

		// Ketiga pemilih Daftar Detail Dokumen Travel dirakit bersamaan, meniru
		// kenyataannya: detail, master dokumen, dan master plan hidup di basis data yang
		// SAMA pada satu entitas.
		store.detailTravelSelector = detailTravelSelectorMemory(cfg.PrimaryPortal)
		store.travelChoiceSelector = travelChoiceSelectorMemory(cfg.PrimaryPortal)
		store.travelPlanSelector = travelPlanSelectorMemory(cfg.PrimaryPortal)

		// Kedua pemilih Daftar Detail Tipe Dokumen. Pembaca masternya dirakit LEBIH DULU
		// lalu disambungkan ke repo detailnya — tanpa itu, baris yang baru disimpan akan
		// tampil tanpa nama tipe dokumen dan tanpa nama bisnis, karena di Oracle ketiga
		// keterangan itu datang dari join view dan tidak tersimpan di barisnya.
		detailDocumentTypeReference := daftardetailtipedokumenmemory.NewSampleReferenceRepo()
		store.detailDocumentTypeSelector = detailDocumentTypeSelectorMemory(cfg.PrimaryPortal, detailDocumentTypeReference)
		store.detailDocumentTypeReferenceSelector = detailDocumentTypeReferenceMemory(cfg.PrimaryPortal, detailDocumentTypeReference)

		// Kelima pemilih Daftar Tipe Dokumen Bisnis, dirakit bersamaan dengan alasan yang
		// sama: aturan dokumen beserta keempat masternya hidup di basis data yang SAMA
		// pada satu entitas.
		//
		// Keempat master pilihan memakai instans TERSENDIRI meski dua di antaranya
		// membaca tabel yang sama dengan modul lain di atas — tipenya memang berbeda,
		// karena setiap modul mendeklarasikan seam-nya sendiri.
		store.businessDocumentRuleSelector = businessDocumentRuleSelectorMemory(cfg.PrimaryPortal)
		store.businessDocumentRuleBusiness = businessDocumentRuleBusinessMemory(cfg.PrimaryPortal)
		store.businessDocumentRuleDocType = businessDocumentRuleDocTypeMemory(cfg.PrimaryPortal)
		store.businessDocumentRuleDetailDoc = businessDocumentRuleDetailDocMemory(cfg.PrimaryPortal)
		store.businessDocumentRuleObjectDoc = businessDocumentRuleObjectDocMemory(cfg.PrimaryPortal)

		store.autoClaimSelector = autoClaimSelectorMemory(cfg.PrimaryPortal)
		store.progressStatus2Selector = progressStatus2SelectorMemory(store.progressStatusSelector)
		store.masterAutoClaimSelector = masterAutoClaimSelectorMemory(cfg.PrimaryPortal)
		store.claimHistorySelector = claimHistorySelectorMemory(cfg.PrimaryPortal)
		store.claimProtectionSelector = claimProtectionSelectorMemory(cfg.PrimaryPortal)
		store.archiveSelector = archiveSelectorMemory(cfg.PrimaryPortal)
		store.workshopSelector = workshopSelectorMemory(cfg.PrimaryPortal)
		store.panelSelector = panelSelectorMemory(cfg.PrimaryPortal)
		store.sparepartSelector = sparepartSelectorMemory(cfg.PrimaryPortal)
		store.groupingSelector = groupingSelectorMemory(cfg.PrimaryPortal)
		store.partCategorySelector = partCategorySelectorMemory(cfg.PrimaryPortal)
		store.partTypeSelector = partTypeSelectorMemory(cfg.PrimaryPortal)
		store.surveyorLoginSelector = surveyorLoginSelectorMemory(cfg.PrimaryPortal)
		store.clauseAISelector = clauseAISelectorMemory(cfg.PrimaryPortal)
		store.aiReportSelector = aiReportSelectorMemory(cfg.PrimaryPortal)
		store.causeOfLossDetailSelector = causeOfLossDetailSelectorMemory(cfg.PrimaryPortal)
		store.reinsurerMemberSelector = reinsurerMemberSelectorMemory(cfg.PrimaryPortal)
		store.investigatorInboxSelector = investigatorInboxSelectorMemory(cfg.PrimaryPortal)
		store.receiveTKASelector = receiveTKASelectorMemory(cfg.PrimaryPortal)
		store.clauseSelector = clauseSelectorMemory(cfg.PrimaryPortal)
		store.supplierSelector = supplierSelectorMemory(cfg.PrimaryPortal)
		store.rejectionSelector = rejectionSelectorMemory(cfg.PrimaryPortal)
		store.rejectionKomiteSelector = rejectionKomiteSelectorMemory(cfg.PrimaryPortal)
		store.claimReportSelector = claimReportSelectorMemory(cfg.PrimaryPortal)

		// memisahkan di produksi adalah KONEKSI basis data yang berbeda.
		//
		// Klaim contohnya mencakup lima Group Panel, sehingga batas data per lini dapat
		// dicoba tanpa Oracle dan tanpa menunggu migrasi 0004. Seluruh isinya karangan —
		// lihat repo/memory/sample.go.
		outstandingMemory := inboxoutstandingmemory.NewRepoWithSamples()
		store.outstandingSelector = func(string) (inboxoutstanding.Repo, error) {
			return outstandingMemory, nil
		}

		// Master tipe proteksi ikut dimuat berisi kesembilan tipe yang benar-benar ada di
		// `POOLDATA.M_CLAIM_PROTECTION_TYPE`, sehingga pilihan tipe pada form menampilkan
		// nama yang SAMA dengan produksi tanpa Oracle. Ia salinan, bukan cadangan — tidak
		// ada jalur yang jatuh ke sini saat master di Oracle kosong.
		protectionRequestMemory := inputreqprotectionmemory.NewRepoWithSamples()
		protectionTypeMemory := inputreqprotectionmemory.NewTypeRepoWithSamples()
		// Klaim contoh ikut dimuat supaya form tipe 7 dan 8 dapat dicoba utuh tanpa
		// Oracle — termasuk field turunan yang tidak dapat diketik.
		protectionClaimMemory := inputreqprotectionmemory.NewClaimRepoWithSamples()
		store.protectionRequestSelector = func(string) (inputreqprotection.Stores, error) {
			return inputreqprotection.Stores{
				Protections: protectionRequestMemory,
				Types:       protectionTypeMemory,
				Claims:      protectionClaimMemory,
			}, nil
		}

		protectionAcceptMemory := inboxacceptopenprotectionmemory.NewRepoWithSamples()
		store.protectionAcceptSelector = func(string) (inboxacceptopenprotection.Repo, error) {
			return protectionAcceptMemory, nil
		}
		// Inbox Close Claim memakai SATU penyimpanan untuk klaim dan permintaannya.
		//
		// Berbeda dari perakitan SQL di atas, yang memisahkan keduanya karena tabelnya
		// dimiliki sistem yang berbeda. Di memori tidak ada kepemilikan tabel, dan
		// menyatukannya justru yang membuat permintaan yang dicatat langsung terlihat pada
		// daftar — persis perilaku yang hendak dicoba tanpa Oracle.
		//
		// Seluruh isinya karangan — lihat repo/memory/sample.go.
		closeClaimMemory := inboxcloseclaimmemory.NewStoreWithSamples()
		store.closeClaimSelector = func(string) (inboxcloseclaim.Repo, error) {
			return closeClaimMemory, nil
		}
		store.closeClaimRequests = func(string) (inboxcloseclaim.RequestRepo, error) {
			return closeClaimMemory, nil
		}
		// Keempat tipe surveyor nyata ikut dimuat, sehingga layar Master Tipe Surveyors
		// dapat dicoba lengkap tanpa Oracle.
		store.surveyorTypeSelector = surveyorTypeSelectorMemory(cfg.PrimaryPortal)
		// Empat contoh surveyor ikut dimuat, satu per posisi persetujuan, sehingga kelima
		// tab layar Master Surveyors dapat dicoba tanpa Oracle DAN tanpa menunggu migrasi
		// 0004 — yang berbeda dari migrasi 0003 bersifat WAJIB.
		store.surveyorSelector = surveyorSelectorMemory(cfg.PrimaryPortal)
		// Ke-33 status nyata ikut dimuat, sehingga layar Master Status Klaim dapat dicoba
		// lengkap tanpa Oracle dan tanpa menunggu migrasi 0002.
		store.claimStatusSelector = claimStatusSelectorMemory(cfg.PrimaryPortal)
		// Sepuluh contoh faktor dominan ikut dimuat — dan TEKS-nya dikarang, bukan isi
		// master yang sebenarnya. Isi aslinya belum pernah diterima dari DBA; lihat
		// peringatan di masterdominanfactor/repo/memory/sample.go.
		store.dominantFactorSelector = dominantFactorSelectorMemory(cfg.PrimaryPortal)
		// Empat contoh Master XOL ikut dimuat, dan bentuknya MENIRU produksi termasuk
		// keanehannya: nomor berlubang, Type XOL kosong, lapisan tanpa isi, dan satu
		// lapisan yang total share-nya nol. Keempatnya nyata — lihat peringatan di
		// masterxol/repo/memory/sample.go.
		store.xolSelector = xolSelectorMemory(cfg.PrimaryPortal)
		// Sepuluh contoh penyebab kerugian ikut dimuat — dan KETERANGANNYA dikarang,
		// bukan isi master yang sebenarnya; hanya bentuk ID-nya yang diturunkan dari
		// bukti. Salah satunya berketerangan KOSONG, supaya keputusan "keterangan kosong
		// diterima" dapat dicoba sungguhan tanpa Oracle. Lihat peringatan di
		// masterpenyebabkerugian/repo/memory/sample.go.
		store.causeOfLossSelector = causeOfLossSelectorMemory(cfg.PrimaryPortal)
		// Empat contoh PIC teknik ikut dimuat — salah satunya NONAKTIF, supaya keputusan
		// "daftar hanya menampilkan yang aktif" dapat dicoba sungguhan tanpa Oracle.
		store.picTeknikSelector = picTeknikSelectorMemory(cfg.PrimaryPortal)
		// Dua principal contoh dan dua acuan polis ikut dimuat, sehingga layar Master
		// Recovery dapat dicoba utuh tanpa Oracle — termasuk penolakan "polis tidak
		// ditemukan" untuk nomor selain keduanya.
		store.recoverySelector = recoverySelectorMemory(cfg.PrimaryPortal)
		// Lima contoh masking ikut dimuat — satu di antaranya NONAKTIF dan satu tanpa sub
		// modul, supaya dua keputusan yang paling mudah salah dapat dicoba sungguhan tanpa
		// Oracle: bahwa "hapus" hanya menonaktifkan, dan bahwa sub modul boleh kosong.
		store.maskingSelector = maskingSelectorMemory(cfg.PrimaryPortal)
		setExtraMemorySelectors(cfg.PrimaryPortal, &store)
		store.claimReportBranch = inboxlaporanklaimmemory.NewBranchResolver(
			inboxlaporanklaimmemory.SampleBranchOfLogin())

		// Pemetaan contohnya SENGAJA berbeda dari milik Inbox Laporan Klaim: di sana
		// `adminpnc` berada di cabang 1001, di sini ia berada di kantor pusat.
		//
		// Perbedaan itu bukan kelalaian. Layar ini punya jalur kantor pusat yang tidak ada
		// di layar itu, dan tanpa saksi yang cabangnya BENAR-BENAR terbaca sebagai kantor
		// pusat, jalur itu hanya dapat dicapai lewat kegagalan penerjemahan — sehingga
		// "petugas pusat" dan "cabang tidak terbaca" tidak akan pernah dapat dibedakan saat
		// pengembangan.
		store.komunikasiCabangBranch = inboxkomunikasicabangmemory.NewSampleBranchResolver()
		// NewDevRepo, bukan NewSampleRepo: isi contoh m_login_group_pnc.csv hanya
		// memuat satu login, dan login provider tiruan tidak ada di dalamnya. Tanpa
		// itu, masuk saat pengembangan menghasilkan menu kosong yang tampak rusak.
		store.menu = menumemory.NewDevRepo()
		store.inboxXOLSelector = inboxXOLSelectorMemory(cfg.PrimaryPortal)
		store.claimTreatyPropSelector = claimTreatyPropSelectorMemory(cfg.PrimaryPortal)
		store.claimTreatyNonPropSelector = claimTreatyNonPropSelectorMemory(cfg.PrimaryPortal)
		// Sepuluh baris contoh ikut dimuat, dan lima di antaranya sengaja TIDAK muncul di
		// tab mana pun — berkas tanpa Group Panel, klaim yang bocor ke tabel penugasan per
		// orang, klaim yang sudah selesai, dan klaim di antrean bersama lain. Tanpa baris
		// yang tertolak, layar pengembangan tidak dapat menunjukkan bahwa penyaringnya
		// benar-benar bekerja.
		store.managerReceivePUCLSelector = managerReceivePUCLSelectorMemory(cfg.PrimaryPortal)
		store.rclPUCLSelector = rclPUCLSelectorMemory(cfg.PrimaryPortal)
		// Sembilan pengajuan salvage dan delapan klaim contoh, disusun supaya KETIGA BELAS
		// daftarnya punya isi — daftar yang kosong saat pengembangan berarti penyaringnya
		// salah, bukan datanya habis, dan tanpa isi keduanya terlihat sama.
		store.salvageSelector = salvageSelectorMemory(cfg.PrimaryPortal)
		store.reportKPISelector = reportKPISelectorMemory(cfg.PrimaryPortal)
		store.reportKlaimSelector = reportklaimmemory.Selector(cfg.PrimaryPortal)
		// Sepuluh percakapan contoh ikut dimuat, empat di antaranya SENGAJA tertolak —
		// percakapan yang sudah ditutup, yang tanpa pengirim, yang tanpa pesan, dan yang
		// milik cabang lain. Tanpa keempatnya, penyaring yang hilang tidak akan ketahuan
		// saat pengembangan.
		store.komunikasiCabangSelector = komunikasiCabangSelectorMemory(cfg.PrimaryPortal)
		store.inboxProgressClaimSelector = inboxProgressClaimSelectorMemory(cfg.PrimaryPortal)
		store.inboxAnalystDoctorSelector = inboxAnalystDoctorSelectorMemory(cfg.PrimaryPortal)
	}

	switch cfg.Storage {
	case config.StorageOracle:
		primary := store.legacy.DB()
		store.user = sqlstore.NewUserRepo(primary)
		store.session = sqlstore.NewSessionRepo(primary)

	case config.StorageMemory:
		// Catatan dan sesi di memori. Dipakai bersama IDENTITAS_ADAPTER=hcq, ini
		// memungkinkan masuk dengan kredensial SUNGGUHAN sebelum migrasi 0001
		// dijalankan DBA — yang hilang hanya ketahanan sesi terhadap restart dan
		// pengenalan sesi lintas instans.
		if cfg.IdentityAdapter == config.IdentityAdapterHCQ {
			logger.Warn("identitas nyata dengan penyimpanan memori",
				slog.String("akibat", "sesi hilang saat restart dan tidak dikenali instans lain; hanya untuk pengujian"))
		}
		store.user = memory.NewUserRepo()
		store.session = memory.NewSessionRepo()

	default:
		store.close()
		return storage{}, fmt.Errorf("penyimpanan %q tidak dikenal", cfg.Storage)
	}

	return store, nil
}

// progressStatusSelectorMemory menyusun penyimpanan master status progres di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali. Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab.
//
// Hanya portal utama yang dilayani di sini, sejalan dengan readyAliases pada cabang
// tanpa Oracle yang juga menyebut portal utama saja. Memilih portal lain tanpa basis
// data karena itu ditolak dengan galat yang sama seperti di produksi: perilaku
// penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
func progressStatusSelectorMemory(primaryAlias string) masterstatusprogres.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterstatusprogres.Repo{}

	return func(alias string) (masterstatusprogres.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterstatusprogresmemory.NewRepo(masterstatusprogresmemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// travelDocumentSelectorMemory menyusun penyimpanan master dokumen travel di memori.
//
// Bentuknya sengaja sama persis dengan progressStatusSelectorMemory di atas, termasuk
// alasannya: satu portal mendapat satu penyimpanan yang dibuat saat pertama diminta
// lalu dipakai kembali, dan portal selain portal utama ditolak dengan galat yang sama
// seperti di produksi — sehingga perilaku penolakannya ikut teruji saat pengembangan.
func travelDocumentSelectorMemory(primaryAlias string) masterdokumentravel.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterdokumentravel.Repo{}

	return func(alias string) (masterdokumentravel.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterdokumentravelmemory.NewRepo(masterdokumentravelmemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// autoClaimSelectorMemory menyusun penyimpanan Inbox Auto Claim di memori.
//
// Bentuknya sengaja sama persis dengan progressStatusSelectorMemory, termasuk
// penyimpanan per portal yang dibuat sekali lalu dipakai kembali: unggahan yang baru
// disimpan harus tetap ada pada permintaan berikutnya, dan penyimpanan yang dibuat ulang
// tiap permintaan akan membuat layar tampak kehilangan data tanpa sebab.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain karena itu ditolak dengan galat yang sama seperti di produksi —
// perilaku penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
func autoClaimSelectorMemory(primaryAlias string) inboxautoclaim.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxautoclaim.Repo{}

	return func(alias string) (inboxautoclaim.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxautoclaimmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// claimReportSelectorMemory menyusun penyimpanan berkas laporan klaim di memori.
//
// Bentuknya sama persis dengan progressStatusSelectorMemory, dan alasannya pun sama:
// satu penyimpanan per portal, dibuat saat pertama diminta lalu dipakai kembali. Kalau
// dibuat ulang setiap permintaan, berkas yang baru dibuat lewat tombol "Buat Baru" akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab.
func claimReportSelectorMemory(primaryAlias string) inboxlaporanklaim.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxlaporanklaim.Repo{}
	systemClock := clock.System{}

	return func(alias string) (inboxlaporanklaim.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxlaporanklaimmemory.NewRepo(inboxlaporanklaimmemory.SampleOptions(systemClock))
		store[clean] = fresh
		return fresh, nil
	}
}

// surveyorTypeSelectorMemory menyusun penyimpanan master tipe surveyor di memori.
//
// Bentuknya sama dengan progressStatusSelectorMemory dan alasannya pun sama: satu portal
// mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali. Kalau
// dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang pada
// permintaan berikutnya dan layarnya tampak rusak tanpa sebab.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti
// di produksi — perilaku penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
func surveyorTypeSelectorMemory(primaryAlias string) mastertipesurveyors.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastertipesurveyors.Repo{}

	return func(alias string) (mastertipesurveyors.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := mastertipesurveyorsmemory.NewRepo(mastertipesurveyorsmemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// documentTypeSelectorMemory menyusun penyimpanan daftar tipe dokumen di memori.
//
// Bentuknya sengaja sama persis dengan travelDocumentSelectorMemory di atas, termasuk
// alasannya: satu portal mendapat satu penyimpanan yang dibuat saat pertama diminta lalu
// dipakai kembali, dan portal selain portal utama ditolak dengan galat yang sama seperti
// di produksi — sehingga perilaku penolakannya ikut teruji saat pengembangan.
func documentTypeSelectorMemory(primaryAlias string) daftartipedokumen.RepoSelector {
	var lock sync.Mutex
	store := map[string]daftartipedokumen.Repo{}

	return func(alias string) (daftartipedokumen.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := daftartipedokumenmemory.NewRepo(daftartipedokumenmemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// groupingSelectorMemory menyusun penyimpanan Master Grouping Sparepart di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
// Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang pada
// permintaan berikutnya dan layarnya tampak rusak tanpa sebab — dan pada modul ini akibatnya
// lebih jauh: pencacah ID DAN nomor grup ikut mundur, sehingga dua grouping dapat lahir dengan
// kunci yang sama dan dua kendaraan berbeda dapat berbagi satu nomor grup.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti di
// produksi.
//
// Penyimpanan contoh memuat ketiga status persetujuan sekaligus beserta keempat sumber
// acuannya, dan DUA baris yang berbagi satu nomor grup — tanpa itu, layar tidak pernah
// memperlihatkan apa gunanya modul ini.
func groupingSelectorMemory(primaryAlias string) mastergroupingsparepart.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastergroupingsparepart.Store{}

	return func(alias string) (mastergroupingsparepart.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := mastergroupingsparepartmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// partCategorySelectorMemory menyusun penyimpanan Master Kategori Sparepart di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
// Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang pada
// permintaan berikutnya — dan pada modul ini akibatnya lebih jauh: nomor urut yang lahir
// dari `max+1` ikut mundur, sehingga dua kategori dapat lahir dengan kunci yang sama.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti di
// produksi.
//
// # Satu hal yang TIDAK terjadi di modus memori, dan itu perlu disadari saat mencoba
//
// Modul ini dan Master Sparepart membaca tabel yang SAMA di produksi, tetapi punya
// penyimpanan memori SENDIRI-SENDIRI di sini. Kategori yang ditambahkan lewat layar ini
// karena itu tidak muncul di dropdown Kategori pada layar Master Sparepart selama aplikasi
// berjalan tanpa Oracle.
//
// Menyatukan keduanya akan menuntut salah satu modul mengimpor penyimpanan modul lain —
// tautan yang tidak ada di produksi, dan yang membuat modul selesai harus disunting setiap
// kali tetangganya berubah. Keterbatasan ini dibiarkan dan dicatat, bukan ditambal.
func partCategorySelectorMemory(primaryAlias string) masterkategorisparepart.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterkategorisparepart.Store{}

	return func(alias string) (masterkategorisparepart.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterkategorisparepartmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// partTypeSelectorMemory menyusun penyimpanan Master Tipe Sparepart di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
// Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang pada
// permintaan berikutnya — dan pada modul ini akibatnya lebih jauh: nomor urut yang lahir
// dari `max+1` ikut mundur, sehingga dua tipe dapat lahir dengan kunci yang sama.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti di
// produksi.
//
// # Dua hal yang TIDAK terjadi di modus memori, dan keduanya perlu disadari saat mencoba
//
//  1. Modul ini dan Master Sparepart membaca tabel tipe yang SAMA di produksi, tetapi punya
//     penyimpanan memori SENDIRI-SENDIRI di sini. Tipe yang ditambahkan lewat layar ini
//     tidak muncul di dropdown Tipe pada layar Master Sparepart selama berjalan tanpa
//     Oracle.
//  2. Dropdown Kategori pada layar ini dilayani SALINAN acuan milik penyimpanan ini sendiri
//     (lihat SampleCategoryList), bukan oleh penyimpanan Master Kategori Sparepart.
//     Kategori yang ditambahkan di layar itu karena itu tidak muncul di sini.
//
// Menyatukan keduanya akan menuntut salah satu modul mengimpor penyimpanan modul lain —
// tautan yang tidak ada di produksi, dan yang membuat modul selesai harus disunting setiap
// kali tetangganya berubah. Keterbatasan ini dibiarkan dan dicatat, bukan ditambal.
func partTypeSelectorMemory(primaryAlias string) mastertipesparepart.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastertipesparepart.Store{}

	return func(alias string) (mastertipesparepart.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := mastertipesparepartmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// surveyorLoginSelectorMemory menyusun penyimpanan Master Login di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
// Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang pada
// permintaan berikutnya — dan pada modul ini akibatnya khas: LOGINLEADER baris baru
// diturunkan dengan MEMBACA penyimpanan yang sama, sehingga penyimpanan yang lahir kembali
// akan membuat setiap penambahan tampak seolah pemanggilnya tidak pernah terdaftar.
func surveyorLoginSelectorMemory(primaryAlias string) masterlogin.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterlogin.Repo{}

	return func(alias string) (masterlogin.Repo, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterloginmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// clauseAISelectorMemory menyusun penyimpanan Master Pasal AI di memori.
//
// Alasannya sama dengan progressStatusSelectorMemory: satu portal satu penyimpanan, dibuat
// saat pertama diminta lalu dipakai kembali, dan hanya portal utama yang dilayani.
//
// Bedanya satu: **inilah satu-satunya jalur yang benar-benar menampilkan layar ini hari
// ini.** Pada penyimpanan Oracle, modul ini menjawab 503 sampai kueri `GetListDataPasalAI`
// dan `CountDataPasalAI` diterima dari Tim Pega.
func clauseAISelectorMemory(primaryAlias string) masterpasalai.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterpasalai.Repo{}

	return func(alias string) (masterpasalai.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterpasalaimemory.NewRepo(masterpasalaimemory.SampleClause())
		store[clean] = fresh
		return fresh, nil
	}
}

// aiReportSelectorMemory menyusun penyimpanan Laporan Hasil AI di memori.
//
// Alasannya sama dengan clauseAISelectorMemory: satu portal satu penyimpanan, dibuat saat
// pertama diminta lalu dipakai kembali, dan hanya portal utama yang dilayani.
//
// Modul ini baca-saja, sehingga "dipakai kembali" tidak menjaga apa pun yang ditulis — ia
// hanya menghemat pembentukan baris contoh pada setiap permintaan. Bentuknya tetap
// disamakan dengan pemilih lain supaya tidak ada satu modul yang merakit dirinya dengan
// cara yang berbeda tanpa alasan.
func aiReportSelectorMemory(primaryAlias string) laporanhasilai.RepoSelector {
	var lock sync.Mutex
	store := map[string]laporanhasilai.Repo{}

	return func(alias string) (laporanhasilai.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := laporanhasilaimemory.NewRepo(laporanhasilaimemory.SampleRows()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// causeOfLossDetailSelectorMemory menyusun penyimpanan Detail Penyebab Kerugian di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
// Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang pada
// permintaan berikutnya — dan pada modul ini akibatnya khas: nomor urut D_COL_ID ikut lahir
// kembali, sehingga baris berikutnya menerima ID yang sudah dipakai dan ditolak sebagai
// kunci ganda.
func causeOfLossDetailSelectorMemory(primaryAlias string) detailpenyebab.RepoSelector {
	var lock sync.Mutex
	store := map[string]detailpenyebab.Store{}

	return func(alias string) (detailpenyebab.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := detailpenyebabmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// investigatorInboxSelectorMemory menyusun penyimpanan Inbox Investigator di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
//
// Seperti Master Reas, alasannya BUKAN menjaga penambahan agar tidak hilang — modul ini
// tidak menulis apa pun. Yang dijaga adalah IDENTITAS penyimpanannya: repo yang lahir
// kembali setiap permintaan membuat galat yang dipasang lewat SetError menghilang di antara
// dua permintaan, sehingga jalur gagal tidak dapat dicoba saat pengembangan.
//
// Portal selain yang utama DITOLAK dengan `portal.ErrNotReady` lewat memoryPortal — bukan
// diberi penyimpanan kosong. Pembedaannya penting: "belum tersedia" dan "antreannya memang
// kosong" adalah dua hal berbeda, dan menjawab yang pertama dengan daftar kosong membuat
// petugas menyimpulkan tidak ada pekerjaan padahal basis datanya belum tersambung.
func investigatorInboxSelectorMemory(primaryAlias string) inboxinvestigator.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxinvestigator.Repo{}

	return func(alias string) (inboxinvestigator.Repo, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxinvestigatormemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// receiveTKASelectorMemory menyusun penyimpanan Inbox Receive TKA di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
//
// Di sini alasannya BUKAN sekadar menjaga identitas penyimpanan seperti pada modul
// baca-saja: modul ini MENULIS. Repo yang lahir kembali setiap permintaan akan memunculkan
// lagi baris yang barusan diisi, sehingga jalur "baris hilang setelah dikerjakan" — ciri
// kedua Inbox pada `D-79` — tidak dapat dicoba sama sekali saat pengembangan.
//
// Portal selain yang utama DITOLAK dengan `portal.ErrNotReady` lewat memoryPortal — bukan
// diberi penyimpanan kosong. Pembedaannya penting: "belum tersedia" dan "memang tidak ada
// pekerjaannya" adalah dua hal berbeda.
func receiveTKASelectorMemory(primaryAlias string) inboxreceivetka.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxreceivetka.Repo{}

	return func(alias string) (inboxreceivetka.Repo, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxreceivetkamemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// reinsurerMemberSelectorMemory menyusun penyimpanan Master Reas di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
//
// Berbeda dari pemilih modul lain, alasannya BUKAN menjaga penambahan agar tidak hilang —
// modul ini tidak menulis apa pun. Yang dijaga adalah IDENTITAS penyimpanannya: repo yang
// lahir kembali setiap permintaan membuat galat yang dipasang lewat SetError menghilang di
// antara dua permintaan, sehingga jalur gagal tidak dapat dicoba saat pengembangan.
func reinsurerMemberSelectorMemory(primaryAlias string) masterreas.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterreas.Repo{}

	return func(alias string) (masterreas.Repo, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterreasmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// clauseSelectorMemory menyusun penyimpanan Master Pasal Kerugian di memori.
// documentObjectSelectorMemory menyusun penyimpanan Daftar Objek Dokumen di memori.
//
// Bentuknya sama persis dengan documentTypeSelectorMemory di atas, dan kesamaannya
// disengaja: hanya portal UTAMA yang dilayani, dan alias lain DITOLAK — bukan diam-diam
// dialihkan ke portal utama. Menjalankan tanpa basis data tidak boleh mengubah aturan
// pemisahan entitas, karena justru di lingkungan itulah pelanggarannya paling mudah lolos
// (`R-20`).
func documentObjectSelectorMemory(primaryAlias string) daftarobjekdokumen.RepoSelector {
	var lock sync.Mutex
	store := map[string]daftarobjekdokumen.Repo{}

	return func(alias string) (daftarobjekdokumen.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := daftarobjekdokumenmemory.NewRepo(daftarobjekdokumenmemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// documentObjectBusinessSelectorMemory melayani master bisnis milik modul Daftar Objek
// Dokumen di memori.
//
// Master yang sama dipakai untuk setiap portal yang dilayani — dan karena hanya portal utama
// yang dilayani tanpa basis data, tidak ada dua entitas yang berbagi satu master di sini.
// Penolakan portal lain tetap sama seperti di produksi.
func documentObjectBusinessSelectorMemory(
	primaryAlias string,
	business *daftarobjekdokumenmemory.BusinessRepo,
) daftarobjekdokumen.BusinessRepoSelector {
	return func(alias string) (daftarobjekdokumen.BusinessRepo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}
		return business, nil
	}
}

// surveyorSelectorMemory menyusun penyimpanan Master Surveyors di memori.
//
// Bentuknya sama persis dengan surveyorTypeSelectorMemory, dan kesamaannya disengaja:
// keduanya melayani modul yang datanya hidup per entitas, sehingga keduanya WAJIB menolak
// portal selain portal utama alih-alih diam-diam melayaninya dari satu tempat (`R-20`).
func surveyorSelectorMemory(primaryAlias string) mastersurveyors.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastersurveyors.Repo{}

	return func(alias string) (mastersurveyors.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := mastersurveyorsmemory.NewRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// businessDocumentRuleSelectorMemory menyusun penyimpanan aturan dokumen bisnis di
// memori.
//
// Bentuknya sama persis dengan pemilih memori modul lain, dan kesamaannya disengaja:
// seluruhnya melayani modul yang datanya hidup per entitas, sehingga seluruhnya WAJIB
// menolak portal selain portal utama alih-alih diam-diam melayaninya dari satu tempat
// (`R-20`).
func businessDocumentRuleSelectorMemory(primaryAlias string) daftartipedokumenbisnis.RepoSelector {
	var lock sync.Mutex
	store := map[string]daftartipedokumenbisnis.Repo{}

	return func(alias string) (daftartipedokumenbisnis.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := daftartipedokumenbisnismemory.NewRepo(daftartipedokumenbisnismemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// businessDocumentRuleBusinessMemory menyusun pembaca master lini bisnis di memori.
func businessDocumentRuleBusinessMemory(primaryAlias string) daftartipedokumenbisnis.BusinessRepoSelector {
	shared := daftartipedokumenbisnismemory.NewBusinessRepo(daftartipedokumenbisnismemory.SampleBusinessList()...)

	return func(alias string) (daftartipedokumenbisnis.BusinessRepo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}
		return shared, nil
	}
}

// Ketiga pemilih daftar pilihan di bawah sengaja ditulis terpisah meski isinya nyaris
// sama.
//
// Menyatukannya menuntut satu fungsi yang mengembalikan tiga tipe antarmuka berbeda, dan
// itu hanya mungkin dengan generik atau dengan menukar tipe kembaliannya menjadi tipe
// beton — yang pertama menambah satu konsep demi menghemat enam baris, yang kedua
// melemahkan seam-nya. Ketiganya dibiarkan apa adanya.

// businessDocumentRuleDocTypeMemory menyusun pembaca master tahap dokumen di memori.
func businessDocumentRuleDocTypeMemory(primaryAlias string) daftartipedokumenbisnis.DocumentTypeRepoSelector {
	shared := daftartipedokumenbisnismemory.NewReferenceRepo(daftartipedokumenbisnismemory.SampleDocumentTypeList()...)

	return func(alias string) (daftartipedokumenbisnis.DocumentTypeRepo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}
		return shared, nil
	}
}

// businessDocumentRuleDetailDocMemory menyusun pembaca master rincian dokumen di memori.
func businessDocumentRuleDetailDocMemory(primaryAlias string) daftartipedokumenbisnis.DetailTypeDocRepoSelector {
	shared := daftartipedokumenbisnismemory.NewReferenceRepo(daftartipedokumenbisnismemory.SampleDetailTypeDocList()...)

	return func(alias string) (daftartipedokumenbisnis.DetailTypeDocRepo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}
		return shared, nil
	}
}

// businessDocumentRuleObjectDocMemory menyusun pembaca master objek dokumen di memori.
func businessDocumentRuleObjectDocMemory(primaryAlias string) daftartipedokumenbisnis.ObjectDocRepoSelector {
	shared := daftartipedokumenbisnismemory.NewReferenceRepo(daftartipedokumenbisnismemory.SampleObjectDocList()...)

	return func(alias string) (daftartipedokumenbisnis.ObjectDocRepo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}
		return shared, nil
	}
}

// detailTravelSelectorMemory menyusun penyimpanan detail dokumen travel di memori.
//
// Bentuknya sengaja sama persis dengan documentTypeSelectorMemory di atas, termasuk
// alasannya: satu portal mendapat satu penyimpanan yang dibuat saat pertama diminta lalu
// dipakai kembali, dan portal selain portal utama ditolak dengan galat yang sama seperti
// di produksi — sehingga perilaku penolakannya ikut teruji saat pengembangan.
// detailDocumentTypeSelectorMemory menyiapkan penyimpanan Daftar Detail Tipe Dokumen di
// memori untuk portal utama saja.
//
// Pembaca masternya diterima sebagai argumen, bukan dibuat di dalam, supaya repo yang
// dikembalikan membaca DAFTAR YANG SAMA dengan yang dilayani rute `/pilihan`. Bila
// keduanya dibuat terpisah, layar akan menampilkan keterangan yang tidak pernah cocok
// dengan pilihan yang ditawarkannya sendiri — kelas kebingungan yang tidak ada di Oracle,
// tempat keduanya memang satu basis data.
func detailDocumentTypeSelectorMemory(
	primaryAlias string,
	reference *daftardetailtipedokumenmemory.ReferenceRepo,
) daftardetailtipedokumen.RepoSelector {
	var lock sync.Mutex
	store := map[string]daftardetailtipedokumen.Repo{}

	return func(alias string) (daftardetailtipedokumen.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := daftardetailtipedokumenmemory.NewRepo(daftardetailtipedokumenmemory.SampleList()...)
		fresh.UseReferences(reference)
		store[clean] = fresh
		return fresh, nil
	}
}

// detailDocumentTypeReferenceMemory melayani keempat master rujukan di memori.
//
// Satu instans dibagi seluruh permintaan portal utama — bukan satu per permintaan —
// karena isinya memang data acuan yang sama, dan karena repo detailnya sudah memegang
// instans yang sama ini lewat UseReferences.
func detailDocumentTypeReferenceMemory(
	primaryAlias string,
	reference *daftardetailtipedokumenmemory.ReferenceRepo,
) daftardetailtipedokumen.ReferenceRepoSelector {
	return func(alias string) (daftardetailtipedokumen.ReferenceRepo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}
		return reference, nil
	}
}

func detailTravelSelectorMemory(primaryAlias string) daftardetaildokumentravel.RepoSelector {
	var lock sync.Mutex
	store := map[string]daftardetaildokumentravel.Repo{}

	return func(alias string) (daftardetaildokumentravel.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := daftardetaildokumentravelmemory.NewRepo(daftardetaildokumentravelmemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// buildArchiveGateway menyusun seam layanan Arsip milik modul Archive Dokumen Klaim.
//
// # Kenapa alamatnya dibaca dari katalog, bukan dari konfigurasi
//
// Karena begitulah alamat layanan luar sudah diperlakukan di aplikasi ini — provider
// HCC/HCQ dan direktori pegawai membacanya dari tabel yang sama. Tiga hal mengikuti dari
// itu: perpindahan endpoint menjadi perubahan data yang dilakukan DBA, tiap portal
// entitas boleh punya alamat Arsip sendiri, dan hostname produksi tidak pernah masuk ke
// berkas yang di-commit (`D-69`).
//
// `legacy` adalah pembaca katalog yang dipakai ulang apa adanya dari modul auth. Ia
// disuntikkan sebagai antarmuka sempit yang dideklarasikan modul archivedokumenklaim
// sendiri, sehingga kedua modul tetap tidak saling mengimpor — yang tahu keduanya hanyalah
// berkas perakitan ini.
//
// # Kapan perekam dipakai sebagai gantinya
//
// Saat koneksi basis data tidak dibuka, sehingga katalognya pun tidak ada. Perekam
// MENCATAT pengiriman alih-alih mengirimkannya, dan perbedaannya diumumkan di log, bukan
// dibiarkan senyap: layar yang melaporkan "berkas dikirim ke sistem Arsip" padahal tidak
// ada yang terkirim adalah kegagalan yang baru ketahuan saat berkas fisiknya dicari.
func buildArchiveGateway(
	legacy *sqlstore.Legacy,
	logger *slog.Logger,
) archivedokumenklaim.Gateway {
	if legacy == nil {
		logger.Warn("layanan Arsip memakai perekam; pengiriman TIDAK sampai ke sistem Arsip",
			slog.String("modul", "archivedokumenklaim"),
			slog.String("sebab", "koneksi basis data tidak dibuka, sehingga katalog alamat layanan tidak ada"),
		)
		return archivedokumenklaimgateway.NewRecorder()
	}

	client, err := archivedokumenklaimgateway.NewHTTP(archivedokumenklaimgateway.Options{
		Catalog: legacy,
	})
	if err != nil {
		// Tidak mungkin terjadi selama legacy bukan nil, dan justru karena itu ia tidak
		// boleh menjatuhkan aplikasi: modul lain tidak ada urusannya dengan layanan
		// Arsip, dan mematikan seluruh aplikasi karena satu seam adalah harga yang tidak
		// sepadan. Pengirimannya gagal dengan pesan yang jelas; sisanya tetap berjalan.
		logger.Error("klien layanan Arsip gagal dirakit; memakai perekam",
			slog.String("modul", "archivedokumenklaim"),
			slog.String("galat", err.Error()),
		)
		return archivedokumenklaimgateway.NewRecorder()
	}

	return client
}

// archiveSelectorMemory menyusun penyimpanan Archive Dokumen Klaim di memori.
//
// Bentuknya sama persis dengan claimHistorySelectorMemory: hanya portal UTAMA yang
// dilayani, dan alias lain DITOLAK — bukan diam-diam dialihkan ke portal utama.
// Menjalankan tanpa basis data tidak boleh mengubah aturan pemisahan entitas, karena
// justru di lingkungan itulah pelanggarannya paling mudah lolos (`R-20`).
//
// Instans disimpan per alias supaya berkas yang disimpan pengguna bertahan selama aplikasi
// hidup; membuat repo baru setiap permintaan akan membuang setiap penyimpanan.
func archiveSelectorMemory(primaryAlias string) archivedokumenklaim.RepoSelector {
	var lock sync.Mutex
	store := map[string]archivedokumenklaim.Repo{}

	return func(alias string) (archivedokumenklaim.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := archivedokumenklaimmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// buildEmployeeDirectory menyusun seam direktori pegawai milik modul Master PIC Teknik.
//
// # Kenapa pilihannya mengikuti adapter identitas
//
// Keduanya menembak API yang sama dan membaca alamatnya dari baris POOLDATA.GCNM_CONNECT_REST
// yang sama. Menyalakan salah satunya saja akan menghasilkan keadaan yang membingungkan:
// pengguna dapat MASUK lewat HCQ sungguhan, tetapi nama yang dicari layar Master PIC
// Teknik datang dari daftar karangan — atau sebaliknya.
//
// `legacy` adalah pembaca katalog layanan yang dipakai ulang apa adanya dari modul auth.
// Ia disuntikkan sebagai antarmuka sempit `ServiceCatalog` yang dideklarasikan modul
// masterpicteknik sendiri, sehingga kedua modul tetap tidak saling mengimpor — yang tahu
// keduanya hanyalah berkas perakitan ini.
func buildEmployeeDirectory(
	cfg config.Config,
	legacy *sqlstore.Legacy,
	logger *slog.Logger,
) (masterpicteknik.EmployeeDirectory, error) {
	if cfg.IdentityAdapter != config.IdentityAdapterHCQ || legacy == nil {
		// Direktori tiruan. Ia menjaga layar tetap dapat dicoba utuh tanpa Oracle,
		// termasuk jalur penolakan "ID operator tidak terdaftar".
		//
		// Perbedaannya diumumkan di log, bukan dibiarkan senyap: layar yang tampak bekerja
		// padahal nama yang ditemukannya karangan adalah kegagalan yang tidak terlihat
		// siapa pun sampai petugas pertama tersimpan dengan nama yang salah.
		logger.Warn("direktori pegawai memakai daftar tiruan",
			slog.String("modul", "masterpicteknik"),
			slog.String("sebab", "IDENTITAS_ADAPTER bukan hcq, atau koneksi basis data tidak dibuka"),
		)
		return masterpicteknikdirectory.NewFake(masterpicteknikdirectory.SampleEmployees()...), nil
	}

	return masterpicteknikdirectory.New(masterpicteknikdirectory.Options{
		Catalog:  legacy,
		User:     cfg.HCQ.User,
		Password: cfg.HCQ.Password,
		Timeout:  cfg.HCQ.Timeout,
		// Galat "baris tidak terdaftar" milik modul auth diteruskan sebagai nilai, bukan
		// diimpor tipenya: itulah yang membuat modul ini dapat MEMBEDAKAN katalog yang
		// belum diisi dari jaringan yang sedang putus, tanpa bergantung pada modul auth.
		NotRegistered: provider.ErrServiceNotRegistered,
	})
}

// picTeknikSelectorMemory menyusun penyimpanan master PIC teknik di memori.
//
// Bentuknya sama persis dengan surveyorTypeSelectorMemory, dan kesamaannya disengaja:
// hanya portal UTAMA yang dilayani, dan alias lain DITOLAK — bukan diam-diam dialihkan ke
// portal utama. Menjalankan tanpa basis data tidak boleh mengubah aturan pemisahan
// entitas, karena justru di lingkungan itulah pelanggarannya paling mudah lolos (`R-20`).
//
// Instans disimpan per alias supaya perubahan yang disimpan pengguna bertahan selama
// aplikasi hidup; membuat repo baru setiap permintaan akan membuang setiap penyuntingan.
func picTeknikSelectorMemory(primaryAlias string) masterpicteknik.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterpicteknik.Repo{}

	return func(alias string) (masterpicteknik.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterpicteknikmemory.NewRepo(masterpicteknikmemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// travelChoiceSelectorMemory menyusun pembaca daftar pilihan ID Dokumen di memori.
//
// Isinya SAMA dengan contoh modul Master Dokumen Travel, dan itu disengaja: keduanya
// membaca POOLDATA.M_DOCTRAVEL yang sama. Memakai isi yang berbeda akan menampilkan
// isian ID Dokumen yang tidak pernah cocok dengan daftarnya sendiri — kelas kebingungan
// yang tidak ada di produksi.
//
// Ia BACA-SAJA, sehingga instansnya tidak perlu dipakai kembali antarpermintaan seperti
// ketiga pemilih penyimpanan: tidak ada keadaan yang dapat hilang.
func travelChoiceSelectorMemory(primaryAlias string) daftardetaildokumentravel.DocumentRepoSelector {
	shared := daftardetaildokumentravelmemory.NewDocumentRepo(daftardetaildokumentravelmemory.SampleDocumentList()...)

	return func(alias string) (daftardetaildokumentravel.DocumentRepo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}
		return shared, nil
	}
}

// travelPlanSelectorMemory menyusun pembaca plan dan jaminan Travel di memori.
//
// BACA-SAJA dengan alasan yang sama seperti travelChoiceSelectorMemory di atas.
func travelPlanSelectorMemory(primaryAlias string) daftardetaildokumentravel.PlanRepoSelector {
	shared := daftardetaildokumentravelmemory.NewPlanRepo(
		daftardetaildokumentravelmemory.SamplePlanList(),
		daftardetaildokumentravelmemory.SampleCoverageList(),
	)

	return func(alias string) (daftardetaildokumentravel.PlanRepo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}
		return shared, nil
	}
}

// simasOnlineCauseOfLossSelectorMemory menyusun penyimpanan master COL Simas Online di
// memori.
//
// Bentuknya sama dengan kedua pemilih di atas, dengan satu tambahan: master bisnis
// disuntikkan supaya nama bisnis yang tampil di layar berasal dari master yang sama
// dengan yang dipakai memeriksa keberadaannya. Tanpa itu, layar akan menampilkan nama
// kosong untuk bisnis yang sebenarnya ada — dan gejalanya akan tampak seperti cacat
// penyimpanan, padahal hanya perakitannya yang kurang.
//
// Namanya dibedakan dari causeOfLossSelectorMemory di bawah dengan sengaja: keduanya
// melayani modul yang berbeda meski sama-sama bernama "penyebab kerugian".
func simasOnlineCauseOfLossSelectorMemory(
	primaryAlias string,
	business *mastercolmemory.BusinessRepo,
) mastercolsimasonline.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastercolsimasonline.Repo{}

	return func(alias string) (mastercolsimasonline.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := mastercolmemory.NewRepo(mastercolmemory.SampleList()...).WithBusiness(business)
		store[clean] = fresh
		return fresh, nil
	}
}

// recoverySelectorMemory menyusun penyimpanan Master Recovery di memori.
//
// Bentuknya sama persis dengan picTeknikSelectorMemory, dan kesamaannya disengaja: hanya
// portal UTAMA yang dilayani, dan alias lain DITOLAK — bukan diam-diam dialihkan ke portal
// utama. Menjalankan tanpa basis data tidak boleh mengubah aturan pemisahan entitas,
// karena justru di lingkungan itulah pelanggarannya paling mudah lolos (`R-20`).
//
// Instans disimpan per alias supaya batch dan virtual account yang baru dicatat bertahan
// selama aplikasi hidup; membuat repo baru setiap permintaan akan membuat penerbitan VA
// tampak selalu "baru" dan perilaku pakai-ulangnya tidak pernah dapat dicoba.
func recoverySelectorMemory(primaryAlias string) masterrecovery.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterrecovery.Repo{}

	return func(alias string) (masterrecovery.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterrecoverymemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// businessSelectorMemory melayani master bisnis di memori.
//
// Master yang sama dipakai untuk setiap portal yang dilayani — dan karena hanya portal
// utama yang dilayani tanpa basis data, tidak ada dua entitas yang berbagi satu master
// di sini. Penolakan portal lain tetap sama seperti di produksi.
func businessSelectorMemory(
	primaryAlias string,
	business *mastercolmemory.BusinessRepo,
) mastercolsimasonline.BusinessRepoSelector {
	return func(alias string) (mastercolsimasonline.BusinessRepo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}
		return business, nil
	}
}

// buildVirtualAccountIssuer menyusun seam penerbit rekening virtual milik Master Recovery.
//
// # Kenapa pilihannya mengikuti PENYIMPANAN, bukan adapter identitas
//
// Berbeda dari direktori pegawai, yang mengikuti `IDENTITAS_ADAPTER` karena keduanya
// menembak API yang sama. Penerbit VA menembak layanan yang berbeda, dan alamatnya dibaca
// dari POOLDATA.GCNM_CONNECT_REST — baris `TYPESERVICE='GENERATEDVA'`. Tanpa koneksi
// Oracle, alamat itu tidak dapat dibaca sama sekali, sehingga yang menentukan adalah ada
// atau tidaknya koneksi.
//
// # Kenapa tiruan BUKAN sekadar kenyamanan di sini
//
// Alamat yang terdaftar menunjuk layanan Pega yang MENERBITKAN REKENING SUNGGUHAN.
// Menembaknya dari lingkungan pengembangan meninggalkan rekening nyata yang tidak diminta
// siapa pun, pada sistem yang dipakai orang lain.
//
// Perbedaannya diumumkan di log, bukan dibiarkan senyap: layar yang tampak bekerja padahal
// nomor yang ditampilkannya karangan adalah kegagalan yang tidak terlihat siapa pun sampai
// dana pertama dikirim ke nomor itu.
func buildVirtualAccountIssuer(
	cfg config.Config,
	legacy *sqlstore.Legacy,
	logger *slog.Logger,
) (masterrecovery.VirtualAccountIssuer, error) {
	if legacy == nil {
		logger.Warn("penerbit virtual account memakai nomor tiruan",
			slog.String("modul", "masterrecovery"),
			slog.String("sebab", "koneksi basis data tidak dibuka, sehingga alamat layanan pada POOLDATA.GCNM_CONNECT_REST tidak dapat dibaca"),
		)
		return masterrecoveryva.NewFake(), nil
	}

	return masterrecoveryva.NewPega(masterrecoveryva.Options{
		Catalog: legacy,
		// Kredensial HCQ dipakai ulang sebagai Basic Auth bila terisi. Layanan ini tidak
		// diketahui menuntut autentikasi — 18 dari 21 Connect REST di sistem lama
		// ber-`pyUseAuthentication=false` (`D-73`) — dan bila keduanya kosong, header
		// Authorization tidak dikirim sama sekali.
		User:     cfg.HCQ.User,
		Password: cfg.HCQ.Password,
		// Galat "baris tidak terdaftar" milik modul auth diteruskan sebagai nilai, bukan
		// diimpor tipenya: itulah yang membuat modul ini dapat MEMBEDAKAN katalog yang
		// belum diisi dari jaringan yang sedang putus, tanpa bergantung pada modul auth.
		NotRegistered: provider.ErrServiceNotRegistered,
	})
}

// accountSelectorMemory menyusun penyimpanan master rekening di memori.
//
// Repo dan BankRepo dipilih bersamaan karena keduanya hidup di basis data yang sama;
// memisahkannya akan membuka kemungkinan rekening satu entitas dipasangkan dengan daftar
// bank entitas lain.
func accountSelectorMemory(primaryAlias string) func(string) (masterrekening.Repo, masterrekening.BankRepo, error) {
	var lock sync.Mutex
	type pair struct {
		account masterrekening.Repo
		bank    masterrekening.BankRepo
	}
	store := map[string]pair{}

	return func(alias string) (masterrekening.Repo, masterrekening.BankRepo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing.account, existing.bank, nil
		}
		fresh := pair{
			account: masterrekeningmemory.NewRepo(),
			bank:    masterrekeningmemory.NewBankRepo(masterrekeningmemory.SampleBanks()...),
		}
		store[clean] = fresh
		return fresh.account, fresh.bank, nil
	}
}

// claimStatusSelectorMemory menyusun penyimpanan master status klaim di memori.
//
// Bentuknya sama dengan kedua pemilih memori lainnya, dan alasannya pun sama: satu portal
// mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali. Kalau
// dibuat ulang setiap permintaan, perubahan yang baru disimpan akan hilang pada permintaan
// berikutnya dan layarnya tampak rusak tanpa sebab.
func claimStatusSelectorMemory(primaryAlias string) masterstatus.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterstatus.Repo{}

	return func(alias string) (masterstatus.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterstatusmemory.NewRepo(masterstatusmemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// buildXOLNotifier memilih pengisi seam masterxol.Notifier.
//
// Pengirim SMTP dipasang hanya bila konfigurasinya LENGKAP — termasuk daftar penerimanya.
// Bila belum, yang dipasang adalah tiruan yang mencatat, dan penyimpanan Master XOL tetap
// berhasil: datanya sudah tersimpan dan pengajuannya sudah tercatat sebelum pemberitahuan
// dikirim, sehingga menggagalkan permintaan pada titik itu hanya akan membuat pengguna
// menyimpan ulang — dan mengajukan dua kali.
//
// Penerimanya datang dari XOL_PENERIMA_KOMITE, bukan dari kode. Sistem lama menuliskan
// dua alamat perorangan langsung di dalam activity-nya dan menimpa hasil pencariannya
// dengan keduanya; `D-15` melarang pola itu dibawa dan `D-67` melarang akun pribadi.
//
// Tempat yang benar bagi daftar ini kelak adalah master Penerima Notifikasi (`F-4`), yang
// belum dibangun.
func buildXOLNotifier(cfg config.Config, logger *slog.Logger) masterxol.Notifier {
	smtpConfig := masterxolnotif.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		User:     cfg.SMTP.User,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
		To:       cfg.SMTP.XOLCommitteeRecipients,
		Timeout:  cfg.SMTP.Timeout,
	}
	if smtpConfig.Complete() {
		return masterxolnotif.NewSender(smtpConfig)
	}

	logger.Warn("pemberitahuan komite Master XOL tidak dikirim: SMTP atau penerimanya belum lengkap",
		slog.String("modul", "masterxol"),
		slog.String("perbaikan", "isi SMTP_HOST, SMTP_PORT, SMTP_DARI, dan XOL_PENERIMA_KOMITE"))
	return masterxolnotif.NewFake()
}

// xolSelectorMemory menyusun penyimpanan Master XOL di memori.
//
// Bentuknya sama dengan pemilih memori lainnya, dan alasannya pun sama: satu portal
// mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali. Kalau
// dibuat ulang setiap permintaan, induk yang baru disimpan akan hilang pada permintaan
// berikutnya dan layarnya tampak rusak tanpa sebab.
func xolSelectorMemory(primaryAlias string) masterxol.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterxol.Repo{}

	return func(alias string) (masterxol.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterxolmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// dominantFactorSelectorMemory menyusun penyimpanan master faktor dominan di memori.
//
// Bentuknya sama dengan pemilih memori lainnya, dan alasannya pun sama: satu portal
// mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali. Kalau
// dibuat ulang setiap permintaan, perubahan yang baru disimpan akan hilang pada permintaan
// berikutnya dan layarnya tampak rusak tanpa sebab.
func dominantFactorSelectorMemory(primaryAlias string) masterdominanfactor.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterdominanfactor.Repo{}

	return func(alias string) (masterdominanfactor.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterdominanfactormemory.NewRepo(masterdominanfactormemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// causeOfLossSelectorMemory menyusun penyimpanan master penyebab kerugian di memori.
//
// Bentuknya sama dengan pemilih memori lainnya, dan alasannya pun sama: satu portal
// mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali. Kalau
// dibuat ulang setiap permintaan, perubahan yang baru disimpan akan hilang pada permintaan
// berikutnya dan layarnya tampak rusak tanpa sebab.
func causeOfLossSelectorMemory(primaryAlias string) masterpenyebabkerugian.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterpenyebabkerugian.Repo{}

	return func(alias string) (masterpenyebabkerugian.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterpenyebabkerugianmemory.NewRepo(masterpenyebabkerugianmemory.SampleList()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// maskingSelectorMemory memilih penyimpanan Master Masking di memori.
//
// Hanya portal utama yang dilayani. Portal lain menghasilkan ErrNotReady, bukan diam-diam
// memakai penyimpanan yang sama — tanpa basis data, memakai satu penyimpanan untuk semua
// entitas akan membuat perpindahan portal tampak berhasil padahal datanya itu-itu juga,
// dan justru menyembunyikan kelas kesalahan yang `R-20` peringatkan.
func maskingSelectorMemory(primaryAlias string) mastermasking.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastermasking.Repo{}

	return func(alias string) (mastermasking.Repo, error) {
		clean := strings.ToUpper(strings.TrimSpace(alias))
		if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
			return nil, fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		// Cabang tambahan ikut dimuat supaya menambah data untuk cabang yang BELUM punya
		// baris dapat dicoba. Tanpa itu, satu-satunya cabang yang dapat dipilih adalah yang
		// sudah terpakai — dan setiap penambahan akan ditolak sebagai pasangan ganda.
		fresh := mastermaskingmemory.NewRepo(mastermaskingmemory.SampleList()...).
			WithBranches(mastermaskingmemory.SampleBranches()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// buildIdentity menyusun rantai sumber identitas.
//
// Urutannya adalah aturan bisnis yang ditetapkan Work Owner 2026-09-16: HCC/HCQ lebih
// dulu untuk karyawan, lalu POOLDATA.M_LOGIN_PNC untuk non-karyawan.
func buildIdentity(cfg config.Config, production bool, legacy *sqlstore.Legacy) (auth.Identity, error) {
	switch cfg.IdentityAdapter {
	case config.IdentityAdapterFake:
		// Penolakan terhadap produksi ada di dalam provider, bukan hanya di sini —
		// satu nilai konfigurasi tidak boleh cukup untuk menyalakannya di produksi.
		return provider.NewFake(production, nil)

	case config.IdentityAdapterHCQ:
		if legacy == nil {
			return nil, errors.New("IDENTITAS_ADAPTER=hcq menuntut koneksi basis data: alamat layanan HCQ dan daftar login non-karyawan keduanya dibaca dari sana")
		}
		hcq, err := provider.NewHCQ(provider.HCQOptions{
			Katalog:     legacy,
			PortalAlias: cfg.PrimaryPortal,
			User:        cfg.HCQ.User,
			Password:    cfg.HCQ.Password,
			Timeout:     cfg.HCQ.Timeout,
		})
		if err != nil {
			return nil, err
		}
		local, err := provider.NewLocal(legacy)
		if err != nil {
			return nil, err
		}
		return provider.NewChain(
			provider.Link{Name: "hcq", Sumber: hcq},
			provider.Link{Name: "lokal", Sumber: local},
		)

	default:
		return nil, fmt.Errorf("adapter identitas %q tidak dikenal", cfg.IdentityAdapter)
	}
}

// portalParameters mengubah konfigurasi portal menjadi parameter koneksi, melewati
// portal yang variabel wajibnya belum terisi.
func portalParameters(cfg config.Config) []db.Parameter {
	return databaseParameters(cfg.Portal)
}

// anekaParameters menyusun parameter koneksi KEDUA tiap portal.
//
// Kumpulan ini boleh kosong, dan itu keadaan yang sah — lihat config.anekaPrefix dan
// db.NewOptionalPool. Yang belum lengkap dilewati dengan diam, sama seperti portal yang
// belum lengkap: pengisiannya berjalan bertahap.
func anekaParameters(cfg config.Config) []db.Parameter {
	return databaseParameters(cfg.Aneka)
}

// databaseParameters mengubah peta konfigurasi menjadi parameter koneksi, terurut.
//
// Terurut supaya urutan pembukaan koneksi — dan karena itu urutan barisnya di log —
// tidak berubah-ubah antar start. Peta Go tidak menjamin urutan, dan log yang barisnya
// berpindah-pindah membuat perbandingan dua start menjadi pekerjaan tersendiri.
func databaseParameters(source map[string]config.Database) []db.Parameter {
	alias := make([]string, 0, len(source))
	for a := range source {
		alias = append(alias, a)
	}
	sort.Strings(alias)

	parameter := make([]db.Parameter, 0, len(alias))
	for _, a := range alias {
		b := source[a]
		if !b.Complete() {
			continue
		}
		parameter = append(parameter, db.Parameter{
			Alias:              b.Alias,
			Host:               b.Host,
			Port:               b.Port,
			Service:            b.Service,
			User:               b.User,
			Password:           b.Password,
			MaxConnections:     b.MaxConnections,
			MaxIdle:            b.MaxIdle,
			ConnectionLifetime: b.ConnectionLifetime,
		})
	}
	return parameter
}

// inboxXOLSelectorMemory menyusun penyimpanan Inbox XOL di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali — alasannya sama dengan selector memori lain di berkas ini.
//
// Isinya contoh yang mencakup SELURUH jalur layar, termasuk dua yang paling mudah
// terlewat: perjanjian tanpa group business, dan baris treaty inward yang kursnya tidak
// ditemukan. Seluruhnya karangan — lihat inboxxol/repo/memory/sample.go.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti
// di produksi.
func inboxXOLSelectorMemory(primaryAlias string) inboxxol.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxxol.Repo{}

	return func(alias string) (inboxxol.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxxolmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// claimTreatyPropSelectorMemory menyusun penyimpanan Inbox Claim Treaty Prop di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali — alasannya sama dengan selector memori lain di berkas ini.
//
// Isinya contoh yang mencakup ketiga penyaring sekaligus: penugasan milik dua petugas
// berbeda, antrean teknik, satu baris tanpa penanda `CLMP`, dan satu baris di antrean
// lain. Seluruhnya karangan — lihat inboxclaimtreatyprop/repo/memory/sample.go.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
func claimTreatyPropSelectorMemory(primaryAlias string) inboxclaimtreatyprop.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxclaimtreatyprop.Repo{}

	return func(alias string) (inboxclaimtreatyprop.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxclaimtreatypropmemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// claimTreatyNonPropSelectorMemory menyusun penyimpanan Inbox Claim Treaty Non Prop di
// memori; alasannya sama dengan claimTreatyPropSelectorMemory di atas.
//
// Isi contohnya mencakup KEEMPAT penyaring layar ini sekaligus: penugasan milik dua petugas
// berbeda, antrean teknik, satu baris yang nomor polisnya belum terbit, satu baris
// ber-awalan `CLMP-` milik layar saudaranya, dan satu baris ber-awalan `KMTNP-`. Dua yang
// terakhir yang membuktikan penyaring awalan tidak mencampur kedua layar treaty — lihat
// inboxclaimtreatynonprop/repo/memory/sample.go.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
func claimTreatyNonPropSelectorMemory(
	primaryAlias string,
) inboxclaimtreatynonprop.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxclaimtreatynonprop.Repo{}

	return func(alias string) (inboxclaimtreatynonprop.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxclaimtreatynonpropmemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// managerReceivePUCLSelectorMemory menyusun penyimpanan Inbox Manager Receive / PUCL di
// memori; alasannya sama dengan claimTreatyPropSelectorMemory di atas.
//
// Isi contohnya mencakup KEEMPAT penyaring layar ini sekaligus, dan lima dari sepuluh
// barisnya sengaja TERTOLAK: berkas tanpa Group Panel, klaim yang berada di tabel penugasan
// per orang, klaim yang sudah selesai, dan klaim di antrean bersama lain. Baris yang lolos
// saja tidak membuktikan apa pun — yang membuktikan penyaringnya bekerja adalah baris yang
// seharusnya tidak muncul dan memang tidak muncul. Lihat
// inboxmanagerreceivepucl/repo/memory/sample.go.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
func managerReceivePUCLSelectorMemory(
	primaryAlias string,
) inboxmanagerreceivepucl.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxmanagerreceivepucl.Repo{}

	return func(alias string) (inboxmanagerreceivepucl.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxmanagerreceivepuclmemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// rclPUCLSelectorMemory menyusun penyimpanan Inbox RCL/PUCL di memori; alasannya sama
// dengan claimTreatyPropSelectorMemory di atas.
//
// Isi contohnya mencakup KEENAM penyaring layar ini, dan enam dari sebelas barisnya sengaja
// TERTOLAK: penanda kasus yang berbeda, klaim yang sudah disetujui, klaim yang penanda
// persetujuannya KOSONG, klaim yang sudah selesai, klaim di antrean bersama lain, dan klaim
// Personal Accident di luar antrean. Baris yang lolos saja tidak membuktikan apa pun — yang
// membuktikan penyaringnya bekerja adalah baris yang seharusnya tidak muncul dan memang
// tidak muncul.
//
// Baris terakhir punya tugas tambahan: ia TIDAK muncul di tab mana pun tetapi IKUT di
// laporan harian, dan itulah satu-satunya hal yang membuktikan laporan dan tabel memang
// berbeda isinya. Lihat inboxrclpucl/repo/memory/sample.go.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
func rclPUCLSelectorMemory(primaryAlias string) inboxrclpucl.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxrclpucl.Repo{}

	return func(alias string) (inboxrclpucl.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxrclpuclmemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// reportKPISelectorMemory menyusun penyimpanan Report KPI PNC di memori; alasannya sama
// dengan claimTreatyPropSelectorMemory di atas.
//
// Isi contohnya dipilih supaya EMPAT keadaan yang paling mudah salah dapat dilihat langsung
// di layar pengembangan, bukan hanya di uji: satu adjuster yang punya kedua tipe sekaligus,
// komponen yang belum dinilai, satu nilai yang bukan angka, dan dua baris tepat di tepi
// rentang periode. Ditambah dua baris yang sengaja berada DI LUAR periode contoh — tanpa
// baris yang tertolak, layar tidak dapat menunjukkan bahwa penyaringnya bekerja.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
func reportKPISelectorMemory(primaryAlias string) reportkpi.RepoSelector {
	var lock sync.Mutex
	store := map[string]reportkpi.Repo{}

	return func(alias string) (reportkpi.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := reportkpimemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// salvageSelectorMemory menyusun penyimpanan Inbox Salvage di memori; alasannya sama
// dengan rclPUCLSelectorMemory di atas.
//
// Isi contohnya disusun supaya KETIGA BELAS daftarnya punya baris, dan beberapa baris punya
// tugas khusus yang tidak terlihat dari jumlahnya:
//
//   - satu klaim yang sudah SELESAI, untuk membuktikan hanya daftar Salvage Outstanding
//     yang menyaring status kerja;
//   - satu pengajuan ber-`NILAIAKSEP` NOL, untuk membuktikan "Status Lelang" membaca
//     NILAINYA — bukan sekadar keberadaannya;
//   - dua pengajuan berstatus 7 milik PIC yang BERBEDA, sehingga penyaring "hanya milik
//     saya" pada daftar Request Balai Lelang terlihat gagal sebagai baris tambahan alih-alih
//     sebagai daftar kosong;
//   - satu pengajuan berstatus 6, yang tidak punya daftar sendiri tetapi IKUT terhitung
//     pencacah "Histori Salvage" — itulah yang membuat selisih antara pencacah dan daftarnya
//     terlihat.
//
// Berbeda dari modul inbox lain, penyimpanan ini MENERIMA TULISAN. Satu Store dipegang
// seluruh permintaan satu portal, sehingga pengajuan yang disimpan lewat tombol Submit
// langsung muncul di daftar Checker pada permintaan berikutnya — persis seperti di Oracle.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
func salvageSelectorMemory(primaryAlias string) inboxsalvage.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxsalvage.Repo{}

	return func(alias string) (inboxsalvage.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxsalvagememory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// komunikasiCabangSelectorMemory menyusun penyimpanan percakapan cabang di memori.
//
// Sepuluh baris contohnya memegang satu janji: SETIAP penyaring punya baris yang cocok
// MAUPUN yang tidak. Yang membuktikan penyaringnya bekerja bukan baris yang muncul,
// melainkan baris yang seharusnya tidak muncul dan memang tidak muncul — di sini ada empat,
// masing-masing untuk kanal percakapan, pengirim kosong, pesan kosong, dan batas cabang.
//
// Satu baris punya tugas tambahan: KOM-0006 dibalas TANPA penjawab tercatat, sehingga ia
// muncul di tabel tetapi tidak terhitung di pencacah mana pun. Itulah satu-satunya hal yang
// membuktikan selisih satu kolom antara grid dan pencacah benar-benar ditiru.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
func komunikasiCabangSelectorMemory(primaryAlias string) inboxkomunikasicabang.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxkomunikasicabang.Repo{}

	return func(alias string) (inboxkomunikasicabang.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxkomunikasicabangmemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// inboxProgressClaimSelectorMemory menyusun penyimpanan progres klaim di memori;
// alasannya sama dengan claimTreatyPropSelectorMemory di atas.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti di
// produksi: perilaku penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
func inboxProgressClaimSelectorMemory(primaryAlias string) inboxprogressclaim.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxprogressclaim.Repo{}

	return func(alias string) (inboxprogressclaim.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxprogressclaimmemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// inboxAnalystDoctorSelectorMemory menyusun penyimpanan antrean penilaian medis di memori;
// alasannya sama dengan inboxProgressClaimSelectorMemory di atas.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti di
// produksi: perilaku penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
//
// Satu salinan per portal, bukan satu yang dibagi. Modul ini memang tidak menulis, sehingga
// hari ini tidak ada yang dapat saling menimpa — tetapi berbagi penyimpanan antarportal
// adalah bentuk kebocoran yang persis dilarang `R-20`, dan mencegahnya sejak awal jauh lebih
// murah daripada menemukannya kelak.
func inboxAnalystDoctorSelectorMemory(primaryAlias string) inboxanalystdoctor.RepoSelector {
	var lock sync.Mutex
	store := map[string]inboxanalystdoctor.Repo{}

	return func(alias string) (inboxanalystdoctor.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := inboxanalystdoctormemory.NewSampleStore()
		store[clean] = fresh
		return fresh, nil
	}
}

// matchPrimaryPortal menyeragamkan alias dan menolak portal selain portal utama.
//
// Penolakannya memakai portal.ErrNotReady, galat yang sama dengan yang dihasilkan
// produksi saat kredensial sebuah entitas belum diisi — sehingga jalur penolakannya
// berperilaku sama di kedua lingkungan.
func matchPrimaryPortal(alias, primaryAlias string) (string, error) {
	clean := strings.ToUpper(strings.TrimSpace(alias))
	if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
		return "", fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
	}
	return clean, nil
}

// spaVersionText menyebut kapan antarmuka tersemat dibangun, dalam bentuk yang aman
// ditampilkan meski penandanya tidak ada.
//
// Binary yang dikompilasi sebelum penanda ini diperkenalkan tetap dapat berjalan; yang
// hilang hanyalah kemampuan menjawab "antarmuka versi mana yang sedang disajikan".
func spaVersionText() string {
	if v := spa.Version(); v != "" {
		return v
	}
	return "tidak diketahui (dibangun sebelum penanda versi ada)"
}

// masterAutoClaimSelectorMemory menyusun penyimpanan Master Auto Claim di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali. Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa
// Oracle. Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang
// sama seperti di produksi.
func masterAutoClaimSelectorMemory(primaryAlias string) masterautoclaim.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterautoclaim.Store{}

	return func(alias string) (masterautoclaim.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterautoclaimmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// workshopSelectorMemory menyusun penyimpanan Master Bengkel di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali. Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab — dan pada
// modul ini akibatnya lebih jauh: nomor urut ID_BENGKEL ikut mundur, sehingga dua
// bengkel dapat lahir dengan kunci yang sama.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa
// Oracle. Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama
// seperti di produksi.
//
// Penyimpanan contoh memuat ketiga tabel acuannya sekaligus — cabang, kota, dan bank —
// sehingga seluruh alur layar dapat dicoba tanpa Oracle: menambah, menolak nama yang
// sudah ada, menyunting, lalu menyetujui borongan lewat tab Waiting Approval.
func workshopSelectorMemory(primaryAlias string) masterbengkel.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterbengkel.Store{}

	return func(alias string) (masterbengkel.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterbengkelmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// panelSelectorMemory menyusun penyimpanan Master Panel di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali. Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang
// pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab — dan pada modul ini
// akibatnya lebih jauh: nomor urut ID_PANEL ikut mundur, sehingga dua panel dapat lahir
// dengan kunci yang sama.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti
// di produksi.
//
// Penyimpanan contoh memuat panel dengan DUA, SATU, dan NOL lokasi, sehingga seluruh alur
// layar dapat dicoba tanpa Oracle — termasuk panel tanpa lokasi sama sekali, keadaan sah
// yang paling mudah terlupa diuji.
func panelSelectorMemory(primaryAlias string) masterpanel.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterpanel.Store{}

	return func(alias string) (masterpanel.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterpanelmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// sparepartSelectorMemory menyusun penyimpanan Master Sparepart di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai kembali.
// Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan hilang pada
// permintaan berikutnya dan layarnya tampak rusak tanpa sebab — dan pada modul ini
// akibatnya lebih jauh: nomor urut ID ikut mundur, sehingga dua sparepart dapat lahir
// dengan kunci yang sama.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti di
// produksi.
//
// Penyimpanan contoh memuat ketiga status persetujuan sekaligus beserta kedua daftar
// acuannya, sehingga seluruh alur layar dapat dicoba tanpa Oracle — termasuk baris yang
// kolom pilihannya kosong dan baris yang belum pernah distempel tanggal harga, dua keadaan
// sah yang paling mudah terlupa diuji.
func sparepartSelectorMemory(primaryAlias string) mastersparepart.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastersparepart.Store{}

	return func(alias string) (mastersparepart.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := mastersparepartmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// clauseSelectorMemory menyusun penyimpanan Master Pasal Kerugian di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali. Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab — dan pada
// modul ini akibatnya lebih jauh: nomor urut IDDATA ikut mundur, sehingga dua pasal dapat
// lahir dengan kunci yang sama.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa
// Oracle. Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama
// seperti di produksi.
//
// Penyimpanan contoh memuat master lini bisnisnya sekaligus, sehingga seluruh alur layar
// dapat dicoba tanpa Oracle: menambah, memilih lini bisnis dari daftar, menyunting, lalu
// menghapus.
func clauseSelectorMemory(primaryAlias string) masterpasal.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterpasal.Store{}

	return func(alias string) (masterpasal.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterpasalmemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// supplierSelectorMemory menyusun penyimpanan Master Supplier di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali. Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab.
//
// Di modul ini pemakaian ulang itu punya akibat kedua yang tidak dimiliki master lain:
// antrean permintaan persetujuan ikut tersimpan di instans yang sama, sehingga alur
// "tambah supplier lalu periksa antreannya" dapat dicoba tanpa Oracle sama sekali.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti
// di produksi: perilaku penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
func supplierSelectorMemory(primaryAlias string) mastersupplier.RepoSelector {
	var lock sync.Mutex
	store := map[string]mastersupplier.Store{}

	return func(alias string) (mastersupplier.Store, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := mastersuppliermemory.NewSampleRepo()
		store[clean] = fresh
		return fresh, nil
	}
}

// rejectionSelectorMemory menyusun penyimpanan Master Penolakan Klaim di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali. Kalau dibuat ulang setiap permintaan, penambahan yang baru disimpan akan
// hilang pada permintaan berikutnya dan layarnya tampak rusak tanpa sebab.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti
// di produksi: perilaku penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
func rejectionSelectorMemory(primaryAlias string) masterpenolakan.RepoSelector {
	var lock sync.Mutex
	store := map[string]masterpenolakan.Repo{}

	return func(alias string) (masterpenolakan.Repo, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterpenolakanmemory.NewRepo(
			masterpenolakanmemory.SampleParents(),
			masterpenolakanmemory.SampleList()...,
		)
		store[clean] = fresh
		return fresh, nil
	}
}

// rejectionKomiteSelectorMemory menyusun penyimpanan Master Penolakan Komite di memori.
//
// Ia TIDAK menumpang pada rejectionSelectorMemory seperti halnya tingkat 2 master status
// progres menumpang pada tingkat 1: kedua tabel itu memang tidak berhubungan, dan
// menautkannya di sini akan menyiratkan hubungan yang tidak ada.
func rejectionKomiteSelectorMemory(primaryAlias string) masterpenolakan.RepoSelectorKomite {
	var lock sync.Mutex
	store := map[string]masterpenolakan.RepoKomite{}

	return func(alias string) (masterpenolakan.RepoKomite, error) {
		clean, err := memoryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterpenolakanmemory.NewRepoKomite(masterpenolakanmemory.SampleListKomite()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// memoryPortal menormalkan alias portal dan menolak yang bukan portal utama.
//
// Satu fungsi untuk kedua pemilih di atas supaya keduanya menolak dengan galat yang sama
// persis. Dua pemeriksaan terpisah atas hal yang sama akan berbeda begitu salah satunya
// disunting — dan yang dipertaruhkan pada R-20 adalah pemisahan data antar badan hukum.
func memoryPortal(alias, primaryAlias string) (string, error) {
	clean := strings.ToUpper(strings.TrimSpace(alias))
	if clean != strings.ToUpper(strings.TrimSpace(primaryAlias)) {
		return "", fmt.Errorf("%w: portal %q tidak tersedia tanpa basis data", portal.ErrNotReady, alias)
	}
	return clean, nil
}

// claimHistorySelectorMemory menyusun penyimpanan riwayat klaim di memori.
//
// Satu portal mendapat satu penyimpanan, dibuat saat pertama diminta lalu dipakai
// kembali — alasannya sama dengan progressStatusSelectorMemory.
//
// Hanya portal utama yang dilayani, sejalan dengan readyAliases pada cabang tanpa Oracle.
// Memilih portal lain tanpa basis data karena itu ditolak dengan galat yang sama seperti
// di produksi: perilaku penolakannya ikut teruji saat pengembangan, bukan hanya nanti.
func claimHistorySelectorMemory(primaryAlias string) riwayatklaim.RepoSelector {
	var lock sync.Mutex
	store := map[string]riwayatklaim.Repo{}

	return func(alias string) (riwayatklaim.Repo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := riwayatklaimmemory.NewRepo(riwayatklaimmemory.SampleClaims()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// claimProtectionSelectorMemory menyusun gerbang proteksi data di memori.
//
// Ia dipakai kembali antar permintaan, dan itu MENENTUKAN di sini: jatah pencarian
// dihitung dari pemakaian yang tercatat, dan penyimpanan yang dibuat ulang setiap
// permintaan akan mengembalikan jatah penuh setiap kali — sehingga jalur "jatah habis"
// tidak akan pernah dapat dicoba tanpa Oracle.
//
// Baris proteksi contohnya sengaja berbeda keadaan supaya keempat jalur gerbang dapat
// dicoba: lolos, hampir habis, sudah habis, dan belum terdaftar. Lihat
// riwayatklaim/repo/memory/sample.go.
func claimProtectionSelectorMemory(primaryAlias string) riwayatklaim.ProtectionRepoSelector {
	var lock sync.Mutex
	store := map[string]riwayatklaim.ProtectionRepo{}

	return func(alias string) (riwayatklaim.ProtectionRepo, error) {
		clean, err := matchPrimaryPortal(alias, primaryAlias)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := riwayatklaimmemory.NewProtectionRepo(riwayatklaimmemory.SampleProtections()...)
		store[clean] = fresh
		return fresh, nil
	}
}

// progressStatus2SelectorMemory menyusun penyimpanan tingkat 2 di memori.
//
// Ia TIDAK memeriksa alias portalnya sendiri, melainkan menanyakannya ke pemilih tingkat
// 1: portal yang ditolak di sana ditolak di sini dengan galat yang sama persis. Dua
// pemeriksaan terpisah atas hal yang sama akan berbeda begitu salah satunya disunting —
// dan yang dipertaruhkan pada R-20 adalah pemisahan data antar badan hukum.
//
// Repo tingkat 1 yang dikembalikan pemilih itu DIPAKAI LANGSUNG sebagai induk, bukan
// disalin. Dengan begitu status progres 1 yang baru ditambahkan lewat layarnya langsung
// muncul di dropdown tingkat 2 — perilaku yang sama dengan adapter SQL, yang membaca
// tabel induk di dalam transaksi yang sama.
func progressStatus2SelectorMemory(parentSelector masterstatusprogres.RepoSelector) masterstatusprogres.RepoSelector2 {
	var lock sync.Mutex
	store := map[string]masterstatusprogres.Repo2{}

	return func(alias string) (masterstatusprogres.Repo2, error) {
		parent, err := parentSelector(alias)
		if err != nil {
			return nil, err
		}

		memoryParent, usable := parent.(*masterstatusprogresmemory.Repo)
		if !usable {
			// Tidak mungkin terjadi pada rakitan yang ada; dinyatakan supaya cacat
			// perakitan gagal keras, bukan diam-diam menyajikan induk yang kosong.
			return nil, fmt.Errorf("perakitan: repo tingkat 1 portal %q bukan adapter memori", alias)
		}

		clean := strings.ToUpper(strings.TrimSpace(alias))

		lock.Lock()
		defer lock.Unlock()
		if existing, already := store[clean]; already {
			return existing, nil
		}
		fresh := masterstatusprogresmemory.NewRepo2(memoryParent, masterstatusprogresmemory.SampleList2()...)
		store[clean] = fresh
		return fresh, nil
	}
}
