CREATE OR REPLACE PROCEDURE          PEGA_CONVERT_JSONKLAIM_PNC(TempCLAIMID IN VARCHAR2 ,ErrMsg OUT VARCHAR2)
AS
id_count INTEGER;
id_count2 INTEGER;
id_count3 INTEGER;
id_count4 INTEGER;
id_count5 INTEGER;
id_count6 INTEGER;
id_count7 INTEGER;
id_count8 INTEGER;
id_countnull INTEGER;
datajson CLOB;
datajson_JSONKLAIM      JSON_OBJECT_T;
datajson_ObjectList    JSON_OBJECT_T;
datajson_ReceiverList    JSON_OBJECT_T;
datajson_ObjectCovList    JSON_OBJECT_T;
datajson_ObjectItemList    JSON_OBJECT_T;
datajson_EstimationList    JSON_OBJECT_T;
datajson_AdjustmentList    JSON_OBJECT_T;
datajson_SpredingsList    JSON_OBJECT_T;
datajson_PanelList    JSON_OBJECT_T;
datajson_PenawaranList  JSON_OBJECT_T;
datajson_TjhList  JSON_OBJECT_T;
datajson_Sparepart  JSON_OBJECT_T;
datajson_TjhVehicle  JSON_OBJECT_T;
datajson_PLAList    JSON_OBJECT_T;
datajson_DLAList    JSON_OBJECT_T;
datajson_CoinsList       JSON_OBJECT_T;
PolicyData          JSON_OBJECT_T;
Quotation          JSON_OBJECT_T;
OfferFacIn      JSON_OBJECT_T;
SurveyData          JSON_OBJECT_T;
PUCLData            JSON_OBJECT_T;
datajson_komitelist  JSON_OBJECT_T;
komitedatalist           JSON_ARRAY_T;
objectdataLIST          JSON_ARRAY_T;
ReceiverDataLIST          JSON_ARRAY_T;
ObjectCovDataLIST          JSON_ARRAY_T;
ObjectItemLIST          JSON_ARRAY_T;
EstimationLIST          JSON_ARRAY_T;
AdjustmentDataLIST         JSON_ARRAY_T;
SpreadingsLIST         JSON_ARRAY_T;
DLADataLIST         JSON_ARRAY_T;
PLADataLIST         JSON_ARRAY_T;
CoinsDataLIST   JSON_ARRAY_T;
PanelListLIST          JSON_ARRAY_T;
ListPenawaranLIST       JSON_ARRAY_T;
TjhLIST       JSON_ARRAY_T;
dol     varchar2(50);
TGLAKSEP     varchar2(50);
jm       varchar2(5);
jm2       varchar2(5);
tgl       varchar2(5);
tQQNAME        varchar2(2000);
tSOBNAME       varchar2(200);
tSOBNAMEID       varchar2(50);
tBRANCHCODE       varchar2(100);
tBRANCHNAME       varchar2(100);
tBUSINESSCODE       varchar2(50);
tBUSINESSNAME       varchar2(50);
tSTATUSCLAIM       varchar2(100);
tSTATUSWORK       varchar2(100);
tPRODKE       varchar2(100);
tNOPOLIS       varchar2(20);
tCLAIMNO       varchar2(30);
tTKI    varchar2(5);
tOBJECTITEMID       varchar2(30);
tESTIMATIONVALUE       number;
tPricetreat         number;
tPPN    number;
tTTLPPN number;
tKURSVALUE       number;
tCONVERTVALUE       number;
tCURRENCYEST       varchar2(10);
tESTIMATIONTYPE       varchar2(5);
tESTIMATIONID       varchar2(10);
tCFSDATE        date;
tDOCLENGKAP     date;
tOBJECTID       varchar2(50);
tOBJECTNAME       varchar2(2300);
tOBJECTCOVERAGEID       varchar2(30);
tSLIKNO         varchar2(50);
tCAUSEOFLOSSID       varchar2(30);
tCAUSEOFLOSS       varchar2(500);
tCOVERAGEID         varchar2(30);
tCOVERAGENAME       varchar2(4000);
tPXCREATEDATETIME   DATE;
tADJUSTMENTID       varchar2(30);
tNOAKSEPTASI       varchar2(100);
tNILAIAKSEPTASI       number;
tSTATUSREJECT         varchar2(100);
tSTATUSAKSEPTASI       varchar2(10);
tALASANKETERLAMBATAN    varchar2(300);
tAlasan varchar2(4000);
tlainya varchar2(2000);
tkeyword varchar2(300);
tSTSSALVAGE varchar2(5);
tSTATUSAKSEPTASILOD       varchar2(10);
tRECEIVER       varchar2(10);
tRECEIVERNAME       varchar2(4000);
tRECEIVERID       varchar2(10);
tTGLAKSEPTASI       varchar2(50);
tNODLA       varchar2(100);
tTGLDLA       varchar2(50);
tNILAIDLA       VARCHAR2(50);
tDLAREINSURER       varchar2(100);
tTIPEDLA       varchar2(50);
tREVISIDLA       varchar2(10);
tNOPLA       varchar2(30);
tPLAREINSURER        varchar2(500);
tTIPEPLA       varchar2(50);
tREVISI        varchar2(10);
tTGLPLA       varchar2(50);
tTYPEOFCOINS       varchar2(5);
tADJUSTMENT       number;
tSALVAGE       VARCHAR2(50);
tADJUSTER       VARCHAR2(50);
tGROSSVALUE       number;
tPAYMENTTYPE       VARCHAR2(50);
tTANGGALBAYAR           DATE;
tSTATUSBAYAR        VARCHAR(100);
tTOTALCLAIM             number;
tNILAISALVAGE_A       number;
tINDIVIDUAL_RISKTYPE     VARCHAR(15);
tINDIVIDUAL_RISKVALUE    number;
tINDIVIDUAL_RISKPERCENT  number;
tPROPOSE_VALUE number;
tLOC                            number;
tASM_SHARE               number;
tASM_SHARE_VALUE   number;
tSHARE number;
tREGISTERDATE       date;
tCLOSECLAIMDATE       varchar2(50);
tCLOSECLAIMNOTE       varchar2(4000);
tREMARKRECOMENDATION       varchar2(4000);
tKRONOLOGI      varchar2(4000);
tGROUPPANEL        varchar2(20);
tCURRENCY        number;
tCURRENCYDOL number;
tRECEIVEDATE date;
tANALYST_TRANSFERDATE date;
tINVESTIGATOR_TF_DATE date;
tSURVEYDATE date;
tTRFPICDATE date;
tCOMPLIANCE_CREATEDATE date;
tPOSTAUDIT_TF_ANALYSTDATE date;
tCPLVALID_DATE date;
tCPLPOSTAUDIT_VALIDDATE date;
tCOMITEE_APPROVEDATE date;
tSENDTOANALYSTDATE date;
tFINISHREGISTERDATE date;
tDATEOFREQUESTDOCUMENT date;
tPICTEKNIK varchar2(100);
tMANUALACCEPTANCEDATECOMITEE date;
tPRINTLOD_DATE date;
tRECEIVELOD_DATE date;
tANALYSTTORCLPUCL_DATE date;
tRCLPUCL_TF_TOANALYST date;
tRCLPUCL varchar(20);
tACCEPTANCE_DATECOMITEE date;
tANALYST_TFKOMITEDATE date;
tCOINSNAME varchar2(150);
tLEADER varchar2(20);
tPANELID      varchar2(30);
tPANELIDX   varchar2(5);
tExclusion  varchar2(5);
tIsLiablePanel  varchar2(5);
tPanelFrom  varchar2(20);
tPanelName   varchar2(100);
tPanelQty   varchar2(5);
tPanelSide  varchar2(5);
tPanelSupplier  varchar2(5);
tPanelType  varchar2(5);
tPanelVehicle   varchar2(5);
tDiscPanel varchar(20);
tCaseBengkel    varchar2(15);
tWorkShopID     varchar2(30);
tWorkShopName varchar2(100);
tCaseProcurement     varchar2(15);
tpodate     date;
tnopo       varchar2(30);
tSparepartNo        varchar2(30);
tSparepartName      varchar2(100);
tSparepartSupplierID        varchar2(30);
tSparepartSupplierName      varchar2(100);
tID_Penawaran       varchar2(30);
tTTL_Harga_Jasa     number;
tTTL_Harga_Part     number;
tTTL_Discount       number;
tOngkos_Kirim       number;
tTTL_Nett           number;
tEst_Perbaikan      number;
tNama_Bengkel       varchar2(100);
tWilayah_Bengkel    varchar2(100);
tTANGGAL_PENAWARAN DATE;
tTJHID       varchar2(10);
tJenisTJH      varchar2(10);
tEstTJHClaim     number;
tBrand        varchar2(20);
tBrandName       varchar2(150);
tModel           varchar2(50);
tModelName     varchar2(150);
tType       varchar2(50);
tTypeName     varchar2(3000);
tserialno    varchar2(50);
tManufactureYear   varchar2(100);
tEngineNumber    varchar2(50);
tChassisNumber     varchar2(70);
tNOWO   varchar2(30);
tWODate     date;
tTreatmentType      varchar2(10);
tStatusJasa     varchar2(10);
tTreatmentAproval   varchar2(100);
tNoAksep    varchar2(100);
tIsGanti    varchar2(10);
tIsJasa     varchar2(10);
tdol  date;
tTANGGALAPPROVEKOMITELIABILITY date;
tTANGGALAPPROVEKOMITEFINAL date;
tBLASTEMAILDATE date;
tJANJI_KIRIM number;
tTRANSFERCASHIERDATE date;
tCLAIMPAIDSTATUS varchar2(2);
tTGLRCV date;
tTGLTERLAMBAT date;
tTGLTRFINVESTIGATOR date;
tLOSS_ADJUSTER_FEE       number;
tLEADERMEMBER varchar(50);
tASM varchar(100);
tTOLAK varchar(4000);
tEXGRATIA varchar(5);
tCircumCauseOfLoss varchar2(4000);
tNotes varchar2(10000);
tCurr varchar2(10);
tLocation varchar2(1000);
tLocationID varchar2(100);
tReportAddress varchar2(1000);
tReportDate date;
tReporterName varchar2(1000);
tReportType varchar2(2);
tNameOfBank varchar2(100);
tNoAccount varchar2(100);
tNOREFBROKER  VARCHAR2(50);
tSTS_BANDING  VARCHAR2(50);
tSTS_KEPUASAN  VARCHAR2(50);
tEFFORT_CLOSE  VARCHAR2(100);
tKENDALA_CLOSE  VARCHAR2(100);
tSTS_PAPERLESS  VARCHAR2(50);
tUSULAN  VARCHAR2(100);
tkodecabang varchar2(100);
tadminklaim varchar2(1000);
tpolisleader varchar2(1000);
tkodercv varchar2(100);
tCaseIDCashier VARCHAR2(100);
tispendingclose VARCHAR2(100);
tSUMTSI NUMBER;
tCURICUMOFLOSS VARCHAR2(1000);
tEXTENTOFLOSS VARCHAR2(10000);
tInpatientDay NUMBER;
tInpatientDayMax NUMBER;
tRemarks VARCHAR2(1000);
tCurrencyPolicy VARCHAR2(1000);
ttanggalkomites varchar2(1000); 
tCoverInsKey varchar2(1000);
tKomiteAproval varchar2(1000);
tKomiteComment varchar2(10000);
tKomiteID varchar2(100);
counts_komites number;
tlistkomite number;
tnoklaimnum varchar2(100);
tSTATUSCASE varchar2(100);
tSTPenolakan varchar2(100);
tNDPenolakan varchar2(100);
tTREATYNAME    VARCHAR2(200);
tTREATYTYPE    VARCHAR2(200);
tTREATYYEAR    VARCHAR2(200);
tTSISPREADED    NUMBER;
tTREATYGROUP    VARCHAR2(200);
tSHAREPERCENTAGE    NUMBER;
tCONTRACTNO VARCHAR2(1000);
tKodeKondisi VARCHAR2(100);
tJumlahHariTunggakan NUMBER;
tKodeSebabMacet VARCHAR2(100);
tKodeKolektibilitas VARCHAR2(100);
tSukuBunga NUMBER;
tSumberDana VARCHAR2(100);
tKodeJenisFasilitas VARCHAR2(100);
tNOKTP VARCHAR2(100);


BEGIN
    BEGIN
        SELECT TO_CLOB(DATA_JSONBLOB)
        INTO datajson
        FROM JSON_KLAIM
        WHERE IDPEGA = TempCLAIMID;
    EXCEPTION WHEN NO_DATA_FOUND THEN
        ErrMsg := 'Error select claim data ' || sqlerrm;
        rollback;
        return;
    END;
    
    datajson_JSONKLAIM     :=  JSON_OBJECT_T.parse(datajson);
    PolicyData              :=  datajson_JSONKLAIM.get_object('PolicyData'); 
    tQQNAME                 := '';
    tNOPOLIS                := '';
    tTYPEOFCOINS            := '';
    tSOBNAME                := '';
    tSOBNAMEID                := '';
    tBRANCHCODE             := '';
    tBRANCHNAME             := '';
    tBUSINESSCODE           := '';
    tBUSINESSNAME           := '';
    tGROUPPANEL             := '';
    tCoinsName              := 'ASURANSI SINAR MAS';
    
    IF PolicyData IS NOT NULL THEN  
        Quotation               := PolicyData.get_object('Quotation');
        OfferFacIn              := PolicyData.get_object('OfferFacIn');
        tQQNAME                 := PolicyData.get_string('QQName');
        tNOPOLIS                := PolicyData.get_string('PolicyNo');
        tCurrencyPolicy         := PolicyData.get_string('Currency');
        tTYPEOFCOINS            := PolicyData.get_string('TypeOfCoins');
        tSOBNAME                := Quotation.get_string('SobName');
        tSOBNAMEID              := Quotation.get_string('SourceOfBusiness');
        tBRANCHCODE             := Quotation.get_string('BranchCode');
        tBRANCHNAME             := Quotation.get_string('BranchName');
        tBUSINESSCODE           := Quotation.get_string('BusinessCode');
        tBUSINESSNAME           := Quotation.get_string('BusinessName');
        tGROUPPANEL             := Quotation.get_string('GroupPanel');
      
        CoinsDataLIST := PolicyData.get_array('CoinsList');
        IF CoinsDataLIST IS NOT NULL THEN
            FOR y IN 0 .. CoinsDataLIST.get_size - 1 LOOP
                datajson_CoinsList     :=  JSON_OBJECT_T(CoinsDataLIST.get(y));
                tLeader :=  datajson_CoinsList.get_string('Leader');
                tASM := datajson_CoinsList.get_string('CoinsName');
                IF tLeader = 'true' then
                    tCoinsName := datajson_CoinsList.get_string('CoinsName');
                end if;
                
                IF tASM like '%ASURANSI SINAR MAS%' then
                    tSHARE := datajson_CoinsList.get_string('PercentShare');
                end if;
                
                IF tLeader = 'true' and tASM like '%ASURANSI SINAR MAS%' then
                    tLEADERMEMBER := 'LEADER';
                ELSIF tLeader = 'true' and tASM  NOT like '%ASURANSI SINAR MAS%' then
                    tLEADERMEMBER := 'MEMBER';
                ELSIF tASM is null then
                    tLEADERMEMBER := 'LEADER';
                    tSHARE := 100;
                end if;

            END loop;
        ELSE
            tCoinsName := 'ASURANSI SINAR MAS';
            tLEADERMEMBER := 'LEADER';
            tSHARE := 100;
        END IF;
    END IF;
    
    IF  tLEADERMEMBER IS NULL THEN
        tLEADERMEMBER :='LEADER';
        tCoinsName := 'ASURANSI SINAR MAS';
        tSHARE := 100;
    END IF;  
    
    IF tTYPEOFCOINS = 'F' then
        tLEADERMEMBER := 'FAC IN';
        tCoinsName := PolicyData.get_string('CedingCoName');
        tSHARE := OfferFacIn.get_string('PercentShare');
    END IF;
    
    BEGIN    
        SurveyData              := datajson_JSONKLAIM.get_object('SurveyData'); 
        PUCLData                := datajson_JSONKLAIM.get_object('PUCLStatus');
        tSTATUSWORK             := datajson_JSONKLAIM.get_string('StatusWork');
        tSTATUSCLAIM            := datajson_JSONKLAIM.get_string('StatusClaim');
        tSTATUSREJECT           := datajson_JSONKLAIM.get_string('StatusReject');
        tAlasan                 := datajson_JSONKLAIM.get_string('AlasanKlaim');
        tlainya                 := datajson_JSONKLAIM.get_string('Other');
        tkeyword                := datajson_JSONKLAIM.get_string('Keyword');
        tPRODKE                 := datajson_JSONKLAIM.get_string('ProdKe');
        tCLAIMNO                := datajson_JSONKLAIM.get_string('ClaimNo');
        tALASANKETERLAMBATAN    := datajson_JSONKLAIM.get_string('AlasanTerlambat');
        tREGISTERDATE           := TO_DATE(datajson_JSONKLAIM.get_string('RegisterDate'),'yyyymmdd');
        tCLOSECLAIMDATE         := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('CloseClaimDate'),1,8),'yyyymmdd');
        tTGLRCV                 := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('RCVCreateDateTime'),1,8),'yyyymmdd');
        tTGLTERLAMBAT           := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('TanggalTerlambat'),1,8),'yyyymmdd');
        tDOCLENGKAP             := TO_DATE(datajson_JSONKLAIM.get_string('TanggalDokLengkap'),'yyyymmdd');
        tCLOSECLAIMNOTE         := datajson_JSONKLAIM.get_string('CloseClaimNote');
        tREMARKRECOMENDATION    := datajson_JSONKLAIM.get_string('RemarkRecommendation');
        tKRONOLOGI              := datajson_JSONKLAIM.get_string('ReportDescription');
        tRECEIVEDATE            := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('DateReceived'),1,8),'yyyymmdd');
        tANALYST_TRANSFERDATE   := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('AnalystTransferDate'),1,8),'yyyymmdd');
        tTRFPICDATE             := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('TransferPICDate'),1,8),'yyyymmdd');
        tINVESTIGATOR_TF_DATE   := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('InvestTfDate'),1,8),'yyyymmdd');
        tTGLTRFINVESTIGATOR     := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('TransferToInvestDateTime'),1,8),'yyyymmdd');
        tSTSSALVAGE             := datajson_JSONKLAIM.get_string('StatusSalvage');  
        tTKI                    := datajson_JSONKLAIM.get_string('TKI');  
        tCurr                   := datajson_JSONKLAIM.get_string('Currency');
        tLocation               := datajson_JSONKLAIM.get_string('Location');
        tReportAddress          := datajson_JSONKLAIM.get_string('ReportAddress'); 
        tReportDate             := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('ReportDate'),1,8),'yyyymmdd');
        tReporterName           := datajson_JSONKLAIM.get_string('ReporterName'); 
        tReportType             := datajson_JSONKLAIM.get_string('ReportType');
        tispendingclose         :=  datajson_JSONKLAIM.get_string('IsPendingClosed');
        tkodecabang             :=  datajson_JSONKLAIM.get_string('KodeCabang');
        tadminklaim             :=  datajson_JSONKLAIM.get_string('UserAdmin');
        tpolisleader             :=  datajson_JSONKLAIM.get_string('PolicyLeaderNo');
        tkodercv                :=  datajson_JSONKLAIM.get_string('RCV_ID');
        
        
        IF SurveyData IS NOT NULL THEN
            tSURVEYDATE := TO_DATE(SUBSTR(SurveyData.get_string('SurveyDate'),1,8),'yyyymmdd');
        end if;

        tCOMPLIANCE_CREATEDATE := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('TanggalBuatCompliance'),1,8),'yyyymmdd');
        tPOSTAUDIT_TF_ANALYSTDATE := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('PostAudtiTfAnalyst'),1,8),'yyyymmdd');
        tCPLVALID_DATE := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('CPLValidDate'),1,8),'yyyymmdd');
        tCPLPOSTAUDIT_VALIDDATE := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('CPLPostAuditValidDate'),1,8),'yyyymmdd');
        tCOMITEE_APPROVEDATE := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('KomiteApproveDate'),1,8),'yyyymmdd');

        IF PUCLData IS NOT NULL THEN
            tSENDTOANALYSTDATE := TO_DATE(SUBSTR(PUCLData.get_string('TanggalKirimPUCL'),1,8),'yyyymmdd');
            tANALYSTTORCLPUCL_DATE := TO_DATE(SUBSTR(PUCLData.get_string('TanggalKirimPUCL'),1,8),'yyyymmdd');
            tRCLPUCL_TF_TOANALYST := TO_DATE(SUBSTR(PUCLData.get_string('SendtoAnalystDate'),1,8),'yyyymmdd');
            tRCLPUCL               := PUCLData.get_string('RCL_PUCL');
        end if;
        
        
        tFINISHREGISTERDATE := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('FinishRegisterDate'),1,8),'yyyymmdd');
        tDATEOFREQUESTDOCUMENT := TO_DATE(SUBSTR(datajson_JSONKLAIM.get_string('DateOfRequestDocument'),1,8),'yyyymmdd');
        tPICTEKNIK              := datajson_JSONKLAIM.get_string('UserTeknis');

        tRECEIVERID := 1;
        tADJUSTMENTID := 1;
    
    
        /*jm2 := substr(datajson_JSONKLAIM.get_string('DateOfLoss'),10,2)+7;
        if jm2 >= 24 then
            jm := jm2 - 24;
            tgl := substr(datajson_JSONKLAIM.get_string('DateOfLoss'),7,2)+1 ;
       else
            jm := jm2;
             tgl := substr(datajson_JSONKLAIM.get_string('DateOfLoss'),7,2);
        end if;
        
        
        jm := substr(datajson_JSONKLAIM.get_string('DateOfLoss'),10,2);
        tgl := substr(datajson_JSONKLAIM.get_string('DateOfLoss'),7,2);
        dol := tgl || '/' 
                    || substr(datajson_JSONKLAIM.get_string('DateOfLoss'),5,2) 
                    || '/' 
                    || substr(datajson_JSONKLAIM.get_string('DateOfLoss'),1,4) 
                    || ' ' 
                    || jm || ':' || substr(datajson_JSONKLAIM.get_string('DateOfLoss'),12,2) || ':' || substr(datajson_JSONKLAIM.get_string('DateOfLoss'),14,2) || substr(datajson_JSONKLAIM.get_string('DateOfLoss'),16,4) ;
        */
        
        IF datajson_JSONKLAIM.get_string('DateOfLoss') IS NULL and tSTATUSWORK != 'Resolved-Rejected' THEN
            UPDATE JSON_KLAIM SET IDPROD=1 WHERE IDPEGA = TempCLAIMID;
            COMMIT;
            RETURN;           
        END IF;
        
        IF datajson_JSONKLAIM.get_string('DateOfLoss') IS NOT NULL then
            tdol := POOLDATA.CONVERT_PEGA_DATE(datajson_JSONKLAIM.get_string('DateOfLoss'));
        END if;
        
        if datajson_JSONKLAIM.get_string('DateOfLoss') IS NULL then
            tdol :=null;
        end if;
        
        
        
        
        IF datajson_JSONKLAIM.get_string('TanggalApproveKomiteLiability')IS NOT NULL THEN
        tTANGGALAPPROVEKOMITELIABILITY :=POOLDATA.CONVERT_PEGA_DATE(datajson_JSONKLAIM.get_string('TanggalApproveKomiteLiability'));
        END IF;
        
        IF datajson_JSONKLAIM.get_string('TanggalApproveKomiteFinal')IS NOT NULL THEN
        tTANGGALAPPROVEKOMITEFINAL := POOLDATA.CONVERT_PEGA_DATE(datajson_JSONKLAIM.get_string('TanggalApproveKomiteFinal'));
        END IF;
    EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error Set Data Claim Data : ' ||dol|| sqlerrm ;
        ROLLBACK;
        RETURN ;
    END;  
    
    DELETE FROM T_CLAIM_PNC WHERE CLAIMID = TempCLAIMID;
    DELETE FROM T_CLAIM_OBJECTLIST WHERE CLAIMID = TempCLAIMID;
    DELETE FROM T_CLAIM_OBJECTCOVERAGE WHERE CLAIMID = TempCLAIMID;
    DELETE FROM T_CLAIM_ESTIMASI WHERE CLAIMID = TempCLAIMID;
    DELETE FROM T_CLAIM_ADJUSTMENT WHERE CLAIMID = TempCLAIMID;
    --DELETE FROM T_DLALIST WHERE CLAIMID = TempCLAIMID;
    --DELETE FROM T_PLALIST WHERE CLAIMID = TempCLAIMID;
    DELETE FROM T_CLAIM_TJHLIST WHERE CLAIMID = TempCLAIMID;
    DELETE FROM T_CLAIM_PANEL WHERE CLAIMID = TempCLAIMID;
    DELETE FROM T_CLAIM_PENAWARAN WHERE CLAIMID = TempCLAIMID;
    DELETE FROM T_CLAIM_TREATMENT WHERE CLAIMID = TempCLAIMID;
    DELETE FROM T_CLAIM_RECEIVER WHERE CLAIMID = TempCLAIMID;
    
    
    
    BEGIN
    --errmsg :=TO_TIMESTAMP(TO_TIMESTAMP(dol,'dd/MM/rrrr HH24: MI:SS:FF') + (1/24*7),'dd/MM/rrrr HH24: MI:SS:FF');
    --return;
        
        INSERT INTO T_CLAIM_PNC a (CLAIMID, NOPOLIS, CLAIMNO, QQNAME, SOBNAME,SOBNAMEID, DATEOFLOSS, BRANCHCODE, BRANCHNAME, BUSINESSCODE, BUSINESSNAME, 
                    PRODKE, STATUSCLAIM, STATUSWORK, TYPEOFCOINS, COINSNAME, ALASANKETERLAMBATAN, REGISTERDATE, CLOSECLAIMDATE, CLOSECLAIMNOTE, 
                    GROUPPANEL, RECEIVEDATE, ANALYST_TRANSFERDATE, INVESTIGATOR_TF_DATE, SURVEYDATE, COMPLIANCE_CREATEDATE, POSTAUDIT_TF_ANALYSTDATE, 
                    CPLVALID_DATE, CPLPOSTAUDIT_VALIDDATE, COMITEE_APPROVEDATE, SENDTOANALYSTDATE, FINISHREGISTERDATE, DATEOFREQUESTDOCUMENT, PICTEKNIK, 
                    ANALYSTTORCLPUCL_DATE, RCLPUCL_TF_TOANALYST, RCLPUCL,REMARKRECOMENDATION,KRONOLOGI,TANGGALAPPROVEKOMITELIABILITY,TANGGALAPPROVEKOMITEFINAL,
                    RCVDATE,TGLTERLAMBAT,TRF_TO_INVESTIGATOR,TRANSFERPIC_DATE,STSSALVAGE,TGLDOKLENGKAP,STS_TKI,STATUSREJECT,ALASAN,LAINYA,KEYWORD,LEADER_MEMBER,SHAREASM,
                    CURRENCY, LOCATION, REPORTADDRESS, REPORTDATE, REPORTERNAME, REPORTTYPE,NOREFBROKER, STS_BANDING, STS_KEPUASAN, EFFORT_CLOSE, KENDALA_CLOSE, STS_PAPERLESS, USULAN,ISPENDINGCLOSE,KODE_CABANG,ADMINKLAIM,
                    POLISLEADER,RCVID) 
        VALUES (TempCLAIMID, tNOPOLIS, tCLAIMNO, tQQNAME, tSOBNAME,tSOBNAMEID, tdol, tBRANCHCODE, tBRANCHNAME, 
                tBUSINESSCODE, tBUSINESSNAME, tPRODKE, tSTATUSCLAIM, tSTATUSWORK, tTYPEOFCOINS, tCOINSNAME, tALASANKETERLAMBATAN, tREGISTERDATE, 
                tCLOSECLAIMDATE, tCLOSECLAIMNOTE, tGROUPPANEL, tRECEIVEDATE, tANALYST_TRANSFERDATE, tINVESTIGATOR_TF_DATE, tSURVEYDATE, tCOMPLIANCE_CREATEDATE,
                tPOSTAUDIT_TF_ANALYSTDATE, tCPLVALID_DATE, tCPLPOSTAUDIT_VALIDDATE, tCOMITEE_APPROVEDATE, tSENDTOANALYSTDATE, tFINISHREGISTERDATE,
                tDATEOFREQUESTDOCUMENT, tPICTEKNIK, tANALYSTTORCLPUCL_DATE, tRCLPUCL_TF_TOANALYST, tRCLPUCL, tREMARKRECOMENDATION,tKRONOLOGI,
                tTANGGALAPPROVEKOMITELIABILITY,tTANGGALAPPROVEKOMITEFINAL,tTGLRCV,tTGLTERLAMBAT,tTGLTRFINVESTIGATOR,tTRFPICDATE,tSTSSALVAGE,tDOCLENGKAP,
                tTKI,tSTATUSREJECT,tAlasan,tlainya,tkeyword,tLEADERMEMBER,tSHARE,tCurr, tLocation, tReportAddress, tReportDate, tReporterName, tReportType,
                tNOREFBROKER, tSTS_BANDING, tSTS_KEPUASAN, tEFFORT_CLOSE, tKENDALA_CLOSE, tSTS_PAPERLESS, tUSULAN,tispendingclose,tkodecabang,tadminklaim,tpolisleader,tkodercv);
                
                
    EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'INSERT T_CLAIM_PNC Error : ' ||dol|| sqlerrm ;
        ROLLBACK;
        RETURN ;
    END;   
    
    ReceiverDataLIST := datajson_JSONKLAIM.get_array('ReceiverClaim');
    
    
   
    IF ReceiverDataLIST IS NOT NULL THEN
        FOR p IN 0 .. ReceiverDataLIST.get_size - 1 LOOP
            datajson_ReceiverList     :=  JSON_OBJECT_T(ReceiverDataLIST.get(p));
            tRECEIVERID := datajson_ReceiverList.get_string('IDReceiver');
            tNameOfBank     := datajson_ReceiverList.get_string('NameOfBank');
            tRECEIVERNAME   := datajson_ReceiverList.get_string('Name');
            tNoAccount      := datajson_ReceiverList.get_string('NoAccount');
            IF tRECEIVERID IS NULL THEN
              tRECEIVERID := p + 1;
            END IF;
            
            
             BEGIN
                INSERT INTO T_CLAIM_RECEIVER (CLAIMID, IDRECEIVER, NAME, NAMEOFBANK, NOACCOUNT)
                VALUES (TempCLAIMID, tRECEIVERID, tRECEIVERNAME,tNameOfBank,tNoAccount);
             EXCEPTION
                WHEN OTHERS THEN
                ErrMsg := 'INSERT T_CLAIM_RECEIVER Error : ' || sqlerrm;
                ROLLBACK;
                RETURN;
             END;
                                         
        END LOOP;
    END IF;
    
    
    
    ObjectDataLIST := datajson_JSONKLAIM.get_array('ObjectList');
    IF ObjectDataLIST IS NOT NULL THEN
        FOR j IN 0 .. ObjectDataLIST.get_size - 1 LOOP
            
            BEGIN
                datajson_ObjectList     :=  JSON_OBJECT_T(ObjectDataLIST.get(j));
                tOBJECTID :=  datajson_ObjectList.get_string('ObjectID');
                tOBJECTNAME := substr(datajson_ObjectList.get_string('ObjectName'),1,2000);
                tBrand       := datajson_ObjectList.get_string('ObjectVehicleBrandID');
                tBrandName       := datajson_ObjectList.get_string('ObjectVehicleBrand');
                tModel           := datajson_ObjectList.get_string('ObjectVehicleModelID');
                tModelName      := datajson_ObjectList.get_string('ObjectVehicleModel');
                tType       := datajson_ObjectList.get_string('ObjectVehicleTypeID');
                tTypeName    := datajson_ObjectList.get_string('ObjectVehicleType');
                tserialno    := datajson_ObjectList.get_string('SerialNo');
                tManufactureYear    := datajson_ObjectList.get_string('ObjectVehicleYear');
                tEngineNumber    := datajson_ObjectList.get_string('ObjectVehicleEngineNumber');
                tChassisNumber    := datajson_ObjectList.get_string('ObjectVehicleChasis');
                tSLIKNO      := datajson_ObjectList.get_string('NoSLIK');
                tLocationID  := datajson_ObjectList.get_string('LocationID');
                tCONTRACTNO := datajson_ObjectList.get_string('ContractNo');
                tKodeKondisi := datajson_ObjectList.get_string('KodeKondisi');
                tJumlahHariTunggakan :=  REPLACE(datajson_ObjectList.get_string('JumlahHariTunggakan'),',','.');
                tKodeSebabMacet := datajson_ObjectList.get_string('KodeSebabMacet');
                tKodeKolektibilitas := datajson_ObjectList.get_string('KodeKolektibilitas');
                tSukuBunga := REPLACE(datajson_ObjectList.get_string('SukuBunga'),',','.');
                tSumberDana := datajson_ObjectList.get_string('SumberDana');
                tKodeJenisFasilitas := datajson_ObjectList.get_string('KodeJenisFasilitas');
                tNOKTP := datajson_ObjectList.get_string('NoKTP');
            EXCEPTION
                WHEN OTHERS THEN
                ErrMsg := 'INSERT Declare ObjectList : ' || sqlerrm;
                ROLLBACK;
                RETURN;
            END; 
             
            
            IF tOBJECTID IS NOT NULL THEN
                
                
                BEGIN
                    INSERT INTO T_CLAIM_OBJECTLIST (CLAIMID, OBJECTID, OBJECTNAME, BRANDID, BRANDNAME,
                        MODELID, MODELNAME, TYPEID, TYPENAME, SERIALNO, MANUFACTUREYEAR, ENGINENUMBER, CHASISSNUMBER,SLIKNO,LOCATIONID,INSERTDATE,
                        CONTRACTNO,KodeKondisi,JumlahHariTunggakan,KodeSebabMacet,KodeKolektibilitas,SukuBunga,SumberDana,KodeJenisFasilitas,NOKTP) 
                    VALUES (TempCLAIMID, tOBJECTID, tOBJECTNAME, tBrand, tBrandName,
                        tModel, tModelName, tType, tTypeName, tserialno, tManufactureYear, tEngineNumber, tChassisNumber,tSLIKNO,tLocationID,sysdate,
                        tCONTRACTNO,tKodeKondisi,tJumlahHariTunggakan,tKodeSebabMacet,tKodeKolektibilitas,tSukuBunga,tSumberDana,tKodeJenisFasilitas,tNOKTP);
                        
                        
                EXCEPTION
                    WHEN OTHERS THEN
                    ErrMsg := 'INSERT T_CLAIM_OBJECTLIST Error : ' || sqlerrm;
                    ROLLBACK;
                    RETURN;
                END;  
                 
                ObjectCovDataLIST := datajson_ObjectList.get_array('ObjectCoverageList');
                IF ObjectCovDataLIST IS NOT NULL THEN
                    FOR k IN 0 .. ObjectCovDataLIST.get_size - 1 LOOP
                        datajson_ObjectCovList     :=  JSON_OBJECT_T(ObjectCovDataLIST.get(k));
                        tOBJECTCOVERAGEID := datajson_ObjectCovList.get_string('CoverageID');
                        tCAUSEOFLOSSID := datajson_ObjectCovList.get_string('CauseOfLossID');
                        tCAUSEOFLOSS := datajson_ObjectCovList.get_string('CauseOfLoss');
                        tCOVERAGEID := datajson_ObjectCovList.get_string('CoverageOldID');
                        tCOVERAGENAME := datajson_ObjectCovList.get_string('CoverageNote');
                        tSUMTSI := REPLACE(datajson_ObjectCovList.get_string('SumTSI'),',','.');
                        tCURICUMOFLOSS := datajson_ObjectCovList.get_string('CircumCauseOfLoss');
                        tEXTENTOFLOSS := datajson_ObjectCovList.get_string('ExtentOfLoss');
                        tInpatientDay := datajson_ObjectCovList.get_string('InpatientDay');
                        tInpatientDayMax := datajson_ObjectCovList.get_string('InpatientDayMax');
                        tRemarks := datajson_ObjectCovList.get_string('Remarks');
                        
                        
                        IF datajson_ObjectCovList.get_string('pxCreateDateTime') IS NOT NULL THEN
                        tPXCREATEDATETIME := POOLDATA.CONVERT_PEGA_DATE(datajson_ObjectCovList.get_string('pxCreateDateTime'));
                        END IF;
                        
                        IF datajson_ObjectCovList.get_string('BlastEmailDate')IS NOT NULL THEN
                        tBLASTEMAILDATE   := POOLDATA.CONVERT_PEGA_DATE(datajson_ObjectCovList.get_string('BlastEmailDate'));
                        END IF;
                        
                        if tPXCREATEDATETIME is null then
                        tPXCREATEDATETIME := sysdate;
                        end if; 
                        
                        
                        IF tOBJECTCOVERAGEID IS NOT NULL THEN
                            
                            
                            
                            BEGIN
                                INSERT INTO T_CLAIM_OBJECTCOVERAGE (CLAIMID, OBJECTID, OBJECTCOVERAGEID, CAUSEOFLOSSID, CAUSEOFLOSS, COVERAGEID, COVERAGENAME, CREATEDATETIME,BLASTEMAILDATE,SUMTSI,CURICUMOFLOSS,EXTENTOFLOSS,InpatientDay,
                                InpatientDayMax,Remarks,Currency,LOCATIONID,CONTRACTNO)
                                VALUES (TempCLAIMID, tOBJECTID, tOBJECTCOVERAGEID, tCAUSEOFLOSSID, tCAUSEOFLOSS, tCOVERAGEID, tCOVERAGENAME, tPXCREATEDATETIME,tBLASTEMAILDATE,tSUMTSI,tCURICUMOFLOSS,tEXTENTOFLOSS,
                                tInpatientDay,tInpatientDayMax,tRemarks,tCurrencyPolicy,tLocationID,tCONTRACTNO);
                            EXCEPTION
                                WHEN OTHERS THEN
                                ErrMsg := 'INSERT T_CLAIM_OBJECTCOVERAGE Error : ' || sqlerrm;
                                ROLLBACK;
                                RETURN;
                            END;  
                            --END IF;
                            
                            ListPenawaranLIST := datajson_ObjectCovList.get_array('ListPenawaranBengkel');
                            IF ListPenawaranLIST IS NOT NULL THEN
                                FOR p IN 0 .. ListPenawaranLIST.get_size-1 LOOP
                                    datajson_PenawaranList      := JSON_OBJECT_T(ListPenawaranLIST.get(p));
                                    tID_Penawaran       := datajson_PenawaranList.get_string('WorkShopID');
                                    tTTL_Harga_Jasa     := datajson_PenawaranList.get_string('Estimation');
                                    tTTL_Harga_Part     := datajson_PenawaranList.get_string('SparepartPrice');
                                    tTTL_Discount       := datajson_PenawaranList.get_string('SumEstimation');
                                    tOngkos_Kirim       := datajson_PenawaranList.get_string('NilaiPembayaranAdjustment');
                                    tTTL_Nett           := datajson_PenawaranList.get_string('SparepartNettPrice');
                                    tEst_Perbaikan      := datajson_PenawaranList.get_string('IsEstimate');
                                    tNama_Bengkel       := datajson_PenawaranList.get_string('WorkShopName');
                                    tWilayah_Bengkel    := datajson_PenawaranList.get_string('WONumber');
                                    
                                    IF datajson_PenawaranList.get_string('pxCreateDateTime') IS NOT NULL THEN
                                    tTANGGAL_PENAWARAN  := POOLDATA.CONVERT_PEGA_DATE(datajson_PenawaranList.get_string('pxCreateDateTime'));
                                    END IF;
                                                                        
                                    BEGIN
                                        INSERT INTO T_CLAIM_PENAWARAN (CLAIMID, OBJECTID, COVERAGEID,ID_PENAWARAN,TTL_HARGA_JASA,TTL_HARGA_PART,
                                        TTL_DISCOUNT,ONGKOS_KIRIM,TTL_NETT,EST_PERBAIKAN,NAMA_BENGKEL,WILAYAH_BENGKEL, TIPE, TJH_ID,OD_TJH,TANGGAL_PENAWARAN)
                                        VALUES (TempCLAIMID, tOBJECTID, tOBJECTCOVERAGEID,tID_Penawaran,tTTL_Harga_Jasa,tTTL_Harga_Part,tTTL_Discount,tOngkos_Kirim,
                                        tTTL_Nett,tEst_Perbaikan,tNama_Bengkel,tWilayah_Bengkel,'BENGKEL','','OD',tTANGGAL_PENAWARAN);
                                    EXCEPTION
                                    WHEN OTHERS THEN
                                        ErrMsg := 'INSERT T_CLAIM_PENAWARAN (BENGKEL) Error : ' || sqlerrm;
                                        ROLLBACK;
                                        RETURN;
                                    END;
                                END LOOP;
                            END IF;
                            
                            
                            ListPenawaranLIST := datajson_ObjectCovList.get_array('ListPenawaranSupplier');
                            IF ListPenawaranLIST IS NOT NULL THEN
                                FOR p IN 0 .. ListPenawaranLIST.get_size-1 LOOP
                                    datajson_PenawaranList      := JSON_OBJECT_T(ListPenawaranLIST.get(p));
                                    tID_Penawaran       := datajson_PenawaranList.get_string('WorkShopID');
                                    tTTL_Harga_Part     := datajson_PenawaranList.get_string('SparepartPrice');
                                    tTTL_Discount       := datajson_PenawaranList.get_string('SumEstimation');
                                    tOngkos_Kirim       := datajson_PenawaranList.get_string('NilaiPembayaranAdjustment');
                                    tTTL_Nett           := datajson_PenawaranList.get_string('SparepartNettPrice');
                                    tEst_Perbaikan      := datajson_PenawaranList.get_string('IsEstimate');
                                    tNama_Bengkel       := datajson_PenawaranList.get_string('WorkShopName');
                                    tWilayah_Bengkel    := datajson_PenawaranList.get_string('WONumber');
                                    IF datajson_PenawaranList.get_string('pxCreateDateTime') IS NOT NULL THEN
                                    tTANGGAL_PENAWARAN  := POOLDATA.CONVERT_PEGA_DATE(datajson_PenawaranList.get_string('pxCreateDateTime'));
                                    END IF;
                                                                        
                                    BEGIN
                                        INSERT INTO T_CLAIM_PENAWARAN (CLAIMID, OBJECTID, COVERAGEID,ID_PENAWARAN,TTL_HARGA_JASA,TTL_HARGA_PART,
                                            TTL_DISCOUNT,ONGKOS_KIRIM,TTL_NETT,EST_PERBAIKAN,NAMA_BENGKEL,WILAYAH_BENGKEL,TIPE,TJH_ID,OD_TJH,TANGGAL_PENAWARAN)
                                        VALUES (TempCLAIMID, tOBJECTID, tOBJECTCOVERAGEID,tID_Penawaran,'0',tTTL_Harga_Part,tTTL_Discount,tOngkos_Kirim,
                                            tTTL_Nett,tEst_Perbaikan,tNama_Bengkel,tWilayah_Bengkel,'SUPPLIER','','OD',tTANGGAL_PENAWARAN);
                                    EXCEPTION
                                    WHEN OTHERS THEN
                                        ErrMsg := 'INSERT T_CLAIM_PENAWARAN(SUPPLIER) Error : ' || sqlerrm;
                                        ROLLBACK;
                                        RETURN;
                                    END;
                                END LOOP;
                            END IF;
                            
                            TjhLIST := datajson_ObjectCovList.get_array('ThirdPartyLossList');
                            
                            id_countnull := 1;
                            IF TjhLIST IS NULL THEN
                                id_countnull := 0;
                            end if;
                            
                            
                            IF id_countnull=1 THEN

                                FOR p IN 0 .. TjhLIST.get_size-1 LOOP
                                    datajson_TjhList      := JSON_OBJECT_T(TjhLIST.get(p));
                                    datajson_TjhVehicle   :=  datajson_TjhList.get_object('ThirdPartyVehicle');
                                    tTJHID       := datajson_TjhList.get_string('pxListSubscript');
                                    tJenisTJH     := datajson_TjhList.get_string('JenisTJH');
                                    tEstTJHClaim     := datajson_TjhList.get_string('EstTJHClaim'); 
                                    tBrand       := datajson_TjhVehicle.get_string('Brand');
                                    tBrandName       := datajson_TjhVehicle.get_string('BrandName'); 
                                    tModel           := datajson_TjhVehicle.get_string('Model');
                                    tModelName      := datajson_TjhVehicle.get_string('ModelName');
                                    tType       := datajson_TjhVehicle.get_string('Type');
                                    tTypeName    := datajson_TjhVehicle.get_string('TypeName');
                                    tserialno    := datajson_TjhVehicle.get_string('LicensePlate'); 
                                    tManufactureYear    := datajson_TjhVehicle.get_string('ManufactureYear');
                                    tEngineNumber    := datajson_TjhVehicle.get_string('EngineNumber');
                                    tChassisNumber    := datajson_TjhVehicle.get_string('ChassisNumber'); 
                                                                    
                                    BEGIN
                                        INSERT INTO T_CLAIM_TJHLIST (CLAIMID, OBJECTID, COVERAGEID, TJHID, JENISTJH, ESTIMASITJH, BRANDID, BRANDNAME,
                                        MODELID, MODELNAME, TYPEID, TYPENAME, SERIALNO, MANUFACTUREYEAR, ENGINENUMBER, CHASISSNUMBER)
                                        VALUES (TempCLAIMID, tOBJECTID, tOBJECTCOVERAGEID, tTJHID, tJenisTJH, tEstTJHClaim, tBrand, tBrandName,
                                        tModel, tModelName, tType, tTypeName, tserialno, tManufactureYear, tEngineNumber, tChassisNumber);
                                    EXCEPTION
                                    WHEN OTHERS THEN
                                        ErrMsg := 'INSERT T_CLAIM_TJHLIST Error : ' || sqlerrm;
                                        ROLLBACK;
                                        RETURN;
                                    END;
                                   
                               
                                    PanelListLIST := datajson_TjhList.get_array('ObjectItemList'); 
                                    IF PanelListLIST IS NOT NULL THEN
                                        FOR k IN 0 .. PanelListLIST.get_size - 1 LOOP 
                                            datajson_PanelList     :=  JSON_OBJECT_T(PanelListLIST.get(k)); 
                                            tPANELID := datajson_PanelList.get_string('PanelID'); 
                                            tPANELIDX := datajson_PanelList.get_string('ObjectItemID');
                                            tExclusion := datajson_PanelList.get_string('ExclusionC');
                                            tIsLiablePanel := datajson_PanelList.get_string('IsLiablePanel');
                                            tPanelFrom := datajson_PanelList.get_string('PanelFrom');
                                            tPanelName := datajson_PanelList.get_string('PanelName');
                                            tPanelQty := datajson_PanelList.get_string('PanelQty');
                                            tPanelSide := datajson_PanelList.get_string('PanelSide');
                                            tPanelSupplier := datajson_PanelList.get_string('PanelSupplier');
                                            tPanelType := datajson_PanelList.get_string('PanelType');
                                            tPanelVehicle := datajson_PanelList.get_string('PanelVehicle');
                                            tCaseBengkel := datajson_PanelList.get_string('pyID');
                                            tWorkShopID := datajson_PanelList.get_string('WorkShopID');
                                            tWorkShopName := datajson_PanelList.get_string('WorkShopName'); 
                                            tCaseProcurement := datajson_PanelList.get_string('CaseID');
                                            tnopo := datajson_PanelList.get_string('PurchaseOrderNumber');
                                            IF tnopo IS NOT NULL THEN
                                                tpodate := POOLDATA.CONVERT_PEGA_DATE(datajson_PanelList.get_string('PurchaseOrderDate'));
                                            END IF;
                                            datajson_Sparepart               :=  datajson_PanelList.get_object('Sparepart');
                                            tSparepartNo := datajson_Sparepart.get_string('SparepartNo');
                                            tSparepartName := datajson_Sparepart.get_string('SparepartName');
                                            tSparepartSupplierID := datajson_Sparepart.get_string('SparepartSupplierID');
                                            tSparepartSupplierName := datajson_Sparepart.get_string('SparepartSupplierName');
                                            tJANJI_KIRIM        :=datajson_Sparepart.get_string('SparepartTime');
                                            
                                            BEGIN
                                                INSERT INTO T_CLAIM_PANEL (CLAIMID, OBJECTID, COVERAGEID, PANELID,PANELINDEX,EXCLUSIONC,ISLIABLEPANEL,PANELFROM,PANELNAME,PANELQTY,PANELSIDE,
                                                    PANELSUPPLIER, PANELTYPE,PANELVEHICLE,CASEBENGKEL,IDBENGKEL,BENGKELNAME,CASEPROCUREMENT, TJHID, SPAREPARTNO, SPAREPARTNAME,
                                                    NOPO, PODATE, SUPPLIERID, SUPPLIERNAME,JANJI_KIRIM) 
                                                VALUES (TempCLAIMID, tOBJECTID, tOBJECTCOVERAGEID, tPANELID, tPANELIDX, tExclusion, tIsLiablePanel, tPanelFrom, tPanelName, tPanelQty, tPanelSide,
                                                    tPanelSupplier, tPanelType, tPanelVehicle, tCaseBengkel, tWorkShopID, tWorkShopName, tCaseProcurement, tTJHID, tSparepartNo, tSparepartName,
                                                    tnopo, tpodate, tSparepartSupplierID, tSparepartSupplierName,tJANJI_KIRIM);
                                            EXCEPTION
                                                WHEN OTHERS THEN
                                                ErrMsg := 'INSERT T_CLAIM_PANEL TJH Error : ' || sqlerrm;
                                                ROLLBACK;
                                                RETURN;
                                            END; 
                                           
                                            EstimationLIST := datajson_PanelList.get_array('EstimationList');
                                            IF EstimationLIST.get_size>0 then
                                                FOR q IN 0 .. EstimationLIST.get_size - 1 LOOP
                                                    datajson_EstimationList     :=  JSON_OBJECT_T(EstimationLIST.get(q));
                                                    tIsGanti := datajson_EstimationList.get_string('IsGanti');
                                                    tIsJasa := datajson_EstimationList.get_string('IsJasa');
                                                    IF tIsGanti = 'true' or tIsJasa = 'true' THEN
                                                        tNOWO := datajson_EstimationList.get_string('NoWO');
                                                        
                                                        IF datajson_EstimationList.get_string('WODate') IS NOT NULL THEN
                                                        tWODate := POOLDATA.CONVERT_PEGA_DATE(datajson_EstimationList.get_string('WODate'));
                                                        END IF;
                                                        tTreatmentType := datajson_EstimationList.get_string('TreatmentType');
                                                        tStatusJasa := datajson_EstimationList.get_string('StatusJasa');
                                                        tTreatmentAproval := datajson_EstimationList.get_string('TreatmentAproval');
                                                        tNoAksep := datajson_EstimationList.get_string('NoAksep');
                                                        tESTIMATIONVALUE := REPLACE(datajson_EstimationList.get_string('EstimationValue'),',','.');
                                                        tPricetreat := REPLACE(datajson_EstimationList.get_string('ConvertValue'),',','.');
                                                        tDiscPanel := datajson_EstimationList.get_string('SparepartDisc');
                                                    
                                                        BEGIN
                                                            INSERT INTO T_CLAIM_TREATMENT (CLAIMID, OBJECTID, COVERAGEID, PANELID,PANELIDX,TJHID,NOWO,WODATE,
                                                                TREATMENTTYPE,STATUSJASA,TREATMENTAPROVAL,NOAKSEP,TREATMENTVALUE,PANELFROM,DISKON,PRICE) 
                                                            VALUES (TempCLAIMID, tOBJECTID, tOBJECTCOVERAGEID, tPANELID, tPANELIDX, tTJHID, tNOWO, tWODate,
                                                                tTreatmentType, tStatusJasa, tTreatmentAproval, tNoAksep, tESTIMATIONVALUE, tPanelFrom,tDiscPanel,tPricetreat);
                                                        EXCEPTION
                                                            WHEN OTHERS THEN
                                                            ErrMsg := 'INSERT T_CLAIM_TREATMENT TJH Error : ' || sqlerrm;
                                                            ROLLBACK;
                                                            RETURN;
                                                        END;  
                                                    END IF;
                                                END Loop;
                                            END IF;
                                        END Loop;
                                    END IF;       
                                    ListPenawaranLIST := datajson_TjhList.get_array('ListPenawaranBengkelTJH');
                                    IF ListPenawaranLIST IS NOT NULL THEN
                                    
                                        FOR p IN 0 .. ListPenawaranLIST.get_size-1 LOOP
                                            datajson_PenawaranList      := JSON_OBJECT_T(ListPenawaranLIST.get(p));
                                            tID_Penawaran       := datajson_PenawaranList.get_string('WorkShopID');
                                            tTTL_Harga_Jasa     := REPLACE(datajson_PenawaranList.get_string('Estimation'),',','.');
                                            tTTL_Harga_Part     := REPLACE(datajson_PenawaranList.get_string('SparepartPrice'),',','.');
                                            tTTL_Discount       := datajson_PenawaranList.get_string('SumEstimation');
                                            tOngkos_Kirim       := datajson_PenawaranList.get_string('NilaiPembayaranAdjustment');
                                            tTTL_Nett           := datajson_PenawaranList.get_string('SparepartNettPrice');
                                            tEst_Perbaikan      := datajson_PenawaranList.get_string('IsEstimate');
                                            tNama_Bengkel       := datajson_PenawaranList.get_string('WorkShopName');
                                            tWilayah_Bengkel    := datajson_PenawaranList.get_string('WONumber');
                                            IF datajson_PenawaranList.get_string('pxCreateDateTime') IS NOT NULL THEN
                                            tTANGGAL_PENAWARAN  := POOLDATA.CONVERT_PEGA_DATE(datajson_PenawaranList.get_string('pxCreateDateTime'));
                                            END IF;

                                                                            
                                            BEGIN
                                                INSERT INTO T_CLAIM_PENAWARAN (CLAIMID, OBJECTID, COVERAGEID,ID_PENAWARAN,TTL_HARGA_JASA,TTL_HARGA_PART,
                                                TTL_DISCOUNT,ONGKOS_KIRIM,TTL_NETT,EST_PERBAIKAN,NAMA_BENGKEL,WILAYAH_BENGKEL,TIPE,TJH_ID,OD_TJH,TANGGAL_PENAWARAN)
                                                VALUES (TempCLAIMID, tOBJECTID, tOBJECTCOVERAGEID,tID_Penawaran,tTTL_Harga_Jasa,tTTL_Harga_Part,tTTL_Discount,tOngkos_Kirim,
                                                tTTL_Nett,tEst_Perbaikan,tNama_Bengkel,tWilayah_Bengkel,'BENGKEL',tTJHID,'TJH',tTANGGAL_PENAWARAN);
                                            EXCEPTION
                                            WHEN OTHERS THEN
                                                ErrMsg := 'INSERT T_CLAIM_PENAWARAN (BENGKEL TJH) Error : ' || sqlerrm;
                                                ROLLBACK;
                                                RETURN;
                                            END;
                                        END LOOP;
                                    END IF;    
                                    ListPenawaranLIST := datajson_TjhList.get_array('ListPenawaranSupplierTJH');
                                    IF ListPenawaranLIST IS NOT NULL THEN
                                        FOR p IN 0 .. ListPenawaranLIST.get_size-1 LOOP
                                            datajson_PenawaranList      := JSON_OBJECT_T(ListPenawaranLIST.get(p));
                                            tID_Penawaran       := datajson_PenawaranList.get_string('WorkShopID');
                                            tTTL_Harga_Part     := datajson_PenawaranList.get_string('SparepartPrice');
                                            tTTL_Discount       := datajson_PenawaranList.get_string('SumEstimation');
                                            tOngkos_Kirim       := datajson_PenawaranList.get_string('NilaiPembayaranAdjustment');
                                            tTTL_Nett           := datajson_PenawaranList.get_string('SparepartNettPrice');
                                            tEst_Perbaikan      := datajson_PenawaranList.get_string('IsEstimate');
                                            tNama_Bengkel       := datajson_PenawaranList.get_string('WorkShopName');
                                            tWilayah_Bengkel    := datajson_PenawaranList.get_string('WONumber');
                                            
                                            IF datajson_PenawaranList.get_string('pxCreateDateTime') IS NOT NULL THEN
                                            tTANGGAL_PENAWARAN  := POOLDATA.CONVERT_PEGA_DATE(datajson_PenawaranList.get_string('pxCreateDateTime'));
                                            END IF;
                             
                                            BEGIN
                                                INSERT INTO T_CLAIM_PENAWARAN (CLAIMID, OBJECTID, COVERAGEID,ID_PENAWARAN,TTL_HARGA_JASA,TTL_HARGA_PART,
                                                TTL_DISCOUNT,ONGKOS_KIRIM,TTL_NETT,EST_PERBAIKAN,NAMA_BENGKEL,WILAYAH_BENGKEL,TIPE, TJH_ID,OD_TJH,TANGGAL_PENAWARAN)
                                                VALUES (TempCLAIMID, tOBJECTID, tOBJECTCOVERAGEID,tID_Penawaran,'0',tTTL_Harga_Part,tTTL_Discount,tOngkos_Kirim,
                                                tTTL_Nett,tEst_Perbaikan,tNama_Bengkel,tWilayah_Bengkel,'SUPPLIER',tTJHID,'TJH',tTANGGAL_PENAWARAN);
                                            EXCEPTION
                                            WHEN OTHERS THEN
                                                ErrMsg := 'INSERT T_CLAIM_PENAWARAN(SUPPLIER TJH) Error : ' || sqlerrm;
                                                ROLLBACK;
                                                RETURN;
                                            END;
                                        END LOOP;
                                    END IF;
                                    
                                END LOOP;
                            END IF;
                            
                            
                            
                            PanelListLIST := datajson_ObjectCovList.get_array('PanelList');
                            id_countnull := 1;
                            IF PanelListLIST IS NULL THEN
                                id_countnull := 0;
                            end if;
                            IF id_countnull =1 THEN
                                FOR p IN 0 .. PanelListLIST.get_size - 1 LOOP
                                    datajson_PanelList     :=  JSON_OBJECT_T(PanelListLIST.get(p));
                                    tPANELID := datajson_PanelList.get_string('PanelID');
                                    tPANELIDX := datajson_PanelList.get_string('ObjectItemID');
                                    tExclusion := datajson_PanelList.get_string('ExclusionC');
                                    tIsLiablePanel := datajson_PanelList.get_string('IsLiablePanel');
                                    tPanelFrom := datajson_PanelList.get_string('PanelFrom');
                                    tPanelName := datajson_PanelList.get_string('PanelName');
                                    tPanelQty := datajson_PanelList.get_string('PanelQty');
                                    tPanelSide := datajson_PanelList.get_string('PanelSide');
                                    tPanelSupplier := datajson_PanelList.get_string('PanelSupplier');
                                    tPanelType := datajson_PanelList.get_string('PanelType');
                                    tPanelVehicle := datajson_PanelList.get_string('PanelVehicle');
                                    tCaseBengkel := datajson_PanelList.get_string('pyID');
                                    tWorkShopID := datajson_PanelList.get_string('WorkShopID');
                                    tWorkShopName := datajson_PanelList.get_string('WorkShopName');
                                    tCaseProcurement := datajson_PanelList.get_string('CaseID');
                                    tnopo := datajson_PanelList.get_string('PurchaseOrderNumber');
                                    IF tnopo IS NOT NULL THEN
                                        tpodate := POOLDATA.CONVERT_PEGA_DATE(datajson_PanelList.get_string('PurchaseOrderDate'));
                                    END IF;
                                    datajson_Sparepart               :=  datajson_PanelList.get_object('Sparepart');
                                    tSparepartNo := datajson_Sparepart.get_string('SparepartNo');
                                    tSparepartName := datajson_Sparepart.get_string('SparepartName');
                                    tSparepartSupplierID := datajson_Sparepart.get_string('SparepartSupplierID');
                                    tSparepartSupplierName := datajson_Sparepart.get_string('SparepartSupplierName');
                                    tJANJI_KIRIM        :=datajson_Sparepart.get_string('SparepartTime');
                                      
                                            
                                    BEGIN
                                        INSERT INTO T_CLAIM_PANEL (CLAIMID, OBJECTID, COVERAGEID, PANELID,PANELINDEX,EXCLUSIONC,ISLIABLEPANEL,PANELFROM,PANELNAME,PANELQTY,PANELSIDE,
                                            PANELSUPPLIER, PANELTYPE,PANELVEHICLE,CASEBENGKEL,IDBENGKEL,BENGKELNAME,CASEPROCUREMENT, TJHID, SPAREPARTNO, SPAREPARTNAME,
                                            NOPO, PODATE, SUPPLIERID, SUPPLIERNAME,JANJI_KIRIM) 
                                        VALUES (TempCLAIMID, tOBJECTID, tOBJECTCOVERAGEID, tPANELID, tPANELIDX, tExclusion, tIsLiablePanel, tPanelFrom, tPanelName, tPanelQty, tPanelSide,
                                            tPanelSupplier, tPanelType, tPanelVehicle, tCaseBengkel, tWorkShopID, tWorkShopName, tCaseProcurement, tTJHID, tSparepartNo, tSparepartName,
                                            tnopo, tpodate, tSparepartSupplierID, tSparepartSupplierName,tJANJI_KIRIM);
                                    EXCEPTION
                                        WHEN OTHERS THEN
                                        ErrMsg := 'INSERT T_CLAIM_PANEL TJH Error : ' || sqlerrm;
                                        ROLLBACK;
                                        RETURN;
                                    END; 
                                    
                                    EstimationLIST := datajson_PanelList.get_array('EstimationList');
                                    IF EstimationLIST IS NOT NULL THEN
                                        FOR q IN 0 .. EstimationLIST.get_size - 1 LOOP
                                            
                                            datajson_EstimationList     :=  JSON_OBJECT_T(EstimationLIST.get(q));
                                            tIsGanti := datajson_EstimationList.get_string('IsGanti');
                                            tIsJasa := datajson_EstimationList.get_string('IsJasa');
                                            IF tIsGanti = 'true' or tIsJasa = 'true' THEN
                                                tNOWO := datajson_EstimationList.get_string('NoWO');
                                                
                                                IF datajson_EstimationList.get_string('WODate') IS NOT NULL THEN
                                                tWODate := POOLDATA.CONVERT_PEGA_DATE(datajson_EstimationList.get_string('WODate'));
                                                END IF;
                                                
                                                tTreatmentType := datajson_EstimationList.get_string('TreatmentType');
                                                tStatusJasa := datajson_EstimationList.get_string('StatusJasa');
                                                tTreatmentAproval := datajson_EstimationList.get_string('TreatmentAproval');
                                                tNoAksep := datajson_EstimationList.get_string('NoAksep');
                                                tDiscPanel := datajson_EstimationList.get_string('SparepartDisc');
                                                tESTIMATIONVALUE := REPLACE(datajson_EstimationList.get_string('EstimationValue'),',','.');
                                                tPricetreat := REPLACE(datajson_EstimationList.get_string('ConvertValue'),',','.');
                                                tPPN := REPLACE(datajson_EstimationList.get_string('PercentPLA'),',','.');
                                                tTTLPPN := REPLACE(datajson_EstimationList.get_string('ResultPLA'),',','.');
                                            
                                                BEGIN
                                                    INSERT INTO T_CLAIM_TREATMENT (CLAIMID, OBJECTID, COVERAGEID, PANELID,PANELIDX,TJHID,NOWO,WODATE,
                                                        TREATMENTTYPE,STATUSJASA,TREATMENTAPROVAL,NOAKSEP,TREATMENTVALUE,PANELFROM,DISKON,PRICE,PPN,TOTALPPN) 
                                                    VALUES (TempCLAIMID, tOBJECTID, tOBJECTCOVERAGEID, tPANELID, tPANELIDX, '', tNOWO, tWODate,
                                                        tTreatmentType, tStatusJasa, tTreatmentAproval, tNoAksep, tESTIMATIONVALUE, tPanelFrom,tDiscPanel,tPricetreat,tPPN,tTTLPPN);
                                                EXCEPTION
                                                    WHEN OTHERS THEN
                                                    ErrMsg := 'INSERT T_CLAIM_TREATMENT Error : ' || sqlerrm;
                                                    ROLLBACK;
                                                    RETURN;
                                                END; 
                                            END IF;
                                        END Loop;
                                    END IF;
                                END Loop;
                            END IF;
                            
                            
                            ObjectItemLIST := datajson_ObjectCovList.get_array('ObjectItemList');
                            
                            IF ObjectItemLIST IS NOT NULL THEN
                                FOR p IN 0 .. ObjectItemLIST.get_size - 1 LOOP
                                    datajson_ObjectItemList     :=  JSON_OBJECT_T(ObjectItemLIST.get(p));
                                    tOBJECTITEMID := p + 1;
                                    EstimationLIST := datajson_ObjectItemList.get_array('EstimationList');
                                    
                                    IF EstimationLIST IS NOT NULL THEN
                                        
                                        
                                        FOR q IN 0 .. EstimationLIST.get_size - 1 LOOP
                                            datajson_EstimationList     :=  JSON_OBJECT_T(EstimationLIST.get(q));
                                            tESTIMATIONVALUE := REPLACE(datajson_EstimationList.get_string('EstimationValue'),',','.');
                                            tKURSVALUE := REPLACE(datajson_EstimationList.get_string('KursValue'),',','.');
                                            tCONVERTVALUE := REPLACE(datajson_EstimationList.get_string('ConvertValue'),',','.');
                                            tCURRENCYEST := datajson_EstimationList.get_string('Currency');
                                            tESTIMATIONTYPE := datajson_EstimationList.get_string('EstimationType');
                                            tCFSDATE := TO_DATE(SUBSTR(datajson_EstimationList.get_string('CFSDate'),1,8),'yyyymmdd');
                                            tESTIMATIONID := q + 1;
                                            
                                            
                                            BEGIN
                                                INSERT INTO T_CLAIM_ESTIMASI (CLAIMID, OBJECTID, OBJECTCOVERAGEID, OBJECTITEMID, ESTIMASIID, ESTIMATIONTYPE, ESTIMATIONVALUE, KURSID, KURSVALUE, CONVERTVALUE, CFSDATE,LOCATIONID,
                                                CONTRACTNO) 
                                                VALUES (TempCLAIMID, tOBJECTID, tOBJECTCOVERAGEID, tOBJECTITEMID, tESTIMATIONID, tESTIMATIONTYPE, tESTIMATIONVALUE, tCURRENCYEST, tKURSVALUE, tCONVERTVALUE, tCFSDATE,tLocationID,
                                                tCONTRACTNO);
                                            EXCEPTION
                                                WHEN OTHERS THEN
                                                ErrMsg := 'INSERT T_CLAIM_ESTIMASI Error : ' || sqlerrm;
                                                ROLLBACK;
                                                RETURN;
                                            END;  
                                        END Loop;
                                    END IF;
                                END Loop;
                            END IF;

                            
                            AdjustmentDataLIST := datajson_ObjectCovList.get_array('AdjustmentList');

                            IF AdjustmentDataLIST IS NOT NULL THEN
                                    
                                FOR l IN 0 .. AdjustmentDataLIST.get_size - 1 LOOP
                                    datajson_AdjustmentList     :=  JSON_OBJECT_T(AdjustmentDataLIST.get(l));
                                    tADJUSTMENTID := datajson_AdjustmentList.get_string('pxListSubscript');
                                    IF tADJUSTMENTID IS NULL THEN
                                        tADJUSTMENTID := l + 1;
                                    END IF;
                                    
                                    
                                    BEGIN 
                                   
                                        tNOAKSEPTASI := datajson_AdjustmentList.get_string('AcceptedNo');
                                        tADJUSTMENT := REPLACE(datajson_AdjustmentList.get_string('AdjustmentValue'),',','.');
                                        tSALVAGE := REPLACE(datajson_AdjustmentList.get_string('SalvageValue'),',','.');
                                         tNILAISALVAGE_A  := REPLACE(datajson_AdjustmentList.get_string('SalvageValueA'),',','.');
                                        tADJUSTER := REPLACE(datajson_AdjustmentList.get_string('AdjusterFeeValue'),',','.');
                                        tGROSSVALUE := REPLACE(datajson_AdjustmentList.get_string('GrossValue'),',','.');
                                        tPAYMENTTYPE := datajson_AdjustmentList.get_string('PaymentType');
                                        tCURRENCY := datajson_AdjustmentList.get_string('Currency');
                                        tCURRENCYDOL := datajson_AdjustmentList.get_string('CurrencyDol');
                                        tSTATUSAKSEPTASI := datajson_AdjustmentList.get_string('AcceptanceStatus');
                                        tSTATUSAKSEPTASILOD := datajson_AdjustmentList.get_string('AcceptationStatusLOD');
                                        tRECEIVER := datajson_AdjustmentList.get_string('Receiver');
                                        tTGLAKSEPTASI :=TO_DATE(SUBSTR(datajson_AdjustmentList.get_string('AcceptedDate'),1,8),'yyyymmdd');
                                        tTANGGALBAYAR :=TO_DATE(datajson_AdjustmentList.get_string('TanggalBayar'),'yyyymmdd');
                                        tSTATUSBAYAR  :=  datajson_AdjustmentList.get_string('StatusBayar');
                                        tTOTALCLAIM :=  REPLACE(datajson_AdjustmentList.get_string('ProposeAdjustmentValue'),',','.');
                                        tINDIVIDUAL_RISKTYPE  := datajson_AdjustmentList.get_string('IndividualRiskType');
                                        tINDIVIDUAL_RISKVALUE  :=  REPLACE(datajson_AdjustmentList.get_string('IndividualRiskValue'),',','.');
                                        tPROPOSE_VALUE := REPLACE(datajson_AdjustmentList.get_string('ProposeValue'),',','.');
                                        tLOC      := datajson_AdjustmentList.get_string('LOC');
                                        tASM_SHARE       := datajson_AdjustmentList.get_string('ShareASM');
                                        tASM_SHARE_VALUE   :=  REPLACE(datajson_AdjustmentList.get_string('AdjustmentValue'),',','.');
                                        tINDIVIDUAL_RISKPERCENT := datajson_AdjustmentList.get_string('IndividualRiskPercentage');
                                        tTRANSFERCASHIERDATE    := TO_DATE(SUBSTR(datajson_AdjustmentList.get_string('TransferCashierDate'),1,8),'yyyymmdd');
                                        tCLAIMPAIDSTATUS        := datajson_AdjustmentList.get_string('ClaimPaidStatus');
                                        tCaseIDCashier        := datajson_AdjustmentList.get_string('CaseIDCashier');
                                        tRECEIVERNAME := '';
                                        tMANUALACCEPTANCEDATECOMITEE :=  TO_DATE(datajson_AdjustmentList.get_string('AcceptedDateKomiteManual'),'yyyymmdd');
                                        tPRINTLOD_DATE :=  TO_DATE(SUBSTR(datajson_AdjustmentList.get_string('PrintDateLOD'),1,8),'yyyymmdd');
                                        tRECEIVELOD_DATE :=  TO_DATE(SUBSTR(datajson_AdjustmentList.get_string('ReceiveDateLOD'),1,8),'yyyymmdd');
                                        tACCEPTANCE_DATECOMITEE := TO_DATE(SUBSTR(datajson_AdjustmentList.get_string('AcceptedDateKomite'),1,8),'yyyymmdd'); 
                                        ttanggalkomites := datajson_AdjustmentList.get_string('TanggalComitee');
                                        tSTPenolakan := datajson_AdjustmentList.get_string('KodePenolakanND');
                                        tNDPenolakan := datajson_AdjustmentList.get_string('KodePenolakanST');
                                        tANALYST_TFKOMITEDATE := TO_DATE(SUBSTR(datajson_AdjustmentList.get_string('TanggalComitee'),1,8),'yyyymmdd');
                                        
                                    
                                        if  datajson_AdjustmentList.get_string('pxListSubscript') is null then
                                            IF ttanggalkomites IS NULL THEN
                                                tANALYST_TFKOMITEDATE    := TO_DATE(SUBSTR(datajson_AdjustmentList.get_string('pxCreateDateTime'),1,8),'yyyymmdd');
                                            END IF;
                                         end if;    
                                        
                                        
                                        
                                        ReceiverDataLIST := datajson_JSONKLAIM.get_array('ReceiverClaim');
                                        tLOSS_ADJUSTER_FEE   :=  REPLACE(datajson_AdjustmentList.get_string('LossAdjusterFee'),',','.');
                                        --select utl_encode.base64_encode(utl_raw.cast_to_raw(datajson_AdjustmentList.get_string('RejectClaimNote'))) into tTOLAK from dual;
                                        --dbms_output.put_line(SUBSTR(datajson_AdjustmentList.get_string('RejectClaimNote'),1,4000));
                                        tTOLAK := datajson_AdjustmentList.get_string('RejectClaimNote');
                                        tEXGRATIA := datajson_AdjustmentList.get_string('ExGratia');
                                        tCircumCauseOfLoss := datajson_AdjustmentList.get_string('CircumCauseOfLoss');
                                        tNotes := SUBSTR(translate(datajson_AdjustmentList.get_string('Notes'), chr(10) || chr(13) || chr(09), ' '),0,4000);
                                        tCoverInsKey := datajson_AdjustmentList.get_string('CaseIDKomite');
                                        --tCoverInsKey := substr(datajson_AdjustmentList.get_string('CaseIDKomite'),20,length(datajson_AdjustmentList.get_string('CaseIDKomite')));
                                        
                                        
                                    EXCEPTION
                                           WHEN OTHERS THEN
                                           ErrMsg := 'INSERT Declare Adjustment : ' ||tREVISIDLA|| sqlerrm;
                                           ROLLBACK;
                                           RETURN;
                                    END;  
                                    
                                            
                                    IF ReceiverDataLIST IS NOT NULL THEN
                                        FOR p IN 0 .. ReceiverDataLIST.get_size - 1 LOOP
                                            datajson_ReceiverList     :=  JSON_OBJECT_T(ReceiverDataLIST.get(p));
                                            
                                            tRECEIVERID := datajson_ReceiverList.get_string('IDReceiver');
                                            IF tRECEIVERID = tRECEIVER THEN
                                                tRECEIVERNAME := datajson_ReceiverList.get_string('Name');
                                            END IF;
                                        END LOOP;
                                    END IF;


                                    IF tTYPEOFCOINS IN ('1','F') THEN
                                        IF tPAYMENTTYPE = '3' THEN
                                            tNILAIAKSEPTASI := tSALVAGE;
                                        ELSIF tPAYMENTTYPE = '4' THEN
                                            tNILAIAKSEPTASI := tADJUSTER;
                                        ELSE
                                            tNILAIAKSEPTASI := tADJUSTMENT;
                                        END IF;
                                    ELSE
                                        tNILAIAKSEPTASI := tGROSSVALUE;
                                    END IF;
                                    
                                    --Insert Komite Biar dapat ID Komite
                                    komitedatalist   := datajson_AdjustmentList.get_array('ComiteeClaim');
                                    if komitedatalist is not null then 
                                    
                                        for km IN 0 .. komitedatalist.get_size - 1 LOOP
                                            tlistkomite := km+1;
                                            datajson_komitelist := JSON_OBJECT_T(komitedatalist.get(km));
                                            tKomiteAproval := datajson_komitelist.get_string('KomiteAproval');
                                            tKomiteComment := SUBSTR(datajson_komitelist.get_string('KomiteComment'),1,4000);
                                            tKomiteID := datajson_komitelist.get_string('KomiteID');
                                            tnoklaimnum := substr(TempCLAIMID,20,length(TempCLAIMID));
                                            
                                            if tCoverInsKey is null then 
                                                tCoverInsKey := substr(datajson_komitelist.get_string('CoverInsKey'),20,length(datajson_komitelist.get_string('CoverInsKey')));
                                            end if;
                                            
                                            
                                            if tKomiteAproval ='1' then 
                                                tSTATUSCASE := 'Resolved-Completed';
                                            else    
                                                   tSTATUSCASE :='New';
                                            end if;
                                            
                                            
                                            BEGIN
                                                    SELECT COUNT (1) INTO counts_komites FROM POOLDATA.T_CLAIM_KOMITE_LIST WHERE KOMITE_ID = tCoverInsKey AND NAMAKOMITE = tKomiteID;
                                                    
                                                    IF counts_komites<=0 then 
                                                        INSERT INTO POOLDATA.T_CLAIM_KOMITE_LIST (KOMITE_ID,NO_KLAIM,NAMAKOMITE,STATUSCASE,STATUSAPPROVE,NOTEKOMITE,DATEOFCOMMITE_CREATE,TANGGALKOMITE,KOMITEKE)
                                                        VALUES (tCoverInsKey,tnoklaimnum,tKomiteID,tSTATUSCASE,tKomiteAproval,tKomiteComment,tANALYST_TFKOMITEDATE,tACCEPTANCE_DATECOMITEE,tlistkomite);
                                                    
                                                    else
                                                        UPDATE POOLDATA.T_CLAIM_KOMITE_LIST SET STATUSAPPROVE = tKomiteAproval,NOTEKOMITE = tKomiteComment,TANGGALKOMITE=tACCEPTANCE_DATECOMITEE, STATUSCASE=tSTATUSCASE WHERE KOMITE_ID = tCoverInsKey AND NAMAKOMITE = tKomiteID;
                                                    
                                                    END IF;
                                                    
                                            
                                            EXCEPTION
                                                    WHEN OTHERS THEN
                                                    ErrMsg := 'INSERT KomiteList Error : ' ||tREVISIDLA|| sqlerrm;
                                                    ROLLBACK;
                                                    RETURN;
                                            END;   
                                        
                                        
                                        end Loop;
                                    
                                    
                                    end if;


                                    
                                    
                                    BEGIN
                                        if tOBJECTCOVERAGEID is not null then
                                            
                                            
                                        
                                            INSERT INTO T_CLAIM_ADJUSTMENT (CLAIMID, OBJECTID, OBJECTCOVERAGEID, ADJUSTMENTID, NOAKSEPTASI, NILAIAKSEPTASI, STATUSAKSEPTASI, TGLAKSEPTASI, PAYMENTTYPE, GROSSVALUE, STATUSAKSEPTASILOD, 
                                                            RECEIVER, RECEIVERNAME, CURRENCY, CURRENCYVALUE, MANUALACCEPTANCEDATECOMITEE, PRINTLOD_DATE, ACCEPTANCE_DATECOMITEE,  ANALYST_TFKOMITEDATE,TOTAL_CLAIM,NILAI_SALVAGE_A,
                                                            INDIVIDUAL_RISK_TYPE,INDIVIDUAL_RISK_VALUE,INDIVIDUAL_RISK_PERCENT,PROPOSE_VALUE,LOC,ASM_SHARE,ASM_SHARE_VALUE, RECEIVEDATELOD,TRANSFER_CASHIER_DATE,CLAIM_STATUS_PAID,
                                                            TGLBAYAR,LOSS_ADJUSTER_FEE,ALASANTOLAK,EXGRATIA,CIRCUMCAUSEOFLOSS,NOTES,IDCHASIER,CaseIDKomite,IDPENOLAKANST,IDPENOLAKANDUA,LOCATIONID,CONTRACTNO) 
                                            VALUES (TempCLAIMID, tOBJECTID, tOBJECTCOVERAGEID, tADJUSTMENTID, tNOAKSEPTASI, tNILAIAKSEPTASI, tSTATUSAKSEPTASI, tTGLAKSEPTASI, tPAYMENTTYPE, tGROSSVALUE, tSTATUSAKSEPTASILOD, 
                                                            tRECEIVER, tRECEIVERNAME, tCURRENCY, tCURRENCYDOL, tMANUALACCEPTANCEDATECOMITEE, tPRINTLOD_DATE, tACCEPTANCE_DATECOMITEE, tANALYST_TFKOMITEDATE,tTOTALCLAIM,tNILAISALVAGE_A,
                                                            tINDIVIDUAL_RISKTYPE,tINDIVIDUAL_RISKVALUE,tINDIVIDUAL_RISKPERCENT,tPROPOSE_VALUE,tLOC,tASM_SHARE,tASM_SHARE_VALUE, tRECEIVELOD_DATE,tTRANSFERCASHIERDATE,tCLAIMPAIDSTATUS,
                                                            tTANGGALBAYAR,tLOSS_ADJUSTER_FEE,tTOLAK,tEXGRATIA,tCircumCauseOfLoss,tNotes,tCaseIDCashier,tCoverInsKey,tSTPenolakan,tNDPenolakan,tLocationID,tCONTRACTNO);
                                            
                                        
                                        end if;
                                    
                                    
                                
                                     EXCEPTION
                                        WHEN OTHERS THEN
                                        ErrMsg := 'INSERT T_CLAIM_ADJUSTMENT Error : ' || sqlerrm;
                                        ROLLBACK;
                                        RETURN;
                                    END;  
                                    
                                    -- detail spreding
                                    SpreadingsLIST := datajson_AdjustmentList.get_array('SpreadingList');
                                    id_countnull := 1;
                                    IF SpreadingsLIST IS NULL THEN
                                        id_countnull := 0;
                                    end if;
                                    IF tNOAKSEPTASI is not null and id_countnull=1 then
                                            FOR s IN 0 .. SpreadingsLIST.get_size - 1 LOOP
                                                
                                                datajson_SpredingsList :=  JSON_OBJECT_T(SpreadingsLIST.get(s));
                                                tTREATYNAME :=datajson_SpredingsList.get_string('TreatyName');
                                                tTREATYTYPE :=datajson_SpredingsList.get_string('TreatyType');
                                                tTREATYYEAR :=datajson_SpredingsList.get_string('TreatyYear');
                                                tTSISPREADED:=REPLACE(datajson_SpredingsList.get_string('TSISpreaded'),',','.');
                                                tTREATYGROUP :=datajson_SpredingsList.get_string('TreatyGroup');
                                                tSHAREPERCENTAGE:=REPLACE(datajson_SpredingsList.get_string('SharePercentage'),',','.');
                                            
                                                BEGIN
                                                    INSERT INTO POOLDATA.T_CLAIM_DETAIL_SPREDING
                                                        (CLAIMID,NOAKSEPTASI,
                                                        TREATYNAME,
                                                        TREATYTYPE,
                                                        TREATYYEAR,
                                                        TSISPREADED,
                                                        TREATYGROUP,
                                                        SHAREPERCENTAGE,CONTRACTNO)
                                                        VALUES
                                                        (TempCLAIMID,tNOAKSEPTASI,
                                                        tTREATYNAME,
                                                        tTREATYTYPE,
                                                        tTREATYYEAR,
                                                        tTSISPREADED,
                                                        tTREATYGROUP,
                                                        tSHAREPERCENTAGE,tCONTRACTNO);
                                                EXCEPTION
                                                    WHEN OTHERS THEN
                                                    ErrMsg := 'INSERT Insert Spreding List Error : ' || sqlerrm;
                                                    ROLLBACK;
                                                    RETURN;
                                                END;
                                    
                                            END Loop;
                                    end if;
                                    
                                    
                                    
                                    /*                                 
                                    DLADataLIST := datajson_AdjustmentList.get_array('DLAList');
                                    IF DLADataLIST IS NOT NULL THEN
                                        FOR m IN 0 .. DLADataLIST.get_size - 1 LOOP
                                            datajson_DLAList     :=  JSON_OBJECT_T(DLADataLIST.get(m));
                                            tNODLA :=datajson_DLAList.get_string('NO_DLA');
                                            tTGLDLA := TO_DATE(SUBSTR(datajson_DLAList.get_string('Date'),1,8),'yyyymmdd');
                                            tNILAIDLA :=REPLACE(datajson_DLAList.get_string('DLAShare'),',','.');
                                            tDLAREINSURER :=datajson_DLAList.get_string('DLAReinsurer');
                                            tTIPEDLA :=datajson_DLAList.get_string('DLAType');
                                            tREVISIDLA := datajson_DLAList.get_string('RevisiDLA');
                                            IF tREVISIDLA IS NULL THEN
                                                tREVISIDLA := 0;
                                            END IF;
--                                            
                                            IF tNODLA IS NOT NULL THEN
                                                BEGIN
                                                    INSERT INTO T_DLALIST (CLAIMID, OBJECTID, OBJECTCOVERAGEID, ADJUSTMENTID, NODLA, NILAIDLA, DLAREINSURER, TGLDLA, TIPEDLA, REVISI) 
                                                    VALUES (TempCLAIMID, tOBJECTID, tOBJECTCOVERAGEID, tADJUSTMENTID, tNODLA, tNILAIDLA, tDLAREINSURER, tTGLDLA, tTIPEDLA, tREVISIDLA);
                                                EXCEPTION
                                                    WHEN OTHERS THEN
                                                    ErrMsg := 'INSERT T_DLALIST Error : ' ||tREVISIDLA|| sqlerrm;
                                                    ROLLBACK;
                                                    RETURN;
                                                END;  
                                               
                                            END IF;
                                        END Loop;
                                    END IF;*/
                                END Loop; 
                            END IF;
                            
                        END IF;
                        /*
                        PLADataLIST := datajson_ObjectCovList.get_array('PLAList');
                        IF PLADataLIST IS NOT NULL THEN
                            FOR n IN 0 .. PLADataLIST.get_size - 1 LOOP
                                datajson_PLAList     :=  JSON_OBJECT_T(PLADataLIST.get(n));
                                tNOPLA :=datajson_PLAList.get_string('NoPLA');
                                tPLAREINSURER := datajson_PLAList.get_string('PLAReinsurer');
                                tTIPEPLA := datajson_PLAList.get_string('TipePLA');
                                tREVISI := datajson_PLAList.get_string('Revisi');
                                tTGLPLA := TO_DATE(SUBSTR(datajson_PLAList.get_string('Date'),1,8),'yyyymmdd');
                                
                                IF tNOPLA IS NOT NULL THEN
                                    BEGIN
                                        INSERT INTO T_PLALIST (CLAIMID, OBJECTID, OBJECTCOVERAGEID, NOPLA, PLAREINSURER, TIPEPLA, REVISI, TGLPLA) 
                                        VALUES (TempCLAIMID, tOBJECTID, tOBJECTCOVERAGEID, tNOPLA, tPLAREINSURER, tTIPEPLA, tREVISI, tTGLPLA);
                                    EXCEPTION
                                        WHEN OTHERS THEN
                                        ErrMsg := 'INSERT T_PLALIST Error : ' || sqlerrm;
                                        ROLLBACK;
                                        RETURN;
                                    END;
                                END IF;
                            END Loop;
                        END IF;*/
                    END Loop;
                END IF;
            END IF;
        END Loop;
    END IF;
    
    UPDATE JSON_KLAIM SET IDPROD=1 WHERE IDPEGA = TempCLAIMID;
    COMMIT;
EXCEPTION
    WHEN OTHERS THEN
        ErrMsg := 'PEGA_CONVERT_JSONKLAIM_PNC Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/