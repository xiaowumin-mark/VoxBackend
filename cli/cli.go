package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/xiaowumin-mark/VoxBackend/player"
	"github.com/xiaowumin-mark/VoxBackend/separator"
)

func Run(args []string, stdin io.Reader) error {
	cfg, inputPath, probeONNX, probeFrames, err := parseConfig(args)
	if err != nil {
		return err
	}

	if probeONNX {
		return separator.ProbeONNXModel(separator.ONNXProbeConfig{
			ModelPath:          cfg.ONNX.ModelPath,
			RuntimeLibraryPath: cfg.ONNX.RuntimeLibraryPath,
			ProbeFrames:        probeFrames,
			Profile:            cfg.ONNX.Profile,
		})
	}

	cfg.Callbacks.OnEvent = func(ev player.Event) {
		if ev.Type == player.EventError {
			log.Printf("[%s] %s: %v", ev.Type, ev.Message, ev.Err)
			return
		}
		if ev.TrackPath != "" {
			log.Printf("[%s] #%d %s - %s", ev.Type, ev.TrackIndex, ev.TrackPath, ev.Message)
			return
		}
		log.Printf("[%s] %s", ev.Type, ev.Message)
	}
	p := player.New(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := p.Start(ctx); err != nil {
		return err
	}

	if inputPath != "" {
		tracks, err := player.ResolvePlaylist(inputPath)
		if err != nil {
			log.Printf("扫描路径失败: %v", err)
		} else if len(tracks) > 0 {
			p.AddPaths(tracks...)
			p.Play()
		}
	}

	printHelp()
	go watchCLI(stdin, p, cancel)
	return p.Wait()
}

func parseConfig(args []string) (player.Config, string, bool, int, error) {
	cfg := player.DefaultConfig()
	fs := flag.NewFlagSet("voxbackend", flag.ContinueOnError)

	var separatorType string
	var crossfadeSec float64
	var prewarmSec float64
	var rampMs float64
	var inputPath string
	var probeONNX bool
	var probeFrames int

	separatorType = string(cfg.SeparatorMode)
	crossfadeSec = cfg.Crossfade.Seconds()
	prewarmSec = cfg.Prewarm.Seconds()
	rampMs = float64(cfg.VocalGainRamp) / float64(time.Millisecond)

	fs.StringVar(&inputPath, "input", "", "输入音频路径（wav/mp3/flac/ogg 文件或目录）")
	fs.StringVar(&separatorType, "separator", separatorType, "分离器类型：fake 或 onnx")
	fs.Float64Var(&cfg.VocalGain, "vocal-gain", cfg.VocalGain, "人声音量")
	fs.Float64Var(&rampMs, "vocal-gain-ramp-ms", rampMs, "人声音量平滑时间（毫秒）")
	fs.Float64Var(&cfg.FakeVocalAmount, "fake-vocal-amount", cfg.FakeVocalAmount, "fake 分离器的人声占比")
	fs.Float64Var(&cfg.FakeAccompBleed, "accomp-bleed", cfg.FakeAccompBleed, "fake 分离器伴奏扣减强度")
	fs.IntVar(&cfg.AILatency, "ai-latency-samples", cfg.AILatency, "fake 分离器延迟（采样点）")
	fs.StringVar(&cfg.ONNX.Profile, "onnx-profile", cfg.ONNX.Profile, "onnx 模型配置：umx-vocals / mdx-inst-hq3 / mdx23c-vocals / mdx-voc-ft / mdx-kara / mdx-kara2")
	fs.IntVar(&cfg.ONNX.StepFrames, "onnx-step-frames", cfg.ONNX.StepFrames, "onnx 重叠窗口步进帧数，0 表示按配置自动选择")
	fs.Float64Var(&cfg.ONNX.Compensation, "onnx-compensation", cfg.ONNX.Compensation, "onnx 补偿系数，0 表示使用配置默认值")
	fs.BoolVar(&probeONNX, "probe-onnx", false, "仅运行一次 onnx 探测后退出")
	fs.StringVar(&cfg.ONNX.ModelPath, "onnx-model", cfg.ONNX.ModelPath, "onnx 主模型路径")
	fs.StringVar(&cfg.ONNX.OtherModelPath, "onnx-other-model", cfg.ONNX.OtherModelPath, "可选 onnx other 模型路径（仅 umx 可用）")
	fs.StringVar(&cfg.ONNX.RuntimeLibraryPath, "onnx-runtime", cfg.ONNX.RuntimeLibraryPath, "onnxruntime.dll 路径")
	fs.IntVar(&probeFrames, "onnx-probe-frames", separator.DefaultONNXProbeFrames, "onnx 探测帧数，0 表示自动")
	fs.Float64Var(&crossfadeSec, "crossfade-sec", crossfadeSec, "淡入淡出过渡时间（秒）")
	fs.Float64Var(&prewarmSec, "prewarm-sec", prewarmSec, "提前预热下一首 AI 模型的时间（秒）")
	fs.Float64Var(&cfg.MasterVolume, "volume", cfg.MasterVolume, "主音量")

	if err := fs.Parse(args); err != nil {
		return cfg, "", false, 0, err
	}
	if fs.NArg() > 0 && inputPath == "" {
		inputPath = fs.Arg(0)
	}

	cfg.SeparatorMode = player.SeparatorMode(strings.ToLower(strings.TrimSpace(separatorType)))
	cfg.Crossfade = time.Duration(crossfadeSec * float64(time.Second))
	cfg.Prewarm = time.Duration(prewarmSec * float64(time.Second))
	cfg.VocalGainRamp = time.Duration(rampMs * float64(time.Millisecond))
	return cfg, inputPath, probeONNX, probeFrames, nil
}

func printHelp() {
	log.Printf("实时控制命令：")
	log.Printf("  gain <值>             设置人声音量")
	log.Printf("  ramp <毫秒>           设置人声变化平滑时间")
	log.Printf("  v <值>                设置主音量")
	log.Printf("  p                     暂停/恢复")
	log.Printf("  n/next                下一首")
	log.Printf("  b/prev                上一首")
	log.Printf("  jump <索引>           跳转到指定曲目")
	log.Printf("  s <秒>                seek 到指定播放时间")
	log.Printf("  list                  列出播放列表")
	log.Printf("  add <路径>            添加歌曲（文件或目录）")
	log.Printf("  rm <索引>             从列表移除歌曲")
	log.Printf("  mv <from> <to>        调整播放顺序")
	log.Printf("  play [索引]           开始播放（无参数 = 第一首）")
	log.Printf("  crossfade <秒>        热更新淡入淡出时长")
	log.Printf("  prewarm <秒>          热更新预热提前量")
	log.Printf("  fake <人声占比> <扣减> 热更新 fake 分离参数")
	log.Printf("  comp <值>             热更新 ONNX 补偿系数")
	log.Printf("  dsp <off|on|auto>     切换 DSP 模式")
	log.Printf("  status                输出当前播放状态")
	log.Printf("  q                     退出")
	log.Printf("也支持直接输入数字设置人声音量，例如 1.2")
}

func watchCLI(stdin io.Reader, p *player.Player, cancel context.CancelFunc) {
	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if tryLegacyGainInput(line, p) {
			continue
		}

		fields := strings.Fields(line)
		cmd := strings.ToLower(fields[0])
		switch cmd {
		case "gain":
			v, ok := parseOneFloat(fields, "用法: gain <值>")
			if ok {
				p.SetVocalGain(v)
			}
		case "ramp":
			ms, ok := parseOneFloat(fields, "用法: ramp <毫秒>")
			if ok {
				p.SetVocalRamp(time.Duration(ms * float64(time.Millisecond)))
			}
		case "v":
			vol, ok := parseOneFloat(fields, "用法: v <值>")
			if ok {
				p.SetMasterVolume(vol)
			}
		case "p":
			p.TogglePaused()
		case "n", "next":
			p.Next()
		case "b", "prev":
			p.Previous()
		case "jump":
			if len(fields) != 2 {
				log.Printf("用法: jump <索引>")
				continue
			}
			idx, err := strconv.Atoi(fields[1])
			if err != nil {
				log.Printf("无效索引 %q", fields[1])
				continue
			}
			p.Jump(idx)
		case "s":
			sec, ok := parseOneFloat(fields, "用法: s <秒>")
			if ok {
				p.SeekTo(time.Duration(sec * float64(time.Second)))
			}
		case "list":
			tracks := p.Playlist()
			state := p.Snapshot()
			if len(tracks) == 0 {
				fmt.Println("播放列表为空")
				continue
			}
			for i, t := range tracks {
				marker := " "
				if i == state.TrackIndex {
					marker = ">"
				}
				dur := ""
				if t.Duration > 0 {
					dur = t.Duration.Round(time.Millisecond).String()
				}
				fmt.Printf(" %s %3d  %-40s  %s\n", marker, i, t.Title, dur)
			}
		case "add":
			if len(fields) < 2 {
				log.Printf("用法: add <路径>")
				continue
			}
			rest := strings.TrimSpace(line[len("add"):])
			rest = strings.Trim(rest, `"`)
			tracks, err := player.ResolvePlaylist(rest)
			if err != nil {
				log.Printf("扫描 %q 失败: %v", rest, err)
				continue
			}
			p.AddPaths(tracks...)
			log.Printf("已添加 %d 首歌", len(tracks))
		case "rm":
			if len(fields) != 2 {
				log.Printf("用法: rm <索引>")
				continue
			}
			idx, err := strconv.Atoi(fields[1])
			if err != nil {
				log.Printf("无效索引 %q", fields[1])
				continue
			}
			p.RemoveTrack(idx)
		case "mv":
			if len(fields) != 3 {
				log.Printf("用法: mv <from> <to>")
				continue
			}
			from, err1 := strconv.Atoi(fields[1])
			to, err2 := strconv.Atoi(fields[2])
			if err1 != nil || err2 != nil {
				log.Printf("需要两个数字索引")
				continue
			}
			p.MoveTrack(from, to)
		case "play":
			if len(fields) >= 2 {
				idx, err := strconv.Atoi(fields[1])
				if err != nil {
					log.Printf("无效索引 %q", fields[1])
					continue
				}
				p.PlayIndex(idx)
			} else {
				p.Play()
			}
		case "crossfade":
			sec, ok := parseOneFloat(fields, "用法: crossfade <秒>")
			if ok {
				p.SetCrossfade(time.Duration(sec * float64(time.Second)))
			}
		case "prewarm":
			sec, ok := parseOneFloat(fields, "用法: prewarm <秒>")
			if ok {
				p.SetPrewarm(time.Duration(sec * float64(time.Second)))
			}
		case "fake":
			if len(fields) != 3 {
				log.Printf("用法: fake <人声占比> <扣减>")
				continue
			}
			vocalAmount, err1 := strconv.ParseFloat(fields[1], 64)
			accompBleed, err2 := strconv.ParseFloat(fields[2], 64)
			if err1 != nil || err2 != nil {
				log.Printf("fake 参数无效")
				continue
			}
			p.SetFakeParams(vocalAmount, accompBleed)
		case "comp":
			v, ok := parseOneFloat(fields, "用法: comp <值>")
			if ok {
				p.SetONNXCompensation(v)
			}
		case "dsp":
			if len(fields) != 2 {
				log.Printf("用法: dsp <off|on|auto>")
				continue
			}
			switch strings.ToLower(fields[1]) {
			case "off":
				p.SetDSPMode(player.DSPModeOff)
			case "on":
				p.SetDSPMode(player.DSPModeOn)
			case "auto":
				p.SetDSPMode(player.DSPModeAuto)
			default:
				log.Printf("无效 DSP 模式 %q，可选: off / on / auto", fields[1])
			}
		case "status":
			printStatus(p.Snapshot())
		case "help":
			printHelp()
		case "q":
			cancel()
			p.Stop()
			return
		default:
			log.Printf("未知命令: %q，输入 help 查看帮助", cmd)
		}
	}
}

func parseOneFloat(fields []string, usage string) (float64, bool) {
	if len(fields) != 2 {
		log.Printf("%s", usage)
		return 0, false
	}
	v, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		log.Printf("无效数字 %q", fields[1])
		return 0, false
	}
	return v, true
}

func tryLegacyGainInput(line string, p *player.Player) bool {
	v, err := strconv.ParseFloat(line, 64)
	if err != nil {
		return false
	}
	p.SetVocalGain(v)
	return true
}

func printStatus(state player.State) {
	fmt.Printf(
		"曲目=%d/%d 路径=%s 位置=%s/%s 暂停=%v 空闲=%v 音量=%.3f 人声=%.3f 预热=%v 淡入淡出=%v\n",
		state.TrackIndex,
		state.TrackCount,
		state.TrackPath,
		state.Position.Round(time.Millisecond),
		state.Duration.Round(time.Millisecond),
		state.Paused,
		state.Idle,
		state.Volume,
		state.VocalGain,
		state.Prewarm,
		state.Crossfade,
	)
}
