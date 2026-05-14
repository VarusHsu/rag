package logx

import "testing"

func TestToZapFieldsEmpty(t *testing.T) {
	if got := toZapFields(nil); got != nil {
		t.Fatalf("expected nil for nil input, got len=%d", len(got))
	}

	if got := toZapFields(Fields{}); got != nil {
		t.Fatalf("expected nil for empty input, got len=%d", len(got))
	}
}

func TestToZapFieldsNonEmpty(t *testing.T) {
	fields := Fields{
		"request_id": "req-1",
		"status":     200,
	}

	got := toZapFields(fields)
	if len(got) != 2 {
		t.Fatalf("expected 2 zap fields, got %d", len(got))
	}
}

func TestSyncNoPanic(t *testing.T) {
	Sync()
}
