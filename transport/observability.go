package transport

// Emit fires the observer hook if one is configured. Attributes are
// redacted before being passed to the hook.
func Emit(hook ObservabilityHook, event string, attrs map[string]any) {
	if hook == nil {
		return
	}
	hook(event, Redact(attrs))
}
