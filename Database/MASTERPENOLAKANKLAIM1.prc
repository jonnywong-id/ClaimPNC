CREATE OR REPLACE PROCEDURE          MasterPenolakanKlaim1(tid_st in varchar2, tnotest in varchar2,ErrMsg OUT VARCHAR2)

AS
    id_mst varchar2(2000);
BEGIN
    select nvl(max(to_number(ID_ST)),0)+1 into id_mst from MST_PENOLAKAN_KLAIM_1;

    IF tid_st is null or tid_st='' then
        INSERT INTO POOLDATA.MST_PENOLAKAN_KLAIM_1 (ID_ST,NOTE_ST)
        VALUES (id_mst,tnotest);
        ErrMsg := id_mst;
        commit;
    elsif tid_st is not null or tid_st!='' then
        INSERT INTO POOLDATA.MST_PENOLAKAN_KLAIM_1 (ID_ST,NOTE_ST)
        VALUES (tid_st,tnotest);
        ErrMsg :=tid_st;
        commit;
    end if;


EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error PROGRESS_CLAIM_PNC : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/