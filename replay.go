package theoryloop

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ErrPending is returned by FileCaller when a prompt has no reply yet.
var ErrPending = errors.New("reply pending")

// FileCaller answers model calls from files, so the llm backend can be driven
// by anything that can read a file and write one (a person, another process,
// a Claude subagent) rather than by the Messages API.
//
// Each call is identified by a hash of its model, system and user text. If
// Dir/replies/<id>.txt exists it is the reply. Otherwise the prompt is written
// to Dir/pending/<id>.txt and the call fails with ErrPending. Run the benchmark
// again after supplying replies; identical prompts (for example the proposer's,
// which every condition shares) are answered once.
type FileCaller struct {
	Dir string
	mu  sync.Mutex
}

// PromptID is the file identifier for a prompt.
func PromptID(system, user, model string) string {
	sum := sha256.Sum256([]byte(model + "\x00" + system + "\x00" + user))
	return hex.EncodeToString(sum[:8])
}

// Call implements Caller.
func (f *FileCaller) Call(system, user, model string, maxTokens int) (string, error) {
	id := PromptID(system, user, model)
	text := "SYSTEM PROMPT:\n" + system + "\n\nUSER MESSAGE:\n" + user + "\n"
	if b, err := os.ReadFile(filepath.Join(f.Dir, "replies", id+".txt")); err == nil && len(b) > 0 {
		// keep the prompt beside its reply so a finished run is a complete record
		f.mu.Lock()
		defer f.mu.Unlock()
		pdir := filepath.Join(f.Dir, "prompts")
		if err := os.MkdirAll(pdir, 0o755); err == nil {
			os.WriteFile(filepath.Join(pdir, id+".txt"), []byte(text), 0o644)
		}
		return string(b), nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	dir := filepath.Join(f.Dir, "pending")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, id+".txt"), []byte(text), 0o644); err != nil {
		return "", err
	}
	return "", fmt.Errorf("%w: %s", ErrPending, id)
}
