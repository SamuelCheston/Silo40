package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
	"silo40/internal/repository"
	"silo40/internal/service"
)

// serviceAPI 把 GameService 适配成 GameAPI（等价于 app.go 里 Wails 绑定层
// 的包装；GetEndingNarrative 签名差异在此对齐）。
type serviceAPI struct {
	svc *service.GameService
}

func (a serviceAPI) CreateGame(req model.CreateGameRequest) (*model.GameState, error) {
	return a.svc.CreateGame(req)
}

func (a serviceAPI) GetGameState() (*model.GameState, error) { return a.svc.GetState() }

func (a serviceAPI) GetEventHistory(limit int) (*model.EventHistoryResult, error) {
	return a.svc.GetEventHistory(limit)
}

func (a serviceAPI) PassTime(months int) (*model.TickResult, error) { return a.svc.PassTime(months) }

func (a serviceAPI) ExecuteAction(action model.AgentAction) (*model.ActionOutcome, error) {
	return a.svc.ExecuteAction(action)
}

func (a serviceAPI) GetEndingNarrative() (string, error) { return a.svc.GetEndingNarrative(), nil }

func (a serviceAPI) HasActiveGame() (bool, error) { return a.svc.HasActiveGame(), nil }

func (a serviceAPI) GetProfessionActions(profession string) ([]model.ProfessionActionMeta, error) {
	return engine.GetProfessionActionMeta(profession), nil
}

func (a serviceAPI) GetProfessionAction(id string) (*model.ProfessionActionMeta, error) {
	def := engine.GetProfessionAction(id)
	if def == nil {
		return nil, nil
	}
	return &model.ProfessionActionMeta{
		ID:               def.ID,
		Profession:       def.Profession,
		Label:            def.Label,
		Description:      def.Description,
		APCost:           int(def.APCost),
		TargetType:       string(def.TargetType),
		SuspicionPenalty: def.SuspicionPenalty,
	}, nil
}

// newTestServer 起一个与真实调试面同构的 HTTP 服务（无 GUI 依赖）。
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := repository.InitDB(filepath.Join(t.TempDir(), "api-test.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	svc := service.NewGameService(db)
	srv := NewDebugHTTPServer("127.0.0.1:0", serviceAPI{svc: svc})
	ts := httptest.NewServer(srv.server.Handler)
	t.Cleanup(ts.Close)
	return ts
}

func doPost(t *testing.T, url string, body any, out any) {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("post %s -> HTTP %d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}

func doGet(t *testing.T, url string, out any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %s -> HTTP %d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}

// TestSimContractSeededReplayOverHTTP 以模拟器相同的 HTTP 协议验证确定性：
// 同种子 + 同操作序列（逐月推进）=> 相同的状态轨迹。
func TestSimContractSeededReplayOverHTTP(t *testing.T) {
	play := func() model.Silo {
		ts := newTestServer(t)
		var created model.GameState
		doPost(t, ts.URL+"/debug/game/create", model.CreateGameRequest{
			SiloName:   "SimContract",
			AgentName:  "SimAgent",
			Profession: "IT",
			Seed:       7,
		}, &created)
		for m := 0; m < 12; m++ {
			var tick model.TickResult
			doPost(t, ts.URL+"/debug/game/pass-time", map[string]int{"months": 1}, &tick)
		}
		var final model.GameState
		doGet(t, ts.URL+"/debug/game/state", &final)
		return final.Silo
	}

	a, b := play(), play()
	if a.TotalPopulation != b.TotalPopulation ||
		a.Legitimacy != b.Legitimacy ||
		a.Cohesion != b.Cohesion ||
		a.Rebellion != b.Rebellion ||
		a.CurrentYear != b.CurrentYear || a.CurrentMonth != b.CurrentMonth {
		t.Fatalf("same seed + same ops diverged over HTTP:\n A=%+v\n B=%+v", a, b)
	}
}
