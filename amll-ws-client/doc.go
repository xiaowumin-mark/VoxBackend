// Package amllwsclient 提供 AMLL Player 协议的 WebSocket 客户端。
//
// 连接 AMLL Player 服务端，实现 Protocol V2 的双向通信：
//   - 接收服务器的控制命令（暂停/播放/切歌/跳转/音量/循环模式等）
//   - 推送播放状态（曲目信息/进度/歌词/封面/音频数据等）
//
// 用法:
//
//	client := amllwsclient.New(amllwsclient.Config{
//	    URL:            "ws://localhost:11444",
//	    OnCommand:      func(cmd *amllwsclient.Command) {
//	        switch cmd.Type {
//	        case amllwsclient.CmdPause:
//	            // 暂停播放
//	        case amllwsclient.CmdResume:
//	            // 恢复播放
//	        case amllwsclient.CmdForwardSong:
//	            // 下一首
//	        case amllwsclient.CmdBackwardSong:
//	            // 上一首
//	        case amllwsclient.CmdSetVolume:
//	            // cmd.Volume 新音量 (0.0-1.0)
//	        case amllwsclient.CmdSeekPlayProgress:
//	            // cmd.Progress 跳转位置 (ms)
//	        case amllwsclient.CmdSetRepeatMode:
//	            // cmd.Mode 重复模式 ("off"/"all"/"one")
//	        case amllwsclient.CmdSetShuffleMode:
//	            // cmd.Enabled 随机播放开关
//	        }
//	    },
//	    OnStatusChange: func(st amllwsclient.Status) {
//	        // 连接状态变更
//	    },
//	})
//	client.Connect()           // 异步连接，立即返回
//	defer client.Close()       // 优雅关闭
//
//	client.SendMusic(amllwsclient.MusicInfo{
//	    MusicName: "Song Name",
//	    Artists:   []amllwsclient.Artist{{Name: "Artist"}},
//	    Duration:  240000,
//	})
//	client.SendProgress(12345)
package amllwsclient
