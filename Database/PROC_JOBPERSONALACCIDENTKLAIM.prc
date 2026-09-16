CREATE OR REPLACE PROCEDURE          Proc_JobPersonalAccidentKlaim(tFlags in varchar2,tNOS in number,tIDJOB in varchar2,tNAMAOBJECT in varchar2,tCLAIMID in varchar2,tKOMITEID in varchar2,tNOPOLIS in varchar2,tSOBCODE in varchar2,tSOBNAME in varchar2,tFLAGBISNIS in varchar2,
tSTATUSLOD in varchar2,tEMAIL in varchar2,ErrMsg OUT VARCHAR2)

AS
 id_mst number;
 jumlahdata number;
    
BEGIN
    select nvl(max(b.NOS),0)+1 into id_mst from T_CLAIM_JOB_PERSONALACCIDENT b;
    select count(1) into jumlahdata from POOLDATA.T_CLAIM_JOB_PERSONALACCIDENT where KOMITEID=tKOMITEID;
    
    IF tFlags='insert' AND jumlahdata=0 then
        INSERT INTO POOLDATA.T_CLAIM_JOB_PERSONALACCIDENT b (B.NOS,b.IDJOB,b.NAMAOBJECT,B.CLAIMID,B.KOMITEID,B.NOPOLIS,B.SOBCODE,B.SOBNAME,B.FLAGBISNIS,B.STATUSLOD,B.EMAIL)
        VALUES (id_mst,tIDJOB,tNAMAOBJECT,tCLAIMID,tKOMITEID,tNOPOLIS,tSOBCODE,tSOBNAME,tFLAGBISNIS,'0',tEMAIL);
        ErrMsg := 'success';
        commit;
    elsif tFlags='update' then
        update POOLDATA.T_CLAIM_JOB_PERSONALACCIDENT b set b.STATUSLOD=tSTATUSLOD,TANGGALPROSESLOD=sysdate
        where b.NOS=tNOS and B.CLAIMID=tCLAIMID;
        ErrMsg :='success';
        commit;
    end if;
    
    
EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error PROGRESS_CLAIM_PNC : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/