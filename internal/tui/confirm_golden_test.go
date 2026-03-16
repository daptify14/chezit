package tui

import (
	"testing"

	"github.com/charmbracelet/x/exp/golden"
)

func TestGoldenApplyConfirm(t *testing.T) {
	t.Run("force_selected", func(t *testing.T) {
		m := newTestModel(WithSize(80, 24))
		m = m.showConfirmScreen(chezmoiActionApplyAll, "apply all changes to destination")
		output := stripForGolden(m.renderConfirmScreen())
		golden.RequireEqual(t, []byte(output))
	})

	t.Run("interactive_selected", func(t *testing.T) {
		m := newTestModel(WithSize(80, 24))
		m = m.showConfirmScreen(chezmoiActionApplyAll, "apply all changes to destination")
		m.overlays.applyForce = false
		output := stripForGolden(m.renderConfirmScreen())
		golden.RequireEqual(t, []byte(output))
	})
}
