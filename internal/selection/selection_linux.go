package selection

import (
	"context"
	_ "embed"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/res"
	"github.com/jezek/xgb/xproto"
)

//go:embed focused_selection.py
var focusedSelectionScript string

// A process may keep PRIMARY after focus moves to another of its windows.
func Text(ctx context.Context) string {
	if os.Getenv("DISPLAY") == "" || os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland" {
		return ""
	}
	conn, err := openX11(ctx)
	if err != nil {
		return ""
	}
	stop := context.AfterFunc(ctx, conn.Close)
	defer func() {
		if stop() {
			conn.Close()
		}
	}()
	owner, err := xproto.GetSelectionOwner(conn, xproto.AtomPrimary).Reply()
	if err != nil || owner == nil || owner.Owner == 0 {
		return ""
	}
	focus, err := xproto.GetInputFocus(conn).Reply()
	if err != nil || focus == nil || focus.Focus <= xproto.InputFocusPointerRoot || focus.Focus == xproto.Setup(conn).DefaultScreen(conn).Root {
		return ""
	}
	if !sameClientWindow(conn, owner.Owner, focus.Focus) {
		// GTK and other toolkits may own PRIMARY on a hidden, process-wide window.
		// Read the focused accessible control directly, never that shared selection.
		pid := windowPID(conn, focus.Focus)
		if pid == 0 {
			return ""
		}
		python := "/usr/bin/python3"
		if _, err := os.Stat(python); os.IsNotExist(err) {
			python = "python3"
		}
		out, err := exec.CommandContext(ctx, python, "-I", "-c", focusedSelectionScript, strconv.FormatUint(uint64(pid), 10)).Output()
		if err != nil {
			return ""
		}
		lastFocus, err := xproto.GetInputFocus(conn).Reply()
		if err != nil || lastFocus == nil || lastFocus.Focus != focus.Focus {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	window, err := xproto.NewWindowId(conn)
	if err != nil {
		return ""
	}
	root := xproto.Setup(conn).DefaultScreen(conn).Root
	if err := xproto.CreateWindowChecked(conn, 0, window, root, 0, 0, 1, 1, 0, xproto.WindowClassInputOnly, 0,
		xproto.CwEventMask, []uint32{xproto.EventMaskPropertyChange}).Check(); err != nil {
		return ""
	}
	utf8, err := xproto.InternAtom(conn, false, 11, "UTF8_STRING").Reply()
	if err != nil || utf8 == nil {
		return ""
	}
	incr, err := xproto.InternAtom(conn, false, 4, "INCR").Reply()
	if err != nil || incr == nil {
		return ""
	}
	// The requestor is private to this lookup. Use the PRIMARY atom itself
	// as its result property to avoid interning a per-request property name.
	for _, target := range []xproto.Atom{utf8.Atom, xproto.AtomString} {
		if ctx.Err() != nil {
			return ""
		}
		text, supported := convertPrimary(conn, window, target, incr.Atom)
		if !supported {
			continue
		}
		lastOwner, err := xproto.GetSelectionOwner(conn, xproto.AtomPrimary).Reply()
		if err != nil || lastOwner == nil || lastOwner.Owner != owner.Owner {
			return ""
		}
		lastFocus, err := xproto.GetInputFocus(conn).Reply()
		if err != nil || lastFocus == nil || lastFocus.Focus != focus.Focus {
			return ""
		}
		return strings.TrimSpace(text)
	}
	return ""
}

// Some Linux displays (including Xvfb in WSL) listen only on an abstract
// socket. XGB's default dialer tries the filesystem path alone. Keep normal X
// authentication when opening that same display through its abstract socket.
func openX11(ctx context.Context) (*xgb.Conn, error) {
	c, err := xgb.NewConn()
	display := os.Getenv("DISPLAY")
	if err == nil || !strings.HasPrefix(display, ":") {
		return c, err
	}
	number, screen, _ := strings.Cut(display[1:], ".")
	n, parseErr := strconv.ParseUint(number, 10, 16)
	if parseErr != nil {
		return nil, err
	}
	path := "/tmp/.X11-unix/X" + strconv.FormatUint(n, 10)
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		return nil, err
	}
	var dialer net.Dialer
	socket, err := dialer.DialContext(ctx, "unix", "@"+path)
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = socket.SetDeadline(deadline)
	}
	// xauth resolves the display's cookie from the user's authority database.
	// The cookie stays in memory and is passed only to this X server.
	records, _ := exec.CommandContext(ctx, "xauth", "-i", "list", display).Output()
	var cookie string
	for _, record := range strings.Split(string(records), "\n") {
		fields := strings.Fields(record)
		if len(fields) == 3 && fields[1] == "MIT-MAGIC-COOKIE-1" {
			cookie = fields[2]
			break
		}
	}
	if cookie != "" {
		c, err = xgb.NewConnNetWithCookieHex(socket, cookie)
	} else {
		c, err = xgb.NewConnNet(socket)
	}
	if err != nil {
		socket.Close()
		return nil, err
	}
	c.DisplayNumber = int(n)
	c.DefaultScreen, _ = strconv.Atoi(screen)
	return c, nil
}

// WM_STATE marks the client window inside a window manager's decoration frame.
// With no window manager, an application's top-level window is a child of root.
func sameClientWindow(c *xgb.Conn, owner, focus xproto.Window) bool {
	if owner == focus {
		return true
	}
	state, err := xproto.InternAtom(c, true, 8, "WM_STATE").Reply()
	if err != nil || state == nil {
		return false
	}
	clientWindow := func(w xproto.Window) xproto.Window {
		for {
			tree, err := xproto.QueryTree(c, w).Reply()
			if err != nil || tree == nil || w == tree.Root {
				return 0
			}
			if state.Atom != 0 {
				prop, err := xproto.GetProperty(c, false, w, state.Atom, state.Atom, 0, 0).Reply()
				if err != nil || prop == nil {
					return 0
				}
				if prop.Type == state.Atom {
					return w
				}
			}
			if tree.Parent == tree.Root {
				return w
			}
			w = tree.Parent
		}
	}
	window := clientWindow(focus)
	return window != 0 && window == clientWindow(owner)
}

func windowPID(c *xgb.Conn, w xproto.Window) uint32 {
	if res.Init(c) != nil {
		return 0
	}
	reply, err := res.QueryClientIds(c, 1, []res.ClientIdSpec{{Client: uint32(w), Mask: res.ClientIdMaskLocalClientPID}}).Reply()
	if err != nil || reply == nil || len(reply.Ids) != 1 || len(reply.Ids[0].Value) != 1 {
		return 0
	}
	return reply.Ids[0].Value[0]
}

func convertPrimary(c *xgb.Conn, requestor xproto.Window, target, incr xproto.Atom) (string, bool) {
	const primary = xproto.AtomPrimary
	xproto.ConvertSelection(c, requestor, primary, target, primary, xproto.TimeCurrentTime)
	var data []byte
	streaming := false
	for {
		event, eventErr := c.WaitForEvent()
		if event == nil || eventErr != nil {
			return "", false
		}
		switch e := event.(type) {
		case xproto.SelectionNotifyEvent:
			if e.Requestor != requestor || e.Selection != primary || e.Target != target {
				continue
			}
			if e.Property != primary {
				return "", false
			}
		case xproto.PropertyNotifyEvent:
			if !streaming || e.Window != requestor || e.Atom != primary || e.State != xproto.PropertyNewValue {
				continue
			}
		default:
			continue
		}
		prop, err := xproto.GetProperty(c, true, requestor, primary, xproto.AtomAny, 0, 0x1fffffff).Reply()
		if err != nil || prop == nil {
			return "", false
		}
		if prop.Type == incr {
			streaming = true
			continue
		}
		if prop.Type != target || prop.Format != 8 {
			return "", false
		}
		data = append(data, prop.Value...)
		if streaming && len(prop.Value) > 0 {
			continue
		}
		if target == xproto.AtomString {
			chars := make([]rune, len(data))
			for i, b := range data {
				chars[i] = rune(b)
			}
			return string(chars), true
		}
		return string(data), true
	}
}
