package proofing_rpt

import "testing"

func TestOpacity(t *testing.T) {
	if Opacity(-0.01) != 0.9 {
		t.Error("Neg faScore should be 0.9")
	}
	if Opacity(0.00001) != 0.49995 {
		t.Error("Opacity should be 0.49995")
	}
	if Opacity(0.01) != 0.45 {
		t.Error("Opacity should be 0.45")
	}
	if Opacity(0.1) != 0.0 {
		t.Error("Opacity should be 0.0")
	}
}
