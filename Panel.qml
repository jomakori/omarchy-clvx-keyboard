import QtQuick
import QtQuick.Controls
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
  readonly property color accentColor: Color.accent
  readonly property color dim: Qt.darker(fg, 1.4)
  readonly property string fontFam: bar ? bar.fontFamily : Style.font.family

  readonly property bool connected: service ? service.connected === true : false
  readonly property string transport: service ? String(service.transport || "") : ""
  readonly property int batteryPercent: service ? Number(service.batteryPercent) : 0
  readonly property bool charging: service ? service.charging === true : false
  readonly property string lastError: service ? String(service.lastError || "") : "CLVX service unavailable"
  readonly property var serverGlobal: (service && service.settings && service.settings.global) ? service.settings.global : null
  readonly property bool settingsReady: serverGlobal !== null
  readonly property bool controlsEnabled: connected && settingsReady

  readonly property string transportText: {
    if (!connected) return "Disconnected"
    return transport === "" ? "Connected" : transport.toUpperCase()
  }
  readonly property string detailText: connected
    ? batteryPercent + "%" + (charging ? " · charging" : "")
    : ""

  // Optimistic local state + coalesced pending writes (mirrors magic-mouse pattern)
  // so rapid toggles become one set_global per pause and failed writes surface
  // service.lastError without snapping controls back prematurely.
  property var localGlobal: null
  property var pending: ({})

  onServerGlobalChanged: {
    if (!serverGlobal) return
    var next = JSON.parse(JSON.stringify(serverGlobal))
    var keys = Object.keys(pending)
    for (var i = 0; i < keys.length; i++) {
      next[keys[i]] = pending[keys[i]]
    }
    localGlobal = next
  }

  function val(key, fallback) {
    if (pending && pending[key] !== undefined) return pending[key]
    if (localGlobal && localGlobal[key] !== undefined) return localGlobal[key]
    if (serverGlobal && serverGlobal[key] !== undefined) return serverGlobal[key]
    return fallback
  }

  function boolVal(key, fallback) {
    return val(key, fallback) === true
  }

  function strVal(key, fallback) {
    var v = val(key, fallback)
    return v === undefined || v === null || v === "" ? fallback : String(v)
  }

  function setField(key, value) {
    if (!controlsEnabled) return
    var nextPending = Object.assign({}, pending)
    nextPending[key] = value
    pending = nextPending

    var base = localGlobal ? JSON.parse(JSON.stringify(localGlobal))
      : (serverGlobal ? JSON.parse(JSON.stringify(serverGlobal)) : {})
    base[key] = value
    localGlobal = base

    writeTimer.restart()
  }

  Timer {
    id: writeTimer
    interval: 120
    repeat: false
    onTriggered: root.flush()
  }

  function flush() {
    var keys = Object.keys(pending)
    if (keys.length === 0) return
    if (!service || typeof service.setGlobal !== "function") return
    var patch = Object.assign({}, pending)
    pending = ({})
    service.setGlobal(patch)
  }

  function pollService() {
    if (!service) return
    if (typeof service.refresh === "function") service.refresh()
    if (typeof service.getSettings === "function") service.getSettings()
  }

  function open() {
    root.controller.show()
    root.pollService()
  }
  function close() { root.controller.hide() }
  function switchPanel(direction) {
    if (root.bar && typeof root.bar.switchPanelFrom === "function")
      return root.bar.switchPanelFrom(root.barIdentity, direction)
    return false
  }

  onOpenedChanged: {
    if (opened) root.pollService()
  }

  Connections {
    target: root.service
    function onConnectedChanged() {
      if (root.opened && root.connected && typeof root.service.getSettings === "function")
        root.service.getSettings()
    }
  }

  component SectionNote: Text {
    width: parent ? parent.width : implicitWidth
    textFormat: Text.PlainText
    color: root.dim
    font.family: root.fontFam
    font.pixelSize: Style.font.caption
    wrapMode: Text.WordWrap
  }

  component TouchToggle: Toggle {
    property string field: ""
    property bool defaultValue: false
    width: parent ? parent.width : implicitWidth
    enabled: root.controlsEnabled
    opacity: enabled ? 1.0 : 0.5
    foreground: root.fg
    accent: root.accentColor
    fontFamily: root.fontFam
    titleSize: Style.font.body
    checked: root.boolVal(field, defaultValue)
    onClicked: {
      if (!enabled || field === "") return
      root.setField(field, !root.boolVal(field, defaultValue))
    }
  }

  component ChoiceRow: Column {
    id: crow
    property string label: ""
    property string field: ""
    property string defaultValue: ""
    property var options: []
    width: parent ? parent.width : implicitWidth
    spacing: Style.space(6)
    enabled: root.controlsEnabled
    opacity: enabled ? 1.0 : 0.5

    Text {
      width: parent.width
      textFormat: Text.PlainText
      text: crow.label
      color: root.fg
      font.family: root.fontFam
      font.pixelSize: Style.font.body
      font.bold: true
      elide: Text.ElideRight
    }

    ButtonGroup {
      options: crow.options
      value: root.strVal(crow.field, crow.defaultValue)
      foreground: root.fg
      accent: root.accentColor
      fontFamily: root.fontFam
      fontSize: Style.font.bodySmall
      onChanged: function(v) {
        if (!crow.enabled || crow.field === "") return
        root.setField(crow.field, v)
      }
    }
  }

  KeyboardPanel {
    id: panel
    anchorItem: root.anchorItem
    owner: root.barIdentity
    bar: root.bar
    open: root.opened
    focusTarget: keyCatcher
    contentWidth: panel.fittedContentWidth(Style.space(360))
    contentHeight: panel.fittedContentHeight(column.implicitHeight, Style.space(640))

    PanelKeyCatcher {
      id: keyCatcher
      anchors.fill: parent
      onCloseRequested: root.close()
      onTabRequested: function(direction) { root.switchPanel(direction) }

      ScrollView {
        id: scrollArea
        anchors.fill: parent
        clip: true
        ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
        ScrollBar.vertical.policy: column.implicitHeight > height ? ScrollBar.AsNeeded : ScrollBar.AlwaysOff
        Binding {
          target: scrollArea.contentItem
          property: "interactive"
          value: column.implicitHeight > scrollArea.height
        }

        Column {
          id: column
          width: scrollArea.availableWidth
          spacing: Style.space(14)

          PanelHero {
            title: "CLVX Keyboard"
            meta: root.transportText
            detail: root.detailText
            foreground: root.fg
            fontFamily: root.fontFam
          }

          Text {
            width: parent.width
            visible: root.lastError !== ""
            text: root.lastError
            textFormat: Text.PlainText
            color: !root.connected ? root.dim : Color.urgent
            font.family: root.fontFam
            font.pixelSize: Style.font.caption
            wrapMode: Text.WordWrap
          }

          PanelSeparator { width: parent.width; foreground: root.fg }

          PanelSectionHeader {
            text: "Touch"
            foreground: root.fg
            fontFamily: root.fontFam
          }

          SectionNote {
            visible: !root.settingsReady
            text: "Loading…"
          }

          Column {
            width: parent.width
            spacing: Style.space(8)

            TouchToggle {
              label: "Single-finger tap clicks"
              field: "tap1fEnable"
              defaultValue: true
            }

            ChoiceRow {
              label: "Single-finger tap button"
              field: "tap1fMouseButton"
              defaultValue: "LEFT_BUTTON"
              options: [
                { value: "LEFT_BUTTON", label: "Left click" },
                { value: "RIGHT_BUTTON", label: "Right click" }
              ]
            }

            TouchToggle {
              label: "Two-finger tap right-clicks"
              field: "tap2fEnable"
              defaultValue: true
            }

            TouchToggle {
              label: "Tap twice and drag to select"
              field: "holdEnable"
              defaultValue: true
            }

            TouchToggle {
              label: "Swap primary/secondary click"
              field: "swapClickButtons"
              defaultValue: false
            }

            TouchToggle {
              label: "Gestures after finger lift"
              field: "touchActivationAfterLiftOff"
              defaultValue: false
            }

            TouchToggle {
              label: "Hold at touchpad edge"
              field: "holdDelayOnBorderEnable"
              defaultValue: true
            }

            TouchToggle {
              label: "Palm rejection"
              field: "keySuppressorEnable"
              defaultValue: true
            }

            TouchToggle {
              label: "Beginner mode"
              field: "newbieModeEnable"
              defaultValue: false
            }

            ChoiceRow {
              label: "Dominant hand"
              field: "dominantHand"
              defaultValue: "UNSELECTED"
              options: [
                { value: "LEFT", label: "Left" },
                { value: "RIGHT", label: "Right" },
                { value: "UNSELECTED", label: "Unselected" }
              ]
            }
          }

          PanelSeparator { width: parent.width; foreground: root.fg }

          PanelSectionHeader {
            text: "Sliders & zones"
            foreground: root.fg
            fontFamily: root.fontFam
          }
          SectionNote {
            text: "Left and right touch slider zone controls land in the next pass."
          }

          PanelSeparator { width: parent.width; foreground: root.fg }

          PanelSectionHeader {
            text: "Backlight"
            foreground: root.fg
            fontFamily: root.fontFam
          }
          SectionNote {
            text: "Brightness, effect mode, and auto-dim controls land in the next pass."
          }

          PanelSeparator { width: parent.width; foreground: root.fg }

          PanelSectionHeader {
            text: "F-row"
            foreground: root.fg
            fontFamily: root.fontFam
          }
          SectionNote {
            text: "Function row default behavior and media key controls land in the next pass."
          }

          PanelSeparator { width: parent.width; foreground: root.fg }

          PanelSectionHeader {
            text: "Device"
            foreground: root.fg
            fontFamily: root.fontFam
          }
          SectionNote {
            text: "Linux OS mode, firmware version, and reset/restart controls land in the next pass."
          }
        }
      }
    }
  }
}
