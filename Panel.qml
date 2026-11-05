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
  readonly property var serverProfile: (service && service.profile) ? service.profile : null
  readonly property bool profileReady: serverProfile !== null
  readonly property bool profileControlsEnabled: connected && profileReady
  readonly property var serverDefaults: (service && service.deviceDefaults) ? service.deviceDefaults : null
  readonly property bool defaultsReady: serverDefaults !== null

  readonly property string transportText: {
    if (!connected) return "Disconnected"
    return transport === "" ? "Connected" : transport.toUpperCase()
  }
  readonly property string detailText: connected
    ? batteryPercent + "%" + (charging ? " · charging" : "")
    : ""

  // Profile optimistic local state + coalesced pending writes (set_profile_settings)
  property var localProfile: null
  property var pendingProfile: ({})

  readonly property var sliderBehaviorOptions: [
    { value: "predefinedShortcut", label: "Def" },
    { value: "custom", label: "Bright" },
    { value: "nothing", label: "Off" },
    { value: "asGlobal", label: "Glob" },
    { value: "customShortcut", label: "Key" }
  ]

  readonly property var touchpadEffectOptions: [
    { value: "solidColor", label: "Solid", tooltip: "Solid" },
    { value: "breathing", label: "Pulse", tooltip: "Breathing" },
    { value: "colorCycle", label: "Cycle", tooltip: "Cycle" },
    { value: "colorWave", label: "Wave", tooltip: "Wave" },
    { value: "aurora", label: "Aura", tooltip: "Aurora" },
    { value: "interactiveAnimation", label: "Touch", tooltip: "Interactive" }
  ]

  readonly property var touchpadColorPresets: [
    { id: "white",   label: "White",   red: 255, green: 255, blue: 255 },
    { id: "red",     label: "Red",     red: 255, green: 0,   blue: 0   },
    { id: "orange",  label: "Orange",  red: 255, green: 102, blue: 0   },
    { id: "amber",   label: "Amber",   red: 255, green: 191, blue: 0   },
    { id: "green",   label: "Green",   red: 0,   green: 255, blue: 0   },
    { id: "cyan",    label: "Cyan",    red: 0,   green: 255, blue: 255 },
    { id: "blue",    label: "Blue",    red: 0,   green: 0,   blue: 255 },
    { id: "magenta", label: "Magenta", red: 255, green: 0,   blue: 255 }
  ]

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
      if (k === "slider.left.sensitivity" || k === "slider.right.sensitivity") {
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
      } else if (k.indexOf("backlight.") === 0) {
        var parts = k.split(".")
        var zKey = parts[1]
        var prop = parts[2]
        if (!target.backlight) target.backlight = {}
        var prevZ = target.backlight[zKey] || {}
        if (prop === "effect") {
          target.backlight[zKey] = buildBacklightZoneVariant(zKey, prevZ, String(v))
        } else if (prop === "color") {
          target.backlight[zKey] = applyBacklightColor(zKey, prevZ, v)
        } else if (prop === "red" || prop === "green" || prop === "blue") {
          target.backlight[zKey] = applyBacklightChannel(zKey, prevZ, prop, v)
        }
      }
    }
  }

  function extractZoneColor(zone) {
    var fallback = { red: 255, green: 255, blue: 255 }
    if (!zone || typeof zone !== "object") return fallback
    var src = (zone.solidColor && zone.solidColor.color) ? zone.solidColor.color
      : ((zone.breathing && zone.breathing.color) ? zone.breathing.color
      : ((zone.interactiveAnimation && zone.interactiveAnimation.color) ? zone.interactiveAnimation.color
      : (zone.color ? zone.color : null)))
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

  function applyBacklightColor(zoneKey, prevZone, colorObj) {
    var r = Math.max(0, Math.min(255, Math.round(Number(colorObj && colorObj.red !== undefined ? colorObj.red : 255))))
    var g = Math.max(0, Math.min(255, Math.round(Number(colorObj && colorObj.green !== undefined ? colorObj.green : 255))))
    var b = Math.max(0, Math.min(255, Math.round(Number(colorObj && colorObj.blue !== undefined ? colorObj.blue : 255))))
    var col = { red: r, green: g, blue: b }
    var eff = detectZoneEffect(prevZone, "solidColor")
    var next = buildBacklightZoneVariant(zoneKey, prevZone, eff)
    if (next.solidColor) {
      next.solidColor.color = col
    } else if (next.breathing) {
      next.breathing.color = col
    } else if (next.interactiveAnimation && typeof next.interactiveAnimation === "object") {
      next.interactiveAnimation.color = col
    } else {
      next.color = col
    }
    return next
  }

  function applyBacklightChannel(zoneKey, prevZone, channel, value) {
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

  function backlightZoneChannel(zoneKey, channel, fallback) {
    var colorKey = "backlight." + zoneKey + ".color"
    if (pendingProfile && pendingProfile[colorKey] && pendingProfile[colorKey][channel] !== undefined) {
      return Math.max(0, Math.min(255, Math.round(Number(pendingProfile[colorKey][channel]) || 0)))
    }
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

  function resetProfileSetting(path) {
    if (!profileControlsEnabled || !defaultsReady) return
    if (!service || typeof service.resetProfileField !== "function") return
    var nextPending = Object.assign({}, pendingProfile)
    delete nextPending[path]
    if (path === "backlight.touchpad.color") {
      delete nextPending["backlight.touchpad.red"]
      delete nextPending["backlight.touchpad.green"]
      delete nextPending["backlight.touchpad.blue"]
    }
    pendingProfile = nextPending
    service.resetProfileField(path)
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
    if (typeof service.getProfileSettings === "function") service.getProfileSettings()
    if (!service.deviceDefaults) {
      if (typeof service.getDefaultSettings === "function") service.getDefaultSettings()
      else if (typeof service.getDeviceDefaults === "function") service.getDeviceDefaults()
    }
  }

  function open() {
    root.controller.show()
    root.pollService()
  }
  function close() {
    root.controller.hide()
  }
  function switchPanel(direction) {
    if (root.bar && typeof root.bar.switchPanelFrom === "function")
      return root.bar.switchPanelFrom(root.barIdentity, direction)
    return false
  }

  onOpenedChanged: {
    if (opened) {
      root.pollService()
    }
  }

  Connections {
    target: root.service
    function onConnectedChanged() {
      if (root.opened && root.connected) {
        if (typeof root.service.getProfileSettings === "function") root.service.getProfileSettings()
        if (!root.service.deviceDefaults) {
          if (typeof root.service.getDefaultSettings === "function") root.service.getDefaultSettings()
          else if (typeof root.service.getDeviceDefaults === "function") root.service.getDeviceDefaults()
        }
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

  component ResetButton: Item {
    id: rbtn
    property string resetPath: ""
    property string resetLabel: "Reset to default"
    readonly property bool canReset: root.profileControlsEnabled && root.defaultsReady
    readonly property string effectiveTooltip: root.defaultsReady ? resetLabel : "Defaults unavailable"
    implicitWidth: innerBtn.implicitWidth
    implicitHeight: innerBtn.implicitHeight

    Button {
      id: innerBtn
      iconText: "󰑐"
      iconSize: Style.font.iconSmall
      horizontalPadding: Style.space(5)
      verticalPadding: Style.space(3)
      foreground: rbtn.canReset ? root.dim : Qt.darker(root.dim, 1.4)
      accent: root.accentColor
      fontFamily: root.fontFam
      enabled: rbtn.canReset
      opacity: rbtn.canReset ? 1.0 : 0.45
      tooltipText: rbtn.effectiveTooltip
      onClicked: {
        if (!rbtn.canReset || rbtn.resetPath === "") return
        root.resetProfileSetting(rbtn.resetPath)
      }
    }

    HoverHandler {
      id: disabledHover
    }

    PanelToolTip {
      visible: !rbtn.canReset && disabledHover.hovered && rbtn.effectiveTooltip !== ""
      text: rbtn.effectiveTooltip
      fontFamily: root.fontFam
    }
  }

  component CompactButtonGroup: ButtonGroup {
    id: cbg
    property real chipPaddingX: Style.space(4)
    property real chipPaddingY: Style.space(3)
    spacing: Style.space(3)
    foreground: root.fg
    accent: root.accentColor
    fontFamily: root.fontFam
    fontSize: Style.font.caption

    function tightenChips() {
      for (var i = 0; i < cbg.children.length; i++) {
        var child = cbg.children[i]
        if (child && child.horizontalPadding !== undefined) {
          child.horizontalPadding = cbg.chipPaddingX
          child.verticalPadding = cbg.chipPaddingY
        }
      }
    }

    Component.onCompleted: tightenChips()
    onOptionsChanged: Qt.callLater(tightenChips)
    onChipPaddingXChanged: tightenChips()
    onChipPaddingYChanged: tightenChips()
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

    Item {
      width: parent.width
      implicitHeight: Math.max(sensSlider.implicitHeight, sensResetBtn.implicitHeight)

      PanelSlider {
        id: sensSlider
        anchors.left: parent.left
        anchors.right: sensResetBtn.left
        anchors.rightMargin: Style.space(6)
        anchors.verticalCenter: parent.verticalCenter
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

      ResetButton {
        id: sensResetBtn
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        resetPath: "slider." + szc.side + ".sensitivity"
        resetLabel: "Reset " + szc.title.toLowerCase() + " sensitivity"
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

    Item {
      width: parent.width
      implicitHeight: Math.max(behGroup.implicitHeight, behResetBtn.implicitHeight)

      CompactButtonGroup {
        id: behGroup
        anchors.left: parent.left
        anchors.verticalCenter: parent.verticalCenter
        chipPaddingX: Style.space(5)
        options: root.sliderBehaviorOptions
        value: szc.behaviorValue
        onChanged: function(v) {
          if (!szc.enabled) return
          root.setProfileField("slider." + szc.side + ".behavior", v)
        }
      }

      ResetButton {
        id: behResetBtn
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        resetPath: "slider." + szc.side + ".behavior"
        resetLabel: "Reset " + szc.title.toLowerCase() + " behavior"
      }
    }

    SectionNote {
      visible: szc.isShortcutVariant
      text: "Shortcut capture lands later."
    }
  }

  component TouchpadLightControl: Column {
    id: tplc
    readonly property string effectValue: root.backlightZoneEffect("touchpad", "solidColor")
    readonly property int redValue: root.backlightZoneChannel("touchpad", "red", 255)
    readonly property int greenValue: root.backlightZoneChannel("touchpad", "green", 255)
    readonly property int blueValue: root.backlightZoneChannel("touchpad", "blue", 255)
    width: parent ? parent.width : implicitWidth
    spacing: Style.space(4)
    enabled: root.profileControlsEnabled
    opacity: enabled ? 1.0 : 0.5

    Text {
      width: parent.width
      textFormat: Text.PlainText
      text: "Effect"
      color: root.dim
      font.family: root.fontFam
      font.pixelSize: Style.font.caption
      font.bold: true
      elide: Text.ElideRight
    }

    Item {
      width: parent.width
      implicitHeight: Math.max(tpEffectGroup.implicitHeight, tpEffectResetBtn.implicitHeight)

      CompactButtonGroup {
        id: tpEffectGroup
        anchors.left: parent.left
        anchors.verticalCenter: parent.verticalCenter
        chipPaddingX: Style.space(4)
        options: root.touchpadEffectOptions
        value: tplc.effectValue
        onChanged: function(v) {
          if (!tplc.enabled) return
          root.setProfileField("backlight.touchpad.effect", v)
        }
      }

      ResetButton {
        id: tpEffectResetBtn
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        resetPath: "backlight.touchpad.effect"
        resetLabel: "Reset touchpad effect"
      }
    }

    Text {
      width: parent.width
      textFormat: Text.PlainText
      text: "Color"
      color: root.dim
      font.family: root.fontFam
      font.pixelSize: Style.font.caption
      font.bold: true
      elide: Text.ElideRight
    }

    Item {
      width: parent.width
      implicitHeight: Math.max(swatchRow.implicitHeight, tpColorResetBtn.implicitHeight)

      Row {
        id: swatchRow
        anchors.left: parent.left
        anchors.verticalCenter: parent.verticalCenter
        spacing: Style.space(6)

        Repeater {
          model: root.touchpadColorPresets
          delegate: Item {
            required property var modelData
            readonly property bool isSelected: tplc.redValue === modelData.red
              && tplc.greenValue === modelData.green
              && tplc.blueValue === modelData.blue
            readonly property color swatchColor: Qt.rgba(
              modelData.red / 255,
              modelData.green / 255,
              modelData.blue / 255,
              1.0
            )
            width: Style.space(22)
            height: Style.space(22)

            Rectangle {
              anchors.fill: parent
              radius: width / 2
              color: "transparent"
              border.width: parent.isSelected
                ? Math.max(2, Style.space(2))
                : (swatchMouse.containsMouse ? Math.max(1, Style.space(1)) : 0)
              border.color: parent.isSelected ? root.fg : root.dim

              Rectangle {
                anchors.centerIn: parent
                width: parent.parent.isSelected ? parent.width - Style.space(6) : parent.width - Style.space(4)
                height: width
                radius: width / 2
                color: parent.parent.swatchColor
                border.width: 1
                border.color: Qt.rgba(root.fg.r, root.fg.g, root.fg.b, 0.22)
                scale: swatchMouse.containsMouse && tplc.enabled ? 1.12 : 1.0

                Behavior on scale {
                  NumberAnimation { duration: 110; easing.type: Easing.OutCubic }
                }
              }
            }

            MouseArea {
              id: swatchMouse
              anchors.fill: parent
              hoverEnabled: true
              enabled: tplc.enabled
              cursorShape: tplc.enabled ? Qt.PointingHandCursor : Qt.ArrowCursor
              onClicked: {
                if (!tplc.enabled) return
                root.setProfileField("backlight.touchpad.color", {
                  red: modelData.red,
                  green: modelData.green,
                  blue: modelData.blue
                })
              }
            }

            PanelToolTip {
              visible: swatchMouse.containsMouse
              text: modelData.label
              fontFamily: root.fontFam
            }
          }
        }
      }

      ResetButton {
        id: tpColorResetBtn
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        resetPath: "backlight.touchpad.color"
        resetLabel: "Reset touchpad color"
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
              text: "Sliders"
              foreground: root.fg
              fontFamily: root.fontFam
            }

            SectionNote {
              visible: !root.profileReady
              text: "Loading…"
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
              text: "Touchpad"
              foreground: root.fg
              fontFamily: root.fontFam
            }

            SectionNote {
              visible: !root.profileReady
              text: "Loading…"
            }

            TouchpadLightControl {}
          }
        }
      }
    }
  }
}
