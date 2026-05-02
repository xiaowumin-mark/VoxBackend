package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"
)

type fileBackedStreamer struct {
	beep.StreamSeekCloser
	file *os.File
}

func (s *fileBackedStreamer) Close() error {
	err := s.StreamSeekCloser.Close()
	if ferr := s.file.Close(); ferr != nil && err == nil {
		err = ferr
	}
	return err
}

// DecodeFile 打开并解码一个音频文件。
//
// 支持 wav、mp3、flac、ogg。返回的 streamer 必须由调用方关闭。
func DecodeFile(path string) (beep.StreamSeekCloser, beep.Format, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".wav":
		s, format, err := wav.Decode(f)
		if err != nil {
			_ = f.Close()
			return nil, beep.Format{}, err
		}
		return &fileBackedStreamer{s, f}, format, nil
	case ".mp3":
		s, format, err := mp3.Decode(f)
		if err != nil {
			_ = f.Close()
			return nil, beep.Format{}, err
		}
		return &fileBackedStreamer{s, f}, format, nil
	case ".flac":
		s, format, err := flac.Decode(f)
		if err != nil {
			_ = f.Close()
			return nil, beep.Format{}, err
		}
		return &fileBackedStreamer{s, f}, format, nil
	case ".ogg":
		s, format, err := vorbis.Decode(f)
		if err != nil {
			_ = f.Close()
			return nil, beep.Format{}, err
		}
		return &fileBackedStreamer{s, f}, format, nil
	default:
		_ = f.Close()
		return nil, beep.Format{}, fmt.Errorf("不支持的音频格式 %q，当前支持 .wav .mp3 .flac .ogg", ext)
	}
}
