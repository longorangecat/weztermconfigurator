package state

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// EmbedPrefix marks the line in wezterm.lua containing the serialized state.
const EmbedPrefix = "-- [state:v1:"

// EmbedLine generates the single-line comment embedding the base64-encoded,
// gzip-compressed JSON of s into wezterm.lua.
func (s *State) EmbedLine() (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("marshaling state for embed: %w", err)
	}
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(b); err != nil {
		return "", fmt.Errorf("compressing state for embed: %w", err)
	}
	if err := gw.Close(); err != nil {
		return "", fmt.Errorf("closing gzip writer: %w", err)
	}
	enc := base64.StdEncoding.EncodeToString(buf.Bytes())
	return fmt.Sprintf("%s%s]", EmbedPrefix, enc), nil
}

// Extract parses an embedded State from the content of a wezterm.lua file.
// Returns ErrNoEmbeddedState if no embed line is present.
var ErrNoEmbeddedState = errors.New("no embedded state found in lua config")

func Extract(luaContent string) (*State, error) {
	scanner := bufio.NewScanner(strings.NewReader(luaContent))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, EmbedPrefix) && strings.HasSuffix(line, "]") {
			payload := line[len(EmbedPrefix) : len(line)-1]
			data, err := base64.StdEncoding.DecodeString(payload)
			if err != nil {
				return nil, fmt.Errorf("decoding embedded state base64: %w", err)
			}
			gr, err := gzip.NewReader(bytes.NewReader(data))
			if err != nil {
				return nil, fmt.Errorf("uncompressing embedded state: %w", err)
			}
			defer gr.Close()
			dec, err := io.ReadAll(gr)
			if err != nil {
				return nil, fmt.Errorf("reading uncompressed state: %w", err)
			}
			var s State
			if err := json.Unmarshal(dec, &s); err != nil {
				return nil, fmt.Errorf("unmarshaling embedded state JSON: %w", err)
			}
			if s.Values == nil {
				s.Values = map[string]any{}
			}
			if s.Raw == nil {
				s.Raw = map[string]string{}
			}
			if s.Version == 0 {
				s.Version = Version
			}
			if s.TargetOS == "" {
				s.TargetOS = RuntimeTarget()
			}
			return &s, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, ErrNoEmbeddedState
}

// LoadOrRecover loads the state from jsonPath if present and valid. If missing,
// corrupt, or from a newer schema version, it attempts to recover by extracting
// the embedded state from luaPath (wezterm.lua). If recovery succeeds, the recovered
// state is persisted to jsonPath. If both fail or neither exists, it returns a fresh
// state for the runtime target.
func LoadOrRecover(jsonPath, luaPath string) (*State, error) {
	s, err := Load(jsonPath)
	// If json file exists and loaded cleanly with valid version, use it
	if err == nil && fileExists(jsonPath) && s.Version <= Version {
		return s, nil
	}

	// If jsonPath is missing, corrupt, or newer, try recovering from luaPath
	if luaBytes, readErr := os.ReadFile(luaPath); readErr == nil {
		if recovered, recErr := Extract(string(luaBytes)); recErr == nil && recovered.Version <= Version {
			_ = recovered.Save(jsonPath)
			return recovered, nil
		}
	}

	// If jsonPath simply didn't exist and lua didn't have embedded state, fresh state
	if !fileExists(jsonPath) || errors.Is(err, fs.ErrNotExist) {
		return New(RuntimeTarget()), nil
	}

	if err != nil {
		return nil, fmt.Errorf("%s: %w", jsonPath, err)
	}
	return s, nil
}
