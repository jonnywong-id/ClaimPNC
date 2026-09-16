CREATE OR REPLACE Procedure          InsertMasterRejectedKomite (tIDMASTER in number,tmasternote in varchar2,ErrMsg out varchar2)as 

count_data number;
BEGIN
    
    select count(1) into count_data from POOLDATA.MST_REJECTED_KOMITE;
    
    if count_data=0 then
        count_data:=111;
    else 
        select max(IDMASTER)+1 into count_data from POOLDATA.MST_REJECTED_KOMITE;
    end if;    

    if tIDMASTER is null then
       INSERT INTO  POOLDATA.MST_REJECTED_KOMITE a(A.IDMASTER,A.NOTEMASTER)
       values(count_data,tmasternote);
        commit;
    ELSE
        UPDATE POOLDATA.MST_REJECTED_KOMITE set NOTEMASTER=tmasternote where IDMASTER=tIDMASTER;
        commit;
    end if;
    

EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error Insert Data Master : ' || sqlerrm;
        ROLLBACK;
        RETURN;

END;

/