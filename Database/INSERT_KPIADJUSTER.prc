CREATE OR REPLACE PROCEDURE          INSERT_KPIADJUSTER (
TADJUSTER in varchar2, 
TCASEID in varchar2, 
TSURVEYLAP in varchar2, 
TIMMEDIATEADVICE in varchar2, 
TPRELIMINARYADVICE in varchar2, 
TINTERIM in varchar2, 
TPROGRESS in varchar2, 
TKOMUNIKASI in varchar2,
TPROPOSE in varchar2, 
TFINALREPORT in varchar2, 
TNILAI in varchar2, 
TTIPE in varchar2,
TTANGGAL in date,
ErrMsg OUT VARCHAR2)
AS

idcount number;

BEGIN           
        select count(1) into idcount from POOLDATA.DETAIL_KPI_ADJUSTER where caseid=TCASEID;
      
            IF idcount < 1 THEN
                      
                INSERT INTO POOLDATA.DETAIL_KPI_ADJUSTER (ADJUSTER,CASEID, SURVEYLAP, IMMEDIATEADVICE, PRELIMINARYADVICE, INTERIM, PROGRESS,
                                                          KOMUNIKASI, PROPOSE, FINALREPORT, NILAI,TIPE,TANGGAL)
                VALUES (TADJUSTER, TCASEID, TSURVEYLAP, TIMMEDIATEADVICE, TPRELIMINARYADVICE, TINTERIM, 
                        TPROGRESS,TKOMUNIKASI, TPROPOSE, TFINALREPORT, TNILAI, TTIPE,TTANGGAL); 
                COMMIT;  
        
            ELSIF idcount > 0 THEN
    
                UPDATE DETAIL_KPI_ADJUSTER SET 
                ADJUSTER = TADJUSTER,
                CASEID = TCASEID,
                SURVEYLAP  = TSURVEYLAP,
                IMMEDIATEADVICE = TIMMEDIATEADVICE,
                PRELIMINARYADVICE = TPRELIMINARYADVICE,
                INTERIM = TINTERIM, 
                PROGRESS = TPROGRESS,
                KOMUNIKASI = TKOMUNIKASI,
                PROPOSE = TPROPOSE,
                FINALREPORT = TFINALREPORT,
                NILAI = TNILAI,
                TIPE = TTIPE,
                TANGGAL = TTANGGAL
                WHERE CASEID=TCASEID;
                COMMIT;
    
            END IF;      
    
EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error INSERT KPI ADJUSTER : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/