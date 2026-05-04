const { atom,getDefaultStore } = Jotai;

export const VoxBackendStates = {
    VocalGain: atom(1),
    MasterVolume: atom(1),
    ShowEleInPlayer: atom(true),
    VocalGainRamp: atom(1000), //ms
    Crossfade: atom(12), //s
    DSPMode : atom('auto'), // auto, on, off
    Crossfadeing : atom(false),

    WsIsConect : atom(false),
}