package main

import (
	"context"
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"time"

	statepkg "github.com/xiaowumin-mark/FluxUI/state"
	"github.com/xiaowumin-mark/FluxUI/system"
	ui "github.com/xiaowumin-mark/FluxUI/ui"
	amllwsclient "github.com/xiaowumin-mark/VoxBackend/amll-ws-client"
)

type guiRuntime struct {
	store      *appStore
	controller *appController
}

type guiPalette struct {
	Surface   color.NRGBA
	Panel     color.NRGBA
	PanelSoft color.NRGBA
	Text      color.NRGBA
	Muted     color.NRGBA
	Border    color.NRGBA
	Accent    color.NRGBA
	Accent2   color.NRGBA
	Warning   color.NRGBA
	Danger    color.NRGBA
	App       color.NRGBA
	Other     color.NRGBA
	Free      color.NRGBA
}

func runAppGUI(rt *guiRuntime) error {
	root := func(ctx *ui.Context) ui.Element {
		return guiApp(ctx, rt)
	}
	return ui.RunElement(
		root,
		ui.Title("AMLL Player VoxBackend"),
		ui.Size(1120, 780),
		ui.MinSize(760, 520),
		ui.WithTheme(ui.LightThemeFromSeed(ui.NRGBA(14, 116, 144, 255))),
		ui.WithDensity(ui.CompactDensityScale()),
	)
}

func guiApp(ctx *ui.Context, rt *guiRuntime) ui.Element {
	pal := palette()
	snapshot := ui.UseState(ctx, rt.store.Snapshot())
	modelPath := ui.UseState(ctx, snapshot.Value().Config.ModelPath)
	pickerBusy := ui.UseState(ctx, false)
	sampler := ui.UseRef(ctx, newResourceSampler())

	ui.UseInterval(ctx, time.Second, func() {
		if sampler.Current != nil {
			rt.store.UpdateResources(sampler.Current.Sample())
		}
		snapshot.Set(rt.store.Snapshot())
	})

	snap := snapshot.Value()
	narrow := ctx.MaxConstraints().X > 0 && ctx.MaxConstraints().X < 900

	var body ui.Element
	if narrow {
		body = ui.ScrollViewElement(
			ui.ColumnElement(
				logPanel(pal, snap),
				ui.SpacerElement(0, 12),
				controlPanel(pal, snap, rt, snapshot, modelPath, pickerBusy),
				ui.SpacerElement(0, 12),
				resourcePanel(pal, snap.Resources),
			),
			ui.ScrollVertical(true),
		)
	} else {
		body = ui.RowElement(
			ui.FlexedElement(1.45, logPanel(pal, snap)),
			ui.SpacerElement(12, 0),
			ui.FixedWidthElement(390, ui.ColumnElement(
				controlPanel(pal, snap, rt, snapshot, modelPath, pickerBusy),
				ui.SpacerElement(0, 12),
				ui.FlexedElement(1, resourcePanel(pal, snap.Resources)),
			)),
		)
	}

	return ui.ContainerDecorationElement(
		ui.Bg(pal.Surface).WithPad(ui.All(14)),
		ui.ColumnElement(
			headerPanel(pal, snap),
			ui.SpacerElement(0, 12),
			ui.FlexedElement(1, body),
			ui.SpacerElement(0, 12),
			playbackPanel(pal, snap),
		),
	)
}

func headerPanel(pal guiPalette, snap appSnapshot) ui.Element {
	statusText, statusColor := runtimeStatus(snap)
	amllText, amllColor := amllStatus(snap.AMLLStatus)
	return surface(pal, ui.ColumnElement(
		ui.RowElement(
			ui.ColumnElement(
				ui.TextElement("AMLL Player VoxBackend", ui.TextSize(23), ui.TextColor(pal.Text), ui.TextFontWeight(ui.FontWeightSemiBold)),
				ui.TextElement("Realtime vocal separation backend", ui.TextSize(12), ui.TextColor(pal.Muted)),
			),
			ui.FlexedElement(1, ui.SpacerElement(0, 0)),
			badge(statusText, statusColor),
			ui.SpacerElement(8, 0),
			badge("AMLL "+amllText, amllColor),
		),
	))
}

func logPanel(pal guiPalette, snap appSnapshot) ui.Element {
	lines := snap.Logs
	if len(lines) == 0 {
		lines = []string{"等待启动。Gin、插件和播放器日志会显示在这里。"}
	}
	children := []ui.Element{
		ui.TextElement("日志", ui.TextSize(17), ui.TextColor(pal.Text), ui.TextFontWeight(ui.FontWeightSemiBold)),
		ui.SpacerElement(0, 8),
	}
	for _, line := range lines {
		children = append(children, ui.PaddingElement(
			ui.Insets{Bottom: 3},
			ui.TextElement(line, ui.TextSize(12), ui.TextColor(pal.Text)),
		))
	}

	return surface(pal, ui.ColumnElement(
		children[0],
		children[1],
		ui.FlexedElement(1, ui.ScrollViewElement(
			ui.ColumnElement(children[2:]...),
			ui.ScrollVertical(true),
			ui.ScrollBarVisible(true),
			ui.ScrollAutoToEndKey(len(lines)),
		)),
	))
}

func controlPanel(pal guiPalette, snap appSnapshot, rt *guiRuntime, snapshot *statepkg.State[appSnapshot], modelPath *statepkg.State[string], pickerBusy *statepkg.State[bool]) ui.Element {
	options := modelOptions(snap)
	canEdit := !snap.Running && !snap.Starting
	fileDialogSupported := system.Supports(system.CapabilityFileDialog)
	startLabel := "启动"
	if snap.Running {
		startLabel = "停止"
	} else if snap.Starting {
		startLabel = "启动中"
	}

	return surface(pal, ui.ColumnElement(
		ui.TextElement("模型", ui.TextSize(17), ui.TextColor(pal.Text), ui.TextFontWeight(ui.FontWeightSemiBold)),
		ui.TextElement("程序会按文件名自动判断模型类型。", ui.TextSize(12), ui.TextColor(pal.Muted)),
		ui.SpacerElement(0, 10),
		ui.SelectElement(
			snap.Config.ModelPath,
			options,
			ui.SelectPlaceholder[string]("选择 ONNX 模型"),
			ui.SelectMaxHeight[string](260),
			ui.SelectDisabled[string](!canEdit || len(options) == 0),
			ui.SelectOnChange[string](func(ctx *ui.Context, value string) {
				rt.store.SelectModel(value)
				modelPath.Set(value)
				snapshot.Set(rt.store.Snapshot())
			}),
		),
		ui.SpacerElement(0, 8),
		ui.TextFieldElement(
			modelPath.Value(),
			ui.InputPlaceholder("当前模型路径"),
			ui.InputSingleLine(true),
			ui.InputDisabled(!canEdit),
			ui.InputOnChange(func(ctx *ui.Context, value string) {
				modelPath.Set(value)
				rt.store.SelectModel(value)
				snapshot.Set(rt.store.Snapshot())
			}),
		),
		ui.SpacerElement(0, 8),
		ui.RowElement(
			ui.FlexedElement(1, filePickerButton(pal, filePickerLabel(fileDialogSupported, pickerBusy.Value()), !canEdit || pickerBusy.Value() || !fileDialogSupported, rt, snapshot, modelPath, pickerBusy)),
			ui.SpacerElement(8, 0),
			ui.FlexedElement(1, secondaryButton(pal, "刷新模型", !canEdit, func() {
				rt.store.RefreshModels()
				next := rt.store.Snapshot()
				modelPath.Set(next.Config.ModelPath)
				snapshot.Set(next)
			})),
		),
		ui.SpacerElement(0, 10),
		infoLine(pal, "识别类型", snap.Profile),
		infoLine(pal, "判断依据", snap.ProfileNote),
		ui.SpacerElement(0, 14),
		ui.FillWidthElement(primaryButton(pal, startLabel, snap.Starting, func() {
			if snap.Running {
				rt.controller.Stop()
			} else {
				rt.store.SelectModel(modelPath.Value())
				rt.controller.Start(rt.store.Snapshot().Config)
			}
			snapshot.Set(rt.store.Snapshot())
		})),
		ui.SpacerElement(0, 10),
		ui.RowElement(
			ui.FlexedElement(1, secondaryButton(pal, "上一首", !snap.Running, func() { rt.controller.Previous() })),
			ui.SpacerElement(8, 0),
			ui.FlexedElement(1, secondaryButton(pal, playPauseLabel(snap.State.Paused), !snap.Running, func() { rt.controller.TogglePause() })),
			ui.SpacerElement(8, 0),
			ui.FlexedElement(1, secondaryButton(pal, "下一首", !snap.Running, func() { rt.controller.Next() })),
		),
		errorLine(pal, snap.LastError),
	))
}

func resourcePanel(pal guiPalette, stats resourceStats) ui.Element {
	cpuLine := "采样中"
	if stats.HasCPU {
		cpuLine = fmt.Sprintf("总 CPU %.1f%%  程序 %.1f%%", stats.TotalCPUPercent, stats.ProcessCPUPercent)
	}
	memLine := fmt.Sprintf("总内存 %.1f%%  程序 %.2f%%", stats.TotalMemoryPercent, stats.ProcessMemoryPercent)
	procMem := fmt.Sprintf("程序内存 %s / %s", formatBytes(stats.ProcessMemoryBytes), formatBytes(stats.TotalMemoryBytes))

	children := []ui.Element{
		ui.TextElement("资源占用", ui.TextSize(17), ui.TextColor(pal.Text), ui.TextFontWeight(ui.FontWeightSemiBold)),
		ui.TextElement("红色=程序  黄色=其他占用  绿色=剩余", ui.TextSize(12), ui.TextColor(pal.Muted)),
		ui.SpacerElement(0, 12),
		metricBlock(pal, "CPU", cpuLine, stats.ProcessCPUPercent, stats.TotalCPUPercent),
		ui.SpacerElement(0, 10),
		metricBlock(pal, "内存", memLine, stats.ProcessMemoryPercent, stats.TotalMemoryPercent),
		ui.SpacerElement(0, 8),
		ui.TextElement(procMem, ui.TextSize(12), ui.TextColor(pal.Muted)),
	}
	if stats.Err != nil {
		children = append(children, ui.SpacerElement(0, 8), ui.TextElement(stats.Err.Error(), ui.TextSize(12), ui.TextColor(pal.Danger)))
	}
	return surface(pal, ui.ColumnElement(children...))
}

func playbackPanel(pal guiPalette, snap appSnapshot) ui.Element {
	title := "等待 AMLL 插件加入歌曲"
	artist := ""
	position := time.Duration(0)
	duration := time.Duration(0)
	if snap.State.Track != nil {
		title = strings.TrimSpace(snap.State.Track.Title)
		artist = strings.TrimSpace(snap.State.Track.Artist)
		position = snap.State.Position
		duration = snap.State.Duration
		if title == "" {
			title = filepath.Base(snap.State.Track.Path)
		}
	}
	name := title
	if artist != "" {
		name += " - " + artist
	}
	if snap.State.Paused {
		name += " [暂停]"
	}

	return ui.FixedHeightElement(92, surface(pal, ui.ColumnElement(
		ui.RowElement(
			ui.TextElement(name, ui.TextSize(15), ui.TextColor(pal.Text), ui.TextFontWeight(ui.FontWeightMedium)),
			ui.FlexedElement(1, ui.SpacerElement(0, 0)),
			ui.TextElement(formatDuration(position)+" / "+formatDuration(duration), ui.TextSize(13), ui.TextColor(pal.Muted)),
		),
		ui.SpacerElement(0, 12),
		progressBar(pal, progressPercent(position, duration), pal.Accent),
	)))
}

func surface(pal guiPalette, child ui.Element) ui.Element {
	deco := ui.Bg(pal.Panel).
		WithPad(ui.All(14)).
		WithRad(14).
		WithBorder(ui.Border{Width: 1, Color: pal.Border}).
		Merge(ui.Shadow(0, 2, 8, ui.NRGBA(15, 23, 42, 24)))
	return ui.ContainerDecorationElement(
		deco,
		child,
	)
}

func badge(label string, col color.NRGBA) ui.Element {
	return ui.ContainerDecorationElement(
		ui.Bg(withAlpha(col, 34)).WithPad(ui.Symmetric(5, 9)).WithRad(20),
		ui.TextElement(label, ui.TextSize(12), ui.TextColor(col), ui.TextFontWeight(ui.FontWeightMedium)),
	)
}

func primaryButton(pal guiPalette, label string, disabled bool, onClick func()) ui.Element {
	return ui.ButtonElement(
		ui.TextElement(label),
		ui.Disabled(disabled),
		ui.ButtonPadding(ui.Symmetric(8, 12)),
		ui.ButtonRadius(10),
		ui.ButtonBackground(pal.Accent),
		ui.ButtonForeground(ui.NRGBA(255, 255, 255, 255)),
		ui.OnClick(func(ctx *ui.Context) {
			if !disabled && onClick != nil {
				onClick()
			}
		}),
	)
}

func secondaryButton(pal guiPalette, label string, disabled bool, onClick func()) ui.Element {
	return ui.ButtonElement(
		ui.TextElement(label),
		ui.Disabled(disabled),
		ui.ButtonPadding(ui.Symmetric(8, 10)),
		ui.ButtonRadius(10),
		ui.ButtonBackground(pal.PanelSoft),
		ui.ButtonForeground(pal.Text),
		ui.OnClick(func(ctx *ui.Context) {
			if !disabled && onClick != nil {
				onClick()
			}
		}),
	)
}

func filePickerButton(pal guiPalette, label string, disabled bool, rt *guiRuntime, snapshot *statepkg.State[appSnapshot], modelPath *statepkg.State[string], pickerBusy *statepkg.State[bool]) ui.Element {
	return ui.ButtonElement(
		ui.TextElement(label),
		ui.Disabled(disabled),
		ui.ButtonPadding(ui.Symmetric(8, 10)),
		ui.ButtonRadius(10),
		ui.ButtonBackground(pal.PanelSoft),
		ui.ButtonForeground(pal.Text),
		ui.OnClick(func(ctx *ui.Context) {
			if !disabled {
				openModelPicker(ctx, rt, snapshot, modelPath, pickerBusy)
			}
		}),
	)
}

func infoLine(pal guiPalette, label, value string) ui.Element {
	if strings.TrimSpace(value) == "" {
		value = "-"
	}
	return ui.PaddingElement(ui.Insets{Bottom: 4}, ui.RowElement(
		ui.FixedWidthElement(70, ui.TextElement(label, ui.TextSize(12), ui.TextColor(pal.Muted))),
		ui.FlexedElement(1, ui.TextElement(value, ui.TextSize(12), ui.TextColor(pal.Text))),
	))
}

func errorLine(pal guiPalette, text string) ui.Element {
	if strings.TrimSpace(text) == "" {
		return ui.SpacerElement(0, 0)
	}
	return ui.PaddingElement(ui.Insets{Top: 10}, ui.TextElement(text, ui.TextSize(12), ui.TextColor(pal.Danger)))
}

func metricBlock(pal guiPalette, label, value string, procPercent, totalPercent float64) ui.Element {
	return ui.ColumnElement(
		ui.RowElement(
			ui.TextElement(label, ui.TextSize(13), ui.TextColor(pal.Text), ui.TextFontWeight(ui.FontWeightMedium)),
			ui.FlexedElement(1, ui.SpacerElement(0, 0)),
			ui.TextElement(value, ui.TextSize(12), ui.TextColor(pal.Muted)),
		),
		ui.SpacerElement(0, 6),
		segmentedBar(pal, procPercent, totalPercent),
	)
}

func segmentedBar(pal guiPalette, procPercent, totalPercent float64) ui.Element {
	width := float32(280)
	proc := clampFloat(procPercent, 0, 100)
	total := clampFloat(totalPercent, 0, 100)
	if proc > total {
		proc = total
	}
	other := clampFloat(total-proc, 0, 100)
	free := clampFloat(100-total, 0, 100)
	return ui.FixedWidthElement(width, ui.FixedHeightElement(10,
		ui.ContainerDecorationElement(
			ui.Bg(pal.Free).WithRad(5),
			ui.RowElement(
				barSegment(width, proc, pal.App),
				barSegment(width, other, pal.Other),
				barSegment(width, free, pal.Free),
			),
		),
	))
}

func barSegment(totalWidth float32, percent float64, col color.NRGBA) ui.Element {
	if percent <= 0 {
		return ui.SpacerElement(0, 0)
	}
	w := totalWidth * float32(percent) / 100
	if w < 2 {
		w = 2
	}
	return ui.FixedWidthElement(w, ui.FixedHeightElement(10,
		ui.ContainerDecorationElement(ui.Bg(col).WithRad(5), ui.SpacerElement(0, 0)),
	))
}

func progressBar(pal guiPalette, percent float64, col color.NRGBA) ui.Element {
	return ui.FillWidthElement(ui.ProgressBarElement(
		float32(clampFloat(percent, 0, 100)),
		ui.ProgressMin(0),
		ui.ProgressMax(100),
		ui.ProgressThickness(12),
		ui.ProgressTrackColor(pal.PanelSoft),
		ui.ProgressFillColor(col),
		ui.ProgressLabelVisible(false),
	))
}

func openModelPicker(ctx *ui.Context, rt *guiRuntime, snapshot *statepkg.State[appSnapshot], modelPath *statepkg.State[string], pickerBusy *statepkg.State[bool]) {
	if ctx == nil || rt == nil || rt.store == nil {
		return
	}
	handle, _ := ui.GetWindow(ui.CurrentWindowID(ctx))
	pickerBusy.Set(true)
	go func(uiCtx *ui.Context) {
		result, err := ui.OpenFileDialogContext(uiCtx, context.Background(),
			system.FileDialogTitle("选择 ONNX 模型"),
			system.FileDialogDefaultDir("."),
			system.FileDialogFilters(
				system.FileFilter{Name: "ONNX 模型", Patterns: []string{"onnx"}},
				system.FileFilter{Name: "所有文件", Patterns: []string{"*.*"}},
			),
		)
		if err != nil {
			rt.store.AppendLog("选择模型失败: %v", err)
		} else if !result.Cancelled && len(result.Paths) > 0 {
			selected := filepath.Clean(result.Paths[0])
			rt.store.SelectModel(selected)
			modelPath.Set(selected)
		}
		pickerBusy.Set(false)
		snapshot.Set(rt.store.Snapshot())
		handle.Invalidate()
	}(ctx)
}

func filePickerLabel(supported, busy bool) string {
	if !supported {
		return "文件选择不可用"
	}
	if busy {
		return "选择中"
	}
	return "选择文件"
}

func modelOptions(snap appSnapshot) []ui.SelectOptionItem[string] {
	options := make([]ui.SelectOptionItem[string], 0, len(snap.Candidates)+1)
	seen := make(map[string]struct{})
	for _, candidate := range snap.Candidates {
		path := strings.TrimSpace(candidate.Path)
		if path == "" {
			continue
		}
		seen[strings.ToLower(filepath.Clean(path))] = struct{}{}
		label := candidate.Name
		if candidate.Profile != "" {
			label += "  (" + candidate.Profile + ")"
		}
		options = append(options, ui.SelectOptionItem[string]{Label: label, Value: path})
	}
	current := strings.TrimSpace(snap.Config.ModelPath)
	if current != "" {
		key := strings.ToLower(filepath.Clean(current))
		if _, ok := seen[key]; !ok {
			options = append([]ui.SelectOptionItem[string]{{Label: filepath.Base(current), Value: current}}, options...)
		}
	}
	return options
}

func runtimeStatus(snap appSnapshot) (string, color.NRGBA) {
	pal := palette()
	if snap.Starting {
		return "启动中", pal.Warning
	}
	if snap.Running {
		return "运行中", pal.Accent2
	}
	return "未启动", pal.Muted
}

func amllStatus(status amllwsclient.Status) (string, color.NRGBA) {
	pal := palette()
	switch status {
	case amllwsclient.StatusConnected:
		return "已连接", pal.Accent2
	case amllwsclient.StatusConnecting:
		return "连接中", pal.Warning
	case amllwsclient.StatusError:
		return "异常", pal.Danger
	default:
		return "未连接", pal.Muted
	}
}

func playPauseLabel(paused bool) string {
	if paused {
		return "播放"
	}
	return "暂停"
}

func progressPercent(pos, dur time.Duration) float64 {
	if dur <= 0 {
		return 0
	}
	return float64(pos) / float64(dur) * 100
}

func formatBytes(v uint64) string {
	const unit = 1024
	if v < unit {
		return fmt.Sprintf("%d B", v)
	}
	div, exp := uint64(unit), 0
	for n := v / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(v)/float64(div), "KMGTPE"[exp])
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func palette() guiPalette {
	return guiPalette{
		Surface:   ui.NRGBA(238, 242, 246, 255),
		Panel:     ui.NRGBA(255, 255, 255, 255),
		PanelSoft: ui.NRGBA(241, 245, 249, 255),
		Text:      ui.NRGBA(15, 23, 42, 255),
		Muted:     ui.NRGBA(100, 116, 139, 255),
		Border:    ui.NRGBA(203, 213, 225, 255),
		Accent:    ui.NRGBA(14, 116, 144, 255),
		Accent2:   ui.NRGBA(22, 163, 74, 255),
		Warning:   ui.NRGBA(217, 119, 6, 255),
		Danger:    ui.NRGBA(220, 38, 38, 255),
		App:       ui.NRGBA(225, 29, 72, 255),
		Other:     ui.NRGBA(245, 158, 11, 255),
		Free:      ui.NRGBA(34, 197, 94, 255),
	}
}

func withAlpha(col color.NRGBA, alpha uint8) color.NRGBA {
	col.A = alpha
	return col
}
