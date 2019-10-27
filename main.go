package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

func main() {
	args := os.Args
	if len(args) != 4 {
		fmt.Println("go run main.go filename format stats/teams")
		return
	}

	format := args[2]

	urls, err := GetURLsFromFile(args[1], format)
	if err != nil {
		fmt.Println(err)
		return
	}

	switch args[3] {
	case "stats":
		res, err := GetStats(urls)
		if err != nil {
			fmt.Println(err)
			return
		}

		for name, val := range res {
			fmt.Printf(name + "\t" + strconv.Itoa(val) + "\n")
		}
	case "teams":
		res, err := GetTeams(urls)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, team := range res {
			displayTeam(team)
		}
	}
}

func displayTeam(team *Team) {
	if team == nil || team.Lead == "" {
		return
	}

	pokes := make([]string, len(team.Pokemons))
	i := 0
	for _, poke := range team.Pokemons {
		pokes[i] = poke.Name
		i++
	}

	sort.Strings(pokes)
	output := team.Player + ";"
	output += team.Pokemons[team.Lead].Name + ";"
	//output += team.Lead + ";"
	for _, poke := range pokes {
		for _, p := range team.Pokemons {
			if poke == p.Name {
				output += poke + ";"
				output += p.Item + ";"
				output += strings.Join(p.Moves, ";") + ";"
			}
		}
	}
	output += team.Result
	fmt.Println(output)
}
