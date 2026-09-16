CREATE OR REPLACE PROCEDURE          MasterPenolakanKlaim2(ID_1 in varchar2,note_1 in varchar2,ID_2 in varchar2,note_2 in varchar2,user_input in varchar2,ErrMsg OUT VARCHAR2)

AS
    id_mst varchar2(2000);
BEGIN
    select nvl(max(to_number(ID_ND)),0)+1 into id_mst from MST_PENOLAKAN_KLAIM_2;

    IF ID_2 is null or ID_2='' then
        INSERT INTO POOLDATA.MST_PENOLAKAN_KLAIM_2 a (A.ID_ST,A.NOTE_ST,A.ID_ND,A.NOTE_ND,A.USER_INPUT,A.TANGGALKIRIM,A.STATUS)
        VALUES (ID_1,note_1,id_mst,note_2,user_input,sysdate,'0');
        ErrMsg := id_mst;
        commit;
    elsif ID_2 is not null or ID_2!='' then
        update POOLDATA.MST_PENOLAKAN_KLAIM_2 a set A.ID_ST=ID_1,A.NOTE_ST=note_1,A.NOTE_ND=note_2,A.USER_INPUT=user_input,A.TANGGALKIRIM=sysdate,A.STATUS='0'
        where A.ID_ND=ID_2;
        ErrMsg :=ID_2;
        commit;
    end if;


EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error PROGRESS_CLAIM_PNC : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/