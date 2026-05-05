const { useAtom, useAtomValue } = Jotai
import { VoxBackendStates } from './store.jsx'
import { GetPlaylists } from './db.jsx'
const { Text, Heading, Card, Flex, Slider, TextField, Switch, Select, Badge, Button } = RadixTheme
const { useEffect, useRef, useState } = React
function Setting() {
    const [vocalGain, setVocalGain] = useAtom(VoxBackendStates.VocalGain)
    const [masterVolume, setMasterVolume] = useAtom(VoxBackendStates.MasterVolume)
    const [showEleInPlayer, setShowEleInPlayer] = useAtom(VoxBackendStates.ShowEleInPlayer)
    const [vocalGainRamp, setVocalGainRamp] = useAtom(VoxBackendStates.VocalGainRamp)
    const [crossfade, setCrossfade] = useAtom(VoxBackendStates.Crossfade)
    const [dspMode, setDspMode] = useAtom(VoxBackendStates.DSPMode)
    const [crossfadeing, setCrossfadeing] = useAtom(VoxBackendStates.Crossfadeing)
    const [wsIsConect] = useAtom(VoxBackendStates.WsIsConect)
    const [eventLog] = useAtom(VoxBackendStates.EventLog)
    const [isCrossfade, setIsCrossfade] = useAtom(VoxBackendStates.IsCrossfade)
    const [needShufflePlay, setNeedShufflePlay] = useAtom(VoxBackendStates.NeedShufflePlay)
    const [nowPlayListName, setNowPlayListName] = useAtom(VoxBackendStates.NowPlayListName)
    const logRef = useRef(null)
    const MusicContextMode = extensionContext.playerStates.MusicContextMode
    const musicMode = useAtomValue(extensionContext.playerStates.musicContextModeAtom)
    const isWsMode = musicMode === MusicContextMode.WSProtocol
    const [toast, setToast] = useState(null)

    useEffect(() => {
        if (!toast) return
        const t = setTimeout(() => setToast(null), 3000)
        return () => clearTimeout(t)
    }, [toast])

    const copyText = (url, msg) => {
        navigator.clipboard.writeText(url).catch(() => {})
        setToast(msg)
    }

    useEffect(() => {
        logRef.current?.scrollTo({ top: 0, behavior: 'smooth' });
    }, [eventLog]);

    return (
        <>
            <Flex align="center" gap="3">
                <Heading size="7" style={{ lineHeight: 1.5 }}>VoxBackend</Heading>
                <Button size="1" variant="ghost" color="gray" onClick={() => copyText(
                    'https://github.com/xiaowumin-mark/VoxBackend',
                    '链接已复制～记得点个 Star ⭐ (๑•̀ㅂ•́)و✧'
                )}>
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor" style={{ marginRight: 4 }}>
                        <path fillRule="evenodd" d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z" />
                    </svg>
                    Star
                </Button>
            </Flex>
            <Card mt="2">
                <Flex align="center" gap="2">
                    <div style={{
                        width: 10,
                        height: 10,
                        borderRadius: '50%',
                        backgroundColor: wsIsConect ? '#22c55e' : '#ef4444',
                        boxShadow: wsIsConect ? '0 0 6px #22c55e' : '0 0 6px #ef4444',
                    }} />
                    <Text size="2" color="gray">
                        {wsIsConect ? '已连接后端程序' : '未连接后端程序'}
                    </Text>
                    <Badge color={wsIsConect ? 'green' : 'red'}>
                        {wsIsConect ? 'Socket.IO :54199' : '等待连接...'}
                    </Badge>
                    {!wsIsConect && (
                        <Button size="1" variant="soft" color="red" onClick={() => copyText(
                            'https://github.com/xiaowumin-mark/VoxBackend/releases',
                            '链接已复制～请在浏览器中下载并按教程操作哦 🎵'
                        )}>
                            下载后端
                        </Button>
                    )}
                </Flex>
            </Card>

            <Card mt="2">
                {isWsMode ? (
                    <Flex align="center" gap="2">
                        <div style={{
                            width: 10, height: 10, borderRadius: '50%',
                            backgroundColor: '#22c55e',
                            boxShadow: '0 0 6px #22c55e',
                        }} />
                        <Text size="2" color="gray">音乐上下文模式正确</Text>
                        <Badge color="green">WSProtocol</Badge>
                    </Flex>
                ) : (
                    <Flex direction="column" gap="2">
                        <Flex align="center" gap="2">
                            <div style={{
                                width: 10, height: 10, borderRadius: '50%',
                                backgroundColor: '#f59e0b',
                                boxShadow: '0 0 6px #f59e0b',
                            }} />
                            <Text size="2" color="gray">模式不匹配，当前为</Text>
                            <Badge color="orange">{musicMode}</Badge>
                            <Text size="2" color="gray">，需要 WSProtocol</Text>
                        </Flex>
                        <Button size="1" variant="soft" color="orange" onClick={() => {
                            extensionContext.jotaiStore.set(
                                extensionContext.playerStates.musicContextModeAtom,
                                MusicContextMode.WSProtocol
                            );
                        }}>
                            切换到 WS 模式
                        </Button>
                    </Flex>
                )}
            </Card>

            <Flex direction="column" gap="0.5">
                <SettingEntry label="显示人声调节控件" description="在歌词界面中显示人声调节控件">
                    <Switch checked={showEleInPlayer} onCheckedChange={setShowEleInPlayer} />
                </SettingEntry>
                <SettingEntry label="人声音量" description="歌曲中的人声音量">
                    <div style={{
                        width: 200,
                    }}>
                        <Slider defaultValue={[vocalGain]} onValueChange={(value) => setVocalGain(value[0])} value={[vocalGain]} max={1} step={0.01} />
                    </div>

                </SettingEntry>
                <SettingEntry label="音频音量" description="播放中的音频音量">
                    <div style={{
                        width: 200,
                    }}>
                        <Slider defaultValue={[masterVolume]} onValueChange={(value) => setMasterVolume(value[0])} value={[masterVolume]} max={1} step={0.01} />
                    </div>
                </SettingEntry>
                <SettingEntry label="人声音量平滑时间" description="音量平滑时间，单位为毫秒">
                    <TextField.Root type='number' value={vocalGainRamp} onChange={(e) => setVocalGainRamp(Number(e.target.value))} style={{
                        width: 200,
                    }}>
                        <TextField.Slot></TextField.Slot>
                        <TextField.Slot>ms</TextField.Slot>
                    </TextField.Root>
                </SettingEntry>
                <SettingEntry label="淡入淡出过渡时间" description="淡入淡出过渡时间，单位为秒">
                    <TextField.Root type='number' value={crossfade} onChange={(e) => setCrossfade(Number(e.target.value))} style={{
                        width: 200,
                    }}>
                        <TextField.Slot></TextField.Slot>
                        <TextField.Slot>s</TextField.Slot>
                    </TextField.Root>
                </SettingEntry>
                <SettingEntry label="DSP增强" description="DSP增强模式，自动模式会根据人声音量自动设置强度">
                    <Select.Root defaultValue={dspMode} onValueChange={setDspMode}>
                        <Select.Trigger />
                        <Select.Content>
                            <Select.Item value="auto">自动</Select.Item>
                            <Select.Item value="on">开启</Select.Item>
                            <Select.Item value="off">关闭</Select.Item>
                        </Select.Content>
                    </Select.Root>

                </SettingEntry>
                <SettingEntry label="平滑过渡" description="切换歌曲时启用淡入淡出过渡">
                    <Switch checked={isCrossfade} onCheckedChange={setIsCrossfade} />
                </SettingEntry>
                <SettingEntry label="随机播放" description="重新随机排列当前播放列表">
                    <Button size="1" variant="soft" onClick={() => setNeedShufflePlay(true)} disabled={needShufflePlay}>
                        随机打乱
                    </Button>
                </SettingEntry>
                <SettingEntry label="歌单选择" description="选择本地歌单并加载到后端播放">
                    <PlaylistSelect nowPlayListName={nowPlayListName} setNowPlayListName={setNowPlayListName} />
                </SettingEntry>

            </Flex>
            <Card mt="3">
                <Heading size="3" mb="2">事件日志</Heading>
                <div ref={logRef} style={{ height: 200, overflowY: 'auto' }}>
                    {eventLog.length === 0 ? (
                        <Text size="2" color="gray">暂无事件</Text>
                    ) : (
                        <Flex direction="column" gap="1">
                            {eventLog.map((entry, i) => {
                                const t = new Date(entry.time);
                                const ts = `${String(t.getHours()).padStart(2, '0')}:${String(t.getMinutes()).padStart(2, '0')}:${String(t.getSeconds()).padStart(2, '0')}`;
                                const isError = entry.type === 'error';
                                return (
                                    <Flex key={i} gap="2" align="center">
                                        <Text size="1" color="gray" style={{ fontFamily: 'monospace', width: 56, flexShrink: 0 }}>{ts}</Text>
                                        <Badge color={isError ? 'red' : 'gray'} size="1">{entry.type}</Badge>
                                        <Text size="1" style={{ wordBreak: 'break-all' }}>{entry.message}</Text>
                                    </Flex>
                                );
                            })}
                        </Flex>
                    )}
                </div>
            </Card>
            {toast && (
                <div style={{
                    position: 'fixed', bottom: 120, left: '50%', transform: 'translateX(-50%)',
                    zIndex: 9999, padding: '10px 20px', borderRadius: 8,
                    background: 'rgba(18,18,18,0.94)', color: '#fff', fontSize: 14,
                    backdropFilter: 'blur(4px)', border: '1px solid rgba(255,255,255,0.1)',
                    animation: 'fadeIn .3s ease', pointerEvents: 'none', maxWidth: '90vw',
                }}>
                    {toast}
                </div>
            )}
        </>
    )
}


const SettingEntry = ({ label, description, children }) => {
    return (
        <Card mt="2">
            <Flex direction="row" align="center" gap="4" wrap="wrap">
                <Flex direction="column" flexGrow="1">
                    <Text as="div">{label}</Text>
                    <Text as="div" color="gray" size="2">
                        {description}
                    </Text>
                </Flex>
                {children}
            </Flex>
        </Card>
    );
};
function PlaylistSelect({ nowPlayListName, setNowPlayListName }) {
    const [playlists, setPlaylists] = useState([]);

    useEffect(() => {
        GetPlaylists().then(setPlaylists);
    }, []);

    return (
        <Select.Root defaultValue={nowPlayListName} onValueChange={setNowPlayListName}>
            <Select.Trigger placeholder="选择歌单" />
            <Select.Content>
                {playlists.map(pl => (
                    <Select.Item key={pl.id} value={pl.name}>{pl.name}</Select.Item>
                ))}
            </Select.Content>
        </Select.Root>
    );
}
export default Setting;