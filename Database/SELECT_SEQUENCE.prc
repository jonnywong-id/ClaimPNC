CREATE OR REPLACE PROCEDURE          select_sequence(p_seq in varchar2)
is
    lastNumber number;
    maxVal number;
begin
    select last_number,max_value into lastNumber, maxVal from user_sequences where sequence_name = p_seq;

    if (lastNumber > maxVal) then
        reset_sequence(p_seq,maxVal);
    end if;
end;

/