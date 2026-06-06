package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xiaowumin-mark/VoxBackend/player"
)

const appConfigFileName = "amll-player-voxbackend.json"

type AppConfig struct {
	ModelPath          string `json:"modelPath"`
	RuntimeLibraryPath string `json:"runtimeLibraryPath,omitempty"`
	AMLLURL            string `json:"amllUrl,omitempty"`
	PluginAddr         string `json:"pluginAddr,omitempty"`
}

func defaultAppConfig() AppConfig {
	modelPath := firstExistingPath(
		filepath.Join(".", "onnx", "umxl_vocals.onnx"),
		filepath.Join(".", "onnx", "UVR-MDX-NET-Inst_HQ_3.onnx"),
		player.DefaultONNXModel,
	)
	if modelPath == "" {
		modelPath = filepath.Join(".", "onnx", "umxl_vocals.onnx")
	}

	return AppConfig{
		ModelPath:          modelPath,
		RuntimeLibraryPath: filepath.Join(".", "onnxruntime", "lib", "onnxruntime.dll"),
		AMLLURL:            "ws://localhost:11444",
		PluginAddr:         ":54199",
	}
}

func loadAppConfig() (AppConfig, string, error) {
	cfg := defaultAppConfig()
	path := appConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg.normalize()
			return cfg, path, nil
		}
		cfg.normalize()
		return cfg, path, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		cfg = defaultAppConfig()
		cfg.normalize()
		return cfg, path, err
	}
	cfg.normalize()
	return cfg, path, nil
}

func saveAppConfig(path string, cfg AppConfig) error {
	cfg.normalize()
	if path == "" {
		path = appConfigPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func appConfigPath() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "VoxBackend", appConfigFileName)
	}
	return filepath.Join(".", appConfigFileName)
}

func (cfg *AppConfig) normalize() {
	def := defaultAppConfig()
	if cfg.ModelPath == "" {
		cfg.ModelPath = def.ModelPath
	}
	if cfg.RuntimeLibraryPath == "" {
		cfg.RuntimeLibraryPath = def.RuntimeLibraryPath
	}
	if cfg.AMLLURL == "" {
		cfg.AMLLURL = def.AMLLURL
	}
	if cfg.PluginAddr == "" {
		cfg.PluginAddr = def.PluginAddr
	}
}

func (cfg AppConfig) validateForStart() error {
	if strings.TrimSpace(cfg.ModelPath) == "" {
		return errors.New("请先选择 ONNX 模型文件")
	}
	if _, err := os.Stat(cfg.ModelPath); err != nil {
		return fmt.Errorf("模型文件不可用: %w", err)
	}
	if strings.TrimSpace(cfg.RuntimeLibraryPath) == "" {
		return errors.New("请先选择 onnxruntime.dll")
	}
	if _, err := os.Stat(cfg.RuntimeLibraryPath); err != nil {
		return fmt.Errorf("ONNX Runtime 不可用: %w", err)
	}
	if _, _, err := resolveConfiguredProfile(cfg.ModelPath); err != nil {
		return err
	}
	return nil
}

func firstExistingPath(paths ...string) string {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

type modelCandidate struct {
	Path    string
	Name    string
	Profile string
	Reason  string
}

func discoverONNXModels() []modelCandidate {
	seen := make(map[string]struct{})
	var candidates []modelCandidate
	for _, root := range []string{filepath.Join(".", "onnx"), "."} {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if root == "." && path != "." && strings.HasPrefix(filepath.Base(path), ".") {
					return filepath.SkipDir
				}
				if root == "." && path != "." && filepath.Clean(path) != filepath.Clean("./onnx") {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.EqualFold(filepath.Ext(path), ".onnx") {
				clean := filepath.Clean(path)
				if _, ok := seen[clean]; ok {
					return nil
				}
				seen[clean] = struct{}{}
				profile, reason := detectProfileFromFilename(clean)
				candidates = append(candidates, modelCandidate{
					Path:    clean,
					Name:    filepath.Base(clean),
					Profile: profile,
					Reason:  reason,
				})
			}
			return nil
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Profile == candidates[j].Profile {
			return strings.ToLower(candidates[i].Name) < strings.ToLower(candidates[j].Name)
		}
		return candidates[i].Profile < candidates[j].Profile
	})
	return candidates
}
