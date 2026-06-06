package main

import (
	"fmt"
	"time"

	"github.com/xiaowumin-mark/VoxBackend/player"
)

var Player *player.Player
var Clientws *amllClient

func main() {
	cfg, cfgPath, err := loadAppConfig()
	store := newAppStore(cfg, cfgPath)
	setLogStore(store)
	installAppLogging()

	store.AppendLog("配置文件: %s", cfgPath)
	if err != nil {
		store.AppendLog("读取配置失败，已使用默认值: %v", err)
	}
	store.AppendLog("请选择模型后点击启动。")

	controller := newAppController(cfgPath, store)
	defer controller.Stop()
	runtime := &guiRuntime{store: store, controller: controller}
	if err := runAppGUI(runtime); err != nil {
		fmt.Printf("GUI 启动失败: %v\n", err)
	}
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func trackMetaValue(t *player.Track, key string) any {
	if t == nil || t.Meta == nil {
		return nil
	}
	return t.Meta[key]
}

func trackMetaString(t *player.Track, key string) string {
	v := trackMetaValue(t, key)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
