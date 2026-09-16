CREATE OR REPLACE PROCEDURE          INSERTT_PERSONLIST (
   p_IDPEGA       IN     VARCHAR2,
   p_NOPOLIS      IN     VARCHAR2,
   p_PRODKE       IN     VARCHAR2,
   p_GROUPPANEL   IN     VARCHAR2,
   p_TAHUN        IN     VARCHAR2,
   p_JSONDATA     IN     CLOB,
   o_message         OUT VARCHAR2)
IS
   /*Person List*/
   INDEXOBJECT               NUMBER;
   Delete_INDEXOBJECT        NUMBER;
   UNAMEDBASESPARTICIPANT    VARCHAR2 (200 BYTE) DEFAULT NULL;
   ASMCCAMOUNT               VARCHAR2 (15 BYTE) DEFAULT NULL;
   PYFULLNAME                VARCHAR2 (500 BYTE) DEFAULT NULL;
   ASMIDCARD                 VARCHAR2 (200 BYTE) DEFAULT NULL;
   ASMDATEOFBIRTH            VARCHAR2 (100 BYTE) DEFAULT NULL;
   ASMPARTICIPANTSTATUS      VARCHAR2 (50 BYTE) DEFAULT NULL;
   ASMGENDER                 VARCHAR2 (30 BYTE) DEFAULT NULL;
   ASMJOBNAME                VARCHAR2 (200 BYTE) DEFAULT NULL;
   ASMJOBDESC                VARCHAR2 (100 BYTE) DEFAULT NULL;
   ASMCLASS                  VARCHAR2 (200 BYTE) DEFAULT NULL;
   ASMCLASSID                VARCHAR2 (100 BYTE) DEFAULT NULL;
   ASMHEIGHT                 VARCHAR2 (15 BYTE) DEFAULT NULL;
   ASMWEIGHT                 VARCHAR2 (15 BYTE) DEFAULT NULL;
   ASMLEFTHANDED             VARCHAR2 (10 BYTE) DEFAULT NULL;
   HEIRTYPE                  VARCHAR2 (10 BYTE) DEFAULT NULL;
   STARTDATETIME             VARCHAR2 (100 BYTE) DEFAULT NULL;
   ENDDATETIME               VARCHAR2 (100 BYTE) DEFAULT NULL;
   EFFECTIVEDATE             VARCHAR2 (100 BYTE) DEFAULT NULL;
   CSVUPLOAD                 VARCHAR2 (10 BYTE) DEFAULT NULL;
   SUMPREMIOBJECTPERSON      VARCHAR2 (500 BYTE) DEFAULT NULL;
   SUMTSIPERSON              VARCHAR2 (500 BYTE) DEFAULT NULL;
   SUMDISCOUNTOBJECTPERSON   VARCHAR2 (500 BYTE) DEFAULT NULL;

   /*PERSON EDM*/
   EDMPREMIUMDIFF            VARCHAR2 (500 BYTE) DEFAULT NULL;
   EDMRATEDIFF               VARCHAR2 (500 BYTE) DEFAULT NULL;
   EDMTSIDIFF                VARCHAR2 (500 BYTE) DEFAULT NULL;
   FLAGOLDDATA               VARCHAR2 (10 BYTE) DEFAULT NULL;
   FLAGDELETE                VARCHAR2 (10 BYTE) DEFAULT NULL;
   FLAGEDITDATA              VARCHAR2 (10 BYTE) DEFAULT NULL;

   /*Heir List*/
   HEIR_PYFULLNAME           VARCHAR2 (1000 BYTE) DEFAULT NULL;
   HEIR_ASMHEIRPERCENTAGE    VARCHAR2 (50 BYTE) DEFAULT NULL;
   HEIR_ASMDATEOFBIRTH       VARCHAR2 (100 BYTE) DEFAULT NULL;
   HEIR_ASMRELATIONNAME      VARCHAR2 (100 BYTE) DEFAULT NULL;
   HEIR_ASMRELATION          VARCHAR2 (10 BYTE) DEFAULT NULL;
   HEIR_ASMGENDER            VARCHAR2 (10 BYTE) DEFAULT NULL;

   datajson_JSONPOLIS        JSON_OBJECT_T DEFAULT NULL;
   PERSONLISTDATA            JSON_OBJECT_T DEFAULT NULL;
   PERSONLIST                JSON_ARRAY_T DEFAULT NULL;
   PERSONLISTBLOB            BLOB;
   COVERAGELISTDATA          JSON_OBJECT_T DEFAULT NULL;
   COVERAGELIST              JSON_ARRAY_T DEFAULT NULL;
   COVERAGELISTBLOB          BLOB;
   HEIRLISTDATA              JSON_OBJECT_T DEFAULT NULL;
   HEIRLIST                  JSON_ARRAY_T DEFAULT NULL;
   HEIRLISTBLOB              BLOB;

   /*Heir EDM*/
   HEIR_FLAGOLDDATA          VARCHAR2 (10 BYTE) DEFAULT NULL;
   HEIR_FLAGDELETE           VARCHAR2 (10 BYTE) DEFAULT NULL;
   HEIR_FLAGEDITDATA         VARCHAR2 (10 BYTE) DEFAULT NULL;

   CNT                       NUMBER DEFAULT NULL;
   vCntPolis                 NUMBER DEFAULT 0;
BEGIN
   o_message := NULL;
   
   SELECT COUNT (1)
     INTO vCntPolis
     FROM pooldata.json_polis
    WHERE idpega = p_IDPEGA
      AND sts_konversi = 1;
      
 IF vCntPolis<=0 THEN

   /*Clear To Renewa Data*/
   BEGIN
      --           DELETE FROM T_PERSONLIST
      --           WHERE
      --           IDPEGA = p_IDPEGA;

      --           DELETE FROM T_COVERAGELIST
      --           WHERE
      --           IDPEGA = p_IDPEGA;
      --
      --           DELETE FROM T_OUTGOLIST
      --           WHERE
      --           IDPEGA = p_IDPEGA;
      --
      --           DELETE FROM T_SPREADINGLIST
      --           WHERE
      --           IDPEGA = p_IDPEGA;

      --           DELETE FROM T_HEIRLIST h
      --           WHERE
      --           h.IDPEGA = p_IDPEGA;

      COMMIT;
   END;

   datajson_JSONPOLIS := JSON_OBJECT_T.parse (p_JSONDATA);

   IF (datajson_JSONPOLIS.get_array ('PersonList') IS NOT NULL)
   THEN
      INDEXOBJECT := 0;

      PERSONLIST := datajson_JSONPOLIS.get_array ('PersonList');

      FOR j IN 0 .. PERSONLIST.get_size - 1
      LOOP
         /*Setting adjust index object */
         SELECT MAX (indexobject)
           INTO CNT
           FROM T_PERSONLIST a
          WHERE (a.IDPEGA = p_IDPEGA);

         /*Untuk data pertama jika index kosong*/
         IF CNT IS NULL
         THEN
            CNT := 0;
         END IF;

         /*Isi data per object*/
         PERSONLISTDATA := JSON_OBJECT_T (PERSONLIST.get (j));

         IF (PERSONLISTDATA.get_string ('IndexObject') IS NULL)
         THEN
            INDEXOBJECT := CNT + 1;
            PERSONLISTDATA.put ('IndexObject', INDEXOBJECT);
         ELSE
            INDEXOBJECT := PERSONLISTDATA.get_number ('IndexObject');
         END IF;

         PERSONLISTDATA.put ('IndexObject', INDEXOBJECT);
         UNAMEDBASESPARTICIPANT :=
            PERSONLISTDATA.get_string ('UnamedBasesParticipant');
         ASMCCAMOUNT := PERSONLISTDATA.get_string ('ASMCCAmount');
         PYFULLNAME := PERSONLISTDATA.get_string ('pyFullName');
         ASMIDCARD := PERSONLISTDATA.get_string ('ASMIDCard');
         ASMDATEOFBIRTH := PERSONLISTDATA.get_string ('ASMDateOfBirth');
         ASMPARTICIPANTSTATUS :=
            PERSONLISTDATA.get_string ('ASMParticipantStatus');
         ASMGENDER := PERSONLISTDATA.get_string ('ASMGender');
         ASMJOBNAME := PERSONLISTDATA.get_string ('ASMJobName');
         ASMJOBDESC := PERSONLISTDATA.get_string ('ASMJobDesc');
         ASMCLASS := PERSONLISTDATA.get_string ('ASMClass');
         ASMCLASSID := PERSONLISTDATA.get_string ('ASMClassID');
         ASMHEIGHT := PERSONLISTDATA.get_string ('ASMHeight');
         ASMWEIGHT := PERSONLISTDATA.get_string ('ASMWeight');
         ASMLEFTHANDED := PERSONLISTDATA.get_string ('ASMLeftHanded');
         HEIRTYPE := PERSONLISTDATA.get_string ('HeirType');
         STARTDATETIME := PERSONLISTDATA.get_string ('StartDateTime');
         ENDDATETIME := PERSONLISTDATA.get_string ('EndDateTime');
         EFFECTIVEDATE := PERSONLISTDATA.get_string ('EffectiveDate');
         CSVUPLOAD := PERSONLISTDATA.get_string ('CSVUpload');
         SUMPREMIOBJECTPERSON :=
            PERSONLISTDATA.get_string ('SumPremiObjectPerson');
         SUMTSIPERSON := PERSONLISTDATA.get_string ('SumTSIPerson');
         SUMDISCOUNTOBJECTPERSON :=
            PERSONLISTDATA.get_string ('SumDiscountObjectPerson');

         /*EDM Property default null*/
         EDMPREMIUMDIFF := PERSONLISTDATA.get_string ('EDMPremiumDiff');
         EDMRATEDIFF := PERSONLISTDATA.get_string ('EDMRateDiff');
         EDMTSIDIFF := PERSONLISTDATA.get_string ('EDMTSIDiff');
         FLAGOLDDATA := PERSONLISTDATA.get_string ('FlagOldData');
         FLAGDELETE := PERSONLISTDATA.get_string ('FlagDelete');
         FLAGEDITDATA := PERSONLISTDATA.get_string ('FlagEditData');


         --                IF(PERSONLISTDATA.get_array('ASMHeir') is not null) then
         --                    HEIRLIST    :=  PERSONLISTDATA.get_array('ASMHeir');
         --                        FOR i IN 0 .. HEIRLIST.get_size - 1 LOOP
         --
         --                             HEIRLISTDATA :=  JSON_OBJECT_T(HEIRLIST.get(i));
         --
         --                             HEIR_PYFULLNAME         := HEIRLISTDATA.get_string('pyFullName');
         --                             HEIR_ASMHEIRPERCENTAGE  := HEIRLISTDATA.get_string('ASMHeirPercentage');
         --                             HEIR_ASMDATEOFBIRTH     := HEIRLISTDATA.get_string('ASMDateOfBirth');
         --                             HEIR_ASMRELATIONNAME    := HEIRLISTDATA.get_string('ASMRelationName');
         --                             HEIR_ASMRELATION        := HEIRLISTDATA.get_string('ASMRelation');
         --                             HEIR_ASMGENDER          := HEIRLISTDATA.get_string('ASMGender');
         --
         --                             /*EDM Heir Prop default null*/
         --                             HEIR_FLAGOLDDATA   := HEIRLISTDATA.get_string('FlagOldData');
         --                             HEIR_FLAGDELETE    := HEIRLISTDATA.get_string('FlagDelete');
         --                             HEIR_FLAGEDITDATA  := HEIRLISTDATA.get_string('FlagEditData');
         --
         --                              BEGIN
         --                                INSERT INTO POOLDATA.T_HEIRLIST ( IDPEGA,
         --                                                                  NOPOLIS,
         --                                                                  PRODKE,
         --                                                                  GROUPPANEL,
         --                                                                  INDEXOBJECT,
         --                                                                  PYFULLNAME,
         --                                                                  ASMHEIRPERCENTAGE,
         --                                                                  ASMDATEOFBIRTH,
         --                                                                  ASMRELATIONNAME,
         --                                                                  ASMRELATION,
         --                                                                  ASMGENDER,
         --                                                                  FLAGOLDDATA,
         --                                                                  FLAGDELETE,
         --                                                                  FLAGEDITDATA,
         --                                                                  OBJECTDATA)
         --                                VALUES( p_IDPEGA,
         --                                        p_NOPOLIS,
         --                                        p_PRODKE,
         --                                        p_GROUPPANEL,
         --                                        INDEXOBJECT,
         --                                        HEIR_PYFULLNAME,
         --                                        HEIR_ASMHEIRPERCENTAGE,
         --                                        HEIR_ASMDATEOFBIRTH,
         --                                        HEIR_ASMRELATIONNAME,
         --                                        HEIR_ASMRELATION,
         --                                        HEIR_ASMGENDER,
         --                                        HEIR_FLAGOLDDATA,
         --                                        HEIR_FLAGDELETE,
         --                                        HEIR_FLAGEDITDATA,
         --                                        HEIRLISTBLOB);
         --                               exception when others then
         --                                    o_message := o_message || 'ERR Gak bisa Insert T_PERSONLIST '||'Peserta Ke : ' || INDEXOBJECT || ' Nama Ahli Waris : ' || HEIR_PYFULLNAME || SQLERRM ;
         --                              END;
         --                commit;
         --
         --                       END LOOP;
         --                    END IF;

         HEIRLISTBLOB := NULL;

         IF (PERSONLISTDATA.get_array ('ASMHeir') IS NOT NULL)
         THEN
            HEIRLIST := PERSONLISTDATA.get_array ('ASMHeir');
            HEIRLISTBLOB :=
               pooldata.clobtoblob (
                  '{"ASMHeir":' || HEIRLIST.TO_CLOB () || '}');
         END IF;

         COVERAGELISTBLOB := NULL;

         IF (PERSONLISTDATA.get_array ('ASMCoverage') IS NOT NULL)
         THEN
            COVERAGELIST := PERSONLISTDATA.get_array ('ASMCoverage');
            COVERAGELISTBLOB :=
               pooldata.clobtoblob (
                  '{"ASMCoverage":' || COVERAGELIST.TO_CLOB () || '}');
         END IF;

         /* Process COVERAGE OUTGO SPREADING*/
         --                begin
         --                    POOLDATA.INSERTT_COVERAGELISTPERSON (p_IDPEGA, p_NOPOLIS, p_PRODKE, INDEXOBJECT,p_GROUPPANEL,p_TAHUN,PERSONLISTDATA,o_message);
         --                exception when others then
         --                    o_message := o_message || 'Tidak Bisa Insert Coverage'||SQLERRM;
         --                END;

         PERSONLISTDATA.remove ('ASMCoverage');
         PERSONLISTDATA.remove ('ASMHeir');

         BEGIN
            PERSONLISTBLOB := pooldata.clobtoblob (PERSONLISTDATA.TO_CLOB ());
         EXCEPTION
            WHEN OTHERS
            THEN
               o_message := o_message || SQLERRM;
         END;

         BEGIN
            Delete_INDEXOBJECT := INDEXOBJECT;

            DELETE FROM T_PERSONLIST a
                  WHERE     a.IDPEGA = p_IDPEGA
                        AND a.INDEXOBJECT = Delete_INDEXOBJECT;

            COMMIT;        /*Jangan hilangin commit nanti data ekhapus semua*/
         END;

         BEGIN
            INSERT INTO POOLDATA.T_PERSONLIST (IDPEGA,
                                               NOPOLIS,
                                               PRODKE,
                                               GROUPPANEL,
                                               INDEXOBJECT,
                                               UNAMEDBASESPARTICIPANT,
                                               ASMCCAMOUNT,
                                               PYFULLNAME,
                                               ASMIDCARD,
                                               ASMDATEOFBIRTH,
                                               ASMPARTICIPANTSTATUS,
                                               ASMGENDER,
                                               ASMJOBNAME,
                                               ASMJOBDESC,
                                               ASMCLASS,
                                               ASMCLASSID,
                                               ASMHEIGHT,
                                               ASMWEIGHT,
                                               ASMLEFTHANDED,
                                               HEIRTYPE,
                                               STARTDATETIME,
                                               ENDDATETIME,
                                               EFFECTIVEDATE,
                                               CSVUPLOAD,
                                               SUMPREMIOBJECTPERSON,
                                               SUMTSIPERSON,
                                               SUMDISCOUNTOBJECTPERSON,
                                               EDMPREMIUMDIFF,
                                               EDMRATEDIFF,
                                               EDMTSIDIFF,
                                               FLAGOLDDATA,
                                               FLAGDELETE,
                                               FLAGEDITDATA,
                                               OBJECTDATA,
                                               COVERAGEDATA,
                                               HEIRDATA)
                 VALUES (p_IDPEGA,
                         p_NOPOLIS,
                         p_PRODKE,
                         p_GROUPPANEL,
                         INDEXOBJECT,
                         UNAMEDBASESPARTICIPANT,
                         ASMCCAMOUNT,
                         PYFULLNAME,
                         ASMIDCARD,
                         ASMDATEOFBIRTH,
                         ASMPARTICIPANTSTATUS,
                         ASMGENDER,
                         ASMJOBNAME,
                         ASMJOBDESC,
                         ASMCLASS,
                         ASMCLASSID,
                         ASMHEIGHT,
                         ASMWEIGHT,
                         ASMLEFTHANDED,
                         HEIRTYPE,
                         STARTDATETIME,
                         ENDDATETIME,
                         EFFECTIVEDATE,
                         CSVUPLOAD,
                         SUMPREMIOBJECTPERSON,
                         SUMTSIPERSON,
                         SUMDISCOUNTOBJECTPERSON,
                         EDMPREMIUMDIFF,
                         EDMRATEDIFF,
                         EDMTSIDIFF,
                         NVL (FLAGOLDDATA, 0),
                         NVL (FLAGDELETE, 0),
                         NVL (FLAGEDITDATA, 0),
                         PERSONLISTBLOB,
                         COVERAGELISTBLOB,
                         HEIRLISTBLOB);

            COMMIT;                                 /*Jangan hilangin commit*/
         EXCEPTION
            WHEN OTHERS
            THEN
               o_message :=
                     o_message
                  || 'ERR Gak bisa Insert T_PERSONLIST '
                  || 'Peserta Ke : '
                  || INDEXOBJECT
                  || ' Nama Peserta : '
                  || PYFULLNAME
                  || SQLERRM;
         END;
      END LOOP;


      BEGIN
         IF p_PRODKE IS NOT NULL OR p_PRODKE != ''
         THEN
            UPDATE T_PERSONLIST
               SET PRODKE = p_PRODKE, NOPOLIS = p_NOPOLIS
             WHERE IDPEGA = p_IDPEGA;

            COMMIT;
         END IF;
      EXCEPTION
         WHEN OTHERS
         THEN
            o_message := o_message || 'Error Update Prodke';
      END;

      COMMIT;
   END IF;

  END IF;

   IF TRIM (o_message) IS NULL OR TRIM (o_message) = ''
   THEN
      o_message := TRIM ('SUCCESS');
   END IF;
EXCEPTION
   WHEN OTHERS
   THEN
      o_message := o_message || 'Error tampung ke local variable ' || SQLERRM;
END;

/