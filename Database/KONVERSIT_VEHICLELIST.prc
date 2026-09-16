CREATE OR REPLACE PROCEDURE          KONVERSIT_VEHICLELIST (
   p_IDPEGA               IN     VARCHAR2,
   o_message              OUT VARCHAR2)
IS
  errmsg clob;
  countvehiclelist number;
  
CURSOR data_rec
IS
  select idpega,nopolis,prodke,to_clob(data_jsonblob) as JSONDATA from json_polis where idpega = p_idpega or nopolis = p_idpega
    ;
    
    BEGIN
        /* Formatted on 10/11/2020 16:25:00 (QP5 v5.215.12089.38647) */
        FOR rec_data IN data_rec
                 LOOP
                    countvehiclelist := 0;
                    IF (rec_data.idpega = rec_data.nopolis) THEN
                        
                        delete from t_vehiclelist
                        WHERE idpega = rec_data.IDPEGA AND prodke = rec_data.PRODKE;
                        
                        SELECT COUNT (1)
                          INTO countvehiclelist
                          FROM t_vehiclelist
                         WHERE idpega = rec_data.IDPEGA AND prodke = rec_data.PRODKE;
                    END IF;
                    
                    IF (rec_data.idpega <> rec_data.nopolis) THEN
                        SELECT COUNT (1)
                          INTO countvehiclelist
                          FROM t_vehiclelist
                         WHERE idpega = rec_data.IDPEGA;
                    END IF;

                    IF(countvehiclelist<1) THEN
                        insertnewt_vehiclelist(rec_data.IDPEGA,rec_data.NOPOLIS,rec_data.PRODKE,rec_data.JSONDATA,errmsg);
                    END IF;
                 END LOOP;
        o_message := errmsg;
    exception when others then
        o_message := 'Error : ' || SQLERRM || errmsg;
    END;

/