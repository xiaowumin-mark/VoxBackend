package main

import (
	"path/filepath"
	"strings"

	"github.com/xiaowumin-mark/VoxBackend/separator"
)

func resolveConfiguredProfile(modelPath string) (string, string, error) {
	profile, reason := detectProfileFromFilename(modelPath)
	parsed, err := separator.ParseONNXProfile(profile)
	if err != nil {
		return "", "", err
	}
	return parsed.Name, reason, nil
}

func detectProfileFromFilename(path string) (string, string) {
	name := strings.ToLower(filepath.Base(path))
	name = strings.NewReplacer("_", "-", " ", "-", ".", "-").Replace(name)

	switch {
	case strings.Contains(name, "mdx23c"):
		return separator.MDX23CVocalsProfile, "文件名包含 mdx23c"
	case strings.Contains(name, "inst-hq-3") ||
		strings.Contains(name, "inst-hq3") ||
		strings.Contains(name, "instrumental") ||
		strings.Contains(name, "mdx-inst"):
		return separator.MDXInstHQ3Profile, "文件名像 MDX instrumental"
	case strings.Contains(name, "voc-ft") ||
		strings.Contains(name, "vocft") ||
		strings.Contains(name, "mdx-net-voc") ||
		strings.Contains(name, "mdx-voc"):
		return separator.MDXVocFTProfile, "文件名像 MDX vocal"
	case strings.Contains(name, "kara2") ||
		strings.Contains(name, "kara-2"):
		return separator.MDXKara2Profile, "文件名包含 kara2/kara-2"
	case strings.Contains(name, "kara"):
		return separator.MDXKaraProfile, "文件名包含 kara"
	case strings.Contains(name, "umxl") ||
		strings.Contains(name, "umx") ||
		strings.Contains(name, "open-unmix"):
		return separator.DefaultUMXVocalsProfile, "文件名像 UMX/Open-Unmix"
	default:
		return separator.DefaultUMXVocalsProfile, "无法识别，使用默认 UMX vocals"
	}
}
