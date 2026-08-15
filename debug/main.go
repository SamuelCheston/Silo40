package main

import (
	"fmt"
	"silo40/internal/engine"
	"silo40/internal/model"
)

func main() {
	// Initialize Silo with "leak" trait as in the debug start
	silo := engine.CreateInitialSilo("Debug Silo", 122, []string{"leak"})
	
	fmt.Printf("Total Population: %d\n", silo.TotalPopulation)
	fmt.Printf("Number of Factions: %d\n", len(silo.Factions))
	
	for _, f := range silo.Factions {
		fmt.Printf("Faction: %s (ID: %d, Signature: %s, Members: %d)\n", f.Name, f.ID, f.Signature, f.MemberCount)
		fmt.Printf("  Tags: %v\n", f.Tags)
	}
}
