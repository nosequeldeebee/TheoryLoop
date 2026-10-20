package theoryloop

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultModel is the Claude model the llm backend uses unless told otherwise.
const DefaultModel = "claude-sonnet-5-5"

const apiURL = "https://api.anthropic.com/v1/messages"

// Caller sends one system + user message to a model and returns its text reply.
// Tests swap in a fake; ClaudeCaller is the real one.
type Caller func(system, user, model string, maxTokens int) (string, error)

var httpClient = &http.Client{Timeout: 120 * time.Second}

// ClaudeCaller calls the Claude Messages API using ANTHROPIC_API_KEY. It
// retries rate limits and server errors with exponential backoff, so a long
// parallel benchmark survives transient failures.
func ClaudeCaller(system, user, model string, maxTokens int) (string, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return "", errors.New("set ANTHROPIC_API_KEY to use the llm backend")
	}
	body, err := json.Marshal(map[string]any{
		"model": model, "max_tokens": maxTokens, "system": system,
		"messages": []map[string]string{{"role": "user", "content": user}},
	})
	if err != nil {
		return "", err
	}
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<attempt) * time.Second)
		}
		req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("content-type", "application/json")
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("api status %d: %.200s", resp.StatusCode, string(raw))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("api status %d: %.300s", resp.StatusCode, string(raw))
		}
		var out struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return "", err
		}
		var sb strings.Builder
		for _, c := range out.Content {
			if c.Type == "text" {
				sb.WriteString(c.Text)
			}
		}
		return sb.String(), nil
	}
	return "", lastErr
}

// nameMap returns forward and inverse maps from variable names to what the
// model sees: the names themselves, or x1..x10 when anonymized.
func nameMap(anonymize bool) (fwd, inv map[string]string) {
	fwd, inv = map[string]string{}, map[string]string{}
	for i, p := range Predictors {
		shown := p
		if anonymize {
			shown = fmt.Sprintf("x%d", i+1)
		}
		fwd[p], inv[shown] = shown, p
	}
	return fwd, inv
}

func resolveName(raw string, inv map[string]string) (string, bool) {
	if real, ok := inv[raw]; ok {
		raw = real
	}
	return raw, isPredictor(raw)
}

// parseJSON pulls the JSON array (or object) out of a model reply that may
// wrap it in prose.
func parseJSON(text string) (any, error) {
	for _, br := range [][2]string{{"[", "]"}, {"{", "}"}} {
		i, j := strings.Index(text, br[0]), strings.LastIndex(text, br[1])
		if i >= 0 && j > i {
			var v any
			if json.Unmarshal([]byte(text[i:j+1]), &v) == nil {
				return v, nil
			}
		}
	}
	return nil, fmt.Errorf("no JSON found in reply: %.200s", text)
}

func asObjects(v any) []map[string]any {
	switch t := v.(type) {
	case []any:
		var out []map[string]any
		for _, e := range t {
			if m, ok := e.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case map[string]any:
		return []map[string]any{t}
	}
	return nil
}

func toInt(v any) (int, bool) {
	switch t := v.(type) {
	case float64:
		return int(math.Round(t)), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		return n, err == nil
	}
	return 0, false
}

func toStr(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

// ------------------------------------------------------------- LLM agents

const proposerSystem = "You are an organizational theorist building theory from data on NFT projects. " +
	"The outcome is a 12-month legitimacy score. You see univariate regression results computed by code. " +
	"Propose up to %d propositions that you believe reflect real mechanisms. If the evidence does not " +
	"support a mechanism, propose fewer, or reply with an empty array [] if nothing is supported. " +
	"Reply with a JSON array of " +
	`objects {"predictor": name, "sign": 1 or -1} and nothing after it.`

// LLMProposer asks a model to draft a theory from the evidence table.
type LLMProposer struct {
	Model     string
	K         int
	Anonymize bool
	Call      Caller
}

func (a LLMProposer) Propose(d Data, l *Ledger) ([]Proposition, error) {
	k := a.K
	if k == 0 {
		k = 3
	}
	fwd, inv := nameMap(a.Anonymize)
	var lines []string
	for _, r := range univariateTable(d, l, "proposer") {
		lines = append(lines, fmt.Sprintf("%s: coef/SD=%.3f, p=%.4g", fwd[r.Predictor], r.Coef, r.P))
	}
	reply, err := a.Call(fmt.Sprintf(proposerSystem, k), strings.Join(lines, "\n"), a.Model, 1500)
	if err != nil {
		return nil, fmt.Errorf("proposer: %w", err)
	}
	v, err := parseJSON(reply)
	if err != nil {
		return nil, fmt.Errorf("proposer: %w", err)
	}
	var props []Proposition
	for _, o := range asObjects(v) {
		name, ok := resolveName(toStr(o["predictor"]), inv)
		s, okSign := toInt(o["sign"])
		if !ok || !okSign || (s != 1 && s != -1) {
			continue
		}
		props = append(props, P(name, s))
	}
	props = dedupe(props)
	if len(props) > k {
		props = props[:k]
	}
	l.Log(Entry{Agent: "proposer", Action: "propose", Theory: strs(labels(props)), Raw: reply})
	return props, nil
}

const falsifierSystem = "You are an adversarial reviewer. For each proposition you get diagnostics computed by code: " +
	"the base test, the effect after adding each control, and interaction tests with binary moderators. " +
	"Raise an objection only when the evidence supports it. Kinds: 'fails' (claim does not hold), " +
	"'confound' (name the variable that explains it away and its sign), 'scope' (name the moderator and " +
	`the level at which the effect holds). Reply with a JSON array of {"index": i, "kind": ..., "var": ..., ` +
	`"sign": ..., "value": ...}; an empty array if the theory survives.`

// LLMFalsifier asks a model which objections the diagnostics support.
type LLMFalsifier struct {
	Model     string
	Anonymize bool
	Call      Caller
}

func (a LLMFalsifier) Critique(theory []Proposition, d Data, l *Ledger) ([]Objection, error) {
	fwd, inv := nameMap(a.Anonymize)
	show := func(s string) string { return fwd[s] }
	var blocks []string
	for i, p := range theory {
		dg := diagnose(p, d, l, "falsifier")
		var ctrl, scp []string
		for _, c := range dg.Controls {
			ctrl = append(ctrl, fmt.Sprintf("+%s: coef=%.3f p=%.3g (control p=%.3g, sign=%d)",
				show(c.Control), c.CoefWith, c.PWith, c.ControlP, c.ControlSign))
		}
		for _, s := range dg.Scopes {
			scp = append(scp, fmt.Sprintf("%s: int p=%.3g, coef@0=%.3f, coef@1=%.3f",
				show(s.Moderator), s.InteractionP, s.Coef0, s.Coef1))
		}
		blocks = append(blocks, fmt.Sprintf("[%d] %s\n base: coef=%.3f p=%.3g\n controls: %s\n moderators: %s",
			i, p.LabelWith(show), dg.Base.Coef, dg.Base.P, strings.Join(ctrl, "; "), strings.Join(scp, "; ")))
	}
	reply, err := a.Call(falsifierSystem, strings.Join(blocks, "\n\n"), a.Model, 2000)
	if err != nil {
		return nil, fmt.Errorf("falsifier: %w", err)
	}
	v, err := parseJSON(reply)
	if err != nil {
		return nil, fmt.Errorf("falsifier: %w", err)
	}
	var obs []Objection
	for _, o := range asObjects(v) {
		i, ok := toInt(o["index"])
		kind := toStr(o["kind"])
		if !ok || i < 0 || i >= len(theory) {
			continue
		}
		ob := Objection{Prop: theory[i], Kind: kind}
		switch kind {
		case "fails":
		case "confound", "scope":
			name, ok := resolveName(toStr(o["var"]), inv)
			if !ok {
				continue // an objection that names no real variable cannot be applied
			}
			ob.Var = name
			if kind == "confound" {
				if s, ok := toInt(o["sign"]); ok && s != 0 {
					ob.Sign = s
				} else {
					ob.Sign = 1
				}
			} else {
				ob.Value = 1
				if val, ok := toInt(o["value"]); ok {
					ob.Value = val
				}
			}
		default:
			continue
		}
		obs = append(obs, ob)
	}
	l.Log(Entry{Agent: "falsifier", Action: "critique", Objections: strs(objectionStrings(obs)), Raw: reply})
	return obs, nil
}
