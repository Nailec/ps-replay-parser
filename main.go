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
	if len(args) != 5 {
		fmt.Println("go run main.go file/url filename/urladdress format stats/teams")
		return
	}

	format := args[3]

	var urls []string
	var err error

	switch args[1] {
	case "url":
		urls, err = GetURLsFromForumsPage(args[2], format)
		if err != nil {
			fmt.Println(err)
			return
		}
	case "file":
		urls, err = GetURLsFromFile(args[2], format)
		if err != nil {
			fmt.Println(err)
			return
		}
	default:
		fmt.Println("go run main.go file/url filename/urladdress")
		return
	}

	switch args[4] {
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
				output += strings.Join(p.Moves, ";") + ";"
			}
		}
	}
	output += team.Result
	fmt.Println(output)
}
