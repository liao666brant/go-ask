"""Read selected text from the active AT-SPI window of an X11 client."""
import sys

import gi

gi.require_version("Atspi", "2.0")
from gi.repository import Atspi


def children(node):
    for index in range(node.get_child_count()):
        child = node.get_child_at_index(index)
        if child is not None:
            yield child


def selected_ranges(node):
    if not node.get_text_iface():
        return []
    parts = []
    for index in range(Atspi.Text.get_n_selections(node)):
        span = Atspi.Text.get_selection(node, index)
        if span and 0 <= span.start_offset < span.end_offset:
            parts.append(Atspi.Text.get_text(node, span.start_offset, span.end_offset))
    return parts


def window_selection(window):
    documents = []
    pending = [window]
    while pending:
        node = pending.pop()
        states = node.get_state_set()
        role = node.get_role()
        if role == Atspi.Role.DOCUMENT_WEB:
            if states.contains(Atspi.StateType.SHOWING) and states.contains(Atspi.StateType.VISIBLE):
                documents.append(node)
        elif states.contains(Atspi.StateType.FOCUSED):
            return "" if role == Atspi.Role.PASSWORD_TEXT else "\n".join(selected_ranges(node))
        elif role != Atspi.Role.PASSWORD_TEXT:
            pending.extend(children(node))

    # WebKit exposes body selections on the visible document's text nodes, even
    # when no individual node has keyboard focus. Exclude other tabs and panes.
    if len(documents) != 1:
        return ""
    parts = []
    pending = documents
    while pending:
        node = pending.pop()
        states = node.get_state_set()
        if not states.contains(Atspi.StateType.SHOWING) or not states.contains(Atspi.StateType.VISIBLE):
            continue
        if node.get_role() == Atspi.Role.PASSWORD_TEXT:
            continue
        selected = selected_ranges(node)
        if selected:
            parts.extend(selected)
        else:
            pending.extend(reversed(list(children(node))))
    return "\n".join(parts)


def focused_selection(pid):
    for app in children(Atspi.get_desktop(0)):
        if app.get_process_id() == pid:
            for window in children(app):
                if window.get_state_set().contains(Atspi.StateType.ACTIVE):
                    return window_selection(window)
    return ""


sys.stdout.buffer.write(focused_selection(int(sys.argv[1])).encode("utf-8"))
