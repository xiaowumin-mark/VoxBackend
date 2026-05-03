package separator

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	ort "github.com/yalue/onnxruntime_go"
)

func getTestONNXConfig() (ONNXConfig, bool) {
	modelPath := os.Getenv("VOXBACKEND_TEST_ONNX_MODEL")
	ortLib := os.Getenv("VOXBACKEND_TEST_ONNX_RUNTIME")
	if modelPath == "" {
		modelPath = "..\\onnx\\UVR-MDX-NET-Inst_HQ_3.onnx"
	}
	if _, err := os.Stat(modelPath); err != nil {
		return ONNXConfig{}, false
	}
	if ortLib == "" {
		ortLib = "..\\onnxruntime\\lib\\onnxruntime.dll"
	}
	cfg := ONNXConfig{
		Profile:            MDXInstHQ3Profile,
		ModelPath:          modelPath,
		RuntimeLibraryPath: ortLib,
		StepFrames:         32,
	}
	return cfg, true
}

func loadTestSharedSession(t *testing.T, parallelMode bool) *SharedONNXSession {
	t.Helper()
	cfg, ok := getTestONNXConfig()
	if !ok {
		t.Skip("ONNX 模型不可用")
	}
	profile, err := ParseONNXProfile(cfg.Profile)
	if err != nil {
		t.Fatalf("解析 profile 失败: %v", err)
	}
	if err := InitONNXRuntime(cfg.RuntimeLibraryPath); err != nil {
		t.Fatalf("初始化 ONNX Runtime 失败: %v", err)
	}
	session, err := loadSessionWithMode(cfg.ModelPath, profile, parallelMode)
	if err != nil {
		t.Fatalf("加载 session 失败: %v", err)
	}
	return session
}

func loadSessionWithMode(modelPath string, profile ONNXModelProfile, parallelMode bool) (*SharedONNXSession, error) {
	inputs, outputs, err := ort.GetInputOutputInfo(modelPath)
	if err != nil {
		return nil, fmt.Errorf("读取 ONNX 信息失败: %w", err)
	}
	if len(inputs) != 1 || len(outputs) != 1 {
		return nil, fmt.Errorf("ONNX 非单输入单输出")
	}
	inputShape, err := resolveProfileInputShape(inputs[0].Dimensions, profile)
	if err != nil {
		return nil, err
	}
	outputShape, err := resolveProfileOutputShape(outputs[0].Dimensions, profile)
	if err != nil {
		return nil, err
	}

	options, err := ort.NewSessionOptions()
	if err != nil {
		return nil, err
	}
	defer options.Destroy()

	options.SetGraphOptimizationLevel(ort.GraphOptimizationLevelEnableAll)

	if parallelMode {
		if err := options.SetExecutionMode(ort.ExecutionModeParallel); err != nil {
			_ = options.Destroy()
			return nil, fmt.Errorf("设置 Parallel 执行模式失败: %w", err)
		}
	} else {
		if err := options.SetExecutionMode(ort.ExecutionModeSequential); err != nil {
			_ = options.Destroy()
			return nil, fmt.Errorf("设置 Sequential 执行模式失败: %w", err)
		}
	}

	options.SetIntraOpNumThreads(3)
	options.SetInterOpNumThreads(1)
	options.AddSessionConfigEntry("session.intra_op.allow_spinning", "0")
	options.AddSessionConfigEntry("session.inter_op.allow_spinning", "0")
	options.AddSessionConfigEntry("session.force_spinning_stop", "1")

	session, err := ort.NewDynamicAdvancedSession(
		modelPath,
		[]string{inputs[0].Name},
		[]string{outputs[0].Name},
		options,
	)
	if err != nil {
		return nil, fmt.Errorf("创建 DynamicAdvancedSession 失败: %w", err)
	}
	return &SharedONNXSession{
		session:     session,
		inputShape:  inputShape,
		outputShape: outputShape,
		inputName:   inputs[0].Name,
		outputName:  outputs[0].Name,
		profile:     profile,
		modelPath:   modelPath,
	}, nil
}

func TestSequentialMutexConcurrent(t *testing.T) {
	session := loadTestSharedSession(t, false)
	defer session.Destroy()

	m1, err := newONNXModelFromShared(session)
	if err != nil {
		t.Fatalf("创建 model 失败: %v", err)
	}
	defer m1.Destroy()
	m2, err := newONNXModelFromShared(session)
	if err != nil {
		t.Fatalf("创建 model 失败: %v", err)
	}
	defer m2.Destroy()

	shape := session.inputShape.FlattenedSize()
	input1 := make([]float32, shape)
	for i := range input1 {
		input1[i] = float32(i%100) / 100.0
	}
	input2 := make([]float32, shape)
	for i := range input2 {
		input2[i] = float32((i+50)%100) / 100.0
	}

	var wg sync.WaitGroup
	var errCount atomic.Int32
	iterations := 2

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_, err := m1.Run(input1)
			if err != nil {
				errCount.Add(1)
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_, err := m2.Run(input2)
			if err != nil {
				errCount.Add(1)
			}
		}
	}()
	wg.Wait()

	if errCount.Load() > 0 {
		t.Errorf("Sequential+mutex 并发测试出现 %d 个错误", errCount.Load())
	}
}

func TestParallelConcurrent(t *testing.T) {
	session := loadTestSharedSession(t, true)
	defer session.Destroy()

	m1, err := newONNXModelFromShared(session)
	if err != nil {
		t.Fatalf("创建 model 失败: %v", err)
	}
	defer m1.Destroy()
	m2, err := newONNXModelFromShared(session)
	if err != nil {
		t.Fatalf("创建 model 失败: %v", err)
	}
	defer m2.Destroy()

	shape := session.inputShape.FlattenedSize()
	input1 := make([]float32, shape)
	for i := range input1 {
		input1[i] = float32(i%100) / 100.0
	}
	input2 := make([]float32, shape)
	for i := range input2 {
		input2[i] = float32((i+50)%100) / 100.0
	}

	// Run WITHOUT mutex to test native Parallel mode safety
	runRaw := func(m *onnxModel, input []float32) ([]float32, error) {
		copy(m.inputTensor.GetData(), input)
		m.outputTensor.ZeroContents()
		err := m.shared.session.Run([]ort.Value{m.inputTensor}, []ort.Value{m.outputTensor})
		if err != nil {
			return nil, err
		}
		out := make([]float32, len(m.outputTensor.GetData()))
		copy(out, m.outputTensor.GetData())
		return out, nil
	}

	var wg sync.WaitGroup
	var errCount atomic.Int32
	iterations := 2

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_, err := runRaw(m1, input1)
			if err != nil {
				errCount.Add(1)
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_, err := runRaw(m2, input2)
			if err != nil {
				errCount.Add(1)
			}
		}
	}()
	wg.Wait()

	if errCount.Load() > 0 {
		t.Errorf("Parallel 并发测试出现 %d 个错误", errCount.Load())
	}
}

func TestParallelCorrectness(t *testing.T) {
	session := loadTestSharedSession(t, true)
	defer session.Destroy()

	m, err := newONNXModelFromShared(session)
	if err != nil {
		t.Fatalf("创建 model 失败: %v", err)
	}
	defer m.Destroy()

	shape := session.inputShape.FlattenedSize()
	input := make([]float32, shape)
	for i := range input {
		input[i] = float32(i+1) / float32(shape+1)
	}

	ref, err := m.Run(input)
	if err != nil {
		t.Fatalf("基准推理失败: %v", err)
	}

	for i := 0; i < 5; i++ {
		out, err := m.Run(input)
		if err != nil {
			t.Fatalf("第 %d 次推理失败: %v", i, err)
		}
		if len(out) != len(ref) {
			t.Fatalf("第 %d 次输出长度不一致", i)
		}
		for j := range ref {
			delta := out[j] - ref[j]
			if delta < 0 {
				delta = -delta
			}
			if delta > 1e-4 {
				t.Errorf("第 %d 次推理输出[%d] 与基准不一致: %f vs %f (delta=%e)", i, j, out[j], ref[j], delta)
				break
			}
		}
	}
}

func TestParallelModeAvailable(t *testing.T) {
	if ort.ExecutionModeParallel == ort.ExecutionModeSequential {
		t.Skip("Parallel execution mode not available (constant same as Sequential)")
	}
	t.Log("ExecutionModeParallel is available")
}
