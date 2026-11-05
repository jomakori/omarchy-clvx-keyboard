import QtQuick
import qs.Commons
import qs.Ui

// Bar button for the CLVX keyboard. Owns the panel loader and hands the
// panel the bar, its own settings, the anchor button, and the plugin's
// service singleton (declared by manifest kind "service").
BarWidget {
  id: root
  moduleName: "io.github.joel.clvx-keyboard"

  readonly property string pluginId: "io.github.joel.clvx-keyboard"
  readonly property var service: {
    var host = bar && bar.shell ? bar.shell : null
    if (!host || typeof host.serviceFor !== "function") return null
    return host.serviceFor(root.pluginId)
  }
  readonly property bool connected: service ? service.connected === true : false
  readonly property int batteryPercent: service ? Number(service.batteryPercent) : 0
  readonly property bool charging: service ? service.charging === true : false
  readonly property string transport: service ? String(service.transport || "") : ""
  readonly property string lastError: service ? String(service.lastError || "") : "CLVX service unavailable"

  function injectPanel() {
    var target = panelLoader.item
    if (!target) return
    if ("bar" in target) target.bar = root.bar
    if ("settings" in target) target.settings = root.settings
    if ("anchorItem" in target) target.anchorItem = button
    if ("hostWidget" in target) target.hostWidget = root
    if ("service" in target) target.service = root.service
  }

  // Shape contract for shell.summon/hide/toggle routing (Bar.findPanelWidget
  // requires open/close/opened on the bar-widget root).
  readonly property bool opened: panelLoader.item ? panelLoader.item.opened === true : false
  function open() { if (panelLoader.item && panelLoader.item.open) panelLoader.item.open() }
  function close() { if (panelLoader.item && panelLoader.item.close) panelLoader.item.close() }
  function togglePanel() { if (panelLoader.item && panelLoader.item.toggle) panelLoader.item.toggle() }
  readonly property bool popoutSwitchClosing: panelLoader.item ? panelLoader.item.popoutSwitchClosing === true : false
  function closeForPopoutSwitch() { if (panelLoader.item) panelLoader.item.closeForPopoutSwitch() }

  onBarChanged: injectPanel()
  onSettingsChanged: injectPanel()
  onServiceChanged: injectPanel()

  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight

  WidgetButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    text: root.connected
      ? "󰌌 " + root.batteryPercent + "%" + (root.charging ? " 󱐋" : "")
      : "󰌌 ✕"
    dimmed: !root.connected
    active: root.opened
    useActiveColor: true
    tooltipText: root.connected
      ? "CLVX Keyboard · " + root.batteryPercent + "%" + (root.charging ? " · charging" : "") + (root.transport !== "" ? " · " + root.transport : "")
      : "CLVX Keyboard disconnected" + (root.lastError !== "" ? " · " + root.lastError : "")
    onPressed: function(mouseButton) {
      if (mouseButton === Qt.LeftButton) root.togglePanel()
    }
  }

  Loader {
    id: panelLoader
    active: true
    source: Qt.resolvedUrl("Panel.qml")
    visible: false
    onLoaded: { root.injectPanel(); Qt.callLater(root.injectPanel) }
  }
}
