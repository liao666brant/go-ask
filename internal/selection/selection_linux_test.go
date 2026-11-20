package selection

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jezek/xgb/xproto"
)

// Run on a private X server (no interaction with a desktop user's focus or PRIMARY).
func TestX11SelectionWindow(t *testing.T) {
	if os.Getenv("GOASK_TEST_X11") == "" {
		t.Skip("set GOASK_TEST_X11 on an isolated X server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	c, err := openX11(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	root := xproto.Setup(c).DefaultScreen(c).Root
	create := func(parent xproto.Window) xproto.Window {
		t.Helper()
		id, err := xproto.NewWindowId(c)
		if err != nil {
			t.Fatal(err)
		}
		if err := xproto.CreateWindowChecked(c, 0, id, parent, 0, 0, 10, 10, 0, xproto.WindowClassInputOnly, 0, 0, nil).Check(); err != nil {
			t.Fatal(err)
		}
		return id
	}
	windowA, windowB := create(root), create(root)
	owner, focus := create(windowA), create(windowA)
	if !sameClientWindow(c, owner, focus) {
		t.Fatal("rejected owner and focus within window A")
	}
	if sameClientWindow(c, owner, create(windowB)) {
		t.Fatal("accepted window A selection while window B has focus")
	}
}
