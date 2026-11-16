package views

import (
	"os"
	"testing"

	"github.com/sthbryan/ftm/internal/i18n"
)

func TestMain(m *testing.M) {
	if err := i18n.Load(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
