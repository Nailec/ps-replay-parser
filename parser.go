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
	Pokemons []string
	Result   string
	Player   string
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
				stats[pokemon+"\t"+teamType]++
			}
		}
	}

	return stats, nil
}

type pokeList struct {
	Data []string `json:"data"`
}

func GetType(team []string) (string, error) {
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
				if p == p2 {
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
	teams := make(map[string]*Team)  // The teams to be returned
	i := map[string]int{}            // Stores the index of the pokemon currently parsed by player
	playerIDs := map[string]string{} // Stores a player ID by name

	expected := regexp.MustCompile(`\|poke\|`)   // regexp to get pokemon names
	endGame := regexp.MustCompile(`\|win\|`)     //regexp to get battle result
	playerID := regexp.MustCompile(`\|player\|`) // rexexp to get players info

	for _, line := range strings.Split(html, "\n") {
		if playerID.MatchString(line) {
			split := strings.Split(line, "|")
			playerIDs[split[3]] = split[2]
			if _, ok := teams[split[2]]; !ok {
				teams[split[2]] = &Team{
					Result:   "L",
					Pokemons: make([]string, 6),
					Player:   split[3],
				}
			}
			continue
		}

		if endGame.MatchString(line) {
			split := strings.Split(line, "|")
			teams[playerIDs[split[2]]].Result = "W"
			break
		}

		if expected.MatchString(line) {
			split := strings.Split(line, "|")
			p := split[2]

			poke := strings.Split(split[3], ",")[0]

			nn := GetNickname(html, p, poke)
			if strings.Contains(html, "|detailschange|"+p+"a: "+nn+"|"+poke+"-Mega") {
				poke = poke + "-Mega"
			}

			if strings.Contains(html, "|detailschange|"+p+"a: "+nn+"|"+poke+"-Y") {
				poke = poke + "-Y"
			}

			if strings.Contains(html, "|detailschange|"+p+"a: "+nn+"|"+poke+"-X") {
				poke = poke + "-X"
			}

			if poke == "Greninja" && CheckAshGreninja(html, p, nn) {
				poke = "Greninja-Ash"
			}

			if poke == "Keldeo-Resolute" {
				poke = "Keldeo"
			}

			if strings.HasPrefix(poke, "Pumpkaboo") {
				poke = "Pumpkaboo"
			}

			if strings.HasPrefix(poke, "Gourgeist") {
				poke = "Gourgeist"
			}

			if strings.HasPrefix(poke, "Sawsbuck") {
				poke = "Sawsbuck"
			}

			if strings.HasPrefix(poke, "Deerling") {
				poke = "Deerling"
			}

			if poke == "Gastrodon-East" {
				poke = "Gastrodon"
			}

			if poke == "Shellos-East" {
				poke = "Shellos"
			}

			if strings.HasSuffix(poke, "-Totem") {
				poke = strings.TrimSuffix(poke, "-Totem")
			}

			teams[p].Pokemons[i[p]] = poke
			i[p]++
		}
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

func CheckAshGreninja(html, player, nn string) bool {
	protean := regexp.MustCompile(`\|-start\|` + player + `a: ` + nn + `\|typechange\|([^\|]*)\|\[from\] Protean`)

	usedMove := regexp.MustCompile(`\|move\|` + player + `a: ` + nn + `\|([^\|]*)\|`)

	return strings.Contains(html, "|detailschange|"+player+"a: "+nn+"|"+"Greninja-Ash") || !protean.MatchString(html) && usedMove.MatchString(html)
}
