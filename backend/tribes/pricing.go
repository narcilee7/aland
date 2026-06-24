package tribes

import "strings"

// TokenPriceTable 已知模型的 USD/M token 价。
// 规则：cache_read 折算 10%（与 Anthropic 官方一致）；cache_creation 算全价。
//
// 未来支持用户自填（写到 ~/.aland/pricing.json）。
type TokenPriceTable struct {
	Input     float64
	Output    float64
	CacheRead float64 // 折算比例
}

// defaultPrices 是 2026 年 6 月的官方公开价。
// 任何新模型在这里加一行即可。
var defaultPrices = map[string]TokenPriceTable{
	"claude-opus-4-7":   {Input: 15, Output: 75, CacheRead: 1.5},
	"claude-opus-4-5":   {Input: 15, Output: 75, CacheRead: 1.5},
	"claude-opus-4-0":   {Input: 15, Output: 75, CacheRead: 1.5},
	"claude-sonnet-4-6": {Input: 3, Output: 15, CacheRead: 0.30},
	"claude-sonnet-4-5": {Input: 3, Output: 15, CacheRead: 0.30},
	"claude-sonnet-4-0": {Input: 3, Output: 15, CacheRead: 0.30},
	"claude-haiku-4-5":  {Input: 1, Output: 5, CacheRead: 0.10},
	"claude-haiku-3-5":  {Input: 0.80, Output: 4, CacheRead: 0.08},
}

// PriceFor 查 model 的价格；找不到给个保守的"高估"默认（避免被低估）。
func PriceFor(model string) TokenPriceTable {
	if p, ok := defaultPrices[model]; ok {
		return p
	}
	// 模糊匹配：前缀
	for k, p := range defaultPrices {
		if strings.HasPrefix(model, k) {
			return p
		}
	}
	// 默认按 Sonnet 价位（中等）
	return TokenPriceTable{Input: 3, Output: 15, CacheRead: 0.30}
}

// CostFor 按 TokenPriceTable 算 USD。
func (p TokenPriceTable) CostFor(in, out, cacheRead, cacheWrite int64) float64 {
	return float64(in)*p.Input/1e6 +
		float64(out)*p.Output/1e6 +
		float64(cacheRead)*p.CacheRead/1e6 +
		float64(cacheWrite)*p.Input/1e6 // cache write 算全价 input
}

// CostFromUsage 一行算价。
func CostFromUsage(model string, in, out, cacheRead, cacheWrite int64) float64 {
	return PriceFor(model).CostFor(in, out, cacheRead, cacheWrite)
}
