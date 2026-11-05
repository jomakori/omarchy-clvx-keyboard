import QtQuick
import qs.Commons
import qs.Ui

// CLVX keyboard settings panel. State comes from the plugin's service
// singleton (manifest kind "service", injected by the shell as `service`).
Panel {
  id: root
  moduleName: "io.github.joel.clvx-keyboard"
  ipcTarget: "io.github.joel.clvx-keyboard"
  manageIpc: false

  // Injected by BarWidget.injectPanel() (and by the shell for keepLoaded panels).
  property var anchorItem: null
  property var hostWidget: null
  property var service: null

  readonly property var barIdentity: hostWidget || root
  readonly property color fg: bar ? bar.foreground : Color.foreground
  readonly property color dim: Qt.darker(fg, 1.4)
  readonly property string fontFam: bar ? bar.fontFamily : Style.font.family

  readonly property bool connected: service ? service.connected === true : false
  readonly property string transport: service ? String(service.transport || "") : ""
  readonly property int batteryPercent: service ? Number(service.batteryPercent) : 0
  readonly property bool charging: service ? service.charging === true : false
  readonly property int activeProfile: service ? Number(service.activeProfile) : 0
  readonly property string lastError: service ? String(service.lastError || "") : "CLVX service unavailable"

  readonly property string transportText: {
    if (!connected) return "Disconnected"
    return transport === "" ? "Connected" : transport.toUpperCase()
  }
  readonly property string detailText: connected
    ? batteryPercent + "%" + (charging ? " · charging" : "")
    : ""

  function open() {
    root.controller.show()
    if (service && typeof service.refresh === "function") service.refresh()
  }
  function close() { root.controller.hide() }
  function switchPanel(direction) {
    if (root.bar && typeof root.bar.switchPanelFrom === "function")
      return root.bar.switchPanelFrom(root.barIdentity, direction)
    return false
  }
  function chooseProfile(value) {
    if (service && typeof service.setProfile === "function") service.setProfile(Number(value))
  }

  KeyboardPanel {
    id: panel
    anchorItem: root.anchorItem
    owner: root.barIdentity
    bar: root.bar
    open: root.opened
    focusTarget: keyCatcher
    contentWidth: panel.fittedContentWidth(Style.space(360))
    contentHeight: panel.fittedContentHeight(column.implicitHeight)

    PanelKeyCatcher {
      id: keyCatcher
      anchors.fill: parent
      onCloseRequested: root.close()
      onTabRequested: function(direction) { root.switchPanel(direction) }

      Column {
        id: column
        anchors.fill: parent
        spacing: Style.space(14)

        PanelHero {
          title: "CLVX Keyboard"
          meta: root.transportText
          detail: root.detailText
          foreground: root.fg
          fontFamily: root.fontFam
        }

        PanelSeparator { width: parent.width; foreground: root.fg }

        PanelSectionHeader {
          text: "Profile"
          foreground: root.fg
          fontFamily: root.fontFam
        }
        ButtonGroup {
          options: [0, 1, 2, 3]
          value: String(root.activeProfile)
          foreground: root.fg
          fontFamily: root.fontFam
          enabled: root.connected
          onChanged: function(value) { root.chooseProfile(value) }
        }

        Text {
          width: parent.width
          visible: !root.connected && root.lastError !== ""
          text: root.lastError
          textFormat: Text.PlainText
          color: root.dim
          font.family: root.fontFam
          font.pixelSize: Style.font.caption
          wrapMode: Text.WordWrap
        }

        PanelSeparator { width: parent.width; foreground: root.fg }

        Text {
          width: parent.width
          text: "Backlight, gestures, sliders and the F-row land in a later version."
          color: root.dim
          font.family: root.fontFam
          font.pixelSize: Style.font.caption
          wrapMode: Text.WordWrap
        }
      }
    }
  }
}
