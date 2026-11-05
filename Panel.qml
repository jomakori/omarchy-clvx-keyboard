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
  readonly property var serverProfile: (service && service.profile) ? service.profile : null
  readonly property bool profileReady: serverProfile !== null
  readonly property bool profileControlsEnabled: connected && profileReady

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

  // Profile optimistic local state + coalesced pending writes (set_profile_settings)
  property var localProfile: null
  property var pendingProfile: ({})

  readonly property var fKeyList: ["F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12"]
  readonly property var sliderBehaviorOptions: [
    { value: "predefinedShortcut", label: "Touchpad default" },
    { value: "custom", label: "Custom brightness" },
    { value: "nothing", label: "Nothing" },
    { value: "asGlobal", label: "Global" },
    { value: "customShortcut", label: "Custom shortcut" }
  ]
  readonly property var fKeyBehaviorOptions: [
    { value: "customShortcut", label: "Custom shortcut" },
    { value: "predefinedShortcut", label: "Predefined" },
    { value: "custom", label: "Custom" },
    { value: "nothing", label: "Nothing" },
    { value: "asGlobal", label: "Global" }
  ]

  onServerProfileChanged: {
    if (!serverProfile) return
    var next = JSON.parse(JSON.stringify(serverProfile))
    root.applyPendingProfileTo(next)
    localProfile = next
  }

  function activeProfile() {
    return localProfile || serverProfile
  }

  function applyPendingProfileTo(target) {
    if (!target || typeof target !== "object") return
    var keys = Object.keys(pendingProfile)
    for (var i = 0; i < keys.length; i++) {
      var k = keys[i]
      var v = pendingProfile[k]
      if (k === "touchpad.enable" || k === "touchpad.oneFingerEnable" || k === "touchpad.twoFingerEnable") {
        if (!target.touchZone) target.touchZone = {}
        if (!target.touchZone.touchpad) target.touchZone.touchpad = {}
        var tf = k.split(".")[1]
        target.touchZone.touchpad[tf] = v === true
      } else if (k === "slider.left.sensitivity" || k === "slider.right.sensitivity") {
        var sideS = k.split(".")[1]
        if (!target.touchZone) target.touchZone = {}
        if (!target.touchZone.slider) target.touchZone.slider = {}
        if (!target.touchZone.slider[sideS]) target.touchZone.slider[sideS] = { nothing: {} }
        target.touchZone.slider[sideS].sensitivity = Math.max(1, Math.min(5, Math.round(Number(v) || 3)))
      } else if (k === "slider.left.behavior" || k === "slider.right.behavior") {
        var sideB = k.split(".")[1]
        if (!target.touchZone) target.touchZone = {}
        if (!target.touchZone.slider) target.touchZone.slider = {}
        var prevS = target.touchZone.slider[sideB] || {}
        target.touchZone.slider[sideB] = buildSliderVariant(prevS, String(v))
      } else if (k.indexOf("fKey.") === 0) {
        var fk = k.split(".")[1]
        if (!target.keyboard) target.keyboard = {}
        if (!target.keyboard.fKey) target.keyboard.fKey = {}
        var prevF = target.keyboard.fKey[fk] || {}
        target.keyboard.fKey[fk] = buildFKeyVariant(prevF, String(v))
      }
    }
  }

  function buildSliderVariant(prevSlider, variant) {
    var next = {}
    var sens = Number(prevSlider && prevSlider.sensitivity !== undefined ? prevSlider.sensitivity : 3)
    next.sensitivity = (isFinite(sens) && sens >= 1 && sens <= 5) ? Math.round(sens) : 3
    var fallbackCombo = { key1: { type: 0, code: 0 } }
    if (variant === "customShortcut") {
      var cs = (prevSlider && prevSlider.customShortcut && typeof prevSlider.customShortcut === "object")
        ? JSON.parse(JSON.stringify(prevSlider.customShortcut))
        : { inc: fallbackCombo, dec: fallbackCombo }
      next.customShortcut = cs
    } else if (variant === "predefinedShortcut") {
      var ps = (prevSlider && prevSlider.predefinedShortcut && typeof prevSlider.predefinedShortcut === "object")
        ? JSON.parse(JSON.stringify(prevSlider.predefinedShortcut))
        : { inc: fallbackCombo, dec: fallbackCombo }
      next.predefinedShortcut = ps
    } else if (variant === "custom") {
      var cVal = Number(prevSlider && prevSlider.custom !== undefined ? prevSlider.custom : 0)
      next.custom = isFinite(cVal) ? Math.max(0, Math.round(cVal)) : 0
    } else if (variant === "asGlobal") {
      next.asGlobal = {}
    } else {
      next.nothing = {}
    }
    return next
  }

  function buildFKeyVariant(prevAction, variant) {
    var next = {}
    var fallbackCombo = { key1: { type: 0, code: 0 } }
    if (variant === "customShortcut") {
      next.customShortcut = (prevAction && prevAction.customShortcut && typeof prevAction.customShortcut === "object")
        ? JSON.parse(JSON.stringify(prevAction.customShortcut))
        : fallbackCombo
    } else if (variant === "predefinedShortcut") {
      next.predefinedShortcut = (prevAction && prevAction.predefinedShortcut && typeof prevAction.predefinedShortcut === "object")
        ? JSON.parse(JSON.stringify(prevAction.predefinedShortcut))
        : fallbackCombo
    } else if (variant === "custom") {
      next.custom = {}
    } else if (variant === "nothing") {
      next.nothing = {}
    } else {
      next.asGlobal = {}
    }
    return next
  }

  function touchpadBool(field, fallback) {
    var k = "touchpad." + field
    if (pendingProfile && pendingProfile[k] !== undefined) return pendingProfile[k] === true
    var p = activeProfile()
    if (p && p.touchZone && p.touchZone.touchpad && p.touchZone.touchpad[field] !== undefined)
      return p.touchZone.touchpad[field] === true
    return fallback
  }

  function sliderSensitivity(side, fallback) {
    var k = "slider." + side + ".sensitivity"
    if (pendingProfile && pendingProfile[k] !== undefined) return Number(pendingProfile[k])
    var p = activeProfile()
    if (p && p.touchZone && p.touchZone.slider && p.touchZone.slider[side]
        && p.touchZone.slider[side].sensitivity !== undefined) {
      var v = Number(p.touchZone.slider[side].sensitivity)
      if (isFinite(v) && v >= 1 && v <= 5) return Math.round(v)
    }
    return fallback
  }

  function sliderBehavior(side, fallback) {
    var k = "slider." + side + ".behavior"
    if (pendingProfile && pendingProfile[k] !== undefined) return String(pendingProfile[k])
    var p = activeProfile()
    var s = (p && p.touchZone && p.touchZone.slider) ? p.touchZone.slider[side] : null
    if (s && typeof s === "object") {
      if (s.customShortcut !== undefined && s.customShortcut !== null) return "customShortcut"
      if (s.predefinedShortcut !== undefined && s.predefinedShortcut !== null) return "predefinedShortcut"
      if (s.custom !== undefined && s.custom !== null) return "custom"
      if (s.nothing !== undefined && s.nothing !== null) return "nothing"
      if (s.asGlobal !== undefined && s.asGlobal !== null) return "asGlobal"
    }
    return fallback
  }

  function fKeyBehavior(keyName, fallback) {
    var k = "fKey." + keyName
    if (pendingProfile && pendingProfile[k] !== undefined) return String(pendingProfile[k])
    var p = activeProfile()
    var a = (p && p.keyboard && p.keyboard.fKey) ? p.keyboard.fKey[keyName] : null
    if (a && typeof a === "object") {
      if (a.customShortcut !== undefined && a.customShortcut !== null) return "customShortcut"
      if (a.predefinedShortcut !== undefined && a.predefinedShortcut !== null) return "predefinedShortcut"
      if (a.custom !== undefined && a.custom !== null) return "custom"
      if (a.nothing !== undefined && a.nothing !== null) return "nothing"
      if (a.asGlobal !== undefined && a.asGlobal !== null) return "asGlobal"
    }
    return fallback
  }

  function setProfileField(key, value) {
    if (!profileControlsEnabled) return
    var nextPending = Object.assign({}, pendingProfile)
    nextPending[key] = value
    pendingProfile = nextPending

    var src = localProfile || serverProfile
    var base = src ? JSON.parse(JSON.stringify(src)) : { id: 0 }
    applyPendingProfileTo(base)
    if (serverProfile && serverProfile.raw !== undefined) base.raw = serverProfile.raw
    else if (service && service.profileRaw !== undefined && service.profileRaw !== null) base.raw = service.profileRaw
    localProfile = base

    profileWriteTimer.restart()
  }

  Timer {
    id: profileWriteTimer
    interval: 120
    repeat: false
    onTriggered: root.flushProfile()
  }

  function flushProfile() {
    var keys = Object.keys(pendingProfile)
    if (keys.length === 0) return
    if (!service || typeof service.setProfileSettings !== "function") return
    var src = localProfile || serverProfile
    if (!src) return
    var payload = JSON.parse(JSON.stringify(src))
    applyPendingProfileTo(payload)
    if (serverProfile && serverProfile.raw !== undefined) payload.raw = serverProfile.raw
    else if (service && service.profileRaw !== undefined && service.profileRaw !== null) payload.raw = service.profileRaw
    pendingProfile = ({})
    localProfile = payload
    service.setProfileSettings(payload)
  }

  function pollService() {
    if (!service) return
    if (typeof service.refresh === "function") service.refresh()
    if (typeof service.getSettings === "function") service.getSettings()
    if (typeof service.getProfileSettings === "function") service.getProfileSettings()
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
      if (root.opened && root.connected) {
        if (typeof root.service.getSettings === "function") root.service.getSettings()
        if (typeof root.service.getProfileSettings === "function") root.service.getProfileSettings()
      }
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

  component TouchpadZoneToggle: Toggle {
    property string field: ""
    property bool defaultValue: true
    width: parent ? parent.width : implicitWidth
    enabled: root.profileControlsEnabled
    opacity: enabled ? 1.0 : 0.5
    foreground: root.fg
    accent: root.accentColor
    fontFamily: root.fontFam
    titleSize: Style.font.body
    checked: root.touchpadBool(field, defaultValue)
    onClicked: {
      if (!enabled || field === "") return
      root.setProfileField("touchpad." + field, !root.touchpadBool(field, defaultValue))
    }
  }

  component SliderZoneControl: Column {
    id: szc
    property string side: "left"
    property string title: "Left slider"
    readonly property int sensValue: root.sliderSensitivity(side, 3)
    readonly property string behaviorValue: root.sliderBehavior(side, "nothing")
    readonly property bool isShortcutVariant: behaviorValue === "customShortcut" || behaviorValue === "predefinedShortcut"
    width: parent ? parent.width : implicitWidth
    spacing: Style.space(6)
    enabled: root.profileControlsEnabled
    opacity: enabled ? 1.0 : 0.5

    Row {
      width: parent.width
      spacing: Style.space(8)

      Text {
        width: parent.width - sensLabel.implicitWidth - parent.spacing
        textFormat: Text.PlainText
        text: szc.title + " sensitivity"
        color: root.fg
        font.family: root.fontFam
        font.pixelSize: Style.font.body
        font.bold: true
        elide: Text.ElideRight
      }

      Text {
        id: sensLabel
        textFormat: Text.PlainText
        text: String(szc.sensValue)
        color: root.dim
        font.family: root.fontFam
        font.pixelSize: Style.font.bodySmall
      }
    }

    PanelSlider {
      width: parent.width
      bar: root.bar
      minimum: 1
      maximum: 5
      step: 1
      integer: true
      tickCount: 5
      value: szc.sensValue
      onReleased: function(v) {
        if (!szc.enabled) return
        root.setProfileField("slider." + szc.side + ".sensitivity", Math.round(v))
      }
    }

    Text {
      width: parent.width
      textFormat: Text.PlainText
      text: szc.title + " behavior"
      color: root.fg
      font.family: root.fontFam
      font.pixelSize: Style.font.body
      font.bold: true
      elide: Text.ElideRight
    }

    ButtonGroup {
      options: root.sliderBehaviorOptions
      value: szc.behaviorValue
      foreground: root.fg
      accent: root.accentColor
      fontFamily: root.fontFam
      fontSize: Style.font.bodySmall
      onChanged: function(v) {
        if (!szc.enabled) return
        root.setProfileField("slider." + szc.side + ".behavior", v)
      }
    }

    SectionNote {
      visible: szc.isShortcutVariant
      text: "Shortcut capture lands later."
    }
  }

  component FKeyChoiceRow: Column {
    id: fkrow
    property string keyName: "F1"
    readonly property string behaviorValue: root.fKeyBehavior(keyName, "asGlobal")
    readonly property bool isShortcutVariant: behaviorValue === "customShortcut" || behaviorValue === "predefinedShortcut"
    width: parent ? parent.width : implicitWidth
    spacing: Style.space(6)
    enabled: root.profileControlsEnabled
    opacity: enabled ? 1.0 : 0.5

    Text {
      width: parent.width
      textFormat: Text.PlainText
      text: fkrow.keyName
      color: root.fg
      font.family: root.fontFam
      font.pixelSize: Style.font.body
      font.bold: true
      elide: Text.ElideRight
    }

    ButtonGroup {
      options: root.fKeyBehaviorOptions
      value: fkrow.behaviorValue
      foreground: root.fg
      accent: root.accentColor
      fontFamily: root.fontFam
      fontSize: Style.font.bodySmall
      onChanged: function(v) {
        if (!fkrow.enabled || fkrow.keyName === "") return
        root.setProfileField("fKey." + fkrow.keyName, v)
      }
    }

    SectionNote {
      visible: fkrow.isShortcutVariant
      text: "Shortcut capture lands later."
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
            visible: !root.profileReady
            text: "Loading…"
          }

          Column {
            width: parent.width
            spacing: Style.space(8)

            TouchpadZoneToggle {
              label: "Touchpad zone enabled"
              field: "enable"
              defaultValue: true
            }

            TouchpadZoneToggle {
              label: "Single-finger touchpad gestures"
              field: "oneFingerEnable"
              defaultValue: true
            }

            TouchpadZoneToggle {
              label: "Two-finger touchpad gestures"
              field: "twoFingerEnable"
              defaultValue: true
            }

            SliderZoneControl {
              side: "left"
              title: "Left slider"
            }

            SliderZoneControl {
              side: "right"
              title: "Right slider"
            }
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
            visible: !root.profileReady
            text: "Loading…"
          }

          Column {
            width: parent.width
            spacing: Style.space(8)

            Repeater {
              model: root.fKeyList
              delegate: FKeyChoiceRow {
                required property string modelData
                keyName: modelData
              }
            }
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
