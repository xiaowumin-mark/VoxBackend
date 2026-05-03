const { useAtom } = Jotai
import { VoxBackendStates } from './store.jsx'
const { Text, Heading, Card, Flex, Slider, TextField, Switch, Select } = RadixTheme
function Setting() {
    const [vocalGain, setVocalGain] = useAtom(VoxBackendStates.VocalGain)
    const [masterVolume, setMasterVolume] = useAtom(VoxBackendStates.MasterVolume)
    const [showEleInPlayer, setShowEleInPlayer] = useAtom(VoxBackendStates.ShowEleInPlayer)
    const [vocalGainRamp, setVocalGainRamp] = useAtom(VoxBackendStates.VocalGainRamp)
    const [crossfade, setCrossfade] = useAtom(VoxBackendStates.Crossfade)
    const [dspMode, setDspMode] = useAtom(VoxBackendStates.DSPMode)
    const [crossfadeing, setCrossfadeing] = useAtom(VoxBackendStates.Crossfadeing)
    return (
        <>
            <Heading size="7" style={{
                lineHeight: 1.5
            }}>VoxBackend</Heading>

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
                <SettingEntry label="淡入淡出" description="是否启用淡入淡出">
                    <Switch checked={crossfadeing} onCheckedChange={setCrossfadeing} />
                </SettingEntry>

            </Flex>
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