package service

import (
	"fmt"
	"math/rand"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// assignedStationNumber uses a separate deterministic stream so assignment
// never consumes gameplay RNG. Silos 1, 18, and 40 are excluded by construction.
func assignedStationNumber(seed int64) int {
	return rand.New(rand.NewSource(seed^0x40a17)).Intn(16) + 2
}

func createAssignedStation(seed int64, startYear int, traitIDs []string) *model.Silo {
	return engine.CreateInitialSilo(fmt.Sprintf("%d 号地堡", assignedStationNumber(seed)), startYear, traitIDs)
}

const initialPlayerFragmentsPerProfession = 2

// limitPlayerKnowledge gives the player a real information gap while leaving
// AI operators' existing knowledge model unchanged.
func limitPlayerKnowledge(agent *model.Agent) {
	if agent == nil {
		return
	}
	prefix := agent.Profession + "_"
	fragments := make([]string, 0, initialPlayerFragmentsPerProfession)
	for _, fragment := range model.ALL_FRAGMENTS {
		if len(fragment) > len(prefix) && fragment[:len(prefix)] == prefix {
			fragments = append(fragments, fragment)
			if len(fragments) == initialPlayerFragmentsPerProfession {
				break
			}
		}
	}
	agent.KnownFragments = fragments
}

func validateStationAssignment(world *model.World, silos []model.Silo, agent *model.Agent) error {
	if agent == nil || agent.StationSiloID == 0 {
		return fmt.Errorf("v6 save has no assigned station")
	}
	if agent.SiloID != agent.StationSiloID || world.PlayerSiloID != agent.StationSiloID {
		return fmt.Errorf("player, agent, and station IDs do not identify the same silo")
	}
	for i := range silos {
		if silos[i].Name == "Silo 40" || silos[i].Name == "40 号地堡" {
			return fmt.Errorf("silo 40 must not exist as a simulation entity")
		}
		if silos[i].ID != agent.StationSiloID {
			continue
		}
		if residentSiloHasTrait(&silos[i], silo1FullTrait) || residentSiloHasTrait(&silos[i], silo18CanonTrait) {
			return fmt.Errorf("station assignment points to protected silo %d", silos[i].ID)
		}
		return nil
	}
	return fmt.Errorf("station silo %d is missing", agent.StationSiloID)
}

// stationSilo returns the ordinary silo where the Silo 40 operative is posted.
func (s *GameService) stationSilo() *model.Silo {
	if s.agent == nil || s.agent.StationSiloID == 0 {
		return nil
	}
	return s.siloByID(s.agent.StationSiloID)
}

// playerActionSilo preserves legacy single-silo behavior while ensuring every
// new assigned-game action affects the station rather than the Silo 40 sender.
func (s *GameService) playerActionSilo() *model.Silo {
	if station := s.stationSilo(); station != nil {
		return station
	}
	return s.silo
}
