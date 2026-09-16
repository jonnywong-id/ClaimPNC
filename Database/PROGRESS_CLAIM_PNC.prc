CREATE OR REPLACE PROCEDURE          PROGRESS_CLAIM_PNC(pidcase in varchar2, prog1 in varchar2, prog2 in varchar2, pnote in varchar2, puser in varchar2, ptglfollow in DATE, pstsprog in varchar2,
                                                                                                pposisi in varchar2, pstsposisi in varchar2, pcatgory in varchar2, pclaimno in varchar2,pJSONstsprog2 in CLOB, pdoctype in varchar2,
                                                                                                idposisi out varchar2,idprogress out varchar2, ErrMsg OUT VARCHAR2)
AS
    tmptglfollowup timestamp;
BEGIN
    IF ptglfollow IS NULL THEN
        tmptglfollowup:=(sysdate+7);
    else
        tmptglfollowup:=ptglfollow;
    END IF;
    IF pcatgory = 'INSERT' THEN
        select nvl(max (ID),0)+1 into idposisi from pooldata.GCNM_PROGRESS_POSISI_PNC WHERE CLAIMNO=pclaimno;
        
        BEGIN
            INSERT INTO POOLDATA.GCNM_PROGRESS_POSISI_PNC (ID, CLAIMNO, CASEID, STATUSPOSISI, PROGRESSDATE, POSISI)
            VALUES (idposisi, pclaimno, pidcase, pstsposisi, SYSDATE, pposisi);
        EXCEPTION
            WHEN OTHERS THEN
            ErrMsg := 'INSERT GCNM_PROGRESS_POSISI_PNC Error : ' || sqlerrm;
            ROLLBACK;
            RETURN;
        END;
            
        BEGIN
            INSERT INTO POOLDATA.GCNM_PROGRESS_CLAIM (ID_UPDATE, PNCCASEID, KETERANGAN, STATUS_PROGRESS1, STATUS_PROGRESS2, STATUS, NEXT_FOLLOWUP, USER_INPUT, POSISIID,JSONSTATUS_PROGRESS2)
            select nvl(max (id_update),0)+1, pclaimno, pnote, prog1, prog2, pstsprog, tmptglfollowup, puser, idposisi,
            '{"pxObjClass":"ASM-FW-GCNMFW-Data-ClaimData","ObjectList":[ {' || TO_CLOB (jsonObjectWriter ('BranchName',prog2,','))
           ||TO_CLOB (jsonObjectWriter ('pxObjClass','ASM-FW-GCNMFW-Data-Object',' '))  || '}]}' from pooldata.gcnm_progress_claim WHERE PNCCASEID= pclaimno;
        EXCEPTION
            WHEN OTHERS THEN
            ErrMsg := 'INSERT GCNM_PROGRESS_CLAIM Error : ' || sqlerrm;
            ROLLBACK;
            RETURN;
        END;
    ELSIF pcatgory = 'UPDATE' THEN
        BEGIN
            UPDATE POOLDATA.GCNM_PROGRESS_POSISI_PNC 
            SET STATUSPOSISI = pstsposisi, 
            PROGRESSDATEDONE = SYSDATE
            WHERE CLAIMNO=pclaimno
            AND ID=pidcase;
        EXCEPTION
            WHEN OTHERS THEN
                ErrMsg := 'UPDATE GCNM_PROGRESS_POSISI_PNC Error : ' || sqlerrm;
                ROLLBACK;
                RETURN;
        END;
        
        BEGIN
            select nvl(max (id_update),0)+1 into idprogress from pooldata.GCNM_PROGRESS_CLAIM WHERE PNCCASEID=pclaimno;
            
            INSERT INTO POOLDATA.GCNM_PROGRESS_CLAIM (ID_UPDATE, PNCCASEID, KETERANGAN, STATUS_PROGRESS1, STATUS_PROGRESS2, STATUS, NEXT_FOLLOWUP, USER_INPUT, POSISIID,JSONSTATUS_PROGRESS2)
            select nvl(max (id_update),0)+1, pclaimno, pnote, prog1, prog2, pstsprog, '', puser, pidcase,
            '{"pxObjClass":"ASM-FW-GCNMFW-Data-ClaimData","ObjectList":[ {' || TO_CLOB (jsonObjectWriter ('BranchName',prog2,','))
           ||TO_CLOB (jsonObjectWriter ('pxObjClass','ASM-FW-GCNMFW-Data-Object',' '))  || '}]}' from pooldata.gcnm_progress_claim WHERE PNCCASEID= pclaimno;
        EXCEPTION
            WHEN OTHERS THEN
            ErrMsg := 'INSERT GCNM_PROGRESS_CLAIM Error : ' || sqlerrm;
            ROLLBACK;
            RETURN;
        END;
    ELSIF pcatgory = 'UPDATE_1' THEN
        BEGIN
            UPDATE POOLDATA.GCNM_PROGRESS_POSISI_PNC 
            SET STATUSPOSISI = pstsposisi, 
            PROGRESSDATEDONE = SYSDATE
            WHERE CLAIMNO=pclaimno
            AND ID=pidcase;
        EXCEPTION
            WHEN OTHERS THEN
                ErrMsg := 'UPDATE GCNM_PROGRESS_POSISI_PNC Error : ' || sqlerrm;
                ROLLBACK;
                RETURN;
        END;
        
        BEGIN
            select nvl(max (id_update),0)+1 into idprogress from pooldata.GCNM_PROGRESS_CLAIM WHERE PNCCASEID=pclaimno;
            
            INSERT INTO POOLDATA.GCNM_PROGRESS_CLAIM (ID_UPDATE, PNCCASEID, KETERANGAN, STATUS_PROGRESS1, STATUS_PROGRESS2, STATUS, NEXT_FOLLOWUP, USER_INPUT, POSISIID,JSONSTATUS_PROGRESS2)
            select nvl(max (id_update),0)+1, pclaimno, pnote, prog1, prog2, pstsprog, ptglfollow, puser, 1,
            '{"pxObjClass":"ASM-FW-GCNMFW-Data-ClaimData","ObjectList":[ {' || TO_CLOB (jsonObjectWriter ('BranchName',prog2,','))
           ||TO_CLOB (jsonObjectWriter ('pxObjClass','ASM-FW-GCNMFW-Data-Object',' '))  || '}]}' from pooldata.gcnm_progress_claim WHERE PNCCASEID= pclaimno;
        EXCEPTION
            WHEN OTHERS THEN
            ErrMsg := 'INSERT GCNM_PROGRESS_CLAIM Error : ' || sqlerrm;
            ROLLBACK;
            RETURN;
        END;
    ELSE
        BEGIN
            select nvl(max (id_update),0)+1 into idprogress from pooldata.GCNM_PROGRESS_CLAIM WHERE PNCCASEID=pclaimno;
            
            INSERT INTO POOLDATA.GCNM_PROGRESS_CLAIM (ID_UPDATE, PNCCASEID, KETERANGAN, STATUS_PROGRESS1, STATUS_PROGRESS2, STATUS, NEXT_FOLLOWUP, USER_INPUT, POSISIID, JSONSTATUS_PROGRESS2,DOCTYPE)
            select nvl(max (id_update),0)+1, pclaimno, pnote, prog1, prog2, pstsprog, tmptglfollowup, puser, pidcase,pJSONstsprog2,pdoctype from pooldata.gcnm_progress_claim WHERE PNCCASEID= pclaimno;
        EXCEPTION
            WHEN OTHERS THEN
            ErrMsg := 'INSERT GCNM_PROGRESS_CLAIM Error : ' || sqlerrm;
            ROLLBACK;
            RETURN;
        END;

    END IF;
    
EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error PROGRESS_CLAIM_PNC : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/