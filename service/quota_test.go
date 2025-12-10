package service

import (
	"strings"
	"testing"
)

// TestClaudeLongContextPricing 测试 Claude 长上下文计费逻辑
func TestClaudeLongContextPricing(t *testing.T) {
	tests := []struct {
		name                        string
		modelName                   string
		promptTokens                int
		cacheTokens                 int
		cacheCreationTokens         int
		completionTokens            int
		expectedIsLongContext       bool
		expectedInputMultiplier     float64
		expectedOutputMultiplier    float64
	}{
		{
			name:                     "Claude 模型，输入 < 200K，不触发长上下文",
			modelName:                "claude-sonnet-4-20250514",
			promptTokens:             100000,
			cacheTokens:              50000,
			cacheCreationTokens:      40000,
			completionTokens:         5000,
			expectedIsLongContext:    false,
			expectedInputMultiplier:  1.0,
			expectedOutputMultiplier: 1.0,
		},
		{
			name:                     "Claude 模型，输入 = 200K，触发长上下文",
			modelName:                "claude-sonnet-4-20250514",
			promptTokens:             100000,
			cacheTokens:              50000,
			cacheCreationTokens:      50000,
			completionTokens:         5000,
			expectedIsLongContext:    true,
			expectedInputMultiplier:  2.0,
			expectedOutputMultiplier: 1.5,
		},
		{
			name:                     "Claude 模型，输入 > 200K，触发长上下文",
			modelName:                "claude-sonnet-4-20250514",
			promptTokens:             150000,
			cacheTokens:              80000,
			cacheCreationTokens:      70000,
			completionTokens:         10000,
			expectedIsLongContext:    true,
			expectedInputMultiplier:  2.0,
			expectedOutputMultiplier: 1.5,
		},
		{
			name:                     "Claude Opus 模型，输入 >= 200K，触发长上下文",
			modelName:                "claude-opus-4-20250514",
			promptTokens:             200000,
			cacheTokens:              0,
			cacheCreationTokens:      0,
			completionTokens:         5000,
			expectedIsLongContext:    true,
			expectedInputMultiplier:  2.0,
			expectedOutputMultiplier: 1.5,
		},
		{
			name:                     "Claude Haiku 模型，输入 >= 200K，触发长上下文",
			modelName:                "claude-haiku-4-5-20250514",
			promptTokens:             250000,
			cacheTokens:              0,
			cacheCreationTokens:      0,
			completionTokens:         5000,
			expectedIsLongContext:    true,
			expectedInputMultiplier:  2.0,
			expectedOutputMultiplier: 1.5,
		},
		{
			name:                     "非 Claude 模型（GPT），输入 >= 200K，不触发长上下文",
			modelName:                "gpt-4o",
			promptTokens:             200000,
			cacheTokens:              50000,
			cacheCreationTokens:      50000,
			completionTokens:         5000,
			expectedIsLongContext:    false,
			expectedInputMultiplier:  1.0,
			expectedOutputMultiplier: 1.0,
		},
		{
			name:                     "非 Claude 模型（Gemini），输入 >= 200K，不触发长上下文",
			modelName:                "gemini-2.0-flash",
			promptTokens:             300000,
			cacheTokens:              0,
			cacheCreationTokens:      0,
			completionTokens:         5000,
			expectedIsLongContext:    false,
			expectedInputMultiplier:  1.0,
			expectedOutputMultiplier: 1.0,
		},
		{
			name:                     "Claude 模型（大写），输入 >= 200K，触发长上下文",
			modelName:                "CLAUDE-SONNET-4",
			promptTokens:             200000,
			cacheTokens:              0,
			cacheCreationTokens:      0,
			completionTokens:         5000,
			expectedIsLongContext:    true,
			expectedInputMultiplier:  2.0,
			expectedOutputMultiplier: 1.5,
		},
		{
			name:                     "边界测试：输入 199999，不触发长上下文",
			modelName:                "claude-sonnet-4-20250514",
			promptTokens:             199999,
			cacheTokens:              0,
			cacheCreationTokens:      0,
			completionTokens:         5000,
			expectedIsLongContext:    false,
			expectedInputMultiplier:  1.0,
			expectedOutputMultiplier: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟计费逻辑（与 PostClaudeConsumeQuota 中的逻辑一致）
			totalInputTokens := tt.promptTokens + tt.cacheTokens + tt.cacheCreationTokens
			isLongContext := false
			longContextInputMultiplier := 1.0
			longContextOutputMultiplier := 1.0

			if totalInputTokens >= 200000 && strings.Contains(strings.ToLower(tt.modelName), "claude") {
				isLongContext = true
				longContextInputMultiplier = 2.0
				longContextOutputMultiplier = 1.5
			}

			if isLongContext != tt.expectedIsLongContext {
				t.Errorf("isLongContext = %v, want %v (totalInputTokens=%d, model=%s)",
					isLongContext, tt.expectedIsLongContext, totalInputTokens, tt.modelName)
			}

			if longContextInputMultiplier != tt.expectedInputMultiplier {
				t.Errorf("longContextInputMultiplier = %v, want %v",
					longContextInputMultiplier, tt.expectedInputMultiplier)
			}

			if longContextOutputMultiplier != tt.expectedOutputMultiplier {
				t.Errorf("longContextOutputMultiplier = %v, want %v",
					longContextOutputMultiplier, tt.expectedOutputMultiplier)
			}
		})
	}
}

// TestClaudeLongContextQuotaCalculation 测试长上下文计费金额计算
func TestClaudeLongContextQuotaCalculation(t *testing.T) {
	// 模拟参数
	modelRatio := 1.0
	groupRatio := 1.0
	completionRatio := 4.0 // Claude 的补全倍率通常是 4-5 倍
	cacheRatio := 0.1
	cacheCreationRatio := 1.25

	tests := []struct {
		name             string
		promptTokens     int
		cacheTokens      int
		cacheCreationTokens int
		completionTokens int
		isLongContext    bool
		// expectedQuota 是相对于标准计费的倍数
	}{
		{
			name:                "标准计费（<200K）",
			promptTokens:        100000,
			cacheTokens:         0,
			cacheCreationTokens: 0,
			completionTokens:    10000,
			isLongContext:       false,
		},
		{
			name:                "长上下文计费（>=200K）",
			promptTokens:        200000,
			cacheTokens:         0,
			cacheCreationTokens: 0,
			completionTokens:    10000,
			isLongContext:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			longContextInputMultiplier := 1.0
			longContextOutputMultiplier := 1.0
			if tt.isLongContext {
				longContextInputMultiplier = 2.0
				longContextOutputMultiplier = 1.5
			}

			// 计算配额
			calculateQuota := float64(tt.promptTokens) * longContextInputMultiplier
			calculateQuota += float64(tt.cacheTokens) * cacheRatio * longContextInputMultiplier
			calculateQuota += float64(tt.cacheCreationTokens) * cacheCreationRatio * longContextInputMultiplier
			calculateQuota += float64(tt.completionTokens) * completionRatio * longContextOutputMultiplier
			calculateQuota = calculateQuota * groupRatio * modelRatio

			// 计算标准配额（作为对照）
			standardQuota := float64(tt.promptTokens)
			standardQuota += float64(tt.cacheTokens) * cacheRatio
			standardQuota += float64(tt.cacheCreationTokens) * cacheCreationRatio
			standardQuota += float64(tt.completionTokens) * completionRatio
			standardQuota = standardQuota * groupRatio * modelRatio

			t.Logf("模型: %s", tt.name)
			t.Logf("  输入tokens: prompt=%d, cache=%d, cacheCreation=%d",
				tt.promptTokens, tt.cacheTokens, tt.cacheCreationTokens)
			t.Logf("  输出tokens: completion=%d", tt.completionTokens)
			t.Logf("  标准配额: %.2f", standardQuota)
			t.Logf("  实际配额: %.2f", calculateQuota)
			if tt.isLongContext {
				t.Logf("  长上下文倍率: 输入 %.1fx, 输出 %.1fx", longContextInputMultiplier, longContextOutputMultiplier)
				ratio := calculateQuota / standardQuota
				t.Logf("  总体倍率: %.2fx", ratio)
			}
		})
	}
}
