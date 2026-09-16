CREATE OR REPLACE PROCEDURE          INSERT_SALVAGE(tCLAIMID in varchar2, tTGLINPUT in date, tTGLTRF in date, 
                                                                tJENIS in varchar2, tQUANTITY in number, tNILAI in number, tLOKASI in varchar2, 
                                                                tSTATUS in varchar2, tNOTE in varchar2, tTGLAKSEP in date,tAKSEP in varchar2,
                                                                tPIC in varchar2, tID in varchar2, tIDOBJ in varchar2, tOBJNAME in varchar2,
                                                                tIDCVG in varchar2, tCVGNAME in varchar2,tcurrency in varchar2, tNILAIAKSEP in number,tEmail in varchar2,
                                                                tPemenangName in varchar2,tTGLLelang in date,tnilaipenawaran in number,tsurveyname in varchar2,tsurveytelp in varchar2,tsurveyemail in varchar2,tTGLterimaa in date,
                                                                tisjabodatabek in varchar2,ErrMsg OUT VARCHAR2)
AS
idcount number;
jum_data number;
idsalvage number;

BEGIN
   select count(1) into jum_data from PNC_salvage where noklaim = tCLAIMID and TRUNC(TGLINPUT,'mi') = TRUNC (SYSDATE,'mi');
    
    
    IF tID is null and jum_data = 0 THEN         
        
        select max(to_number(IDSALVAGE)) into idcount from POOLDATA.PNC_SALVAGE;
        
        idsalvage := idcount+1;
      
                INSERT INTO POOLDATA.PNC_SALVAGE (NOKLAIM, TGLINPUT, TGLTRANSFERGA, JENISSALVAGE, QUANTITYSALVAGE, ESTIMASINILAI, LOKASISALVAGE, 
                                                  STSTRANSFER, REMARK,TGLAKSEPTASI,NOAKSEPTASI,PIC,IDSALVAGE,IDOBJECT,OBJECTNAME,IDCOVERAGE,COVERAGENAME,
                                                  CURRENCY,EMAIL,PEMENANGNAME,TANGGALLELANG,NILAIPENAWARAN,PICSURVEY,EMAILSURVEY,NOTELP,TANGGALTERIMA,ISJABODATABEK)
                VALUES (tCLAIMID, tTGLINPUT, tTGLTRF, tJENIS, tQUANTITY, tNILAI, tLOKASI,tSTATUS, tNOTE,tTGLAKSEP,tAKSEP,tPIC,idsalvage,
                        tIDOBJ, tOBJNAME, tIDCVG, tCVGNAME,tcurrency,tEmail,tPemenangName,tTGLLelang,tnilaipenawaran,tsurveyname,tsurveytelp,tsurveyemail,tTGLterimaa,tisjabodatabek); 
                ErrMsg := idsalvage;
                COMMIT;     
    
    ELSIF tID is not null THEN
    
        BEGIN
        
        select count(1) into idcount from POOLDATA.PNC_SALVAGE where IDSALVAGE=tID;
        
        EXCEPTION WHEN NO_DATA_FOUND THEN  
                idcount := 0;  
                
        END;
        
          
        IF idcount > 0 THEN
                
            UPDATE POOLDATA.PNC_SALVAGE SET NOKLAIM=tCLAIMID, TGLINPUT=tTGLINPUT, TGLTRANSFERGA=tTGLTRF, JENISSALVAGE=tJENIS, 
                QUANTITYSALVAGE=tQUANTITY, ESTIMASINILAI=tNILAI, LOKASISALVAGE=tLOKASI, 
                STSTRANSFER=tSTATUS, REMARK=tNOTE, TGLAKSEPTASI=tTGLAKSEP, NOAKSEPTASI=tAKSEP, IDSALVAGE=idsalvage,
                IDOBJECT=tIDOBJ, OBJECTNAME=tOBJNAME, IDCOVERAGE=tIDCVG, COVERAGENAME=tCVGNAME,CURRENCY = tcurrency, NILAIAKSEP=tNILAIAKSEP,ISJABODATABEK=tisjabodatabek,
                EMAIL=tEmail,PEMENANGNAME=tPemenangName,TANGGALLELANG=tTGLLelang,NILAIPENAWARAN=tnilaipenawaran
            WHERE IDSALVAGE=tID;   
            
            ErrMsg := tID;     
            COMMIT;    

        END IF;

    END IF;    
    
EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error INSERT SALVAGE : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/