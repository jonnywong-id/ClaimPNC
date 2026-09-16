CREATE OR REPLACE PROCEDURE          Pega_D_Pasal_Master(IDdatap varchar2, IDPasal_p varchar2, Datapega clob, ErrMess OUT varchar2)
AS
JUM_PASAL number;

BEGIN
    select count(*) INTO JUM_PASAL from POOLDATA.V_M_DATA_PASAL where IDDATA = IDdatap;
    
    IF JUM_PASAL = 0 then
        BEGIN
            
            select max(TO_NUMBER(IDDATA)) INTO JUM_PASAL from V_M_DATA_PASAL;
            JUM_PASAL := JUM_PASAL+1;
            
            INSERT INTO POOLDATA.V_M_DATA_PASAL(IDDATA,IDPASAL,JSONPASAL) VALUES(to_char(JUM_PASAL),IDPasal_p,Datapega);
            ErrMess := 'INSERT DATA PASAL : '|| IDPasal_p || ' BERHASIL';
        EXCEPTION
            WHEN OTHERS THEN
            ErrMess := 'Data Tidak Dapat Diinsert Error : ' || sqlerrm;
            ROLLBACK;
            RETURN;
        END;
    
    ELSE
        BEGIN
            UPDATE POOLDATA.V_M_DATA_PASAL SET IDPASAL = IDPasal_p, JSONPASAL = Datapega WHERE IDDATA = IDdatap;
            ErrMess := 'UPDATE DATA PASAL : '|| IDPasal_p || ' BERHASIL';
        EXCEPTION
            WHEN OTHERS THEN
            ErrMess := 'Data Tidak Dapat Diupdate Error : ' || sqlerrm;
            ROLLBACK;
            RETURN;
        END;
        
    
    END IF;

EXCEPTION
    WHEN OTHERS THEN
        ErrMess := 'Proc Condition PEGA_D_CAUSE_OF_LOSS Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/