CREATE OR REPLACE PROCEDURE          PEGA_M_SUPPLIER(Datapega IN CLOB, IDPega IN VARCHAR2,ErrMsg OUT VARCHAR2, IDPegaOut OUT VARCHAR2,StsSimpan OUT NUMBER)
AS
id_site M_SITE_DATABASE.ID%type;
id_supplier_ins M_SUPPLIER.ID%type;
id_supplier_out M_SUPPLIER.ID%type;

BEGIN
    
    IF IDPega IS NULL THEN

        BEGIN
                SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
        EXCEPTION
              WHEN OTHERS THEN
              ErrMsg := 'Select Site Database Error : ' || sqlerrm;
              StsSimpan := 0;
              ROLLBACK;
              RETURN;
        END;
        
        id_supplier_ins := id_site || lpad(to_Char(supplier_seq.nextval),11,'0');
        
        BEGIN
            INSERT INTO M_SUPPLIER(ID,JSONDATA) VALUES(id_supplier_ins,replace(DataPega,'UnknownID',id_supplier_ins));
            COMMIT;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'Insert Supplier Error : ' || sqlerrm;
              StsSimpan := 0;
              ROLLBACK;
              RETURN;
        END;
        id_supplier_out := id_supplier_ins;
    ELSE
        BEGIN
            UPDATE M_SUPPLIER SET JSONDATA = DataPega WHERE ID = IDPega;
            COMMIT;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'Update Supplier Error : ' || sqlerrm;
              StsSimpan := 0;
              ROLLBACK;
              RETURN;
        END; 
        id_supplier_out := IDPega;
    END IF;
    ErrMsg := 'Data Sudah Di Simpan dengan ID : ' || id_supplier_out;
    IDPegaOut := id_supplier_out;
    StsSimpan := 1;

EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Simpan Supplier Error : ' || sqlerrm;
        StsSimpan := 0;
        ROLLBACK;
        RETURN;
END;

/