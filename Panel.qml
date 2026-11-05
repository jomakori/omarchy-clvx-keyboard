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
  readonly property var serverDeviceInfo: (service && service.deviceInfo) ? service.deviceInfo : null
  readonly property bool deviceInfoReady: serverDeviceInfo !== null
  property bool factoryResetConfirmed: false

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

  function numVal(key, fallback) {
    var v = Number(val(key, fallback))
    return isFinite(v) ? Math.round(v) : fallback
  }

  function strVal(key, fallback) {
    var v = val(key, fallback)
    return v === undefined || v === null || v === "" ? fallback : String(v)
  }

  function formatBacklightTime(seconds) {
    var s = Math.max(0, Math.round(Number(seconds) || 0))
    if (s === 0) return "Always on (infinite)"
    if (s % 60 === 0 && s >= 60) return Math.round(s / 60) + " min (" + s + "s)"
    return s + "s"
  }

  function formatIdleBacklightTime(seconds) {
    var s = Math.max(0, Math.round(Number(seconds) || 0))
    if (s === 0) return "Disabled"
    return s + "s"
  }

  function formatFwVersion(packed) {
    var v = Number(packed)
    if (!isFinite(v) || v <= 0) return "—"
    var n = Math.round(v)
    return ((n >>> 16) & 0xff) + "." + ((n >>> 8) & 0xff) + "." + (n & 0xff)
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
    { value: "predefinedShortcut", label: "Def" },
    { value: "custom", label: "Bright" },
    { value: "nothing", label: "Off" },
    { value: "asGlobal", label: "Glob" },
    { value: "customShortcut", label: "Key" }
  ]
  readonly property var fKeyBehaviorOptions: [
    { value: "customShortcut", label: "Key" },
    { value: "predefinedShortcut", label: "Preset" },
    { value: "custom", label: "Cust" },
    { value: "nothing", label: "Off" },
    { value: "asGlobal", label: "Glob" }
  ]
  readonly property var backlightZoneOptionsWithAurora: [
    { value: "solidColor", label: "Solid" },
    { value: "breathing", label: "Pulse" },
    { value: "colorCycle", label: "Cycle" },
    { value: "colorWave", label: "Wave" },
    { value: "aurora", label: "Aura" },
    { value: "interactiveAnimation", label: "Touch" }
  ]
  readonly property var backlightZoneOptionsSlider: [
    { value: "solidColor", label: "Solid" },
    { value: "breathing", label: "Pulse" },
    { value: "colorCycle", label: "Cycle" },
    { value: "colorWave", label: "Wave" },
    { value: "interactiveAnimation", label: "Touch" }
  ]
  readonly property var osModeOptions: [
    { value: "LINUX", label: "Linux" },
    { value: "WINDOWS", label: "Win" },
    { value: "MAC", label: "Mac" },
    { value: "IPAD", label: "iPad" }
  ]
  property string selectedOsMode: "LINUX"

  onServerDeviceInfoChanged: {
    if (serverDeviceInfo && serverDeviceInfo.deviceMode && serverDeviceInfo.deviceMode.osMode) {
      var m = String(serverDeviceInfo.deviceMode.osMode)
      if (m === "LINUX" || m === "WINDOWS" || m === "MAC" || m === "IPAD") {
        selectedOsMode = m
      }
    }
  }

  onServerProfileChanged: {
    if (!serverProfile) return
    var next = JSON.parse(JSON.stringify(serverProfile))
    root.applyPendingProfileTo(next)
    localProfile = next
  }

  function workingProfile() {
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
      } else if (k.indexOf("backlight.") === 0) {
        var parts = k.split(".")
        var zKey = parts[1]
        var prop = parts[2]
        if (!target.backlight) target.backlight = {}
        var prevZ = target.backlight[zKey] || {}
        if (prop === "effect") {
          target.backlight[zKey] = buildBacklightZoneVariant(zKey, prevZ, String(v))
        } else if (prop === "transparency") {
          var nextT = JSON.parse(JSON.stringify(prevZ))
          nextT.transparency = Math.max(0, Math.min(100, Math.round(Number(v) || 0)))
          target.backlight[zKey] = nextT
        } else if (prop === "red" || prop === "green" || prop === "blue") {
          target.backlight[zKey] = applyBacklightChannel(zKey, prevZ, prop, v)
        }
      }
    }
  }

  function extractZoneColor(zone) {
    // Channel range unconfirmed on firmware — assume 0-255 per RGB channel.
    var fallback = { red: 255, green: 255, blue: 255 }
    if (!zone || typeof zone !== "object") return fallback
    var src = (zone.solidColor && zone.solidColor.color) ? zone.solidColor.color
      : ((zone.breathing && zone.breathing.color) ? zone.breathing.color
      : ((zone.interactiveAnimation && zone.interactiveAnimation.color) ? zone.interactiveAnimation.color : null))
    if (!src || typeof src !== "object") return fallback
    var r = Number(src.red !== undefined ? src.red : 255)
    var g = Number(src.green !== undefined ? src.green : 255)
    var b = Number(src.blue !== undefined ? src.blue : 255)
    return {
      red: isFinite(r) ? Math.max(0, Math.min(255, Math.round(r))) : 255,
      green: isFinite(g) ? Math.max(0, Math.min(255, Math.round(g))) : 255,
      blue: isFinite(b) ? Math.max(0, Math.min(255, Math.round(b))) : 255
    }
  }

  function buildBacklightZoneVariant(zoneKey, prevZone, variant) {
    var next = {}
    if (prevZone && prevZone.transparency !== undefined) {
      var tr = Number(prevZone.transparency)
      if (isFinite(tr)) next.transparency = Math.max(0, Math.min(100, Math.round(tr)))
    }
    var col = extractZoneColor(prevZone)
    var defaultPicker = {
      markersNumber: 2,
      markersArray: [
        { color: { red: 255, green: 0, blue: 0 }, transparency: 0, position: 0 },
        { color: { red: 0, green: 128, blue: 255 }, transparency: 0, position: 100 }
      ]
    }
    if (variant === "solidColor") {
      next.solidColor = (prevZone && prevZone.solidColor && typeof prevZone.solidColor === "object")
        ? JSON.parse(JSON.stringify(prevZone.solidColor))
        : { color: col }
      if (!next.solidColor.color) next.solidColor.color = col
    } else if (variant === "breathing") {
      next.breathing = (prevZone && prevZone.breathing && typeof prevZone.breathing === "object")
        ? JSON.parse(JSON.stringify(prevZone.breathing))
        : { color: col, period: 1000 }
      if (!next.breathing.color) next.breathing.color = col
    } else if (variant === "colorCycle") {
      next.colorCycle = (prevZone && prevZone.colorCycle && typeof prevZone.colorCycle === "object")
        ? JSON.parse(JSON.stringify(prevZone.colorCycle))
        : { colorLinePicker: defaultPicker, period: 1000 }
    } else if (variant === "colorWave") {
      next.colorWave = (prevZone && prevZone.colorWave && typeof prevZone.colorWave === "object")
        ? JSON.parse(JSON.stringify(prevZone.colorWave))
        : { colorLinePicker: defaultPicker, period: 1000, direction: 180, length: 400 }
    } else if (variant === "aurora" && (zoneKey === "keyboard" || zoneKey === "touchpad")) {
      next.aurora = {}
    } else if (variant === "interactiveAnimation") {
      if (prevZone && prevZone.interactiveAnimation && typeof prevZone.interactiveAnimation === "object") {
        next.interactiveAnimation = JSON.parse(JSON.stringify(prevZone.interactiveAnimation))
      } else if (zoneKey === "leftSlider" || zoneKey === "rightSlider") {
        next.interactiveAnimation = { enable: true }
      } else if (zoneKey === "touchpad") {
        next.interactiveAnimation = { color: col, enable: true, trace: 2 }
      } else {
        next.interactiveAnimation = { color: col, enable: true, duration: 5 }
      }
    } else {
      next.solidColor = { color: col }
    }
    return next
  }

  function applyBacklightChannel(zoneKey, prevZone, channel, value) {
    // Channel range unconfirmed on firmware — assume 0-255 per RGB channel.
    var chVal = Math.max(0, Math.min(255, Math.round(Number(value) || 0)))
    var eff = detectZoneEffect(prevZone, "solidColor")
    var next = buildBacklightZoneVariant(zoneKey, prevZone, eff)
    if (next.solidColor) {
      if (!next.solidColor.color) next.solidColor.color = { red: 255, green: 255, blue: 255 }
      next.solidColor.color[channel] = chVal
    } else if (next.breathing) {
      if (!next.breathing.color) next.breathing.color = { red: 255, green: 255, blue: 255 }
      next.breathing.color[channel] = chVal
    }
    return next
  }

  function detectZoneEffect(zone, fallback) {
    if (zone && typeof zone === "object") {
      if (zone.solidColor !== undefined && zone.solidColor !== null) return "solidColor"
      if (zone.breathing !== undefined && zone.breathing !== null) return "breathing"
      if (zone.colorCycle !== undefined && zone.colorCycle !== null) return "colorCycle"
      if (zone.colorWave !== undefined && zone.colorWave !== null) return "colorWave"
      if (zone.aurora !== undefined && zone.aurora !== null) return "aurora"
      if (zone.interactiveAnimation !== undefined && zone.interactiveAnimation !== null) return "interactiveAnimation"
    }
    return fallback
  }

  function backlightZoneEffect(zoneKey, fallback) {
    var k = "backlight." + zoneKey + ".effect"
    if (pendingProfile && pendingProfile[k] !== undefined) return String(pendingProfile[k])
    var p = workingProfile()
    var z = (p && p.backlight) ? p.backlight[zoneKey] : null
    return detectZoneEffect(z, fallback)
  }

  function backlightZoneTransparency(zoneKey, fallback) {
    var k = "backlight." + zoneKey + ".transparency"
    if (pendingProfile && pendingProfile[k] !== undefined) return Math.max(0, Math.min(100, Math.round(Number(pendingProfile[k]) || 0)))
    var p = workingProfile()
    var z = (p && p.backlight) ? p.backlight[zoneKey] : null
    if (z && z.transparency !== undefined) {
      var v = Number(z.transparency)
      if (isFinite(v)) return Math.max(0, Math.min(100, Math.round(v)))
    }
    return fallback
  }

  function backlightZoneChannel(zoneKey, channel, fallback) {
    // Channel range unconfirmed on firmware — assume 0-255 per RGB channel.
    var k = "backlight." + zoneKey + "." + channel
    if (pendingProfile && pendingProfile[k] !== undefined) return Math.max(0, Math.min(255, Math.round(Number(pendingProfile[k]) || 0)))
    var p = workingProfile()
    var z = (p && p.backlight) ? p.backlight[zoneKey] : null
    var col = extractZoneColor(z)
    return col[channel] !== undefined ? col[channel] : fallback
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
    var p = workingProfile()
    if (p && p.touchZone && p.touchZone.touchpad && p.touchZone.touchpad[field] !== undefined)
      return p.touchZone.touchpad[field] === true
    return fallback
  }

  function sliderSensitivity(side, fallback) {
    var k = "slider." + side + ".sensitivity"
    if (pendingProfile && pendingProfile[k] !== undefined) return Number(pendingProfile[k])
    var p = workingProfile()
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
    var p = workingProfile()
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
    var p = workingProfile()
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
    if (typeof service.getDeviceInfo === "function") service.getDeviceInfo()
  }

  function open() {
    root.factoryResetConfirmed = false
    root.controller.show()
    root.pollService()
  }
  function close() {
    root.factoryResetConfirmed = false
    root.controller.hide()
  }
  function switchPanel(direction) {
    if (root.bar && typeof root.bar.switchPanelFrom === "function")
      return root.bar.switchPanelFrom(root.barIdentity, direction)
    return false
  }

  onOpenedChanged: {
    if (opened) {
      root.factoryResetConfirmed = false
      root.pollService()
    }
  }

  Connections {
    target: root.service
    function onConnectedChanged() {
      if (root.opened && root.connected) {
        if (typeof root.service.getSettings === "function") root.service.getSettings()
        if (typeof root.service.getProfileSettings === "function") root.service.getProfileSettings()
        if (typeof root.service.getDeviceInfo === "function") root.service.getDeviceInfo()
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

  component TouchToggle: Item {
    id: tt
    property string label: ""
    property string field: ""
    property bool defaultValue: false
    width: parent ? parent.width : implicitWidth
    implicitHeight: Math.max(Style.spacing.controlHeight, ttSwitch.implicitHeight)
    enabled: root.controlsEnabled
    opacity: enabled ? 1.0 : 0.5

    function toggleField() {
      if (!tt.enabled || tt.field === "") return
      root.setField(tt.field, !root.boolVal(tt.field, tt.defaultValue))
    }

    Text {
      anchors.left: parent.left
      anchors.right: ttSwitch.left
      anchors.rightMargin: Style.space(8)
      anchors.verticalCenter: parent.verticalCenter
      textFormat: Text.PlainText
      text: tt.label
      color: root.fg
      font.family: root.fontFam
      font.pixelSize: Style.font.bodySmall
      elide: Text.ElideRight

      MouseArea {
        anchors.fill: parent
        enabled: tt.enabled
        cursorShape: Qt.PointingHandCursor
        onClicked: tt.toggleField()
      }
    }

    ToggleSwitch {
      id: ttSwitch
      anchors.right: parent.right
      anchors.verticalCenter: parent.verticalCenter
      trackHeight: Math.max(18, Math.round(Style.spacing.controlHeight * 0.5))
      cursorPad: Style.space(3)
      checked: root.boolVal(tt.field, tt.defaultValue)
      busy: !tt.enabled
      interactive: tt.enabled
      foreground: root.fg
      accent: root.accentColor
      onToggled: tt.toggleField()
    }
  }

  component ChoiceRow: Column {
    id: crow
    property string label: ""
    property string field: ""
    property string defaultValue: ""
    property var options: []
    width: parent ? parent.width : implicitWidth
    spacing: Style.space(2)
    enabled: root.controlsEnabled
    opacity: enabled ? 1.0 : 0.5

    Row {
      width: parent.width
      spacing: Style.space(6)

      Text {
        width: parent.width - crowGroup.implicitWidth - parent.spacing
        anchors.verticalCenter: parent.verticalCenter
        textFormat: Text.PlainText
        text: crow.label
        color: root.fg
        font.family: root.fontFam
        font.pixelSize: Style.font.bodySmall
        elide: Text.ElideRight
      }

      ButtonGroup {
        id: crowGroup
        anchors.verticalCenter: parent.verticalCenter
        spacing: Style.space(4)
        options: crow.options
        value: root.strVal(crow.field, crow.defaultValue)
        foreground: root.fg
        accent: root.accentColor
        fontFamily: root.fontFam
        fontSize: Style.font.caption
        onChanged: function(v) {
          if (!crow.enabled || crow.field === "") return
          root.setField(crow.field, v)
        }
      }
    }
  }

  component TouchpadZoneToggle: Item {
    id: tzt
    property string label: ""
    property string field: ""
    property bool defaultValue: true
    width: parent ? parent.width : implicitWidth
    implicitHeight: Math.max(Style.spacing.controlHeight, tztSwitch.implicitHeight)
    enabled: root.profileControlsEnabled
    opacity: enabled ? 1.0 : 0.5

    function toggleZoneField() {
      if (!tzt.enabled || tzt.field === "") return
      root.setProfileField("touchpad." + tzt.field, !root.touchpadBool(tzt.field, tzt.defaultValue))
    }

    Text {
      anchors.left: parent.left
      anchors.right: tztSwitch.left
      anchors.rightMargin: Style.space(8)
      anchors.verticalCenter: parent.verticalCenter
      textFormat: Text.PlainText
      text: tzt.label
      color: root.fg
      font.family: root.fontFam
      font.pixelSize: Style.font.bodySmall
      elide: Text.ElideRight

      MouseArea {
        anchors.fill: parent
        enabled: tzt.enabled
        cursorShape: Qt.PointingHandCursor
        onClicked: tzt.toggleZoneField()
      }
    }

    ToggleSwitch {
      id: tztSwitch
      anchors.right: parent.right
      anchors.verticalCenter: parent.verticalCenter
      trackHeight: Math.max(18, Math.round(Style.spacing.controlHeight * 0.5))
      cursorPad: Style.space(3)
      checked: root.touchpadBool(tzt.field, tzt.defaultValue)
      busy: !tzt.enabled
      interactive: tzt.enabled
      foreground: root.fg
      accent: root.accentColor
      onToggled: tzt.toggleZoneField()
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
    spacing: Style.space(3)
    enabled: root.profileControlsEnabled
    opacity: enabled ? 1.0 : 0.5

    Row {
      width: parent.width
      spacing: Style.space(6)

      Text {
        width: parent.width - sensLabel.implicitWidth - parent.spacing
        textFormat: Text.PlainText
        text: szc.title + " sensitivity"
        color: root.fg
        font.family: root.fontFam
        font.pixelSize: Style.font.bodySmall
        elide: Text.ElideRight
      }

      Text {
        id: sensLabel
        textFormat: Text.PlainText
        text: String(szc.sensValue)
        color: root.dim
        font.family: root.fontFam
        font.pixelSize: Style.font.caption
      }
    }

    PanelSlider {
      width: parent.width
      implicitHeight: Style.space(18)
      knobSize: Math.max(12, Math.round(Style.spacing.controlHeight * 0.32))
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
      color: root.dim
      font.family: root.fontFam
      font.pixelSize: Style.font.caption
      font.bold: true
      elide: Text.ElideRight
    }

    ButtonGroup {
      spacing: Style.space(4)
      options: root.sliderBehaviorOptions
      value: szc.behaviorValue
      foreground: root.fg
      accent: root.accentColor
      fontFamily: root.fontFam
      fontSize: Style.font.caption
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
    spacing: Style.space(2)
    enabled: root.profileControlsEnabled
    opacity: enabled ? 1.0 : 0.5

    Text {
      width: parent.width
      textFormat: Text.PlainText
      text: fkrow.keyName
      color: root.dim
      font.family: root.fontFam
      font.pixelSize: Style.font.caption
      font.bold: true
      elide: Text.ElideRight
    }

    ButtonGroup {
      spacing: Style.space(4)
      options: root.fKeyBehaviorOptions
      value: fkrow.behaviorValue
      foreground: root.fg
      accent: root.accentColor
      fontFamily: root.fontFam
      fontSize: Style.font.caption
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

  component GlobalSliderRow: Column {
    id: gsrow
    property string label: ""
    property string field: ""
    property int minimum: 0
    property int maximum: 100
    property int step: 1
    property int defaultValue: 0
    property string valueText: String(currentValue)
    readonly property int currentValue: root.numVal(field, defaultValue)
    width: parent ? parent.width : implicitWidth
    spacing: Style.space(2)
    enabled: root.controlsEnabled
    opacity: enabled ? 1.0 : 0.5

    Row {
      width: parent.width
      spacing: Style.space(6)

      Text {
        width: parent.width - gsValueLabel.implicitWidth - parent.spacing
        textFormat: Text.PlainText
        text: gsrow.label
        color: root.fg
        font.family: root.fontFam
        font.pixelSize: Style.font.bodySmall
        elide: Text.ElideRight
      }

      Text {
        id: gsValueLabel
        textFormat: Text.PlainText
        text: gsrow.valueText
        color: root.dim
        font.family: root.fontFam
        font.pixelSize: Style.font.caption
      }
    }

    PanelSlider {
      width: parent.width
      implicitHeight: Style.space(18)
      knobSize: Math.max(12, Math.round(Style.spacing.controlHeight * 0.32))
      tickCount: 0
      bar: root.bar
      minimum: gsrow.minimum
      maximum: gsrow.maximum
      step: gsrow.step
      integer: true
      value: gsrow.currentValue
      onReleased: function(v) {
        if (!gsrow.enabled || gsrow.field === "") return
        var snapped = Math.round(v / gsrow.step) * gsrow.step
        snapped = Math.max(gsrow.minimum, Math.min(gsrow.maximum, snapped))
        root.setField(gsrow.field, snapped)
      }
    }
  }

  component BacklightZoneControl: Column {
    id: bzc
    property string zoneKey: "keyboard"
    property string title: "Keyboard zone"
    property bool allowAurora: true
    readonly property string effectValue: root.backlightZoneEffect(zoneKey, "solidColor")
    readonly property int transparencyValue: root.backlightZoneTransparency(zoneKey, 0)
    // Channel range unconfirmed on firmware — use 0-255 sliders for active solid/breathing color.
    readonly property int redValue: root.backlightZoneChannel(zoneKey, "red", 255)
    readonly property int greenValue: root.backlightZoneChannel(zoneKey, "green", 255)
    readonly property int blueValue: root.backlightZoneChannel(zoneKey, "blue", 255)
    readonly property bool hasColorChannels: effectValue === "solidColor" || effectValue === "breathing"
    width: parent ? parent.width : implicitWidth
    spacing: Style.space(3)
    enabled: root.profileControlsEnabled
    opacity: enabled ? 1.0 : 0.5

    Text {
      width: parent.width
      textFormat: Text.PlainText
      text: bzc.title + " effect"
      color: root.dim
      font.family: root.fontFam
      font.pixelSize: Style.font.caption
      font.bold: true
      elide: Text.ElideRight
    }

    ButtonGroup {
      spacing: Style.space(4)
      options: bzc.allowAurora ? root.backlightZoneOptionsWithAurora : root.backlightZoneOptionsSlider
      value: bzc.effectValue
      foreground: root.fg
      accent: root.accentColor
      fontFamily: root.fontFam
      fontSize: Style.font.caption
      onChanged: function(v) {
        if (!bzc.enabled || bzc.zoneKey === "") return
        root.setProfileField("backlight." + bzc.zoneKey + ".effect", v)
      }
    }

    Row {
      width: parent.width
      spacing: Style.space(6)

      Text {
        width: parent.width - trLabel.implicitWidth - parent.spacing
        textFormat: Text.PlainText
        text: bzc.title + " transparency"
        color: root.fg
        font.family: root.fontFam
        font.pixelSize: Style.font.bodySmall
        elide: Text.ElideRight
      }

      Text {
        id: trLabel
        textFormat: Text.PlainText
        text: bzc.transparencyValue + "%"
        color: root.dim
        font.family: root.fontFam
        font.pixelSize: Style.font.caption
      }
    }

    PanelSlider {
      width: parent.width
      implicitHeight: Style.space(18)
      knobSize: Math.max(12, Math.round(Style.spacing.controlHeight * 0.32))
      tickCount: 0
      bar: root.bar
      minimum: 0
      maximum: 100
      step: 1
      integer: true
      value: bzc.transparencyValue
      onReleased: function(v) {
        if (!bzc.enabled || bzc.zoneKey === "") return
        root.setProfileField("backlight." + bzc.zoneKey + ".transparency", Math.round(v))
      }
    }

    Column {
      width: parent.width
      spacing: Style.space(2)
      visible: bzc.hasColorChannels

      Repeater {
        model: [
          { ch: "red", label: "R", val: bzc.redValue },
          { ch: "green", label: "G", val: bzc.greenValue },
          { ch: "blue", label: "B", val: bzc.blueValue }
        ]
        delegate: Row {
          required property var modelData
          width: parent ? parent.width : implicitWidth
          spacing: Style.space(6)

          Text {
            width: Style.space(14)
            anchors.verticalCenter: parent.verticalCenter
            textFormat: Text.PlainText
            text: modelData.label
            color: root.dim
            font.family: root.fontFam
            font.pixelSize: Style.font.caption
            font.bold: true
          }

          PanelSlider {
            width: parent.width - Style.space(14) - chNumLabel.width - parent.spacing * 2
            implicitHeight: Style.space(16)
            knobSize: Math.max(11, Math.round(Style.spacing.controlHeight * 0.3))
            tickCount: 0
            anchors.verticalCenter: parent.verticalCenter
            bar: root.bar
            minimum: 0
            maximum: 255
            step: 1
            integer: true
            value: modelData.val
            onReleased: function(v) {
              if (!bzc.enabled || bzc.zoneKey === "") return
              root.setProfileField("backlight." + bzc.zoneKey + "." + modelData.ch, Math.round(v))
            }
          }

          Text {
            id: chNumLabel
            width: Style.space(24)
            anchors.verticalCenter: parent.verticalCenter
            horizontalAlignment: Text.AlignRight
            textFormat: Text.PlainText
            text: String(modelData.val)
            color: root.dim
            font.family: root.fontFam
            font.pixelSize: Style.font.caption
          }
        }
      }
    }
  }

  component DeviceInfoRow: Item {
    id: dirow
    property string label: ""
    property string valueText: "—"
    width: parent ? parent.width : implicitWidth
    implicitHeight: Math.max(diLbl.implicitHeight, diVal.implicitHeight)

    Text {
      id: diLbl
      anchors.left: parent.left
      anchors.right: diVal.left
      anchors.rightMargin: Style.space(8)
      anchors.verticalCenter: parent.verticalCenter
      textFormat: Text.PlainText
      text: dirow.label
      color: root.dim
      font.family: root.fontFam
      font.pixelSize: Style.font.caption
      elide: Text.ElideRight
    }

    Text {
      id: diVal
      anchors.right: parent.right
      anchors.verticalCenter: parent.verticalCenter
      width: Math.min(implicitWidth, Math.round(parent.width * 0.62))
      horizontalAlignment: Text.AlignRight
      textFormat: Text.PlainText
      text: dirow.valueText
      color: root.fg
      font.family: root.fontFam
      font.pixelSize: Style.font.caption
      font.bold: true
      elide: Text.ElideRight
    }
  }

  KeyboardPanel {
    id: panel
    anchorItem: root.anchorItem
    owner: root.barIdentity
    bar: root.bar
    open: root.opened
    focusTarget: keyCatcher
    contentWidth: panel.fittedContentWidth(Style.space(300))
    contentHeight: panel.fittedContentHeight(
      column.implicitHeight,
      panel.screenH > 0 ? Math.min(Style.space(520), Math.round(panel.screenH * 0.7)) : Style.space(520)
    )

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
          spacing: Style.space(8)

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

          Column {
            width: parent.width
            spacing: Style.space(4)

            PanelSectionHeader {
              text: "Touch"
              foreground: root.fg
              fontFamily: root.fontFam
            }

            SectionNote {
              visible: !root.settingsReady
              text: "Loading…"
            }

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
                { value: "LEFT_BUTTON", label: "Left" },
                { value: "RIGHT_BUTTON", label: "Right" }
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
                { value: "UNSELECTED", label: "Auto" }
              ]
            }
          }

          PanelSeparator { width: parent.width; foreground: root.fg }

          Column {
            width: parent.width
            spacing: Style.space(4)

            PanelSectionHeader {
              text: "Sliders & zones"
              foreground: root.fg
              fontFamily: root.fontFam
            }

            SectionNote {
              visible: !root.profileReady
              text: "Loading…"
            }

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

          Column {
            width: parent.width
            spacing: Style.space(4)

            PanelSectionHeader {
              text: "Backlight"
              foreground: root.fg
              fontFamily: root.fontFam
            }

            SectionNote {
              visible: !root.settingsReady || !root.profileReady
              text: "Loading…"
            }

            GlobalSliderRow {
              label: "Brightness"
              field: "brightness"
              minimum: 0
              maximum: 100
              step: 1
              defaultValue: 0
              valueText: currentValue + "%"
            }

            TouchToggle {
              label: "Eco brightness"
              field: "ecoBrightness"
              defaultValue: false
            }

            GlobalSliderRow {
              label: "Backlight timeout"
              field: "backlightTime"
              minimum: 0
              maximum: 900
              step: 15
              defaultValue: 300
              valueText: root.formatBacklightTime(currentValue)
            }

            GlobalSliderRow {
              label: "Idle dim timeout"
              field: "idleBacklightTime"
              minimum: 0
              maximum: 300
              step: 5
              defaultValue: 30
              valueText: root.formatIdleBacklightTime(currentValue)
            }

            BacklightZoneControl {
              zoneKey: "keyboard"
              title: "Keyboard"
              allowAurora: true
            }

            BacklightZoneControl {
              zoneKey: "touchpad"
              title: "Touchpad"
              allowAurora: true
            }

            BacklightZoneControl {
              zoneKey: "leftSlider"
              title: "Left slider"
              allowAurora: false
            }

            BacklightZoneControl {
              zoneKey: "rightSlider"
              title: "Right slider"
              allowAurora: false
            }
          }

          PanelSeparator { width: parent.width; foreground: root.fg }

          Column {
            width: parent.width
            spacing: Style.space(4)

            PanelSectionHeader {
              text: "F-row"
              foreground: root.fg
              fontFamily: root.fontFam
            }

            SectionNote {
              visible: !root.profileReady
              text: "Loading…"
            }

            Repeater {
              model: root.fKeyList
              delegate: FKeyChoiceRow {
                required property string modelData
                keyName: modelData
              }
            }
          }

          PanelSeparator { width: parent.width; foreground: root.fg }

          Column {
            width: parent.width
            spacing: Style.space(5)

            PanelSectionHeader {
              text: "Device"
              foreground: root.fg
              fontFamily: root.fontFam
            }

            SectionNote {
              visible: !root.deviceInfoReady
              text: "Loading…"
            }

            Column {
              width: parent.width
              spacing: Style.space(3)

              DeviceInfoRow {
                label: "Device name"
                valueText: (root.serverDeviceInfo && root.serverDeviceInfo.name) ? String(root.serverDeviceInfo.name) : "—"
              }

              DeviceInfoRow {
                label: "Firmware version"
                valueText: root.serverDeviceInfo ? root.formatFwVersion(root.serverDeviceInfo.fwVersion) : "—"
              }

              DeviceInfoRow {
                label: "Serial number"
                valueText: (root.serverDeviceInfo && root.serverDeviceInfo.serialNumber !== undefined) ? String(root.serverDeviceInfo.serialNumber) : "—"
              }

              DeviceInfoRow {
                label: "Bar code"
                valueText: (root.serverDeviceInfo && root.serverDeviceInfo.barCode) ? String(root.serverDeviceInfo.barCode) : "—"
              }

              DeviceInfoRow {
                label: "Diagnostics"
                valueText: "Not reported"
              }
            }

            Column {
              width: parent.width
              spacing: Style.space(3)
              enabled: root.connected
              opacity: enabled ? 1.0 : 0.5

              Text {
                width: parent.width
                textFormat: Text.PlainText
                text: "OS mode"
                color: root.dim
                font.family: root.fontFam
                font.pixelSize: Style.font.caption
                font.bold: true
                elide: Text.ElideRight
              }

              Row {
                width: parent.width
                spacing: Style.space(4)

                ButtonGroup {
                  id: osModeGroup
                  anchors.verticalCenter: parent.verticalCenter
                  spacing: Style.space(4)
                  options: root.osModeOptions
                  value: root.selectedOsMode
                  foreground: root.fg
                  accent: root.accentColor
                  fontFamily: root.fontFam
                  fontSize: Style.font.caption
                  onChanged: function(v) {
                    root.selectedOsMode = v
                  }
                }

                Button {
                  anchors.verticalCenter: parent.verticalCenter
                  text: "Set"
                  bordered: true
                  focusable: true
                  enabled: root.connected
                  opacity: enabled ? 1.0 : 0.5
                  foreground: root.fg
                  accent: root.accentColor
                  fontFamily: root.fontFam
                  fontSize: Style.font.caption
                  horizontalPadding: Style.space(8)
                  verticalPadding: Style.space(4)
                  onClicked: {
                    if (!root.connected || !root.service || typeof root.service.setOsMode !== "function") return
                    root.service.setOsMode(root.selectedOsMode || "LINUX")
                  }
                }
              }
            }

            Item {
              width: parent.width
              implicitHeight: Math.max(Style.spacing.controlHeight, resetConfirmSwitch.implicitHeight)
              enabled: root.connected
              opacity: enabled ? 1.0 : 0.5

              Text {
                anchors.left: parent.left
                anchors.right: resetConfirmSwitch.left
                anchors.rightMargin: Style.space(8)
                anchors.verticalCenter: parent.verticalCenter
                textFormat: Text.PlainText
                text: "Confirm factory reset"
                color: root.fg
                font.family: root.fontFam
                font.pixelSize: Style.font.bodySmall
                elide: Text.ElideRight

                MouseArea {
                  anchors.fill: parent
                  enabled: root.connected
                  cursorShape: Qt.PointingHandCursor
                  onClicked: root.factoryResetConfirmed = !root.factoryResetConfirmed
                }
              }

              ToggleSwitch {
                id: resetConfirmSwitch
                anchors.right: parent.right
                anchors.verticalCenter: parent.verticalCenter
                trackHeight: Math.max(18, Math.round(Style.spacing.controlHeight * 0.5))
                cursorPad: Style.space(3)
                checked: root.factoryResetConfirmed
                busy: !root.connected
                interactive: root.connected
                foreground: root.fg
                accent: Color.urgent
                onToggled: {
                  if (!root.connected) return
                  root.factoryResetConfirmed = !root.factoryResetConfirmed
                }
              }
            }

            Row {
              width: parent.width
              spacing: Style.space(6)

              Button {
                width: Math.floor((parent.width - parent.spacing) / 2)
                text: "Restart keyboard"
                bordered: true
                focusable: true
                enabled: root.connected
                opacity: enabled ? 1.0 : 0.5
                foreground: root.fg
                accent: root.accentColor
                fontFamily: root.fontFam
                fontSize: Style.font.bodySmall
                verticalPadding: Style.space(5)
                onClicked: {
                  if (!root.connected || !root.service || typeof root.service.restartDevice !== "function") return
                  root.service.restartDevice()
                }
              }

              Button {
                width: parent.width - parent.children[0].width - parent.spacing
                text: "Factory reset"
                bordered: true
                focusable: true
                enabled: root.connected && root.factoryResetConfirmed
                opacity: enabled ? 1.0 : 0.5
                foreground: enabled ? Color.urgent : root.dim
                accent: Color.urgent
                fontFamily: root.fontFam
                fontSize: Style.font.bodySmall
                verticalPadding: Style.space(5)
                onClicked: {
                  if (!root.connected || !root.factoryResetConfirmed) return
                  if (!root.service || typeof root.service.factoryReset !== "function") return
                  root.factoryResetConfirmed = false
                  root.service.factoryReset(true)
                }
              }
            }
          }
        }
      }
    }
  }
}
