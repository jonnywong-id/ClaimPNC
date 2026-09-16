CREATE OR REPLACE PROCEDURE          INSERT_SURVEYORLIST (
TCASEID in varchar2, 
TPNCCASEID in varchar2, 
TSRVTYPE in varchar2, 
TLOSSTYPE in varchar2, 
TSRVNAME in varchar2, 
TSRVDATE in date, 
TSRVLOC in varchar2,
TOBJNAME in varchar2, 
TOBJLOC in varchar2, 
TOBJID in varchar2, 
TSRVINDEX in varchar2, 
TDOL in date, 
TPANSIEN in varchar2, 
TSTS in varchar2,
TNOTE in varchar2,
TAKSEP in varchar2,
ErrMsg OUT VARCHAR2)
AS

idcount number;
listsur number;
jumlahsurvey number;
picklaimst varchar2(2000);

BEGIN           
        select count(1) into idcount from POOLDATA.T_SURVEYORLIST where caseid=TCASEID and index_survey=TSRVINDEX;
      
            IF idcount < 1 THEN
                      
                INSERT INTO POOLDATA.T_SURVEYORLIST (TGLINPUT,CASEID, PNCCASEID,SURVEYTYPE, LOSSTYPE, SURVEYOR_NAME, SURVEYDATE, LOCATION_SURVEY, OBJECT_NAME, LOCATION_OBJECT,
                                                     IDOBJECT, INDEX_SURVEY,TGLPERAWATAN,NAMA_PASIEN,STS_SURVEY,KETERANGAN, STSAKSEP)
                VALUES (SYSDATE,TCASEID, TPNCCASEID, TSRVTYPE, TLOSSTYPE, TSRVNAME, TSRVDATE,TSRVLOC, TOBJNAME,TOBJLOC,TOBJID,TSRVINDEX,
                        TDOL,TPANSIEN,TSTS,TNOTE, TAKSEP); 
                COMMIT;  
        
            ELSIF idcount > 0 AND TNOTE IS NOT NULL THEN
    
                UPDATE T_SURVEYORLIST SET CASEID=TCASEID,
                PNCCASEID=TPNCCASEID,
                SURVEYTYPE=TSRVTYPE,
                LOSSTYPE=TLOSSTYPE,
                SURVEYOR_NAME=TSRVNAME,
                SURVEYDATE=TSRVDATE,
                LOCATION_SURVEY=TSRVLOC,
                OBJECT_NAME=TOBJNAME,
                LOCATION_OBJECT=TOBJLOC,
                IDOBJECT=TOBJID,
                INDEX_SURVEY=TSRVINDEX,
                TGLPERAWATAN=TDOL,
                NAMA_PASIEN=TPANSIEN,
                STS_SURVEY=TSTS,
                KETERANGAN=TNOTE,
                STSAKSEP=TAKSEP
                WHERE CASEID=TCASEID AND index_survey=TSRVINDEX;
                COMMIT;
    
            END IF;
            
         --this for update data realtime PIC 
           
         select count(1) into listsur from POOLDATA.T_CLAIM_SURVEY_DATAPEGA where surveyid=TCASEID and  pzinskey=TPNCCASEID; --and surveytype=TSRVTYPE;
         select a.USERTEKNIS_1 into picklaimst FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK a where pzinskey= TPNCCASEID;
         
         select count(1) into jumlahsurvey FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK s
                      WHERE pxobjclass = 'ASM-FW-GCNMFW-Work-SurveyClaim' and pzinskey= TCASEID and pzinskey=TPNCCASEID;
         
         if listsur < 1 and jumlahsurvey > 0 then
            insert into POOLDATA.T_CLAIM_SURVEY_DATAPEGA(SURVEYID,PZINSKEY,PICTEKNIK,SURVEYTYPE,STATUSWORK,CREATEDATE)
            select pzinskey,caseid_1,S.USERTEKNIS_1,s.SURVEYORTYPE_1,S.PYSTATUSWORK,S.PXCREATEDATETIME FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK s
                      WHERE pxobjclass = 'ASM-FW-GCNMFW-Work-SurveyClaim' and pzinskey= TCASEID and pzinskey=TPNCCASEID;
            
            IF TSRVTYPE='2' then
                update POOLDATA.T_CLAIM_SURVEY_KLAIM a set a.lossadjuster=
                (select to_number(b.lossadjuster)+1 from T_CLAIM_SURVEY_KLAIM b where b.userteknik=picklaimst) where a.USERTEKNIK=picklaimst;
                
            elsIF TSRVTYPE='1' then
                update POOLDATA.T_CLAIM_SURVEY_KLAIM a set A.INTERNALSURVEY=
                (select to_number(B.INTERNALSURVEY)+1 from T_CLAIM_SURVEY_KLAIM b where b.userteknik=picklaimst) where a.USERTEKNIK=picklaimst;
            end if;          
            COMMIT;
         end if;
         
                 
    
EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error INSERT SURVEY : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/