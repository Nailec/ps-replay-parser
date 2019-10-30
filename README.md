# ps-replay-parser
A tool to parse replays of pokemon battles !

This programs takes the following parameters : 
 * address # the location of the file containing the replay links
 * format # the format of the battles (useful to filter out a gen in a tour for example)
 * output_type # if replays, list the replays found matching the request. If teams returns a csv of the teams with the format below

examples on how to run the program : <br>
go run *.go ~/Bureau/lcuu_replays gen7lcuu teams

teams output format : <br>
player_name;lead;pokemon1;item;move1;move2;move3;move4;pokemon2;(...);pokemon6;item;move1;move2;move3;move4;result # result is W or L
