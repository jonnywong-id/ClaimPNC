CREATE OR REPLACE PROCEDURE          SET_ATTACHMENT_64BIT(tATTACHMIMETYPE IN VARCHAR2, tATTACHNOTE in varchar2, tATTACHNAME in varchar2, tCOMMAND in varchar2,
                                                                                                        tINPUTOPERATOR in varchar2, tIMAGEID in varchar2, tCategory in varchar2, tSubCategory in varchar2, tIDPEGA in varchar2, tDATAID out varchar2, ERRMSG out varchar2)
AS
pkey varchar2(79);
count_ATTACH NUMBER;

BEGIN
    IF tCOMMAND = 'INSERT' THEN
        BEGIN
            SELECT new_uuid INTO pkey FROM dual;
        EXCEPTION
            WHEN no_data_found THEN
            tDATAID := null;
            RETURN;
        END;
        
        select_sequence('ATTACHFILE_SEQ');
        
        count_ATTACH := to_Char(ATTACHFILE_SEQ.nextval);
        
        INSERT INTO C_COUNTER_ATTACHMENT (KEY, YEAR ,RUNNO)
        VALUES (pkey, to_char(sysdate,'yy'), count_ATTACH);
        
        select year ||  lpad(runno,10,'0')
        into tDATAID
        from C_COUNTER_ATTACHMENT
        where key = pkey;
     
        begin
            insert into pooldata.data_attachfile(DATAID,inputdate, INPUTOPERATOR, ATTACHNAME, ATTACHNOTE, ATTACHMIMETYPE, IMAGEID, CATEGORY, SUB_CATEGORY, IDPEGA)
            values(tDATAID,sysdate, tINPUTOPERATOR, tATTACHNAME, tATTACHNOTE, tATTACHMIMETYPE, tIMAGEID, tCategory, tSubCategory, tIDPEGA);
                
        exception when others then
            ERRMSG := 'Error Insert Attachment ' || sqlerrm;
            ROLLBACK;
            RETURN;
        end;   
    ELSE
        begin
            Delete from pooldata.data_attachfile
            where DATAID = tATTACHNAME;
                
        exception when others then
            ERRMSG := 'Error Delete Attachment ' || sqlerrm;
            ROLLBACK;
            RETURN;
        end;   
    END IF;
    
    commit;
EXCEPTION WHEN OTHERS THEN
    ERRMSG := 'Error Procedure SET_ATTACHMENT_64BIT ' || sqlerrm;
    ROLLBACK;
    RETURN;
END;

/