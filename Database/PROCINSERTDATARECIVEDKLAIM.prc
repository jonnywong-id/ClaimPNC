CREATE OR REPLACE PROCEDURE          ProcInsertDataRecivedKlaim(
tCLAIMID in varchar2,
tTANGGALINPUTDOKUMEN in DATE,
tTANGGALTERIMADOKUMEN in varchar2,
tNAMAPELAPOR in varchar2,
tEMAILPENGIRIM in varchar2,
tTLPPENGIRIM in varchar2,
tNAMAKURIRASM in varchar2,
tNAMATERTANGGUNG in varchar2,
tNOPOLIS in varchar2,
tBUSINESSCODE  in varchar2,
tGROUPPANEL in varchar2,
tNOREFERENSI in varchar2,
tEMAILTERTANGGUNG in varchar2,
tLOKASIKEJADIAN in varchar2,
tSIMPENGENDARA in varchar2,
tKRONOLOGIKEJADIAN in varchar2,
tRINCIANKERUSAKAN in varchar2,
tALASANBLMTRANSFER in varchar2,
tSUBJECTEMAIL in varchar2,
tKETERANGANBLMREGIST in varchar2,
tNOKLAIM in varchar2,
tDOL in Date,
tTRANSFERASM in Date,
tREGISTDATE in Date,
tKODECABANG in varchar2,
tUSERINPUT in varchar2,
ErrMsg OUT VARCHAR2)
AS
idcount number;
tmptglfollowup timestamp;
nokaimss varchar2(1000);

BEGIN
        IF tTANGGALINPUTDOKUMEN IS NULL THEN
            tmptglfollowup:=(sysdate);
        else
            tmptglfollowup:=tTANGGALINPUTDOKUMEN;
        END IF;

        select count(1) into idcount from pooldata.T_CLAIM_RECIVEDCLAIM where claimid=tCLAIMID;
        
        if idcount<=0 then
            BEGIN
                insert into pooldata.T_CLAIM_RECIVEDCLAIM(CLAIMID,TANGGALINPUTDOKUMEN,KODECABANG,USERINPUT)
                values(tCLAIMID,tmptglfollowup,tKODECABANG,tUSERINPUT);
            EXCEPTION
                WHEN OTHERS THEN
                ErrMsg := 'INSERT DATA RCV Error : ' || sqlerrm;
                ROLLBACK;
                RETURN;
            END;

        else
            BEGIN
                IF tNOKLAIM IS NOT NULL THEN
                    nokaimss:='ASM-FW-GCNMFW-WORK '|| tNOKLAIM;
                END IF;
            
                update pooldata.T_CLAIM_RECIVEDCLAIM 
                set TANGGALINPUTDOKUMEN = tmptglfollowup,
                TANGGALTERIMADOKUMEN = tTANGGALTERIMADOKUMEN, 
                NAMAPELAPOR =tNAMAPELAPOR,
                EMAILPENGIRIM = tEMAILPENGIRIM,
                TLPPENGIRIM = tTLPPENGIRIM,
                NAMAKURIRASM = tNAMAKURIRASM,
                NAMATERTANGGUNG = tNAMATERTANGGUNG,
                NOPOLIS = tNOPOLIS,
                USERINPUT =tUSERINPUT,
                BUSINESSCODE = tBUSINESSCODE,
                GROUPPANEL = tGROUPPANEL,
                NOREFERENSI = tNOREFERENSI,
                EMAILTERTANGGUNG = tEMAILTERTANGGUNG,
                LOKASIKEJADIAN = tLOKASIKEJADIAN,
                SIMPENGENDARA = tSIMPENGENDARA,
                KRONOLOGIKEJADIAN = tKRONOLOGIKEJADIAN,
                RINCIANKERUSAKAN = tRINCIANKERUSAKAN,
                ALASANBLMTRANSFER = tALASANBLMTRANSFER,
                SUBJECTEMAIL = tSUBJECTEMAIL,
                KETERANGANBLMREGIST = tKETERANGANBLMREGIST,
                NOKLAIM = nokaimss,
                TRANSFERASM = tTRANSFERASM,
                REGISTDATE = tREGISTDATE,
                KODECABANG= tKODECABANG,
                DOL= tDOL where CLAIMID = tCLAIMID;
            EXCEPTION
                WHEN OTHERS THEN
                ErrMsg := 'Update gagal Pada RCV : ' || sqlerrm;
                ROLLBACK;
                RETURN;
            END;
        
            
        end if;


    
EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error Data RCV : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/