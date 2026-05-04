const { useAtom } = Jotai
import { VoxBackendStates } from './store.jsx'
const { Text, Heading, Card, Flex, Slider, TextField, Switch, Select, Badge } = RadixTheme
const { useEffect, useRef } = React
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
    const logRef = useRef(null)

    useEffect(() => {
        logRef.current?.scrollTo({ top: 0, behavior: 'smooth' });
    }, [eventLog]);

    return (
        <>
            <Heading size="7" style={{
                lineHeight: 1.5
            }}>VoxBackend</Heading>
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
                        {wsIsConect ? '已连接' : '未连接'}
                    </Text>
                    <Badge color={wsIsConect ? 'green' : 'red'}>
                        {wsIsConect ? 'Socket.IO :54199' : '等待连接...'}
                    </Badge>
                </Flex>
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
                                const ts = `${String(t.getHours()).padStart(2,'0')}:${String(t.getMinutes()).padStart(2,'0')}:${String(t.getSeconds()).padStart(2,'0')}`;
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
export default Setting;