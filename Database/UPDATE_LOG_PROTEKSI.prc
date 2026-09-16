CREATE OR REPLACE PROCEDURE          Update_Log_Proteksi(
          T_USERINPUT  IN   VARCHAR2,
          T_LOGIN      IN   VARCHAR2,
          T_LOGSEARCH  IN   INT,
          T_LOGSEEN    IN   INT,
          T_MODUL      IN   VARCHAR2,
          T_SUBMODUL   IN   VARCHAR2,
          T_PASSWORD   IN   VARCHAR2,
          T_STSAKTF    IN   VARCHAR2,
          T_STS_EMAIL  IN   VARCHAR2,
          T_STS_KTP    IN   VARCHAR2,
          T_STS_NOTELP IN   VARCHAR2,
          T_CABANG     IN   VARCHAR2,
          T_KETERANGAN IN   VARCHAR2,
          T_ACTION     IN   VARCHAR2,
          MSG          OUT  VARCHAR2)
          
AS
          DATACOUNT   INTEGER;
          jumdata       INTEGER;
          PASSWORDS   VARCHAR2(12000);
          LOGBERHASIL VARCHAR2(12000);
          KETERANGAN_log VARCHAR2(12000);
          ts_cabang varchar2(10000);
          login varchar2(10000);
BEGIN
   
   IF T_ACTION = 'insert' THEN
        SELECT COUNT(*) INTO DATACOUNT 
            FROM POOLDATA.MST_PROTEKSI_DATA_PNC 
            WHERE CABANG = T_CABANG AND LOGIN =T_LOGIN;
            
        SELECT max(to_number(ID_MST))+1 INTO jumdata 
            FROM POOLDATA.MST_PROTEKSI_DATA_PNC;
        
        IF DATACOUNT =0 THEN
            BEGIN
                INSERT INTO POOLDATA.MST_PROTEKSI_DATA_PNC 
                            (CABANG,
                            TANGGALINPUT,
                            USERINPUT,
                            LOGIN,LOGSEARCH,
                            LOGSEEN,
                            PASSWORD,
                            STS_AKTF,
                            STS_EMAIL,
                            STS_KTP,
                            STS_NOTELP,
                            MODUL,
                            SUBMODUL,ID_MST)
                values
                            (T_CABANG,
                            SYSDATE,
                            T_USERINPUT,
                            T_LOGIN,
                            T_LOGSEARCH,
                            T_LOGSEEN,
                            T_PASSWORD,
                            T_STSAKTF,
                            T_STS_EMAIL,
                            T_STS_KTP,
                            T_STS_NOTELP,
                            T_MODUL,
                            T_SUBMODUL,jumdata);
                COMMIT; 
                MSG := '1';
            
            EXCEPTION
                WHEN OTHERS
                THEN
                    MSG := 'ERROR INSERT PROTEKSI DATA. LOGIN '||T_LOGIN;
                ROLLBACK;
                RETURN;
            END;
            
        ELSIF DATACOUNT >0 THEN
            MSG := 'ERROR INSERT PROTEKSI DATA. LOGIN '||T_LOGIN||' SUDAH ADA';
        END IF; 
   ELSIF T_ACTION = 'update' THEN
        SELECT COUNT(*) INTO DATACOUNT 
            FROM POOLDATA.MST_PROTEKSI_DATA_PNC 
            WHERE CABANG = T_CABANG AND LOGIN =T_LOGIN;
        
        IF DATACOUNT =0 THEN
            MSG := 'ERROR INSERT PROTEKSI DATA. LOGIN '||T_LOGIN||' DATA KOSONG';
            
        ELSIF DATACOUNT >0 THEN
            BEGIN
                UPDATE POOLDATA.MST_PROTEKSI_DATA_PNC 
                SET   CABANG=T_CABANG,
                      USERINPUT=T_USERINPUT,
                      LOGIN=T_LOGIN,
                      LOGSEARCH=T_LOGSEARCH,
                      LOGSEEN=T_LOGSEEN,
                      PASSWORD=T_PASSWORD,
                      STS_AKTF=T_STSAKTF,
                      STS_EMAIL=T_STS_EMAIL,
                      STS_KTP=T_STS_KTP,
                      STS_NOTELP=T_STS_NOTELP,
                      MODUL=T_MODUL,
                      SUBMODUL=T_SUBMODUL
                WHERE CABANG=T_CABANG AND LOGIN =T_LOGIN;
                COMMIT;
                MSG := '1';
            EXCEPTION
            WHEN OTHERS THEN
                MSG:= 'ERROR UPDATE PROTEKSI DATA LOGIN= '|| T_LOGIN;
                ROLLBACK;
                RETURN;
            END; 
        END IF; 
   ELSIF T_ACTION ='log' THEN 
        BEGIN
           /* PASSWORDS := 'MaskingPNC';
            dbms_output.put_line('saya');
        
            IF PASSWORDS=T_PASSWORD THEN
                dbms_output.put_line('berhasil');
                LOGBERHASIL :='1';
            ELSIF PASSWORDS!=T_PASSWORD THEN
                dbms_output.put_line('gagal');
                LOGBERHASIL :='1';
            END IF;*/
            
            
            /*if T_STS_EMAIL='1' then
                KETERANGAN_log := 'Keterangan Password : '||LOGBERHASIL;
            elsif T_STS_KTP='1' then
                 KETERANGAN_log := 'Keterangan KTP : '||LOGBERHASIL;
            elsif T_STS_NOTELP='1' then
                 KETERANGAN_log := 'Keterangan Telephone : '||LOGBERHASIL;
            end if;*/
            select CABANG into ts_cabang from POOLDATA.MST_PROTEKSI_DATA_PNC where LOGIN=T_LOGIN fetch next 1 row only;
            
            INSERT INTO POOLDATA.LOG_DATA_PROTEKSI_KLAIM
                        (CABANG,
                        KETERANGAN,
                        LOGIN,
                        LOGSEARCH,
                        LOGSEEN,
                        MODUL,
                        STS_EMAIL,
                        STS_KTP,
                        STS_NOTELP,
                        SUB_MODUL,
                        TANGGALINPUT,
                        USERINPUT)
            VALUES (ts_cabang,
                    T_KETERANGAN,
                    T_LOGIN,
                    T_LOGSEARCH,
                    T_LOGSEEN,
                    T_MODUL,
                    T_STS_EMAIL,
                    T_STS_KTP,
                    T_STS_NOTELP,
                    T_SUBMODUL,
                    SYSDATE,
                    T_USERINPUT);
            
            
            MSG := '1';
            COMMIT;
            
        EXCEPTION
            WHEN OTHERS THEN
                MSG:= 'ERROR  '|| SQLERRM;
                ROLLBACK;
                RETURN;
        END;
   END IF;
   EXCEPTION
     WHEN NO_DATA_FOUND THEN
       NULL;
     WHEN OTHERS THEN
       -- Consider logging the error and then re-raise
       RAISE;
END Update_Log_Proteksi;

/