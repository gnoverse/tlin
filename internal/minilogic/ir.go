package minilogic

import (
	"sort"
	"strings"
)

// IRReport captures the IR for original and transformed statements.
type IRReport struct {
	Original    string
	Transformed string
}

func (v *Verifier) withDebugIR(report VerificationReport, original, transformed Stmt, env *Env) VerificationReport {
	if !v.config.DebugIR {
		return report
	}

	ir := buildIRReport(original, transformed, env, v.config)
	report.IR = &ir

	report.Detail = strings.TrimSpace(report.Detail)
	if report.Detail != "" {
		report.Detail += "\n"
	}
	report.Detail += "IR(original):\n" + indentIR(ir.Original) + "\nIR(transformed):\n" + indentIR(ir.Transformed)

	return report
}

func buildIRReport(original, transformed Stmt, env *Env, config EvalConfig) IRReport {
	return IRReport{
		Original:    formatStmtIR(original, env, config),
		Transformed: formatStmtIR(transformed, env, config),
	}
}

func formatStmtIR(stmt Stmt, env *Env, config EvalConfig) string {
	ev := NewEvaluator(config)
	base := cloneEnvForEval(env)
	result := ev.EvalStmt(stmt, base)
	return formatResultIRPretty(result)
}

func cloneEnvForEval(env *Env) *Env {
	if env == nil {
		return NewEnv()
	}
	return env.Clone()
}

func formatResultIR(result Result) string {
	switch result.Kind {
	case ResultContinue:
		return "Continue(env=" + formatEnvIR(result.Env) + ", calls=" + formatCallsIR(result.Calls) + ")"
	case ResultReturn:
		if result.Value == nil {
			return "Return(value=nil, calls=" + formatCallsIR(result.Calls) + ")"
		}
		return "Return(value=" + result.Value.String() + ", calls=" + formatCallsIR(result.Calls) + ")"
	case ResultBreak:
		return "Break(calls=" + formatCallsIR(result.Calls) + ")"
	case ResultContinueLoop:
		return "ContinueLoop(calls=" + formatCallsIR(result.Calls) + ")"
	case ResultUnknown:
		return "Unknown"
	default:
		return "Unknown"
	}
}

func formatResultIRPretty(result Result) string {
	switch result.Kind {
	case ResultContinue:
		return "Continue(\n  env = " + formatEnvIRPretty(result.Env) + "\n  calls = " + formatCallsIRPretty(result.Calls) + "\n)"
	case ResultReturn:
		if result.Value == nil {
			return "Return(\n  value = nil\n  calls = " + formatCallsIRPretty(result.Calls) + "\n)"
		}
		return "Return(\n  value = " + result.Value.String() + "\n  calls = " + formatCallsIRPretty(result.Calls) + "\n)"
	case ResultBreak:
		return "Break(\n  calls = " + formatCallsIRPretty(result.Calls) + "\n)"
	case ResultContinueLoop:
		return "ContinueLoop(\n  calls = " + formatCallsIRPretty(result.Calls) + "\n)"
	case ResultUnknown:
		return "Unknown"
	default:
		return "Unknown"
	}
}

func formatEnvIR(env *Env) string {
	if env == nil {
		return "{}"
	}
	keys := make(map[string]struct{})
	env.collectKeys(keys)
	if len(keys) == 0 {
		return "{}"
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	parts := make([]string, 0, len(sorted))
	for _, key := range sorted {
		val := env.Get(key)
		if val == nil {
			parts = append(parts, key+"=nil")
			continue
		}
		parts = append(parts, key+"="+val.String())
	}

	return "{" + strings.Join(parts, ", ") + "}"
}

func formatEnvIRPretty(env *Env) string {
	if env == nil {
		return "{}"
	}
	keys := make(map[string]struct{})
	env.collectKeys(keys)
	if len(keys) == 0 {
		return "{}"
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	lines := make([]string, 0, len(sorted))
	for _, key := range sorted {
		val := env.Get(key)
		if val == nil {
			lines = append(lines, "  "+key+" = nil")
			continue
		}
		lines = append(lines, "  "+key+" = "+val.String())
	}

	return "{\n" + strings.Join(lines, "\n") + "\n}"
}

func formatCallsIR(calls []CallRecord) string {
	if len(calls) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(calls))
	for _, call := range calls {
		parts = append(parts, call.String())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func formatCallsIRPretty(calls []CallRecord) string {
	if len(calls) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(calls))
	for _, call := range calls {
		parts = append(parts, call.String())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func indentIR(value string) string {
	lines := strings.Split(value, "\n")
	for i := range lines {
		lines[i] = "  " + lines[i]
	}
	return strings.Join(lines, "\n")
}
