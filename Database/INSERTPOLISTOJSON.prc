CREATE OR REPLACE PROCEDURE          INSERTPOLISTOJSON (
   p_POLICYNO   IN     VARCHAR2,
   o_message       OUT VARCHAR2)
IS
   maxprodke    VARCHAR2 (5);
   kodebisnis   VARCHAR2 (2);
   nospak       VARCHAR2 (12);
   cnt          NUMBER;
   grpBisnis    varchar2(2);
   MspNoSpak    varchar2(12);

   CURSOR data_polis(NoSpak IN VARCHAR) IS
        SELECT mds_prod_ke as prodke, msp_no_spak as nospak, SUBSTR (mds_no_polis, 0, 2) as kodebisnis, mds_no_polis as noPolis
        FROM collection.mst_det_sales@asmd.sinarmas.co.id
        where msp_no_spak = NoSpak
        and id_pega is null;
        --GROUP BY msp_no_spak, mds_no_polis;

BEGIN
        o_message := 'GUNAKAN YANG ADA DI GLADMIN.ASMD ' || sqlerrm;
         return;  
    
 begin
        select distinct trim(msp_no_spak) into MspNoSpak
        from mst_det_sales@asmd.sinarmas.co.id
        where mds_no_polis = p_POLICYNO
        and id_pega is null;
       exception when others then
         o_message := 'Polis sppa tidak ada di MST_SALES ' || sqlerrm;
         return;
       end;

       FOR rec_polis IN data_polis (MspNoSpak)
       Loop

--                begin
--                    SELECT MAX (mds_prod_ke), msp_no_spak, SUBSTR (mds_no_polis, 0, 2)
--                    INTO maxprodke, nospak, kodebisnis
--                    FROM collection.mst_det_sales@asmd.sinarmas.co.id
--                    WHERE mds_no_polis = p_POLICYNO
--                    GROUP BY msp_no_spak, mds_no_polis;
--               exception when others then
--                    o_message := 'Polis tidak ada di MST_DET_SALES ' || sqlerrm;
--                    return;
--               end;

                SELECT COUNT (1)
                INTO cnt
                FROM json_polis
                WHERE nopolis = rec_polis.noPolis
                and prodke = rec_polis.prodke ;

                IF cnt < 1
                THEN

                        select b.lgb_id
                        into grpBisnis
                        from general.lst_business@asmd.sinarmas.co.id a, general.lst_grp_business@asmd.sinarmas.co.id b
                        where a.lgb_id = b.lgb_id
                        and a.lbu_id = rec_polis.kodebisnis;

                        --POOLDATA.JSON_POLIS_PA (nospak, p_POLICYNO,maxprodke,o_message);

                        if grpBisnis in ('01','17') then
                               ---01,17 = FIRE
                               pooldata.json_fire.set_json_fire(rec_polis.nospak, rec_polis.noPolis, rec_polis.prodke, o_message);
                               --return;
                        elsif grpBisnis in ('02','18') then
                                if rec_polis.kodebisnis in ('04') then
                                    -- Marine Cargo
                                    POOLDATA.JSON_POLIS_MARINE_CARGO (rec_polis.nospak, rec_polis.noPolis,rec_polis.prodke, o_message);
                                elsif rec_polis.kodebisnis in ('17') then
                                    -- CIT (bisnis aneka tapi grpBisnis Marine Cargo)
                                    pooldata.json_aneka.Json_AnekaAllRisk (rec_polis.noPolis,rec_polis.nospak,rec_polis.prodke,o_message);
                                end if;    
                        elsif grpBisnis in ('04','13','19','24') then
                                -- 04, 13,19,24 = MBU
                                begin
                                    POOLDATA.JSON_MBU.set_json_mbu (rec_polis.nospak,rec_polis.noPolis,rec_polis.prodke,o_message);
                                     EXCEPTION WHEN OTHERS THEN
                                     o_message := 'otong njepat' || SQLERRM;
                                END;
                                
                        elsif grpBisnis in ('06','20') then
                           --06,20 = PA, TRAVEL
                            if rec_polis.kodebisnis in ('77','SY') then
                                --Travel
                                 POOLDATA.JSON_POLIS_TRAVEL (rec_polis.nospak,rec_polis.noPolis,rec_polis.prodke, o_message);
                            else
                                --PA
                                POOLDATA.JSON_POLIS_PA (rec_polis.nospak, rec_polis.noPolis,rec_polis.prodke,o_message);
                            end if;
                        elsif grpBisnis not in ('08','12','21') then
                            -- else ini aneka, karena grp bisnis nya banyak udah di else aja dan yang tidak termasuk bisnis setan.
                            pooldata.json_aneka.Json_AnekaAllRisk (rec_polis.noPolis,rec_polis.nospak,rec_polis.prodke,o_message);
                        end if;
                end if;
                
                 COMMIT;
       end loop;

--        bisnis health
--        08 = HEALTH
--        12 = SIMAS SEHAT
--        21 = HEALTH SYARIAH

  
EXCEPTION WHEN OTHERS THEN
      o_message := o_message || SQLERRM;
END;

/