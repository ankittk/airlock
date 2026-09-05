package judge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xdlc-labs/airlock/internal/evalcase"
	"github.com/xdlc-labs/airlock/internal/manifest"
	"github.com/xdlc-labs/airlock/internal/providers"
)

// Spec is a pinned judge declared in .airlock/judges/<id>.json or judges.yml entry.
type Spec struct {
	ID           string             `json:"id"`
	Provider     string             `json:"provider"`
	Model        string             `json:"model"`
	Prompt       string             `json:"prompt"`
	Rubric       string             `json:"rubric"`
	Turns        []evalcase.Message `json:"turns,omitempty"` // multi-turn judge template
	PassContains string             `json:"pass_contains"`   // default "PASS"
	KappaFloor   float64            `json:"kappa_floor"`     // default 0.4
	PromptHash   string             `json:"prompt_hash"`
}

func (s Spec) Pin() string {
	return manifest.HashString(s.Provider + "|" + s.Model + "|" + s.Prompt + "|" + s.Rubric)
}

type CalibrationItem struct {
	Input  string `json:"input"`
	Output string `json:"output"`
	Label  bool   `json:"label"` // human
}

type CalibrationReport struct {
	Kappa   float64 `json:"kappa"`
	N       int     `json:"n"`
	Agree   int     `json:"agree"`
	FloorOK bool    `json:"floor_ok"`
}

// Registry loads judges from a directory of JSON files.
type Registry struct {
	ByID   map[string]Spec
	Client providers.HTTPDoer
}

func LoadDir(dir string) (*Registry, error) {
	r := &Registry{ByID: map[string]Spec{}}
	return r, r.mergeDir(dir)
}

// LoadDirs merges judge specs from dirs (later dirs override earlier on same id).
func LoadDirs(dirs ...string) (*Registry, error) {
	r := &Registry{ByID: map[string]Spec{}}
	for _, d := range dirs {
		if d == "" {
			continue
		}
		if err := r.mergeDir(d); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Registry) mergeDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || strings.Contains(e.Name(), ".calibration.") {
			continue
		}
		if strings.HasSuffix(e.Name(), ".calibration.json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		var s Spec
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		if s.ID == "" {
			s.ID = strings.TrimSuffix(e.Name(), ".json")
		}
		if s.PassContains == "" {
			s.PassContains = "PASS"
		}
		if s.KappaFloor <= 0 {
			s.KappaFloor = 0.4
		}
		if s.PromptHash == "" {
			s.PromptHash = s.Pin()
		}
		r.ByID[s.ID] = s
	}
	return nil
}

func (r *Registry) Score(ctx context.Context, id string, c evalcase.Case, resp providers.Response) (bool, error) {
	s, ok := r.ByID[id]
	if !ok {
		return false, fmt.Errorf("unknown judge %q", id)
	}
	// mock judge: pass if response text contains PASS or is valid JSON when rubric says json
	if s.Provider == "mock" || s.Provider == "" {
		text := resp.Text
		if strings.Contains(strings.ToUpper(text), strings.ToUpper(s.PassContains)) {
			return true, nil
		}
		if s.Rubric == "json_valid" {
			return json.Valid([]byte(text)), nil
		}
		// default mock: accept non-empty
		return strings.TrimSpace(text) != "", nil
	}
	p, err := providers.Resolve(s.Provider, r.Client)
	if err != nil {
		return false, err
	}
	msgs := judgeMessages(s, c, resp)
	out, err := p.Generate(ctx, providers.Request{
		Provider: s.Provider,
		Model:    s.Model,
		Messages: msgs,
	})
	if err != nil {
		return false, err
	}
	return strings.Contains(strings.ToUpper(out.Text), strings.ToUpper(s.PassContains)), nil
}

func judgeMessages(s Spec, c evalcase.Case, resp providers.Response) []providers.Message {
	if len(s.Turns) > 0 {
		out := make([]providers.Message, 0, len(s.Turns)+2)
		for _, t := range s.Turns {
			out = append(out, providers.Message{
				Role:    t.Role,
				Content: expandJudgeTemplate(t.Content, c, resp, s),
			})
		}
		return out
	}
	prompt := s.Prompt
	if prompt == "" {
		prompt = "Score the assistant output. Reply PASS or FAIL only.\nRubric: " + s.Rubric
	}
	user := fmt.Sprintf("Input: %s\nOutput: %s\n%s", lastUser(c), resp.Text, prompt)
	return []providers.Message{{Role: "user", Content: user}}
}

func expandJudgeTemplate(text string, c evalcase.Case, resp providers.Response, s Spec) string {
	text = strings.ReplaceAll(text, "{{input}}", lastUser(c))
	text = strings.ReplaceAll(text, "{{output}}", resp.Text)
	text = strings.ReplaceAll(text, "{{rubric}}", s.Rubric)
	return text
}

func lastUser(c evalcase.Case) string {
	for i := len(c.Input.Messages) - 1; i >= 0; i-- {
		if c.Input.Messages[i].Role == "user" {
			return c.Input.Messages[i].Content
		}
	}
	return ""
}

// Calibrate computes Cohen's kappa vs human labels using mock/live scorer.
func (r *Registry) Calibrate(ctx context.Context, id string, items []CalibrationItem) (CalibrationReport, error) {
	s, ok := r.ByID[id]
	if !ok {
		return CalibrationReport{}, fmt.Errorf("unknown judge %q", id)
	}
	var bothTrue, bothFalse, humanOnly, judgeOnly int
	for _, it := range items {
		c := evalcase.Case{Input: evalcase.Input{Messages: []evalcase.Message{{Role: "user", Content: it.Input}}}}
		resp := providers.Response{Text: it.Output}
		j, err := r.Score(ctx, id, c, resp)
		if err != nil {
			return CalibrationReport{}, err
		}
		switch {
		case it.Label && j:
			bothTrue++
		case !it.Label && !j:
			bothFalse++
		case it.Label && !j:
			humanOnly++
		default:
			judgeOnly++
		}
	}
	n := bothTrue + bothFalse + humanOnly + judgeOnly
	rep := CalibrationReport{N: n, Agree: bothTrue + bothFalse}
	if n == 0 {
		return rep, nil
	}
	po := float64(rep.Agree) / float64(n)
	// pe for binary kappa
	ph := float64(bothTrue+humanOnly) / float64(n)
	pj := float64(bothTrue+judgeOnly) / float64(n)
	pe := ph*pj + (1-ph)*(1-pj)
	if pe == 1 {
		rep.Kappa = 1
	} else {
		rep.Kappa = (po - pe) / (1 - pe)
	}
	rep.FloorOK = rep.Kappa >= s.KappaFloor
	return rep, nil
}

// AttributionRescore scores fixed reference with two judge pins; returns pass-rate delta (new-old).
func AttributionDelta(ctx context.Context, oldJ, newJ *Registry, oldID, newID string, refs []CalibrationItem) (float64, error) {
	if len(refs) == 0 {
		return 0, nil
	}
	oldOK, newOK := 0, 0
	for _, it := range refs {
		c := evalcase.Case{Input: evalcase.Input{Messages: []evalcase.Message{{Role: "user", Content: it.Input}}}}
		resp := providers.Response{Text: it.Output}
		o, err := oldJ.Score(ctx, oldID, c, resp)
		if err != nil {
			return 0, err
		}
		n, err := newJ.Score(ctx, newID, c, resp)
		if err != nil {
			return 0, err
		}
		if o {
			oldOK++
		}
		if n {
			newOK++
		}
	}
	return float64(newOK-oldOK) / float64(len(refs)), nil
}

// LoadLabels reads human-labeled calibration JSONL.
func LoadLabels(path string) ([]CalibrationItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []CalibrationItem
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var it CalibrationItem
		if err := json.Unmarshal([]byte(line), &it); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, nil
}

// WriteCalibration writes report next to judge specs.
func WriteCalibration(dir, id string, rep CalibrationReport) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, id+".calibration.json")
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// LoadCalibration reads a previously written calibration report.
func LoadCalibration(dir, id string) (CalibrationReport, error) {
	data, err := os.ReadFile(filepath.Join(dir, id+".calibration.json"))
	if err != nil {
		return CalibrationReport{}, err
	}
	var rep CalibrationReport
	return rep, json.Unmarshal(data, &rep)
}
