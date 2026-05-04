const { atom, getDefaultStore } = Jotai;

function atomWithLocalStorage(key, defaultValue) {
    const initial = (() => {
        try {
            const raw = localStorage.getItem(key);
            return raw !== null ? JSON.parse(raw) : defaultValue;
        } catch {
            return defaultValue;
        }
    })();
    const base = atom(initial);
    return atom(
        (get) => get(base),
        (_get, set, value) => {
            set(base, value);
            try { localStorage.setItem(key, JSON.stringify(value)); } catch {}
        }
    );
}

export const VoxBackendStates = {
    VocalGain:       atomWithLocalStorage('vox_vocal_gain', 1),
    MasterVolume:    atomWithLocalStorage('vox_master_volume', 1),
    ShowEleInPlayer: atomWithLocalStorage('vox_show_ele', true),
    VocalGainRamp:   atomWithLocalStorage('vox_gain_ramp', 1000),
    Crossfade:       atomWithLocalStorage('vox_crossfade', 12),
    DSPMode:         atomWithLocalStorage('vox_dsp_mode', 'auto'),
    Crossfadeing:    atom(false),
    WsIsConect:      atom(false),
    EventLog:        atom([]),
}