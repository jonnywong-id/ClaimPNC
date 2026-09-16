CREATE OR REPLACE PROCEDURE          InsertDataKlaimCabang (
   tNOKLAIM             IN     VARCHAR2,
   tOBJECTID                IN     VARCHAR2,
   tOBJECTNAME             IN     VARCHAR2,
   tTGL_TRANSFER_PUSAT            IN     DATE,
   tSTATUS          IN     VARCHAR2,
   tKODECABANG            IN     VARCHAR2,
   tUSERNAME            IN     VARCHAR2,
   tRCV_ID               IN     VARCHAR2,
   tGROUPPANEL             IN     VARCHAR2,
   tNOPOLIS                 IN     VARCHAR2,
   tEXPEDISI in varchar2,
   tNORESI in varchar2,
   tTANGGALKIRIMRESI in DATE,
   tTGLESTIMASIRESI in DATE,
   ErrMsg                     OUT VARCHAR2)
AS
   counts_id   NUMBER;
BEGIN
   SELECT COUNT (1)
     INTO counts_id
     FROM POOLDATA.T_CLAIM_DATACABANG
    WHERE NOKLAIM = tNOKLAIM;

   IF counts_id = 0 
   THEN
      INSERT INTO POOLDATA.T_CLAIM_DATACABANG(NOKLAIM,
                                                OBJECTID,
                                                OBJECTNAME,
                                                TGL_TRANSFER_PUSAT,
                                                STATUS,
                                                KODECABANG,
                                                USERNAME,
                                                RCV_ID,
                                                GROUPPANEL,
                                                NOPOLIS,
                                                EXPEDISI,
                                                NORESI,
                                                TANGGALKIRIMRESI,
                                                TGLESTIMASIRESI)
           VALUES (tNOKLAIM,
                   tOBJECTID,
                   tOBJECTNAME,
                   sysdate,
                   tSTATUS,
                   tKODECABANG,
                   tUSERNAME,
                   tRCV_ID,
                   tGROUPPANEL,
                   tNOPOLIS,
                   tEXPEDISI,
                   tNORESI,
                   tTANGGALKIRIMRESI,
                   tTGLESTIMASIRESI);

      ErrMsg := 'Sukses Insert Data';
      COMMIT;
   ELSE
         UPDATE POOLDATA.T_CLAIM_DATACABANG
            SET NOKLAIM = tNOKLAIM, OBJECTID = tOBJECTID,OBJECTNAME=tOBJECTNAME, TGL_TRANSFER_PUSAT=sysdate,STATUS=tSTATUS,KODECABANG=tKODECABANG,
            USERNAME=tUSERNAME,RCV_ID=tRCV_ID,GROUPPANEL=tGROUPPANEL,NOPOLIS=tNOPOLIS,
            EXPEDISI=tEXPEDISI,
            NORESI=tNORESI,
            TANGGALKIRIMRESI=tTANGGALKIRIMRESI,
            TGLESTIMASIRESI=tTGLESTIMASIRESI
          WHERE NOKLAIM = tNOKLAIM;

         ErrMsg := 'Sukses Update Data';
         COMMIT;
   END IF;
EXCEPTION
   WHEN OTHERS
   THEN
      ErrMsg := 'Insert Data Klaim Cabang';
      RETURN;
END;

/