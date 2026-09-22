// Package catalog — model context/output limits fallback (option C).
//
// The upstream OpenCode Go catalog (GET {base-url}/models) publishes only
// id/object/created/owned_by per entry — no context_length or
// max_output_tokens. Without a fallback every routable record reports
// ContextLimit/OutputLimit 0, downstream /v1/models shows a zero context,
// and clients (e.g. Paseo) cannot render a context window.
//
// Precedence for ModelRecord limits (highest first):
//  1. upstream catalog value when positive (a future catalog that gains
//     these fields is honored verbatim);
//  2. config model-limits override for the exact upstream ID;
//  3. the built-in table below (sourced from Zen's own published limits,
//     cross-checked against the models.dev multi-provider aggregate);
//  4. zero (unknown model, unchanged behavior — never fabricated).
package catalog

// ModelLimits carries the context/output token limits for one model.
type ModelLimits struct {
	Context int64
	Output  int64
}

// builtinLimits pins the known Zen (opencode-go) model limits. Values come
// from Zen's own published model listing, cross-checked against independent
// provider aggregates; on disagreement the Zen-published value wins because
// Zen is the serving authority for these IDs. Exact upstream IDs only —
// family-prefix guessing would silently misreport a new variant's window,
// which is worse for clients than reporting zero (config model-limits
// covers new IDs explicitly).
var builtinLimits = map[string]ModelLimits{
	// DeepSeek family (Zen: 1000000/384000 across the family).
	"deepseek-v4-pro":              {Context: 1000000, Output: 384000},
	"deepseek-v4-flash":            {Context: 1000000, Output: 384000},
	"deepseek-flash":               {Context: 1000000, Output: 384000},
	"deepseek-v4.1-flash":          {Context: 1000000, Output: 384000},
	"deepseek-v4-flash-vision-exp": {Context: 1000000, Output: 384000},

	// Qwen family.
	"qwen3.7-max":   {Context: 1000000, Output: 65536},
	"qwen3.8-max":   {Context: 1000000, Output: 131072},
	"qwen3.8-flash": {Context: 1000000, Output: 131072},
	"qwen3.7-plus":  {Context: 1000000, Output: 65536},
	"qwen3.6-plus":  {Context: 262144, Output: 65536},
	"qwen3.5-plus":  {Context: 262144, Output: 65536},

	// GLM (Zhipu) family (Zen: glm-5.2/5.3 at 1M, glm-5/5.1 at 200K).
	"glm-5.2":       {Context: 1000000, Output: 131072},
	"glm-5.3-flash": {Context: 1000000, Output: 131072},
	"glm-5.3":       {Context: 1000000, Output: 131072},
	"glm-5.1":       {Context: 204800, Output: 131072},
	"glm-5":         {Context: 204800, Output: 131072},

	// Kimi (Moonshot) family.
	"kimi-k3":        {Context: 1048576, Output: 131072},
	"kimi-k2.7-code": {Context: 262144, Output: 262144},
	"kimi-k2.6":      {Context: 262144, Output: 65536},
	"kimi-k2.5":      {Context: 262144, Output: 65536},
	"longcat-2.0":    {Context: 1000000, Output: 131072},

	// MiMo (Xiaomi) family.
	"mimo-v2-pro":     {Context: 1048576, Output: 131072},
	"mimo-v2-omni":    {Context: 262144, Output: 131072},
	"mimo-v2.6-pro":   {Context: 1048576, Output: 131072},
	"mimo-v2.6-flash": {Context: 1048576, Output: 131072},
	"mimo-v2.5-pro":   {Context: 1048576, Output: 131072},
	"mimo-v2.5":       {Context: 1048576, Output: 131072},

	// Hunyuan (Tencent) family.
	"hy4-preview": {Context: 1048576, Output: 64000},
	"hy3":         {Context: 262144, Output: 128000},
	"hy3-preview": {Context: 256000, Output: 128000},

	// MiniMax family.
	"minimax-m3":   {Context: 512000, Output: 128000},
	"minimax-m2.7": {Context: 204800, Output: 131072},
	"minimax-m2.5": {Context: 204800, Output: 131072},

	// OpenAI / xAI / Meta contributor models served via Zen.
	"gpt-5.6-luna":               {Context: 1050000, Output: 128000},
	"grok-4.5":                   {Context: 500000, Output: 500000},
	"grok-4.7":                   {Context: 500000, Output: 500000},
	"grok-4.6":                   {Context: 500000, Output: 500000},
	"muse-spark-1.3-contributor": {Context: 1048576, Output: 131072},
	"muse-spark-1.2-contributor": {Context: 1048576, Output: 131072},
	"omen-alpha":                 {Context: 500000, Output: 128000},
}

// LookupBuiltinLimits returns the built-in limits for an exact upstream
// model ID, or false when the model is not in the table.
func LookupBuiltinLimits(upstreamID string) (ModelLimits, bool) {
	lim, ok := builtinLimits[upstreamID]
	return lim, ok
}

// ResolveLimits applies the limits precedence for one catalog entry:
// a positive upstream value wins, then the config override for the exact
// ID, then the built-in table; unknown models resolve to zero limits.
func ResolveLimits(upstreamID string, upstreamCtx, upstreamOut int64, overrides map[string]ModelLimits) (ctx, out int64) {
	if upstreamCtx > 0 {
		ctx = upstreamCtx
	}
	if upstreamOut > 0 {
		out = upstreamOut
	}
	if ctx > 0 && out > 0 {
		return ctx, out
	}
	if ov, ok := overrides[upstreamID]; ok {
		if ctx <= 0 && ov.Context > 0 {
			ctx = ov.Context
		}
		if out <= 0 && ov.Output > 0 {
			out = ov.Output
		}
	}
	if ctx > 0 && out > 0 {
		return ctx, out
	}
	if lim, ok := builtinLimits[upstreamID]; ok {
		if ctx <= 0 {
			ctx = lim.Context
		}
		if out <= 0 {
			out = lim.Output
		}
	}
	return ctx, out
}
