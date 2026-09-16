CREATE OR REPLACE PROCEDURE          INSERT_PNCCHRONOLOGYTAT(TempCLAIMID in varchar2, TPOSITION in varchar2, TUSER in varchar2, TNOTE in varchar2, 
                                                                                        TPROG1 in varchar2, TPROG2 in varchar2, TTIMEIN in date, TTIMEOUT in date, pbisnis IN VARCHAR2, pjenisklaim IN VARCHAR2, 
                                                                                        TTIMEFINAL IN date, TNOKLAIM IN VARCHAR2,
                                                                                        ErrMsg OUT VARCHAR2)
AS
TTIMEDIFFERENCE  varchar2(50);
TAGING number;
idcount number;
tcabang      varchar2(50);
tseksi_name     varchar2(100);
tatas_id        varchar2(100);
tnama       varchar2(100);
tseksi_code     varchar2(10);
flgmitra number;
tJOBCODE  varchar2(15);
yymm   varchar2(10);
TTIMEIN2    date;
tgrouppanel     varchar2(5);
TPOSITION2   varchar2(100);
jobcodetemps varchar2(100);
postionstemps varchar2(1000);
StepPosition varchar2(1000);
    
BEGIN
        StepPosition := 'Mulai Set Posistion';
        select grouppanel into tgrouppanel from pooldata.business where id=pbisnis;
        postionstemps := TPOSITION;
        if TPOSITION='Register Klaim' then
            TPOSITION2:='REGISTRASI';
            jobcodetemps :='000212';
            if tgrouppanel='002' then
                jobcodetemps :='000218';
            END IF;
        ELSE
            TPOSITION2:=TPOSITION;
        end if;
        
        StepPosition := 'Cek PUCL dan Klaim';
        if tgrouppanel='002' and TPOSITION ='PUCL / RCL IP' then
            TPOSITION2:= 'PUCL / RCL IP';
            jobcodetemps :='000337';
        elsif tgrouppanel='002' and TPOSITION !='PUCL / RCL IP' then
            IF upper(postionstemps)='AKSEPTASI' then
                jobcodetemps :='000219';
            END IF;
            TPOSITION2:=upper(TPOSITION2) || ' KLAIM PA';
        else
            TPOSITION2:=upper(TPOSITION2) || ' KLAIM NON MBU';
        end if;

    StepPosition := 'Cek Data DI PNC_CHRONOLOGYTAT';
    if TPOSITION='Register Klaim' and tgrouppanel !='002' then  
        select count(1) into idcount from pooldata.PNC_CHRONOLOGYTAT where idpega=TempCLAIMID and position='REGISTRASI KLAIM NON MBU';  
    elsif TPOSITION='Register Klaim' and tgrouppanel ='002' then   
        select count(1) into idcount from pooldata.PNC_CHRONOLOGYTAT where idpega=TempCLAIMID and position='REGISTRASI KLAIM PA';
    else
        select count(1) into idcount from POOLDATA.PNC_CHRONOLOGYTAT where idpega=TempCLAIMID and position=TPOSITION2 and position not in ('PENERIMAAN DOKUMEN','PUCL / RCL IP');
    end if;
    
    
    IF idcount < 1 THEN
        if TTIMEIN is null then
            TTIMEIN2 := TTIMEOUT;
        else
            TTIMEIN2 := TTIMEIN;
        end if;
        
        if TTIMEFINAL IS NULL THEN
            TTIMEIN2 := TTIMEIN2;
        else
            TTIMEIN2 := TTIMEFINAL;
        end if;
            
         StepPosition := 'Setting Time IN dan OUT';
        SELECT datamining.
                get_working_hours@ASMD.SINARMAS.CO.ID (
                  TO_DATE (TTIMEIN2, 'dd/mm/rrrr hh24:mi:ss'),
                  TO_DATE (TTIMEOUT, 'dd/mm/rrrr hh24:mi:ss'))
        INTO TAGING
        FROM DUAL;
        
        SELECT (   TO_CHAR (TRUNC (TAGING / 3600), 'FM9900')
             || ':'
             || TO_CHAR (TRUNC (MOD (TO_NUMBER (TAGING), 3600) / 60), 'FM00')
             || ':'
             || TO_CHAR (MOD (TO_NUMBER (TAGING), 60), 'FM00'))
        into TTIMEDIFFERENCE FROM DUAL;
        
        StepPosition := 'Insert Data PNC_CHRONOLOGYTAT';
        
        BEGIN
            INSERT INTO POOLDATA.PNC_CHRONOLOGYTAT (IDPEGA, POSITION, USERASSIGN, TIMEIN, TIMEOUT, TIMEDIFFERENCE, AGING, NOTE, STATUSPROGRESS1, STATUSPROGRESS2, INSERTDATE, BISNISID, JENIS_KLAIM, TGLKETERLAMBATAN,NOKLAIM)
            VALUES (TempCLAIMID, TPOSITION2, TUSER, TTIMEIN2, TTIMEOUT, TTIMEDIFFERENCE, TAGING, TNOTE, TPROG1, TPROG2, SYSDATE, pbisnis, pjenisklaim,TTIMEFINAL,TNOKLAIM);
        EXCEPTION
            WHEN OTHERS THEN
            ErrMsg := 'INSERT PNC_CHRONOLOGYTAT Error : ' || sqlerrm;
            ROLLBACK;
            RETURN;
        END;
        
        BEGIN
            select cabang, seksi_name,atas_id, nama,seksi_code
            into tcabang, tseksi_name, tatas_id, tnama, tseksi_code
            from general.lst_mitra@asmd.sinarmas.co.id where login_aplikasi =TUSER;
            
            StepPosition := 'Cari Data Login User';
            flgmitra := 1;
        EXCEPTION WHEN NO_DATA_FOUND THEN
            flgmitra := 0;
        END;
        
        IF flgmitra>0 THEN
            
            StepPosition := 'Set Job CODE ='||jobcodetemps;
            BEGIN
                if jobcodetemps is null or jobcodetemps='' then
                    select JOBCODE into tJOBCODE from general.lst_mitra_jobdesc@asmd.sinarmas.co.id where JOBDESC =TPOSITION2;
                ELSE 
                    select JOBCODE,JOBDESC into tJOBCODE,TPOSITION2 from general.lst_mitra_jobdesc@asmd.sinarmas.co.id where JOBCODE =jobcodetemps;
                END IF;
                flgmitra := 1;
            EXCEPTION WHEN NO_DATA_FOUND THEN  
                flgmitra := 0;  
                
            END;
            
            if flgmitra>0 THEN
            
                SELECT (CASE WHEN TO_NUMBER (TO_CHAR (SYSDATE, 'dd')) >= 20
                    THEN
                        TO_CHAR (ADD_MONTHS (SYSDATE, 1), 'YYYYMM')
                    ELSE
                        TO_CHAR (SYSDATE, 'YYYYMM')
                     END) into yymm
                FROM DUAL;
                   
                BEGIN
                    insert into general.lst_mitra_prod_det@asmd.sinarmas.co.id  (LOGIN_APLIKASI, YYYYMM, SEKSI_NAME, ATAS_ID, MKL_NO_POLIS, JENIS_KLAIM, CABANG, TGL_AWAL, TGL_AKHIR, AGING, 
                            SATUAN, SESUAI_SLA, JML_SESUAI_SLA, JML_TDK_SESUAI_SLA, PERSENTASE_SESUAI_SLA, TOTAL_PRODUKTIVITAS, TOTAL_PROD_TEAM, PERSENTASE_PROD, TGLIU, 
                            NAMA_MITRA, TOTAL_SDM_LOGIN, SEKSI_CODE, JOBCODE, JOBDESC)
                    values(TUSER, yymm, tseksi_name, tatas_id, (case when(substr(TempCLAIMID,1,3)='ASM') then substr(TempCLAIMID,20,100) else TempCLAIMID end), pjenisklaim, tcabang, 
                            TTIMEIN2, TTIMEOUT, (floor(TAGING/28800)), 'AGING', (case when(floor(TAGING/28800)<=1) then 'Y' else 'N' end),  (case when(floor(TAGING/28800)<=1) then '1' else '0' end), 
                            (case when(floor(TAGING/28800)<=1) then '0' else '1' end), (case when(floor(TAGING/28800)<=1) then '100' else '0' end), 
                            '1', '1', '100', sysdate, tnama, '1' , tseksi_code, tJOBCODE, TPOSITION2);
                END;
            end if;
           
        end if;
        
        IF ErrMsg IS NULL THEN
            COMMIT;
        ELSE
            ROLLBACK;
        END IF;
    END IF;
    
EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error INSERT_PNC_CHRONOLOGYTAT : ' || sqlerrm ||' ' ||StepPosition;
        ROLLBACK;
        RETURN;
END;

/