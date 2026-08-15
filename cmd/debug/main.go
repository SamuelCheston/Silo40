package main

import (
	"fmt"
	"silo40/internal/engine"
	"silo40/internal/model"
)

func main() {
	// 模拟开局初始化
	silo := engine.CreateInitialSilo("Debug Silo", 122, []string{"leak"})
	
	fmt.Printf("Initial Factions Count: %d\n", len(silo.Factions))
	for _, f := range silo.Factions {
		fmt.Printf("Faction: %s (Sig: %s, Members: %d, Influence: %.2f, Public: %v)\n", 
			f.Name, f.Signature, f.MemberCount, f.Influence, f.IsPublic)
		
		// 打印成员组成
		for profName, count := range f.TagStats {
			if len(profName) > 5 && profName[:5] == "prof:" {
				fmt.Printf("  - %s: %d\n", profName, count)
			}
		}
	}
	
	fmt.Println("\nCohorts Analysis:")
	for _, c := range silo.Cohorts {
		if c.FactionID != nil {
			fID := *c.FactionID
			var fName string
			for _, f := range silo.Factions {
				if f.ID == fID {
					fName = f.Name
					break
				}
			}
			if fName != "Unaffiliated" {
				fmt.Printf("Cohort %d (%s): Faction=%s, Ideology=%v, Influence=%.2f\n", 
					c.ID, c.Name, fName, c.IdeologyProfile, c.Influence)
			}
		}
	}
}
