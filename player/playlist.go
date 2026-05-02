package player

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ResolvePlaylist(inputPath string) ([]string, error) {
	info, err := os.Stat(inputPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{inputPath}, nil
	}

	var tracks []string
	validExts := map[string]bool{".wav": true, ".mp3": true, ".flac": true, ".ogg": true}

	err = filepath.WalkDir(inputPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && validExts[strings.ToLower(filepath.Ext(path))] {
			tracks = append(tracks, path)
		}
		return nil
	})

	sort.Strings(tracks)
	return tracks, err
}
