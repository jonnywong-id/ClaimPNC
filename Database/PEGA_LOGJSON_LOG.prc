CREATE OR REPLACE PROCEDURE          PEGA_LOGJSON_LOG(No_Klaim IN VARCHAR2, DataPega IN CLOB, 
                                                             ErrMsg OUT VARCHAR2)
AS
id_count INTEGER;
id_count2 INTEGER;

BEGIN

    select count(1) into id_count from POOLDATA.LOG_TABLE_JSON where PZINSKEY=No_Klaim;
    
    if id_count<=0 then
        insert into POOLDATA.LOG_TABLE_JSON(PZINSKEY,PZPVSTREAM,INSERTTIME,USERUPDATE)
        VALUES(No_Klaim,POOLDATA.clobToBlob(DataPega),sysdate,'INSERT');
    else
        select count(1) into id_count2 from POOLDATA.LOG_TABLE_JSON where PZINSKEY=No_Klaim and USERUPDATE='UPDATE';
        if id_count2<=0 then
            insert into POOLDATA.LOG_TABLE_JSON(PZINSKEY,PZPVSTREAM,INSERTTIME,USERUPDATE)
            VALUES(No_Klaim,POOLDATA.clobToBlob(DataPega),sysdate,'UPDATE');
        else 
            UPDATE POOLDATA.LOG_TABLE_JSON SET PZPVSTREAM=POOLDATA.clobToBlob(DataPega),INSERTTIME=sysdate where PZINSKEY=No_Klaim and USERUPDATE='UPDATE';
        end if;
    end if;
        
    
EXCEPTION
    WHEN OTHERS THEN
        ErrMsg := 'LOG PEGA Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/