import { defineStore } from "pinia";
import { reactive, ref } from "vue";

export const useSettingsStore = defineStore("settings", () => {
    const VocalGain = ref(1);
    const VocalGainRamp = ref(1000);
    const DSPMode = ref("auto"); // auto, off, on
    const MasterVolume = ref(1);
    const Crossfade = ref(12);
    const ModelNameIndex = ref(0);
    const ModelNames = ref(["umx-vocals","mdx-kara2"]);
    const SetVocalGain = (value) => {
        VocalGain.value = value;
    };
    const SetVocalGainRamp = (value) => {
        VocalGainRamp.value = value;
    };
    const SetDSPMode = (value) => {
        DSPMode.value = value;
    };
    const SetMasterVolume = (value) => {
        MasterVolume.value = value;
    };
    const SetCrossfade = (value) => {
        Crossfade.value = value;
    };
    const SetModelNameIndex = (value) => {
        ModelNameIndex.value = value;
    };
    return {
        VocalGain,
        VocalGainRamp,
        DSPMode,
        MasterVolume,
        Crossfade,
        ModelNameIndex,
        ModelNames,
        SetVocalGain,
        SetVocalGainRamp,
        SetDSPMode,
        SetMasterVolume,
        SetCrossfade,
        SetModelNameIndex
    };

});