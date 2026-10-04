// Package tray publishes a StatusNotifier item on the session bus.
// A missing StatusNotifier host does not stop the desk window.
package tray

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/mwirges/wx/linux/internal/pixmap"
	"github.com/mwirges/wx/linux/internal/present"
)

const (
	itemIface = "org.kde.StatusNotifierItem"
	menuIface = "com.canonical.dbusmenu"
	propIface = "org.freedesktop.DBus.Properties"
	itemPath  = dbus.ObjectPath("/StatusNotifierItem")
	menuPath  = dbus.ObjectPath("/StatusNotifierItem/Menu")
	guide     = "🔴 [WWWW] 000° ↘000mph"
)

// Pixmap is one IconPixmap entry: width, height, ARGB32 network-order bytes.
type Pixmap struct {
	Width  int32
	Height int32
	Bytes  []byte
}

// ToolTip is the StatusNotifier tooltip tuple.
type ToolTip struct {
	Icon  string
	Icons []Pixmap
	Title string
	Text  string
}

// Snapshot is the desk state the tray draws.
type Snapshot struct {
	Payload   map[string]any
	Units     string
	Format    string
	Error     string
	Location  string
	Favorites []map[string]any
	Cards     []map[string]string
}

// Node is a numbered DBus menu row.
type Node struct {
	ID       int32
	Props    map[string]any
	Children []*Node
	Action   string
	Target   string
}

// Layout is the DBus menu layout tuple (ia{sv}av).
type Layout struct {
	ID       int32
	Props    map[string]dbus.Variant
	Children []dbus.Variant
}

// NumberMenu assigns ids depth-first, starting at 0.
func NumberMenu(item present.Item) *Node {
	counter := int32(0)
	return number(item, &counter)
}

func number(item present.Item, counter *int32) *Node {
	node := &Node{
		ID:     *counter,
		Props:  item.Props(),
		Action: item.Action,
		Target: item.Target,
	}
	*counter++
	for _, child := range item.Children {
		node.Children = append(node.Children, number(child, counter))
	}
	return node
}

// Flatten indexes a menu by id.
func Flatten(node *Node) map[int32]*Node {
	found := map[int32]*Node{}
	var walk func(*Node)
	walk = func(n *Node) {
		if n == nil {
			return
		}
		found[n.ID] = n
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(node)
	return found
}

// LayoutOf encodes a menu node for GetLayout.
func LayoutOf(node *Node) Layout {
	props := map[string]dbus.Variant{}
	for key, value := range node.Props {
		props[key] = propVariant(value)
	}
	kids := make([]dbus.Variant, 0, len(node.Children))
	for _, child := range node.Children {
		kids = append(kids, dbus.MakeVariant(LayoutOf(child)))
	}
	return Layout{ID: node.ID, Props: props, Children: kids}
}

func propVariant(value any) dbus.Variant {
	switch v := value.(type) {
	case bool:
		return dbus.MakeVariant(v)
	case int:
		return dbus.MakeVariant(int32(v))
	case int32:
		return dbus.MakeVariant(v)
	default:
		return dbus.MakeVariant(fmt.Sprint(v))
	}
}

// Tray is one StatusNotifier item and its DBus menu.
type Tray struct {
	snap     func() Snapshot
	onAction func(action, target string)

	mu        sync.Mutex
	conn      *dbus.Conn
	closed    bool
	wg        sync.WaitGroup
	Available bool
	Error     string
	revision  uint32
	menu      *Node
	byID      map[int32]*Node
	pixmapW   int
	pixmapH   int
	pixmap    []byte
}

// New builds a tray. snap and onAction may be nil until set.
func New(snap func() Snapshot, onAction func(action, target string)) *Tray {
	t := &Tray{
		snap:     snap,
		onAction: onAction,
		revision: 0,
	}
	t.rebuild()
	return t
}

// StatusLabel is the Mac menu-bar title, including the hazard pip.
func (t *Tray) StatusLabel() string {
	snap := t.snapshot()
	return present.StatusText(snap.Payload, snap.Units, snap.Format, snap.Error)
}

// Install exports the item on the session bus.
// A missing bus or StatusNotifier host is recorded and does not stop the caller.
func (t *Tray) Install() {
	t.mu.Lock()
	if t.conn != nil || t.closed {
		t.mu.Unlock()
		return
	}
	t.wg.Add(1)
	t.mu.Unlock()
	go func() {
		defer t.wg.Done()
		t.install()
	}()
}

// Shutdown releases the bus name. It is safe to call more than once.
func (t *Tray) Shutdown() {
	t.mu.Lock()
	t.closed = true
	conn := t.conn
	t.conn = nil
	t.Available = false
	t.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	t.wg.Wait()
}

// Sync redraws the label, pixmap, and menu, then tells the host.
func (t *Tray) Sync() {
	t.rebuild()
	t.emitSignals()
}

func (t *Tray) snapshot() Snapshot {
	if t.snap == nil {
		return Snapshot{Units: "imperial", Format: "standard"}
	}
	return t.snap()
}

func (t *Tray) rebuild() {
	snap := t.snapshot()
	menu := NumberMenu(present.TrayMenu(snap.Location, present.DisplayTemp(snap.Payload, snap.Units), snap.Favorites, snap.Format, snap.Cards))
	text := present.PixmapText(snap.Payload, snap.Units, snap.Format, snap.Error)
	w, h, blob := pixmap.RenderStatus(text, present.PipKind(snap.Payload))
	t.mu.Lock()
	t.menu = menu
	t.byID = Flatten(menu)
	t.pixmapW, t.pixmapH, t.pixmap = w, h, blob
	t.revision++
	t.mu.Unlock()
}

func (t *Tray) install() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.note(err)
		return
	}
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		_ = conn.Close()
		return
	}
	t.conn = conn
	t.mu.Unlock()

	item := &sni{tray: t}
	menu := &menuObj{tray: t}
	props := &propsObj{tray: t}
	if err := conn.Export(item, itemPath, itemIface); err != nil {
		t.note(err)
		return
	}
	if err := conn.Export(props, itemPath, propIface); err != nil {
		t.note(err)
		return
	}
	if err := conn.Export(menu, menuPath, menuIface); err != nil {
		t.note(err)
		return
	}
	if err := conn.Export(props, menuPath, propIface); err != nil {
		t.note(err)
		return
	}
	introItem := introspect.NewIntrospectable(&introspect.Node{
		Interfaces: []introspect.Interface{introspect.IntrospectData, itemIntro(), propsIntro()},
	})
	introMenu := introspect.NewIntrospectable(&introspect.Node{
		Interfaces: []introspect.Interface{introspect.IntrospectData, menuIntro(), propsIntro()},
	})
	_ = conn.Export(introItem, itemPath, "org.freedesktop.DBus.Introspectable")
	_ = conn.Export(introMenu, menuPath, "org.freedesktop.DBus.Introspectable")

	name := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	reply, err := conn.RequestName(name, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		if err == nil {
			err = fmt.Errorf("status notifier name %s was not acquired", name)
		}
		t.note(err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	err = conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher").
		CallWithContext(ctx, "org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem", 0, name).Err
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	if err != nil {
		t.Available = false
		t.Error = err.Error()
		return
	}
	t.Available = true
	t.Error = ""
}

func (t *Tray) note(err error) {
	if err == nil {
		return
	}
	t.mu.Lock()
	t.Available = false
	t.Error = err.Error()
	t.mu.Unlock()
}

func (t *Tray) emitSignals() {
	t.mu.Lock()
	conn := t.conn
	rev := t.revision
	t.mu.Unlock()
	if conn == nil {
		return
	}
	_ = conn.Emit(itemPath, itemIface+".NewIcon")
	_ = conn.Emit(itemPath, itemIface+".NewTitle")
	_ = conn.Emit(itemPath, itemIface+".NewToolTip")
	_ = conn.Emit(menuPath, menuIface+".LayoutUpdated", rev, int32(0))
}

func (t *Tray) activate() {
	if t.onAction != nil {
		t.onAction("open-desk", "")
	}
}

func (t *Tray) labelIcon() (string, string, string) {
	snap := t.snapshot()
	label := present.StatusText(snap.Payload, snap.Units, snap.Format, snap.Error)
	icon := ""
	if snap.Format == "standard" {
		icon = present.ConditionIcon(snap.Payload)
	}
	return label, icon, snap.Format
}

func (t *Tray) pix() []Pixmap {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.pixmap) == 0 {
		return []Pixmap{}
	}
	return []Pixmap{{Width: int32(t.pixmapW), Height: int32(t.pixmapH), Bytes: append([]byte(nil), t.pixmap...)}}
}

type sni struct{ tray *Tray }

func (s *sni) Activate(x, y int32) *dbus.Error {
	s.tray.activate()
	return nil
}

func (s *sni) SecondaryActivate(x, y int32) *dbus.Error {
	s.tray.activate()
	return nil
}

func (s *sni) ContextMenu(x, y int32) *dbus.Error {
	s.tray.activate()
	return nil
}

func (s *sni) Scroll(delta int32, orientation string) *dbus.Error { return nil }

type propsObj struct{ tray *Tray }

func (p *propsObj) Get(iface, name string) (dbus.Variant, *dbus.Error) {
	value, ok := p.tray.property(iface, name)
	if !ok {
		return dbus.Variant{}, dbus.MakeFailedError(fmt.Errorf("unknown property %s.%s", iface, name))
	}
	return value, nil
}

func (p *propsObj) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	return p.tray.allProperties(iface), nil
}

func (p *propsObj) Set(iface, name string, value dbus.Variant) *dbus.Error {
	return dbus.MakeFailedError(fmt.Errorf("read-only"))
}

func (t *Tray) property(iface, name string) (dbus.Variant, bool) {
	if iface == menuIface {
		switch name {
		case "Version":
			return dbus.MakeVariant(uint32(3)), true
		case "Status":
			return dbus.MakeVariant("normal"), true
		default:
			return dbus.Variant{}, false
		}
	}
	if iface != itemIface {
		return dbus.Variant{}, false
	}
	label, icon, _ := t.labelIcon()
	switch name {
	case "Category":
		return dbus.MakeVariant("ApplicationStatus"), true
	case "Id":
		return dbus.MakeVariant("wx"), true
	case "Title":
		return dbus.MakeVariant(label), true
	case "Status":
		return dbus.MakeVariant("Active"), true
	case "IconName":
		return dbus.MakeVariant(icon), true
	case "IconPixmap":
		return dbus.MakeVariant(t.pix()), true
	case "OverlayIconName":
		return dbus.MakeVariant(""), true
	case "OverlayIconPixmap":
		return dbus.MakeVariant([]Pixmap{}), true
	case "AttentionIconName":
		return dbus.MakeVariant(""), true
	case "ToolTip":
		tipIcon := icon
		if tipIcon == "" {
			tipIcon = "wx"
		}
		return dbus.MakeVariant(ToolTip{Icon: tipIcon, Icons: []Pixmap{}, Title: "wx", Text: label}), true
	case "Menu":
		return dbus.MakeVariant(menuPath), true
	case "ItemIsMenu":
		return dbus.MakeVariant(false), true
	case "XAyatanaLabel":
		return dbus.MakeVariant(label), true
	case "XAyatanaLabelGuide":
		return dbus.MakeVariant(guide), true
	default:
		return dbus.Variant{}, false
	}
}

func (t *Tray) allProperties(iface string) map[string]dbus.Variant {
	names := []string{"Version", "Status"}
	if iface == itemIface {
		names = []string{
			"Category", "Id", "Title", "Status", "IconName", "IconPixmap",
			"OverlayIconName", "OverlayIconPixmap", "AttentionIconName", "ToolTip",
			"Menu", "ItemIsMenu", "XAyatanaLabel", "XAyatanaLabelGuide",
		}
	}
	out := map[string]dbus.Variant{}
	for _, name := range names {
		if value, ok := t.property(iface, name); ok {
			out[name] = value
		}
	}
	return out
}

type menuObj struct{ tray *Tray }

func (m *menuObj) GetLayout(parentID, depth int32, props []string) (uint32, Layout, *dbus.Error) {
	m.tray.mu.Lock()
	defer m.tray.mu.Unlock()
	node := m.tray.byID[parentID]
	if node == nil {
		node = m.tray.menu
	}
	return m.tray.revision, LayoutOf(node), nil
}

func (m *menuObj) AboutToShow(id int32) (bool, *dbus.Error) { return false, nil }

func (m *menuObj) Event(id int32, eventID string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if eventID == "clicked" && m.tray.onAction != nil {
		m.tray.mu.Lock()
		node := m.tray.byID[id]
		m.tray.mu.Unlock()
		if node != nil && node.Action != "" {
			m.tray.onAction(node.Action, node.Target)
		}
	}
	return nil
}

func (m *menuObj) GetGroupProperties(ids []int32, names []string) ([]struct {
	ID    int32
	Props map[string]dbus.Variant
}, *dbus.Error) {
	return []struct {
		ID    int32
		Props map[string]dbus.Variant
	}{}, nil
}

func (m *menuObj) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	m.tray.mu.Lock()
	node := m.tray.byID[id]
	m.tray.mu.Unlock()
	if node == nil {
		return dbus.MakeVariant(""), nil
	}
	value, ok := node.Props[name]
	if !ok || value == nil {
		return dbus.MakeVariant(""), nil
	}
	return propVariant(value), nil
}

func itemIntro() introspect.Interface {
	return introspect.Interface{
		Name: itemIface,
		Methods: []introspect.Method{
			{Name: "Activate", Args: []introspect.Arg{{Name: "x", Type: "i", Direction: "in"}, {Name: "y", Type: "i", Direction: "in"}}},
			{Name: "SecondaryActivate", Args: []introspect.Arg{{Name: "x", Type: "i", Direction: "in"}, {Name: "y", Type: "i", Direction: "in"}}},
			{Name: "Scroll", Args: []introspect.Arg{{Name: "delta", Type: "i", Direction: "in"}, {Name: "orientation", Type: "s", Direction: "in"}}},
			{Name: "ContextMenu", Args: []introspect.Arg{{Name: "x", Type: "i", Direction: "in"}, {Name: "y", Type: "i", Direction: "in"}}},
		},
		Signals: []introspect.Signal{{Name: "NewTitle"}, {Name: "NewIcon"}, {Name: "NewToolTip"}, {Name: "NewStatus"}},
		Properties: []introspect.Property{
			{Name: "Category", Type: "s", Access: "read"},
			{Name: "Id", Type: "s", Access: "read"},
			{Name: "Title", Type: "s", Access: "read"},
			{Name: "Status", Type: "s", Access: "read"},
			{Name: "IconName", Type: "s", Access: "read"},
			{Name: "IconPixmap", Type: "a(iiay)", Access: "read"},
			{Name: "OverlayIconName", Type: "s", Access: "read"},
			{Name: "OverlayIconPixmap", Type: "a(iiay)", Access: "read"},
			{Name: "AttentionIconName", Type: "s", Access: "read"},
			{Name: "ToolTip", Type: "(sa(iiay)ss)", Access: "read"},
			{Name: "Menu", Type: "o", Access: "read"},
			{Name: "ItemIsMenu", Type: "b", Access: "read"},
			{Name: "XAyatanaLabel", Type: "s", Access: "read"},
			{Name: "XAyatanaLabelGuide", Type: "s", Access: "read"},
		},
	}
}

func menuIntro() introspect.Interface {
	return introspect.Interface{
		Name: menuIface,
		Methods: []introspect.Method{
			{Name: "GetLayout", Args: []introspect.Arg{
				{Name: "parentId", Type: "i", Direction: "in"},
				{Name: "recursionDepth", Type: "i", Direction: "in"},
				{Name: "propertyNames", Type: "as", Direction: "in"},
				{Name: "revision", Type: "u", Direction: "out"},
				{Name: "layout", Type: "(ia{sv}av)", Direction: "out"},
			}},
			{Name: "GetGroupProperties", Args: []introspect.Arg{
				{Name: "ids", Type: "ai", Direction: "in"},
				{Name: "propertyNames", Type: "as", Direction: "in"},
				{Name: "properties", Type: "a(ia{sv})", Direction: "out"},
			}},
			{Name: "GetProperty", Args: []introspect.Arg{
				{Name: "id", Type: "i", Direction: "in"},
				{Name: "name", Type: "s", Direction: "in"},
				{Name: "value", Type: "v", Direction: "out"},
			}},
			{Name: "Event", Args: []introspect.Arg{
				{Name: "id", Type: "i", Direction: "in"},
				{Name: "eventId", Type: "s", Direction: "in"},
				{Name: "data", Type: "v", Direction: "in"},
				{Name: "timestamp", Type: "u", Direction: "in"},
			}},
			{Name: "AboutToShow", Args: []introspect.Arg{
				{Name: "id", Type: "i", Direction: "in"},
				{Name: "needUpdate", Type: "b", Direction: "out"},
			}},
		},
		Signals: []introspect.Signal{{Name: "LayoutUpdated", Args: []introspect.Arg{
			{Name: "revision", Type: "u"},
			{Name: "parent", Type: "i"},
		}}},
		Properties: []introspect.Property{
			{Name: "Version", Type: "u", Access: "read"},
			{Name: "Status", Type: "s", Access: "read"},
		},
	}
}

func propsIntro() introspect.Interface {
	return introspect.Interface{
		Name: propIface,
		Methods: []introspect.Method{
			{Name: "Get", Args: []introspect.Arg{
				{Name: "interface_name", Type: "s", Direction: "in"},
				{Name: "property_name", Type: "s", Direction: "in"},
				{Name: "value", Type: "v", Direction: "out"},
			}},
			{Name: "GetAll", Args: []introspect.Arg{
				{Name: "interface_name", Type: "s", Direction: "in"},
				{Name: "props", Type: "a{sv}", Direction: "out"},
			}},
			{Name: "Set", Args: []introspect.Arg{
				{Name: "interface_name", Type: "s", Direction: "in"},
				{Name: "property_name", Type: "s", Direction: "in"},
				{Name: "value", Type: "v", Direction: "in"},
			}},
		},
	}
}
