CREATE OR REPLACE FUNCTION          GETSELISIHJAM (tglakhir   IN DATE,
                                              tglawal    IN DATE)
   RETURN NUMBER
AS
   tgl   DATE;
   cnt   NUMBER;
   wkt   NUMBER DEFAULT 0;
BEGIN
   SELECT weekends2 (TRUNC (tglawal), TRUNC (tglakhir)) INTO cnt FROM DUAL;

   wkt := wkt + POOLDATA.datediff ('SS', tglawal, tglakhir);

   IF cnt > 0
   THEN
      wkt := wkt - (86400 * cnt);
   END IF;

   RETURN ROUND ( (wkt / 3600), 2);
EXCEPTION
   WHEN OTHERS
   THEN
      RETURN 0;
END getselisihjam;

/