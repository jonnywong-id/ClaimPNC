CREATE OR REPLACE PROCEDURE          UpdateInsert_PengkinianData (
   t_Claimid      IN     VARCHAR2,
   t_Pyid         IN     VARCHAR2,
   t_Noktp        IN     VARCHAR2,
   t_Oldemail     IN     VARCHAR2,
   t_Oldtelp      IN     VARCHAR2,
   t_Newemail     IN     VARCHAR2,
   t_Newtelp      IN     VARCHAR2,
   t_Userinput    IN     VARCHAR2,
   t_pyCategori   IN     VARCHAR2,
   t_dataalamat in clob,
   t_opcid   IN     VARCHAR2,
   ErrMsg          OUT VARCHAR2)
AS
count_data number;
BEGIN
    select count(*) into count_data from POOLDATA.UPDATE_PENGKINIANDATA WHERE PYID = t_Pyid;
   IF count_data = 0
   THEN
      BEGIN
         INSERT INTO POOLDATA.UPDATE_PENGKINIANDATA (CLAIMID,
                                                     PYID,
                                                     OLDEMAIL,
                                                     NOKTP,
                                                     OLDTELP,
                                                     NEWTELP,
                                                     NEWEMAIL,
                                                     TANGGAL_INPUT,
                                                     USERINPUT,DATAALAMAT,OPCID)
              VALUES (t_Claimid,
                      t_Pyid,
                      t_Oldemail,
                      t_Noktp,
                      t_Oldtelp,
                      t_Newtelp,
                      t_Newemail,
                      SYSDATE,
                      t_Userinput,t_dataalamat,t_opcid);
                      ErrMsg := 'SUKSES';
      EXCEPTION
         WHEN OTHERS
         THEN
            ErrMsg := 'INSERT UPDATE_PENGKINIANDATA Error : ' || SQLERRM;
            ROLLBACK;
            RETURN;
      END;
   ELSIF count_data > 0 THEN 
      BEGIN
         UPDATE POOLDATA.UPDATE_PENGKINIANDATA
            SET OLDEMAIL = t_Oldemail,
                NOKTP = t_Noktp,
                OLDTELP = t_Oldtelp,
                NEWTELP = t_Newtelp,
                NEWEMAIL = t_Newemail,
                TANGGAL_INPUT = SYSDATE,
                USERINPUT = t_Userinput,
                DATAALAMAT = t_dataalamat,
                OPCID = t_opcid
          WHERE PYID = t_Pyid;
          ErrMsg := 'SUKSES';
          RETURN;
      EXCEPTION
         WHEN OTHERS
         THEN
            ErrMsg := 'UPDATE UPDATE_PENGKINIANDATA Error : ' || SQLERRM;
            ROLLBACK;
            RETURN;
      END;
   END IF;
EXCEPTION
   WHEN OTHERS
   THEN
      ErrMsg := 'Error UPDATE_PENGKINIANDATA : ' || SQLERRM;
      ROLLBACK;
      RETURN;
END;

/