CREATE OR REPLACE PROCEDURE          PLA_DLA (TAHUN in VARCHAR, KODE in VARCHAR,TIPE in VARCHAR,format OUT VARCHAR, ErrMsg OUT VARCHAR)
AS
pkey varchar2(79);
formatErr varchar2(500);
id_site M_SITE_DATABASE.ID%type;
count_dla varchar2(8);
count_pla varchar2(8);
count_alod varchar2(8);
id_dla varchar(11);
id_pla varchar(11);
id_alod varchar(11);
BEGIN

    BEGIN
        SELECT new_uuid INTO pkey FROM dual;
    EXCEPTION
        WHEN no_data_found THEN
        format := null;
        RETURN;
    END;

    BEGIN
          SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
    EXCEPTION
          WHEN OTHERS THEN
          ErrMsg := 'SELECT PEGA_M_SITE Error : ' || sqlerrm;
          ROLLBACK;
          RETURN;
    END;

    IF TIPE = 'DLA' then
        select_sequence('DLA_SEQ');
            
        count_dla := to_Char(DLA_SEQ.nextval);
            
        INSERT INTO DLA (KEY, ID_DLA,KODE, ID_SITE, TAHUN, COUNT)
        VALUES (pkey, id_dla, KODE, id_site, TAHUN, count_dla);
        
        COMMIT;

        SELECT KODE || TAHUN || id_site || lpad(to_Char(a.count),15,'0')
        into format
        from DLA a where key=pkey;

    ELSIF TIPE = 'PLA' then
        select_sequence('PLA_SEQ');
            
        count_pla := to_Char(PLA_SEQ.nextval);
            
        INSERT INTO PLA (KEY, ID_PLA,KODE, ID_SITE, TAHUN, COUNT)
        VALUES (pkey, id_pla, KODE, id_site, TAHUN, count_pla);

        COMMIT;

        SELECT KODE || TAHUN || id_site || lpad(COUNT,15,'0')
        into format
        from PLA where key=pkey;

    ELSIF TIPE='ALOD' then
        select_sequence('ACCEPTLOD_SEQ');
        
        count_alod := to_Char(AcceptLOD_SEQ.nextval);
            
        INSERT INTO AcceptLOD (KEY, ID_ALOD,KODE, ID_SITE, TAHUN, COUNT)
        VALUES (pkey, id_alod, KODE, id_site, TAHUN, count_alod);
        
        COMMIT;

        SELECT KODE || TAHUN || id_site || lpad(COUNT,15,'0')
        into format
        from AcceptLOD where key=pkey;
        
    ELSIF TIPE='WO' then
        select_sequence('NOWO_HE_SEQ');
        
        count_alod := to_Char(NOWO_HE_SEQ.nextval);
            
        INSERT INTO NOWO_HE (KEY, ID_WO,KODE, ID_SITE, TAHUN, COUNT)
        VALUES (pkey, id_alod, KODE, id_site, TAHUN, count_alod);
        
        COMMIT;

        SELECT KODE || TAHUN || id_site || lpad(COUNT,7,'0')
        into format
        from NOWO_HE where key=pkey;
    ELSIF TIPE='PO' then
        select_sequence('NOPO_HE_SEQ');
        
        count_alod := to_Char(NOPO_HE_SEQ.nextval);
            
        INSERT INTO NOPO_HE (KEY, ID_PO,KODE, ID_SITE, TAHUN, COUNT)
        VALUES (pkey, id_alod, KODE, id_site, TAHUN, count_alod);
        
        COMMIT;

        SELECT KODE || TAHUN || id_site || lpad(COUNT,7,'0')
        into format
        from NOPO_HE where key=pkey;        
    
    ELSIF TIPE = 'PLA_XOL' then
        select_sequence('PLA_XOL_SEQ');
            
        count_pla := to_Char(PLA_XOL_SEQ.nextval);
            
        INSERT INTO PLA_XOL (KEY, ID_PLA_XOL,KODE, ID_SITE, TAHUN, COUNT)
        VALUES (pkey, id_pla, KODE, id_site, TAHUN, count_pla);

        COMMIT;

        SELECT KODE ||'.'|| TAHUN || '.'|| id_site || '.' || lpad(COUNT,7,'0')
        into format
        from PLA_XOL where key=pkey;
    
    ELSIF TIPE = 'DLA_XOL' then
        select_sequence('DLA_XOL_SEQ');
            
        count_dla := to_Char(DLA_XOL_SEQ.nextval);
            
        INSERT INTO DLA_XOL (KEY, ID_DLA_XOL,KODE, ID_SITE, TAHUN, COUNT)
        VALUES (pkey, id_pla, KODE, id_site, TAHUN, count_dla);

        COMMIT;

        SELECT KODE ||'.'|| TAHUN || '.'|| id_site || '.' || lpad(COUNT,7,'0')
        into format
        from DLA_XOL where key=pkey;
    
    END IF;


EXCEPTION WHEN OTHERS THEN
    format := null;
    formatErr := sqlerrm;
    rollback;
    RETURN;
END;

/