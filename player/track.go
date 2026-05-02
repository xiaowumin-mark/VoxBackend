package player

import (
	"path/filepath"
	"strings"
	"time"
)

type Track struct {
	Path     string
	Title    string
	Artist   string
	Album    string
	Duration time.Duration
	Meta     map[string]any
}

func NewTrack(path string) Track {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return Track{
		Path:  path,
		Title: name,
	}
}

func TracksFromPaths(paths []string) []Track {
	tracks := make([]Track, len(paths))
	for i, p := range paths {
		tracks[i] = NewTrack(p)
	}
	return tracks
}

func cloneTrack(t Track) Track {
	c := Track{
		Path:     t.Path,
		Title:    t.Title,
		Artist:   t.Artist,
		Album:    t.Album,
		Duration: t.Duration,
	}
	if t.Meta != nil {
		c.Meta = make(map[string]any, len(t.Meta))
		for k, v := range t.Meta {
			c.Meta[k] = v
		}
	}
	return c
}

func cloneTracks(src []Track) []Track {
	if src == nil {
		return nil
	}
	dst := make([]Track, len(src))
	for i := range src {
		dst[i] = cloneTrack(src[i])
	}
	return dst
}
