CREATE OR REPLACE PROCEDURE POOLDATA.getData_picTeknik(
                                                        txtBusinessType in varchar2,txtTKI in varchar2,
                                                        loginAplikasi out varchar2, errmsg out varchar2)
as
totalJob INT;
counterQuota INT;
teamGroup varchar2(1);
mcoLeader varchar2(5);
userId varchar2(12);

begin

    if txtBusinessType = 'PA' then
        if txtTKI = '1' then
            begin
               select operator_id, team_group,  counter_quota
               into loginAplikasi, teamGroup, counterQuota
               from mst_user_teknik
               where type_business = 'PA' and OPERATOR_ID = 'DIBADYASANTI'
               order by counter_quota
               OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY;
            exception when others then
                errmsg := 'error pada saat ambil pic teknik mst_user_teknis ' || sqlerrm;
                return;
            end;
        else
            begin
               select operator_id, team_group,  counter_quota
               into loginAplikasi, teamGroup, counterQuota
               from mst_user_teknik
               where type_business = 'PA' and OPERATOR_ID = 'ESTHERSIMBOLON'
               order by counter_quota
               OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY;
            exception when others then
                errmsg := 'error pada saat ambil pic teknik mst_user_teknis ' || sqlerrm;
                return;
            end;
        end if;
        
    else
        begin
           select operator_id, team_group,  counter_quota
               into loginAplikasi, teamGroup, counterQuota
           from mst_user_teknik
           where type_business = 'TRAVEL'
           order by counter_quota
           OFFSET 0 ROWS FETCH NEXT 1 ROWS ONLY;
        exception when others then
            errmsg := 'error pada saat ambil pic teknik mst_user_teknis ' || sqlerrm;
            return ;
        end;
    
    end if;
    
    update mst_user_teknik set counter_quota=counter_quota+1 where operator_id=loginAplikasi;
    
    /*
    if (loginAplikasi is not null or loginAplikasi <> '') then
       begin
               update mst_user_teknis
            set total_job = totalJob+1, counter_quota=counterQuota+1
            where operator_id = loginAplikasi;
       exception when others then
               errmsg := 'error pada saat update mst_user_teknis ' || sqlerrm;
       end;
    end if;
   */
   
   
   COMMIT;
   EXCEPTION
      WHEN OTHERS THEN
         errmsg :=
               'Error '
            || SQLERRM;
    ROLLBACK;
         RETURN;
   END;
/
