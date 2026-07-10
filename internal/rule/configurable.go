package rule

// ConfigurableRule extends Rule with rule-specific configuration
// parsing. Rules implement this when they accept a `data:` block in
// .tlin.yaml — the engine calls ParseConfig once at construction with
// the raw value the YAML decoder produced.
//
// The raw argument is whatever the YAML decoder yielded for the
// configured value: typically map[string]any, []any, string, or a
// numeric type. Implementations type-assert the shape they expect
// and return an error when the shape is wrong; the engine logs the
// error and continues with the rule's default behavior.
type ConfigurableRule interface {
	Rule
	ParseConfig(raw any) error
}
