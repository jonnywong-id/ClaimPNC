CREATE OR REPLACE PROCEDURE          InsertDataKomiteList (
   flags                   IN     VARCHAR2,
   tKOMITE_ID               IN     VARCHAR2,
   tNO_KLAIM                IN     VARCHAR2,
   tNAMAKOMITE             IN     VARCHAR2,
   tSTATUSCASE             IN     VARCHAR2,
   tSTATUSAPPROVE          IN     VARCHAR2,
   tNOTEKOMITE             IN     VARCHAR2,
   tGROUP_PANEL            IN     VARCHAR2,
   tPOLICYNO               IN     VARCHAR2,
   tBISNISNAME             IN     VARCHAR2,
   tQQNAME                 IN     VARCHAR2,
   tUSER_TEKNIS            IN     VARCHAR2,
   tSOBNAME                IN     VARCHAR2,
   tBRANCHNAME             IN     VARCHAR2,
   tDATEOFCOMMITE_CREATE   IN     DATE,
   tlistkomite              IN    VARCHAR2,
   tTypeKomite             IN     VARCHAR2,
   tPAYMENTTYPE              IN VARCHAR2,
   tSHAREASM                 IN NUMBER,
   tNILAIKLAIM               IN NUMBER,
   ErrMsg                     OUT VARCHAR2)
AS
   counts_id   NUMBER;
BEGIN
   SELECT COUNT (1)
     INTO counts_id
     FROM POOLDATA.T_CLAIM_KOMITE_LIST
    WHERE KOMITE_ID = tKOMITE_ID AND NAMAKOMITE = tNAMAKOMITE;

   IF counts_id = 0 AND flags != 'update'
   THEN
      INSERT INTO POOLDATA.T_CLAIM_KOMITE_LIST (KOMITE_ID,
                                                NO_KLAIM,
                                                NAMAKOMITE,
                                                STATUSCASE,
                                                STATUSAPPROVE,
                                                NOTEKOMITE,
                                                DATEOFCOMMITE_CREATE,
                                                KOMITEKE,
                                                TYPEKOMITE,
                                                PAYMENTTYPE,SHAREASM,NILAIKLAIM)
           VALUES (tKOMITE_ID,
                   tNO_KLAIM,
                   tNAMAKOMITE,
                   tSTATUSCASE,
                   tSTATUSAPPROVE,
                   tNOTEKOMITE,
                   sysdate,
                   tlistkomite,
                   tTypeKomite,tPAYMENTTYPE,tSHAREASM,tNILAIKLAIM);

      ErrMsg := 'Sukses Insert Data';
      COMMIT;
   ELSIF flags = 'update'
   THEN
      SELECT COUNT (1)
        INTO counts_id
        FROM POOLDATA.T_CLAIM_KOMITE_LIST
       WHERE KOMITE_ID = tKOMITE_ID AND NAMAKOMITE = tNAMAKOMITE;

      IF counts_id > 0
      THEN
         UPDATE POOLDATA.T_CLAIM_KOMITE_LIST
            SET STATUSAPPROVE = tSTATUSAPPROVE, NOTEKOMITE = tNOTEKOMITE,TANGGALKOMITE=tDATEOFCOMMITE_CREATE,TYPEKOMITE=tTypeKomite, STATUSCASE=tSTATUSCASE,PAYMENTTYPE=tPAYMENTTYPE,SHAREASM=tSHAREASM,NILAIKLAIM=tNILAIKLAIM
          WHERE KOMITE_ID = tKOMITE_ID AND NAMAKOMITE = tNAMAKOMITE;

         ErrMsg := 'Sukses Update Data';
         COMMIT;
      END IF;
   END IF;
EXCEPTION
   WHEN OTHERS
   THEN
      ErrMsg := 'Insert Data Komite List';
      RETURN;
END;

/