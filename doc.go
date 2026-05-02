// Package voxbackend 提供一个可复用、可扩展的异步实时音频播放与人声分离库。
//
// 典型用法：
//
//	cfg := voxbackend.DefaultConfig()
//	cfg.InputPath = "music"
//	cfg.Callbacks.OnState = func(state voxbackend.State) {
//		// 默认 60fps 回调，可用于刷新进度条、播放时间、暂停状态等。
//	}
//	cfg.Callbacks.OnEvent = func(event voxbackend.Event) {
//		// 这里可以观察加载、预热、切歌、seek、错误等精细生命周期事件。
//	}
//
//	p := voxbackend.NewPlayer(cfg)
//	_ = p.Start(context.Background())
//
// 播放器启动后，大部分运行时参数都可以热更新，例如主音量、人声增益、人声变化平滑时间、
// 淡入淡出时长、预热提前量、fake 分离器参数和 ONNX 补偿系数。
package voxbackend
