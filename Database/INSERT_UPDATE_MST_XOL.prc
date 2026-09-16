CREATE OR REPLACE PROCEDURE          INSERT_UPDATE_MST_XOL(TTIPE IN VARCHAR2, IDMST IN VARCHAR2,TTAHUN IN VARCHAR2,TNAMAMST IN VARCHAR2, TKURS IN DECIMAL, TBISNIS IN VARCHAR2, 
                                                          TIDBISNIS IN VARCHAR2, TIDLAYER IN VARCHAR2, TLAYER IN VARCHAR2, TLIMIT IN DECIMAL, TEXCESS IN DECIMAL,
                                                          TREAS IN VARCHAR2, TIDREAS IN VARCHAR2, TSHARE IN DECIMAL, TCONVERT IN DECIMAL,TTypes IN DECIMAL, ERRMSG OUT VARCHAR2,IDMST2 OUT VARCHAR2)
AS
idcount number;
idcount2 number;
id_site M_SITE_DATABASE.ID%type;
id_mst_xol MST_XOL_PNC.ID%type;

BEGIN

    IF TTIPE = 'master' then

        IF IDMST = 'UnknownID' THEN

            SELECT max(to_number(id)) INTO idcount from MST_XOL_PNC ;
            
                IF idcount is null OR idcount = 0 THEN
                            idcount2 := '10001';
                ELSE
                            idcount2 := idcount + 1;
                END IF;
            
            BEGIN
                INSERT INTO POOLDATA.MST_XOL_PNC(ID,NAMA,TAHUN,KURSVALUE,TYPEXOL) VALUES(idcount2,TNAMAMST,TTAHUN,TKURS,TTypes);
                ERRMSG := 'Data Sudah Disimpan dengan ID : ' || idcount2 ;
                IDMST2 := idcount2;
                COMMIT;
            EXCEPTION
            WHEN OTHERS THEN
                  ERRMSG := 'INSERT MASTER XOL Error : ' || sqlerrm || idcount2;
                  ROLLBACK;
                  RETURN;
            END;
            
        ELSE
            
                BEGIN
                    UPDATE POOLDATA.MST_XOL_PNC SET NAMA=TNAMAMST, TAHUN=TTAHUN, KURSVALUE=TKURS WHERE ID = IDMST;
                    ERRMSG := 'Data Sudah Diupdate dengan ID : ' || IDMST; 
                EXCEPTION
                WHEN OTHERS THEN
                      ERRMSG := 'UPDATE MASTER XOL Error : ' || sqlerrm;
                      ROLLBACK;
                      RETURN;
                END; 
                
        END IF;       
       
    ELSIF TTIPE = 'bisnis' then
        
        SELECT count(1) INTO idcount from MST_XOL_BUSINESS WHERE ID=IDMST AND IDBUSINESS=TIDBISNIS;
        
        IF idcount = 0 THEN
        
             BEGIN
                        INSERT INTO POOLDATA.MST_XOL_BUSINESS (ID,GROUPBUSINESS,IDBUSINESS) VALUES(IDMST,TBISNIS, TIDBISNIS);
                        ERRMSG := 'Data Sudah Diupdate dengan ID : ' || IDMST;
             EXCEPTION
             WHEN OTHERS THEN
                          ERRMSG := 'UPDATE MASTER XOL Error : ' || sqlerrm;
                          ROLLBACK;
                          RETURN;
             END;

        END IF;     
    
    
    ELSIF TTIPE = 'layer' then
    
        IF TIDLAYER = 'UnknownID' THEN

            SELECT max(to_number(idlayer)) INTO idcount from MST_XOL_LAYER ;
            
                IF idcount is null OR idcount = 0 THEN
                            idcount2 := '10001';
                ELSE
                            idcount2 := idcount + 1;
                END IF;
                
            BEGIN
                    INSERT INTO POOLDATA.MST_XOL_LAYER(ID,IDLAYER,NAMA,LIMIT,EXCESS,CONVERT_LIMIT) VALUES (IDMST,idcount2,TLAYER,TLIMIT,TEXCESS,TCONVERT);
                    ERRMSG := 'Data Sudah Disimpan dengan ID : ' || idcount2 ;
                    IDMST2 := idcount2;
                    COMMIT;
            EXCEPTION
            WHEN OTHERS THEN
                          ERRMSG := 'INSERT MASTER XOL Error : ' || sqlerrm || idcount2;
                          ROLLBACK;
                          RETURN;
            END;
                    
                    
        ELSE
                    
           BEGIN
                UPDATE POOLDATA.MST_XOL_LAYER SET NAMA=TLAYER, LIMIT=TLIMIT, EXCESS=TEXCESS,CONVERT_LIMIT=TCONVERT WHERE IDLAYER = TIDLAYER;
                ERRMSG := 'Data Sudah Diupdate dengan ID : ' || TIDLAYER;
           EXCEPTION
           WHEN OTHERS THEN
                ERRMSG := 'UPDATE MASTER XOL Error : ' || sqlerrm;
                ROLLBACK;
                RETURN;
           END;              
                        
        END IF; 

    ELSIF TTIPE = 'reas' then
        
        SELECT count(1) INTO idcount from MST_XOL_REAS WHERE IDLAYER=TIDLAYER AND IDREAS=TIDREAS;
        
        IF idcount > 0 THEN
        
             BEGIN
                        UPDATE POOLDATA.MST_XOL_REAS SET PERCENTSHARE=TSHARE WHERE IDLAYER=TIDLAYER AND IDREAS=TIDREAS;
                        ERRMSG := 'Data Sudah Diupdate dengan ID : ' || TIDREAS;
             EXCEPTION
             WHEN OTHERS THEN
                          ERRMSG := 'UPDATE MASTER XOL Error : ' || sqlerrm;
                          ROLLBACK;
                          RETURN;
             END;
        ELSE 
             BEGIN
                        INSERT INTO MST_XOL_REAS (IDLAYER, NAMA, IDREAS, PERCENTSHARE) VALUES (TIDLAYER, TREAS, TIDREAS, TSHARE);
                        ERRMSG := 'Data Sudah Diupdate dengan ID : ' || TIDREAS;
             EXCEPTION
             WHEN OTHERS THEN
                          ERRMSG := 'UPDATE MASTER XOL Error : ' || sqlerrm;
                          ROLLBACK;
                          RETURN;
             END;
        END IF;     
        
    END IF;

EXCEPTION
        WHEN OTHERS THEN
        ERRMSG := 'Proc Condition MASTER XOL Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/