package catalog

import (
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"

	"opencode-go-cliproxyapi/internal/config"
)

// The upstream catalog publishes id-only entries, so records must resolve
// limits from the fallback precedence instead of reporting zero —
// downstream /v1/models and clients need a real context denominator.
func TestSwapResolvesLimitsFromBuiltinTable(t *testing.T) {
	cfg := testCfg()
	body := `{"data":[{"id":"deepseek-v4.1-flash"},{"id":"qwen3.6-plus"}]}`
	fc := &fakeClient{resp: pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(body)}}
	m := newManager(cfg, fc)
	mustRefresh(t, m)

	ds := findModel(t, m.Models(), "deepseek-v4.1-flash")
	if ds.ContextLimit != 1000000 || ds.OutputLimit != 384000 {
		t.Fatalf("deepseek limits = %d/%d, want 1000000/384000", ds.ContextLimit, ds.OutputLimit)
	}
	q := findModel(t, m.Models(), "qwen3.6-plus")
	if q.ContextLimit != 262144 || q.OutputLimit != 65536 {
		t.Fatalf("qwen limits = %d/%d, want 262144/65536", q.ContextLimit, q.OutputLimit)
	}
}

func TestSwapUpstreamLimitsWinOverTable(t *testing.T) {
	cfg := testCfg()
	body := `{"data":[{"id":"deepseek-v4.1-flash","context_length":2000000,"max_output_tokens":100}]}` //nolint:lll
	fc := &fakeClient{resp: pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(body)}}
	m := newManager(cfg, fc)
	mustRefresh(t, m)

	ds := findModel(t, m.Models(), "deepseek-v4.1-flash")
	if ds.ContextLimit != 2000000 || ds.OutputLimit != 100 {
		t.Fatalf("upstream limits not honored: %+v", ds)
	}
}

func TestSwapConfigOverrideBeatsBuiltin(t *testing.T) {
	cfg := testCfg()
	cfg.ModelLimits = map[string]config.ModelLimitOverride{
		"deepseek-v4.1-flash": {ContextLength: 128000, MaxOutput: 32000},
	}
	body := `{"data":[{"id":"deepseek-v4.1-flash"}]}`
	fc := &fakeClient{resp: pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(body)}}
	m := newManager(cfg, fc)
	mustRefresh(t, m)

	ds := findModel(t, m.Models(), "deepseek-v4.1-flash")
	if ds.ContextLimit != 128000 || ds.OutputLimit != 32000 {
		t.Fatalf("config override not honored: %+v", ds)
	}
}

func TestSwapUnknownModelKeepsZeroLimits(t *testing.T) {
	// glm-9.9 routes via the glm- prefix but is in neither the upstream
	// payload limits, the config override, nor the built-in table.
	cfg := testCfg()
	body := `{"data":[{"id":"glm-9.9"}]}`
	fc := &fakeClient{resp: pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(body)}}
	m := newManager(cfg, fc)
	mustRefresh(t, m)
	rec := findModel(t, m.Models(), "glm-9.9")
	if rec.ContextLimit != 0 || rec.OutputLimit != 0 {
		t.Fatalf("unknown model limits fabricated: %+v", rec)
	}
}

func TestResolveLimitsPartialMerge(t *testing.T) {
	// Upstream context only: output falls through to the table.
	ctx, out := ResolveLimits("deepseek-v4.1-flash", 2000000, 0, nil)
	if ctx != 2000000 || out != 384000 {
		t.Fatalf("partial merge = %d/%d, want 2000000/384000", ctx, out)
	}
	// Config covers only one side: the other side still falls through.
	ov := map[string]ModelLimits{"deepseek-v4.1-flash": {Context: 128000}}
	ctx, out = ResolveLimits("deepseek-v4.1-flash", 0, 0, ov)
	if ctx != 128000 || out != 384000 {
		t.Fatalf("config partial merge = %d/%d, want 128000/384000", ctx, out)
	}
}

func TestLookupBuiltinLimitsUnknown(t *testing.T) {
	if _, ok := LookupBuiltinLimits("no-such-model"); ok {
		t.Fatal("unknown model must not hit the table")
	}
}
