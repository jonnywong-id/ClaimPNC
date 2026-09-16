CREATE OR REPLACE PROCEDURE          INSERTOBJECTBACKUP(
   p_IDPEGA    IN     VARCHAR2,
   P_JSONDATA   IN     CLOB,
   P_GroupPanel IN     VARCHAR2,
   p_Nopolis IN     VARCHAR2,
   p_Prodke IN VARCHAR2,
   o_message   OUT    VARCHAR2)
IS
   OBJECTCOVERAGEDATACLOB     CLOB;
   datajson_JSONPOLIS         JSON_OBJECT_T;
   datajson_ObjectDataArray   JSON_ARRAY_T;

BEGIN
   IF P_JSONDATA is not null then
    
       BEGIN
          datajson_JSONPOLIS := JSON_OBJECT_T.parse (P_JSONDATA);
       EXCEPTION      
          WHEN OTHERS
          THEN
             o_message := 'Error Parse Data JSON : ' || SQLERRM;         
       END;

       BEGIN
        IF P_GroupPanel = '002' OR P_GroupPanel = '005' then 
          datajson_ObjectDataArray := datajson_JSONPOLIS.get_array ('PersonList');
          OBJECTCOVERAGEDATACLOB :=
             '{"PersonList":' || datajson_ObjectDataArray.TO_CLOB () || '}';
        ELSIF  P_GroupPanel = '007' then 
          datajson_ObjectDataArray := datajson_JSONPOLIS.get_array ('VehicleList');
          OBJECTCOVERAGEDATACLOB :=
             '{"VehicleList":' || datajson_ObjectDataArray.TO_CLOB () || '}';
        ELSIF  P_GroupPanel = '003' then 
          datajson_ObjectDataArray := datajson_JSONPOLIS.get_array ('LocationList');
          OBJECTCOVERAGEDATACLOB :=
             '{"LocationList":' || datajson_ObjectDataArray.TO_CLOB () || '}';
        ELSIF  P_GroupPanel = '006' then 
          datajson_ObjectDataArray := datajson_JSONPOLIS.get_array ('PropertyList');
          OBJECTCOVERAGEDATACLOB :=
             '{"PropertyList":' || datajson_ObjectDataArray.TO_CLOB () || '}';
        END IF;
        
        
       EXCEPTION
          WHEN OTHERS
          THEN
             o_message := 'Error Get Data ' || SQLERRM;
       END;

       BEGIN
          INSERT INTO t_object (IDPEGA,
                                       GROUPPANEL,
                                       OBJECTDATA,
                                       ISSUCESS)
               VALUES (p_IDPEGA,
                       P_GroupPanel,
                       OBJECTCOVERAGEDATACLOB,
                       '0');

          
       EXCEPTION
          WHEN OTHERS
          THEN
             o_message := 'Error Insert Into t_object' || SQLERRM;
       END;
       
       IF o_message is null OR o_message = '' THEN
          o_message := 'SUCCESS';
          COMMIT;
       END IF;
          
       BEGIN
            IF o_message = 'SUCCESS' THEN
            
              DELETE FROM t_object WHERE IDPEGA = p_IDPEGA AND ISSUCESS = '1';
              COMMIT;
              UPDATE t_object set ISSUCESS = '1', NOPOLIS= p_Nopolis, prodke = p_Prodke WHERE IDPEGA = p_IDPEGA;
              COMMIT;
              
              
            ELSE
                ROLLBACK;
                COMMIT;
            END IF;
       END;
   ELSE 
        o_message:= 'PEGA JSON STRING IS NULL';
    
   END IF;
END;

/