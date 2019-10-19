package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
)

func GetURLsFromFile(file, format string) ([]string, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	fileContent := string(b)

	lines := strings.Split(fileContent, "\n")
	urls := make([]string, len(lines))
	i := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "replay") {
			line = strings.Replace(line, "replay", "https://replay", 1)
		}
		line = strings.Replace(line, "http://", "https://", 1)
		if !strings.HasPrefix(line, "https://replay") {
			continue
		}

		urls[i] = line
		i++
	}

	return urls[:i], nil
}

func GetURLsFromForumsPage(url, format string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	sel := doc.Find("a")
	urls := make([]string, sel.Length())
	j := 0
	sel.Each(func(_ int, s *goquery.Selection) {
		ref, _ := s.Attr("href")
		if strings.HasPrefix(ref, "replay") {
			ref = strings.Replace(ref, "replay", "https://replay", 1)
		}
		ref = strings.Replace(ref, "http://", "https://", 1)
		if strings.HasPrefix(ref, "https://replay.pokemonshowdown.com/"+format) ||
			strings.HasPrefix(ref, "https://replay.pokemonshowdown.com/smogtours-"+format) {
			urls[j] = ref
			j++
		}
	})

	return urls[:j], nil
}

type Team struct {
	Pokemons map[string]*Pokemon // Key is Nickname
	Lead     string
	Result   string
	Player   string
}

type Pokemon struct {
	Name  string
	Moves []string
}

func GetTeams(urls []string) ([]*Team, error) {
	allTeams := make([]*Team, 2*len(urls))
	teams := make(map[string]*Team, 2)
	i := 0
	var err error
	for _, url := range urls {
		teams, err = ParsePokemonsFromURL(url)
		if err != nil {
			return nil, err
		}

		for _, team := range teams {
			allTeams[i] = team
			i++
		}
	}

	return allTeams, nil
}

func GetStats(urls []string) (map[string]int, error) {
	stats := map[string]int{}
	teams := make(map[string]*Team, 2)
	var err error
	var teamType string
	for _, url := range urls {
		teams, err = ParsePokemonsFromURL(url)
		if err != nil {
			return nil, err
		}

		for _, team := range teams {
			teamType, err = GetType(team.Pokemons)
			if err != nil {
				return nil, err
			}

			for _, pokemon := range team.Pokemons {
				stats[pokemon.Name+"\t"+teamType]++
			}
		}
	}

	return stats, nil
}

type pokeList struct {
	Data []string `json:"data"`
}

func GetType(team map[string]*Pokemon) (string, error) {
	types := []string{"bug", "dark", "dragon", "electric",
		"fairy", "fighting", "fire", "flying", "ghost",
		"grass", "ground", "ice", "normal", "poison",
		"psychic", "rock", "steel", "water",
	}

	for _, t := range types {
		b, err := ioutil.ReadFile("pokelist/" + t + ".json")
		if err != nil {
			return "", errors.Wrap(err, "could not read type file: "+t)
		}

		var list pokeList
		err = json.Unmarshal(b, &list)
		if err != nil {
			return "", errors.Wrap(err, "could not unmarshal poke list of type: "+t)
		}

		typeMatch := true
		for _, p := range team {
			found := false
			for _, p2 := range list.Data {
				if p.Name == p2 {
					found = true
					break
				}
			}

			if !found {
				typeMatch = false
			}
		}

		if typeMatch {
			return t, nil
		}
	}

	return "Unknown", fmt.Errorf("no type found for team: %+v", team)
}

func ParsePokemonsFromURL(url string) (map[string]*Team, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("could not access: %s, code: %d",
			url, resp.StatusCode)
	}

	defer resp.Body.Close()
	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if strings.Contains(string(html), "Could not connect") {
		return nil, fmt.Errorf("could not connect to: %s", url)
	}

	return ParsePokemonsFromHtml(string(html))
}

func ParsePokemonsFromHtml(html string) (map[string]*Team, error) {
	teams := map[string]*Team{
		"p1": &Team{
			Result:   "L",
			Pokemons: map[string]*Pokemon{},
		},
		"p2": &Team{
			Result:   "L",
			Pokemons: map[string]*Pokemon{},
		},
	} // The teams to be returned

	playerIDs := map[string]string{} // Stores a player ID by name

	lines := strings.Split(html, "\n")
	for i, line := range lines {
		// Init players
		if strings.HasPrefix(line, "|player|") {
			split := strings.Split(line, "|")
			playerIDs[split[3]] = split[2]
			teams[split[2]].Player = split[3]
			continue
		}

		// Init pokemons
		if strings.HasPrefix(line, "|poke|") {
			split := strings.Split(line, "|")
			p := split[2]
			poke := strings.Split(split[3], ",")[0]

			if poke == "Greninja" {
				nn := GetNickname(html, p, poke)
				if checkAshGreninja(html, p, nn) {
					poke = "Greninja-Ash"
				}
			}
			// Pokemon is initialized with its base name as nickname
			teams[p].Pokemons[poke] = &Pokemon{
				Name:  cutName(poke),
				Moves: make([]string, 4),
			}
			continue
		}

		// Init leads
		if strings.HasPrefix(line, "|start") {
			// TODO fix for silvally and -*
			pID, pokeNick, pokeName := getPoke(lines[i+1])
			pokeName = cutName(pokeName)
			teams[pID].Lead = pokeNick
			updatePlayerPoke(teams[pID].Pokemons, pokeNick, pokeName)
			if pokeNick != pokeName {
				delete(teams[pID].Pokemons, pokeName)
			}

			pID, pokeNick, pokeName = getPoke(lines[i+2])
			pokeName = cutName(pokeName)
			teams[pID].Lead = pokeNick
			updatePlayerPoke(teams[pID].Pokemons, pokeNick, pokeName)
			if pokeNick != pokeName {
				delete(teams[pID].Pokemons, pokeName)
			}

			i += 2
			continue
		}

		// Handle end of battle result
		if strings.HasPrefix(line, "|win") {
			split := strings.Split(line, "|")
			teams[playerIDs[split[2]]].Result = "W"
			break // nothing is interesting after we know who won
		}

		// Update nickname and details on forms (silvally, pumpkaboo, ...)
		if strings.HasPrefix(line, "|switch") {
			pID, pokeNick, pokeName := getPoke(line)
			if _, ok := teams[pID].Pokemons[pokeNick]; ok {
				continue // already checked ^^
			}
			pokeName = cutName(pokeName)
			updatePlayerPoke(teams[pID].Pokemons, pokeNick, pokeName)
			if pokeNick != pokeName {
				delete(teams[pID].Pokemons, pokeName)
			}
			continue
		}

		// Update form detail
		if strings.HasPrefix(line, "|detailschange") {
			pID, pokeNick, pokeName := getDetailschange(line)
			teams[pID].Pokemons[pokeNick].Name = pokeName
			continue
		}

		// TODO get moves
		// TODO get items
	}

	return teams, nil
}

func GetNickname(html, player, poke string) string {
	expected := regexp.MustCompile(`\|switch\|` + player + `a: ([^\|]*)\|` + poke)

	res := expected.FindStringSubmatch(html)
	if len(res) < 2 {
		return poke
	}

	return res[1]
}

func checkAshGreninja(html, player, nn string) bool {
	protean := regexp.MustCompile(`\|-start\|` + player + `a: ` + nn + `\|typechange\|([^\|]*)\|\[from\] Protean`)

	usedMove := regexp.MustCompile(`\|move\|` + player + `a: ` + nn + `\|([^\|]*)\|`)

	return strings.Contains(html, "|detailschange|"+player+"a: "+nn+"|"+"Greninja-Ash") || !protean.MatchString(html) && usedMove.MatchString(html)
}

// returns player, nickname and name
func getPoke(line string) (string, string, string) {
	expected := regexp.MustCompile(`\|switch\|(p(1|2))a: ([^\|]*)\|([^,|]*)`)

	res := expected.FindStringSubmatch(line)
	return res[1], res[3], res[4]
}

// returns player, nickname and new form name
func getDetailschange(line string) (string, string, string) {
	expected := regexp.MustCompile(`\|detailschange\|(p(1|2))a: ([^\|]*)\|([^,|]*)`)

	res := expected.FindStringSubmatch(line)
	return res[1], res[3], res[4]
}

func updatePlayerPoke(pokes map[string]*Pokemon, nick, newName string) {
	matched := false
	for oldName, poke := range pokes {
		if !namesMatch(newName, oldName) {
			continue
		}

		matched = true
		if nick == oldName {
			break
		}
		poke.Name = newName
		pokes[nick] = poke
		delete(pokes, oldName)
	}

	if !matched {
		fmt.Println("WTF is " + newName)
	}
}

func namesMatch(a, b string) bool {
	if a == b {
		return true
	}

	as := strings.Split(a, "-")
	bs := strings.Split(b, "-")
	return as[0] == bs[0]
}

func cutName(name string) string {
	if strings.HasPrefix(name, "Pumpkaboo") {
		return "Pumpkaboo"
	}

	if strings.HasPrefix(name, "Gourgeist") {
		return "Gourgeist"
	}

	if strings.HasPrefix(name, "Sawsbuck") {
		return "Sawsbuck"
	}

	if strings.HasPrefix(name, "Deerling") {
		return "Deerling"
	}

	if strings.HasPrefix(name, "Pikachu") {
		return "Pikachu"
	}

	if strings.HasPrefix(name, "Vivillon") {
		return "Vivillon"
	}

	if strings.HasPrefix(name, "Florges") {
		return "Florges"
	}

	if strings.HasPrefix(name, "Flabebe") {
		return "Flabebe"
	}

	if strings.HasPrefix(name, "Floette") {
		return "Floette"
	}

	if strings.HasPrefix(name, "Furfrou") {
		return "Furfrou"
	}

	if name == "Gastrodon-East" {
		return "Gastrodon"
	}

	if name == "Shellos-East" {
		return "Shellos"
	}

	if name == "Basculin-Blue-Striped" {
		return "Basculin"
	}

	if name == "Keldeo-Resolute" {
		return "Keldeo"
	}

	if strings.HasSuffix(name, "-Totem") {
		return strings.TrimSuffix(name, "-Totem")
	}

	return name
}
