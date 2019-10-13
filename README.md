# ps-replay-parser
A tool to parse replays of pokemon battles !

This programs takes the following parameters : 
 * input_type # Whether it parses replays from a file or an url (like a tournament replay thread)
 * address # the location of the said file or the url
 * format # the format of the battles (useful to filter out a gen in a pl for example)
 * output_type # if replays, list the replays found matching the request. If teams returns a csv of the teams with the format below

examples on how to run the program : <br>
go run *.go url https://www.smogon.com/forums/threads/zupl-replays.3653635/\#post-8237446 gen7zu teams <br>
go run *.go file ~/Bureau/lcuu_replays gen7lcuu teams > ~/Bureau/lcuu_teams2

teams output format : <br>
player_name;pokemon1;pokemon2;pokemon3;pokemon4;pokemon5;pokemon6;result # result is W or L
