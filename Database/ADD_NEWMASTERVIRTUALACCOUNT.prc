CREATE OR REPLACE PROCEDURE          add_newmastervirtualaccount (tflags in varchar2,tCLIENTID in varchar2,tCLIENTNAME in varchar2,tSTATUS in varchar2,tMESSAGE in varchar2,tVIRTUALACCOUNTNUMBER in varchar2,
                                                                tEMAILVA in varchar2,ErrMsg OUT varchar2)
AS
idcount number;
vaaccount varchar2(3000);
BEGIN
    
select count(*) into idcount from pooldata.MST_VIRTUAL_ACCOUNT_PNC where upper(CLIENTID)=upper(tCLIENTID) and upper(CLIENTNAME)=upper(tCLIENTNAME);
ErrMsg := '';

if idcount<=0 and tflags='0' then
    INSERT INTO POOLDATA.MST_VIRTUAL_ACCOUNT_PNC(CLIENTID,CLIENTNAME,STATUS,MESSAGE,VIRTUALACCOUNTNUMBER,EMAILVA)
    VALUES (tCLIENTID,tCLIENTNAME,tSTATUS,tMESSAGE,tVIRTUALACCOUNTNUMBER,tEMAILVA);
    commit;

elsif idcount >=1 then
        select VIRTUALACCOUNTNUMBER into vaaccount from pooldata.MST_VIRTUAL_ACCOUNT_PNC where upper(CLIENTID)=upper(tCLIENTID) and upper(CLIENTNAME)=upper(tCLIENTNAME);
        ErrMsg := vaaccount;

end if;
    

EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error INSERT VA : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/