package separator

import (
	"fmt"
	"log"
	"math"

	ort "github.com/yalue/onnxruntime_go"
)

const DefaultONNXProbeFrames = 0

type ONNXProbeConfig struct {
	ModelPath          string
	RuntimeLibraryPath string
	ProbeFrames        int
	Profile            string
}

func ProbeONNXModel(cfg ONNXProbeConfig) error {
	if cfg.ModelPath == "" {
		return fmt.Errorf("ONNX 模型路径不能为空")
	}
	if cfg.RuntimeLibraryPath == "" {
		return fmt.Errorf("ONNX Runtime 动态库路径不能为空")
	}

	profile, err := ParseONNXProfile(cfg.Profile)
	if err != nil {
		return err
	}

	if err := InitONNXRuntime(cfg.RuntimeLibraryPath); err != nil {
		return err
	}

	inputs, outputs, err := ort.GetInputOutputInfo(cfg.ModelPath)
	if err != nil {
		return fmt.Errorf("读取模型输入输出信息失败: %w", err)
	}
	if len(inputs) != 1 || len(outputs) != 1 {
		return fmt.Errorf("当前仅支持单输入单输出模型，实际输入=%d 输出=%d", len(inputs), len(outputs))
	}

	inputInfo := inputs[0]
	outputInfo := outputs[0]
	log.Printf("ONNX 输入:  %s", inputInfo.String())
	log.Printf("ONNX 输出: %s", outputInfo.String())

	probeFrames := profile.Frames
	if cfg.ProbeFrames > 0 {
		probeFrames = cfg.ProbeFrames
	}

	inputShape, err := resolveProbeInputShape(inputInfo.Dimensions, profile, probeFrames)
	if err != nil {
		return fmt.Errorf("解析输入形状失败: %w", err)
	}
	outputShape, err := resolveProbeOutputShape(outputInfo.Dimensions, profile, inputShape)
	if err != nil {
		return fmt.Errorf("解析输出形状失败: %w", err)
	}

	inputData := make([]float32, inputShape.FlattenedSize())
	fillProbeInput(inputData)

	inputTensor, err := ort.NewTensor(inputShape, inputData)
	if err != nil {
		return fmt.Errorf("创建输入张量失败: %w", err)
	}
	defer inputTensor.Destroy()

	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		return fmt.Errorf("创建输出张量失败: %w", err)
	}
	defer outputTensor.Destroy()

	session, err := ort.NewAdvancedSession(
		cfg.ModelPath,
		[]string{inputInfo.Name},
		[]string{outputInfo.Name},
		[]ort.Value{inputTensor},
		[]ort.Value{outputTensor},
		nil,
	)
	if err != nil {
		return fmt.Errorf("创建 ONNX Session 失败: %w", err)
	}
	defer session.Destroy()

	if err := session.Run(); err != nil {
		return fmt.Errorf("运行 ONNX Session 失败: %w", err)
	}

	minValue, maxValue, meanValue := summarizeFloat32(outputTensor.GetData())
	log.Printf(
		"ONNX 探测成功: 配置=%s 帧数=%d 输入形状=%s 输出形状=%s 最小值=%.6f 最大值=%.6f 均值=%.6f",
		profile.Name,
		probeFrames,
		inputShape.String(),
		outputShape.String(),
		minValue,
		maxValue,
		meanValue,
	)
	return nil
}

func resolveProbeInputShape(shape ort.Shape, profile ONNXModelProfile, probeFrames int) (ort.Shape, error) {
	if len(shape) != 4 {
		return nil, fmt.Errorf("输入形状必须是 4 维，当前是 %s", shape.String())
	}
	resolved := shape.Clone()
	expected := []int64{
		1,
		int64(profile.Channels),
		int64(profile.FreqBins),
		int64(probeFrames),
	}
	for i := range resolved {
		if resolved[i] <= 0 {
			resolved[i] = expected[i]
			continue
		}
		if resolved[i] != expected[i] {
			return nil, fmt.Errorf("输入第 %d 维与配置不一致：模型=%d 配置=%d", i, resolved[i], expected[i])
		}
	}
	if err := resolved.Validate(); err != nil {
		return nil, err
	}
	return resolved, nil
}

func resolveProbeOutputShape(shape ort.Shape, profile ONNXModelProfile, inputShape ort.Shape) (ort.Shape, error) {
	if len(shape) == 0 {
		return nil, fmt.Errorf("输出形状不能为空")
	}

	resolved := shape.Clone()
	switch len(resolved) {
	case 4:
		for i := range resolved {
			if resolved[i] > 0 {
				continue
			}
			if i >= len(inputShape) {
				return nil, fmt.Errorf("无法根据输入形状 %s 推断输出第 %d 维", inputShape.String(), i)
			}
			resolved[i] = inputShape[i]
		}
	case 5:
		for i := range resolved {
			if resolved[i] > 0 {
				continue
			}
			switch i {
			case 0:
				resolved[i] = 1
			case 1:
				stems := int64(profile.OutputStemCount)
				if stems <= 0 {
					stems = 1
				}
				resolved[i] = stems
			case 2:
				resolved[i] = inputShape[1]
			case 3:
				resolved[i] = inputShape[2]
			case 4:
				resolved[i] = inputShape[3]
			}
		}
	default:
		return nil, fmt.Errorf("暂不支持 %d 维输出形状 %s", len(resolved), shape.String())
	}

	if err := resolved.Validate(); err != nil {
		return nil, err
	}
	return resolved, nil
}

func fillProbeInput(dst []float32) {
	for i := range dst {
		phase := float64(i%97) / 97.0 * 2.0 * math.Pi
		dst[i] = float32(0.1 * math.Sin(phase))
	}
}

func summarizeFloat32(values []float32) (float32, float32, float32) {
	if len(values) == 0 {
		return 0, 0, 0
	}

	minValue := values[0]
	maxValue := values[0]
	var sum float64
	for _, v := range values {
		if v < minValue {
			minValue = v
		}
		if v > maxValue {
			maxValue = v
		}
		sum += float64(v)
	}
	return minValue, maxValue, float32(sum / float64(len(values)))
}
