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

  readonly property string connectionLabel: {
    if (!connected) return "Not connected"
    var t = transport.toLowerCase()
    if (t.indexOf("ble") === 0) return "Bluetooth"
    if (t.indexOf("usb") === 0) return "USB"
    return "Connected"
  }
  // Pill colors: blue = Bluetooth, green = USB, orange = disconnected.
  // Other errors stay red via the Color.urgent error line below.
  readonly property color connectionDotColor: {
    if (!connected) return "#d9903f"
    var t = transport.toLowerCase()
    if (t.indexOf("usb") === 0) return "#7fb069"
    return "#6aa8ff"
  }
  readonly property string detailText: connected
    ? batteryPercent + "%" + (charging ? " ⚡" : "")
    : ""
  readonly property bool showErrorLine: {
    if (!connected || lastError === "") return false
    var low = lastError.toLowerCase()
    if (low.indexOf("unavailable") >= 0
        || low.indexOf("not connected") >= 0
        || low.indexOf("disconnected") >= 0) {
      return false
    }
    return true
  }

  // Profile optimistic local state + coalesced pending writes (set_profile_settings)
  property var localProfile: null
  property var pendingProfile: ({})
  property string selectedBacklightZone: "keyboard"
  property var touchModeZones: ({})
  property var zonePickedColors: ({})

  readonly property var sliderBehaviorOptions: [
    { value: "keyboardBrightness", label: "Keyboard Brightness", tooltip: "Keyboard Brightness" },
    { value: "volume",             label: "Volume",              tooltip: "Volume" },
    { value: "screenBrightness",   label: "Screen Brightness",   tooltip: "Screen Brightness" }
  ]

  readonly property var backlightZoneMeta: ({
    keyboard:    { key: "keyboard",    label: "Keyboard",     sublabel: "Background", hasAurora: true  },
    touchpad:    { key: "touchpad",    label: "Touchpad",     sublabel: "Touch Area", hasAurora: true  },
    leftSlider:  { key: "leftSlider",  label: "Left Slider",  sublabel: "Left strip", hasAurora: false },
    rightSlider: { key: "rightSlider", label: "Right Slider", sublabel: "Right strip", hasAurora: false }
  })

  readonly property var allBacklightEffectOptions: [
    { value: "solidColor",           label: "Solid",     tooltip: "Solid" },
    { value: "breathing",            label: "Breathing", tooltip: "Breathing" },
    { value: "colorCycle",           label: "Cycle",     tooltip: "Cycle" },
    { value: "colorWave",            label: "Wave",      tooltip: "Wave" },
    { value: "aurora",               label: "Aura",      tooltip: "Aurora" },
    { value: "interactiveAnimation", label: "Touch",     tooltip: "Interactive" }
  ]

  readonly property var sliderBacklightEffectOptions: [
    { value: "solidColor",           label: "Solid",     tooltip: "Solid" },
    { value: "breathing",            label: "Breathing", tooltip: "Breathing" },
    { value: "colorCycle",           label: "Cycle",     tooltip: "Cycle" },
    { value: "colorWave",            label: "Wave",      tooltip: "Wave" },
    { value: "interactiveAnimation", label: "Touch",     tooltip: "Interactive" }
  ]

  readonly property var backlightColorPresets: [
    { id: "white",   label: "White",   red: 255, green: 255, blue: 255 },
    { id: "red",     label: "Red",     red: 255, green: 0,   blue: 0   },
    { id: "orange",  label: "Orange",  red: 255, green: 102, blue: 0   },
    { id: "amber",   label: "Amber",   red: 255, green: 191, blue: 0   },
    { id: "green",   label: "Green",   red: 0,   green: 255, blue: 0   },
    { id: "cyan",    label: "Cyan",    red: 0,   green: 255, blue: 255 },
    { id: "blue",    label: "Blue",    red: 0,   green: 0,   blue: 255 },
    { id: "magenta", label: "Magenta", red: 255, green: 0,   blue: 255 }
  ]

  function effectOptionsForZone(zoneKey) {
    return (zoneKey === "keyboard" || zoneKey === "touchpad")
      ? root.allBacklightEffectOptions
      : root.sliderBacklightEffectOptions
  }

  function effectDisplayLabel(effectVal) {
    if (effectVal === "solidColor") return "Solid"
    if (effectVal === "breathing") return "Breathing"
    if (effectVal === "colorCycle") return "Cycle"
    if (effectVal === "colorWave") return "Wave"
    if (effectVal === "aurora") return "Aurora"
    if (effectVal === "interactiveAnimation") return "Touch"
    return "Solid"
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
      if (k === "slider.left.sensitivity" || k === "slider.right.sensitivity") {
        var sideS = k.split(".")[1]
        if (!target.touchZone) target.touchZone = {}
        if (!target.touchZone.slider) target.touchZone.slider = {}
        if (!target.touchZone.slider[sideS]) target.touchZone.slider[sideS] = { custom: 0 }
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

  function extractZoneColor(zone, zoneKey) {
    var fallback = { red: 255, green: 255, blue: 255 }
    if (zoneKey && zonePickedColors && zonePickedColors[zoneKey]) {
      var picked = zonePickedColors[zoneKey]
      if (picked && typeof picked === "object" && picked.red !== undefined) {
        return {
          red: Math.max(0, Math.min(255, Math.round(Number(picked.red) || 0))),
          green: Math.max(0, Math.min(255, Math.round(Number(picked.green) || 0))),
          blue: Math.max(0, Math.min(255, Math.round(Number(picked.blue) || 0)))
        }
      }
    }
    if (!zone || typeof zone !== "object") return fallback
    // Dots always reflect the base zone color; action (trace) colors are pinned.
    var src = null
    if (zone.solidColor && zone.solidColor.color) {
      src = zone.solidColor.color
    } else if (zone.breathing && zone.breathing.color) {
      src = zone.breathing.color
    } else if (zone.interactiveAnimation && zone.interactiveAnimation.color) {
      src = zone.interactiveAnimation.color
    } else if (zone.colorCycle && zone.colorCycle.colorLinePicker
               && zone.colorCycle.colorLinePicker.markersArray
               && zone.colorCycle.colorLinePicker.markersArray.length > 0
               && zone.colorCycle.colorLinePicker.markersArray[0].color) {
      src = zone.colorCycle.colorLinePicker.markersArray[0].color
    } else if (zone.colorWave && zone.colorWave.colorLinePicker
               && zone.colorWave.colorLinePicker.markersArray
               && zone.colorWave.colorLinePicker.markersArray.length > 0
               && zone.colorWave.colorLinePicker.markersArray[0].color) {
      src = zone.colorWave.colorLinePicker.markersArray[0].color
    } else if (zone.color) {
      src = zone.color
    }
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
    var col = extractZoneColor(prevZone, zoneKey)
    var secondCol = { red: 0, green: 128, blue: 255 }
    var defaultPicker = {
      markersNumber: 2,
      markersArray: [
        { color: col, transparency: 0, position: 0 },
        { color: secondCol, transparency: 0, position: 100 }
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
      next.solidColor = (prevZone && prevZone.solidColor && typeof prevZone.solidColor === "object")
        ? JSON.parse(JSON.stringify(prevZone.solidColor))
        : { color: col }
      if (!next.solidColor.color) next.solidColor.color = col
      if (prevZone && prevZone.interactiveAnimation && typeof prevZone.interactiveAnimation === "object") {
        next.interactiveAnimation = JSON.parse(JSON.stringify(prevZone.interactiveAnimation))
        next.interactiveAnimation.enable = true
        // Action colors are pinned: typing trace blue, glide trace green.
        // Never adopt the picked base color here.
        if (!next.interactiveAnimation.color || typeof next.interactiveAnimation.color !== "object") {
          next.interactiveAnimation.color = zoneKey === "touchpad"
            ? { red: 0, green: 255, blue: 0 }
            : { red: 0, green: 0, blue: 255 }
        }
      } else if (zoneKey === "leftSlider" || zoneKey === "rightSlider") {
        next.interactiveAnimation = { enable: true }
      } else if (zoneKey === "touchpad") {
        next.interactiveAnimation = { color: { red: 0, green: 255, blue: 0 }, enable: true, trace: 2 }
      } else {
        next.interactiveAnimation = { color: { red: 0, green: 0, blue: 255 }, enable: true, duration: 5 }
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
    var eff = detectZoneEffect(prevZone, "solidColor", zoneKey)
    var next = (prevZone && typeof prevZone === "object" && Object.keys(prevZone).length > 0)
      ? JSON.parse(JSON.stringify(prevZone))
      : buildBacklightZoneVariant(zoneKey, prevZone, eff)
    if (eff === "interactiveAnimation") {
      if (!next.solidColor || typeof next.solidColor !== "object") next.solidColor = { color: col }
      else next.solidColor.color = JSON.parse(JSON.stringify(col))
      // Pinned action colors: typing blue, glide green — dots only set the base.
    } else if (next.solidColor && typeof next.solidColor === "object") {
      next.solidColor.color = JSON.parse(JSON.stringify(col))
    } else if (next.breathing && typeof next.breathing === "object") {
      next.breathing.color = JSON.parse(JSON.stringify(col))
    } else if (next.colorCycle && next.colorCycle.colorLinePicker
               && next.colorCycle.colorLinePicker.markersArray
               && next.colorCycle.colorLinePicker.markersArray.length > 0) {
      next.colorCycle.colorLinePicker.markersArray[0].color = JSON.parse(JSON.stringify(col))
    } else if (next.colorWave && next.colorWave.colorLinePicker
               && next.colorWave.colorLinePicker.markersArray
               && next.colorWave.colorLinePicker.markersArray.length > 0) {
      next.colorWave.colorLinePicker.markersArray[0].color = JSON.parse(JSON.stringify(col))
    } else {
      next = buildBacklightZoneVariant(zoneKey, prevZone, "solidColor")
      next.solidColor.color = JSON.parse(JSON.stringify(col))
    }
    return next
  }

  function applyBacklightChannel(zoneKey, prevZone, channel, value) {
    var chVal = Math.max(0, Math.min(255, Math.round(Number(value) || 0)))
    var eff = detectZoneEffect(prevZone, "solidColor", zoneKey)
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

  function detectZoneEffect(zone, fallback, zoneKey) {
    if (zoneKey && touchModeZones && touchModeZones[zoneKey] === true
        && zone && zone.interactiveAnimation !== undefined && zone.interactiveAnimation !== null) {
      return "interactiveAnimation"
    }
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
    return detectZoneEffect(z, fallback, zoneKey)
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
    var col = extractZoneColor(z, zoneKey)
    return col[channel] !== undefined ? col[channel] : fallback
  }

  function buildSliderVariant(prevSlider, variant) {
    var sens = Number(prevSlider && prevSlider.sensitivity !== undefined ? prevSlider.sensitivity : 3)
    var cleanSens = (isFinite(sens) && sens >= 1 && sens <= 5) ? Math.round(sens) : 3
    if (variant === "keyboardBrightness" || variant === "Keyboard Brightness" || variant === "custom") {
      return {
        sensitivity: cleanSens,
        custom: 0
      }
    }
    if (variant === "volume" || variant === "Volume") {
      return {
        sensitivity: cleanSens,
        predefinedShortcut: {
          inc: { key1: { type: 1, code: 233 } },
          dec: { key1: { type: 1, code: 234 } }
        }
      }
    }
    if (variant === "screenBrightness" || variant === "Screen Brightness") {
      return {
        sensitivity: cleanSens,
        predefinedShortcut: {
          inc: { key1: { type: 1, code: 111 } },
          dec: { key1: { type: 1, code: 112 } }
        }
      }
    }
    return {
      sensitivity: cleanSens,
      custom: 0
    }
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

  function matchesShortcutCodes(shortcutObj, incCode, decCode) {
    if (!shortcutObj || typeof shortcutObj !== "object") return false
    var incKey = (shortcutObj.inc && shortcutObj.inc.key1) ? shortcutObj.inc.key1 : null
    var decKey = (shortcutObj.dec && shortcutObj.dec.key1) ? shortcutObj.dec.key1 : null
    if (!incKey || !decKey) return false
    return Number(incKey.type) === 1
      && Number(incKey.code) === incCode
      && Number(decKey.type) === 1
      && Number(decKey.code) === decCode
  }

  function sliderBehavior(side, fallback) {
    var k = "slider." + side + ".behavior"
    if (pendingProfile && pendingProfile[k] !== undefined) return String(pendingProfile[k])
    var p = workingProfile()
    var s = (p && p.touchZone && p.touchZone.slider) ? p.touchZone.slider[side] : null
    if (s && typeof s === "object") {
      if (s.custom !== undefined && s.custom !== null && Number(s.custom) === 0) return "keyboardBrightness"
      if (matchesShortcutCodes(s.predefinedShortcut, 233, 234)) return "volume"
      if (matchesShortcutCodes(s.predefinedShortcut, 111, 112)) return "screenBrightness"
    }
    return fallback
  }

  function setProfileField(key, value) {
    if (!profileControlsEnabled) return
    if (key.indexOf("backlight.") === 0) {
      var parts = key.split(".")
      var zKey = parts[1]
      var prop = parts[2]
      if (prop === "effect") {
        var nextTouch = Object.assign({}, touchModeZones)
        nextTouch[zKey] = (String(value) === "interactiveAnimation")
        touchModeZones = nextTouch
      } else if (prop === "color" && value && typeof value === "object") {
        var nextColors = Object.assign({}, zonePickedColors)
        nextColors[zKey] = {
          red: Number(value.red !== undefined ? value.red : 255),
          green: Number(value.green !== undefined ? value.green : 255),
          blue: Number(value.blue !== undefined ? value.blue : 255)
        }
        zonePickedColors = nextColors
      }
    }

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
    if (path === "slider.left" || path === "slider.right"
        || path === "touchZone.slider.left" || path === "touchZone.slider.right") {
      var side = path.split(".").pop()
      delete nextPending["slider." + side]
      delete nextPending["slider." + side + ".sensitivity"]
      delete nextPending["slider." + side + ".behavior"]
      delete nextPending["touchZone.slider." + side]
    } else if (path.indexOf("backlight.") === 0) {
      var blParts = path.split(".")
      var zKey = blParts[1]
      if (blParts.length === 2) {
        delete nextPending["backlight." + zKey + ".effect"]
        delete nextPending["backlight." + zKey + ".color"]
        delete nextPending["backlight." + zKey + ".red"]
        delete nextPending["backlight." + zKey + ".green"]
        delete nextPending["backlight." + zKey + ".blue"]
        var nextTouch = Object.assign({}, touchModeZones)
        delete nextTouch[zKey]
        touchModeZones = nextTouch
        var nextColors = Object.assign({}, zonePickedColors)
        delete nextColors[zKey]
        zonePickedColors = nextColors
      } else if (blParts[2] === "color") {
        delete nextPending["backlight." + zKey + ".red"]
        delete nextPending["backlight." + zKey + ".green"]
        delete nextPending["backlight." + zKey + ".blue"]
        var nextCols = Object.assign({}, zonePickedColors)
        delete nextCols[zKey]
        zonePickedColors = nextCols
      }
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
    profileWriteTimer.stop()
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
      horizontalPadding: Style.space(4)
      verticalPadding: Style.space(2)
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
    property real chipPaddingX: Style.space(3)
    property real chipPaddingY: Style.space(2)
    spacing: Style.space(2)
    foreground: root.fg
    accent: root.accentColor
    fontFamily: root.fontFam
    fontSize: Math.max(9, Style.font.caption - 1)

    function tightenChips() {
      for (var i = 0; i < cbg.children.length; i++) {
        var child = cbg.children[i]
        if (child && child.horizontalPadding !== undefined) {
          child.horizontalPadding = cbg.chipPaddingX
          child.verticalPadding = cbg.chipPaddingY
          child.fontSize = cbg.fontSize
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
    readonly property string behaviorValue: root.sliderBehavior(side, "")
    width: parent ? parent.width : implicitWidth
    spacing: Style.space(2)
    enabled: root.profileControlsEnabled
    opacity: enabled ? 1.0 : 0.5

    Item {
      width: parent.width
      implicitHeight: Math.max(sliderTitleText.implicitHeight, sliderResetBtn.implicitHeight)

      Text {
        id: sliderTitleText
        anchors.left: parent.left
        anchors.right: sensMetaText.left
        anchors.rightMargin: Style.space(6)
        anchors.verticalCenter: parent.verticalCenter
        textFormat: Text.PlainText
        text: szc.title
        color: root.fg
        font.family: root.fontFam
        font.pixelSize: Style.font.bodySmall
        font.bold: true
        elide: Text.ElideRight
      }

      Text {
        id: sensMetaText
        anchors.right: sliderResetBtn.left
        anchors.rightMargin: Style.space(6)
        anchors.verticalCenter: parent.verticalCenter
        textFormat: Text.PlainText
        text: "Sens " + String(sensSlider.dragging ? Math.round(sensSlider.liveValue) : szc.sensValue)
        color: root.dim
        font.family: root.fontFam
        font.pixelSize: Style.font.caption
      }

      ResetButton {
        id: sliderResetBtn
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        resetPath: "slider." + szc.side
        resetLabel: "Reset " + szc.title.toLowerCase()
      }
    }

    PanelSlider {
      id: sensSlider
      width: parent.width
      implicitHeight: Style.space(14)
      knobSize: Math.max(10, Math.round(Style.spacing.controlHeight * 0.28))
      // The stock selected-fill alpha nearly disappears against a panel card,
      // which makes the filled part read as a half-width control. Keep the
      // full-width groove visible so the track's extent is obvious.
      trackColor: Qt.rgba(root.fg.r, root.fg.g, root.fg.b, 0.24)
      trackHeight: Math.max(6, Style.space(6))
      bar: root.bar
      minimum: 1
      maximum: 5
      step: 1
      integer: true
      tickCount: 5
      value: szc.sensValue
      onMoved: function(v) {
        if (!szc.enabled) return
        var sv = Math.round(v)
        if (sv === szc.sensValue) return
        root.setProfileField("slider." + szc.side + ".sensitivity", sv)
      }
      onReleased: function(v) {
        if (!szc.enabled) return
        var sv = Math.round(v)
        if (sv !== szc.sensValue) {
          root.setProfileField("slider." + szc.side + ".sensitivity", sv)
        }
        root.flushProfile()
      }
    }

    CompactButtonGroup {
      id: behGroup
      anchors.horizontalCenter: parent.horizontalCenter
      chipPaddingX: Style.space(3)
      options: root.sliderBehaviorOptions
      value: szc.behaviorValue
      onChanged: function(v) {
        if (!szc.enabled) return
        root.setProfileField("slider." + szc.side + ".behavior", v)
        root.flushProfile()
      }
    }
  }

  component ZoneMapBlock: BorderSurface {
    id: zmb
    property string zoneKey: "keyboard"
    property string labelText: "Keyboard"
    property string subText: ""
    property int labelAlignment: Qt.AlignVCenter
    property int labelTopMargin: Style.space(6)
    property bool showSliderTicks: false
    readonly property bool isSelected: root.selectedBacklightZone === zoneKey
    readonly property bool mouseHovered: blockMouse.containsMouse
    property bool isHot: mouseHovered
    readonly property int zoneRed: root.backlightZoneChannel(zoneKey, "red", 255)
    readonly property int zoneGreen: root.backlightZoneChannel(zoneKey, "green", 255)
    readonly property int zoneBlue: root.backlightZoneChannel(zoneKey, "blue", 255)
    readonly property color liveZoneColor: Qt.rgba(
      zoneRed / 255,
      zoneGreen / 255,
      zoneBlue / 255,
      1.0
    )
    readonly property real washAlpha: isSelected ? 0.42 : (isHot ? 0.30 : 0.20)

    radius: Math.max(Style.space(5), Style.cornerRadius)
    color: Color.popups.background
    borderSpec: isSelected
      ? Border.flat(root.accentColor, Math.max(2, Style.space(2)))
      : (isHot
          ? Border.flat(Style.hoverBorderFor(root.fg, root.accentColor), Math.max(1, Style.normalBorderWidth))
          : Border.flat(Qt.rgba(liveZoneColor.r, liveZoneColor.g, liveZoneColor.b, 0.42), Math.max(1, Style.normalBorderWidth)))

    // Live zone color wash (brighter when selected, dimmed slightly when unselected)
    Rectangle {
      anchors.fill: parent
      anchors.margins: zmb.isSelected ? Math.max(2, Style.space(2)) : 1
      radius: Math.max(1, zmb.radius - (zmb.isSelected ? Math.max(2, Style.space(2)) : 1))
      color: Qt.rgba(zmb.liveZoneColor.r, zmb.liveZoneColor.g, zmb.liveZoneColor.b, zmb.washAlpha)

      Behavior on color { ColorAnimation { duration: 110 } }
    }

    // Subtle slider strip end ticks when used as a slider strip
    Row {
      visible: zmb.showSliderTicks
      anchors.left: parent.left
      anchors.leftMargin: Style.space(6)
      anchors.verticalCenter: parent.verticalCenter
      spacing: Style.space(2)
      z: 1

      Repeater {
        model: 3
        delegate: Rectangle {
          width: 1
          height: Style.space(8)
          color: Qt.rgba(root.fg.r, root.fg.g, root.fg.b, 0.22)
        }
      }
    }

    Row {
      visible: zmb.showSliderTicks
      anchors.right: parent.right
      anchors.rightMargin: Style.space(6)
      anchors.verticalCenter: parent.verticalCenter
      spacing: Style.space(2)
      z: 1

      Repeater {
        model: 3
        delegate: Rectangle {
          width: 1
          height: Style.space(8)
          color: Qt.rgba(root.fg.r, root.fg.g, root.fg.b, 0.22)
        }
      }
    }

    Row {
      anchors.horizontalCenter: parent.horizontalCenter
      anchors.verticalCenter: zmb.labelAlignment === Qt.AlignVCenter ? parent.verticalCenter : undefined
      anchors.top: zmb.labelAlignment === Qt.AlignTop ? parent.top : undefined
      anchors.topMargin: zmb.labelAlignment === Qt.AlignTop ? zmb.labelTopMargin : 0
      spacing: Style.space(5)
      z: 2

      Rectangle {
        width: Style.space(7)
        height: Style.space(7)
        radius: width / 2
        anchors.verticalCenter: parent.verticalCenter
        color: zmb.liveZoneColor
        opacity: zmb.isSelected ? 1.0 : 0.72
        border.width: 1
        border.color: Qt.rgba(root.fg.r, root.fg.g, root.fg.b, 0.32)
      }

      Text {
        anchors.verticalCenter: parent.verticalCenter
        textFormat: Text.PlainText
        text: zmb.labelText
        color: zmb.isSelected ? root.accentColor : root.fg
        opacity: zmb.isSelected ? 1.0 : 0.85
        font.family: root.fontFam
        font.pixelSize: Style.font.caption
        font.bold: zmb.isSelected
      }

      Text {
        visible: zmb.subText !== ""
        anchors.verticalCenter: parent.verticalCenter
        textFormat: Text.PlainText
        text: "· " + zmb.subText
        color: zmb.isSelected ? root.accentColor : root.dim
        font.family: root.fontFam
        font.pixelSize: Style.font.caption
        opacity: zmb.isSelected ? 0.92 : 0.75
      }
    }

    MouseArea {
      id: blockMouse
      anchors.fill: parent
      z: 2
      hoverEnabled: true
      cursorShape: Qt.PointingHandCursor
      onClicked: {
        root.selectedBacklightZone = zmb.zoneKey
      }
    }
  }

  component BacklightZoneControl: Column {
    id: bzc
    readonly property string zoneKey: root.selectedBacklightZone
    readonly property var zoneMeta: root.backlightZoneMeta[zoneKey] || root.backlightZoneMeta.keyboard
    readonly property string effectValue: root.backlightZoneEffect(zoneKey, "solidColor")
    readonly property int redValue: root.backlightZoneChannel(zoneKey, "red", 255)
    readonly property int greenValue: root.backlightZoneChannel(zoneKey, "green", 255)
    readonly property int blueValue: root.backlightZoneChannel(zoneKey, "blue", 255)
    readonly property color activeColor: Qt.rgba(redValue / 255, greenValue / 255, blueValue / 255, 1.0)
    readonly property bool isColorDotEffect: effectValue === "solidColor" || effectValue === "breathing"

    width: parent ? parent.width : implicitWidth
    spacing: Style.space(4)
    enabled: root.profileControlsEnabled
    opacity: enabled ? 1.0 : 0.5

    // Compact top-view keyboard illustration: left/right slider strips on top,
    // halved-height keyboard deck below (3x10 keycap grid dropped), centered touchpad.
    BorderSurface {
      id: keyboardBody
      width: parent.width
      implicitHeight: mapColumn.implicitHeight + Style.space(8)
      padding: Style.space(4)
      radius: Math.max(Style.space(6), Style.cornerRadius)
      color: Style.normalFillFor(root.fg, root.accentColor)
      borderSpec: Border.flat(Style.normalBorderFor(root.fg, root.accentColor), Math.max(1, Style.normalBorderWidth))

      Column {
        id: mapColumn
        x: Style.space(4)
        y: Style.space(4)
        width: parent.width - Style.space(8)
        spacing: Style.space(3)

        // Top row: Left & Right touch slider strips above the keyboard area
        Row {
          width: parent.width
          spacing: Style.space(4)

          ZoneMapBlock {
            width: Math.floor((parent.width - parent.spacing) / 2)
            height: Style.space(18)
            radius: Math.max(Style.space(4), Math.round(Style.cornerRadius * 0.6))
            zoneKey: "leftSlider"
            labelText: "Left Slider"
            showSliderTicks: true
          }

          ZoneMapBlock {
            width: parent.width - Math.floor((parent.width - parent.spacing) / 2) - parent.spacing
            height: Style.space(18)
            radius: Math.max(Style.space(4), Math.round(Style.cornerRadius * 0.6))
            zoneKey: "rightSlider"
            labelText: "Right Slider"
            showSliderTicks: true
          }
        }

        // Compact keyboard deck with centered touchpad overlay
        ZoneMapBlock {
          id: keyboardDeck
          width: parent.width
          height: Style.space(40)
          radius: Math.max(Style.space(5), Math.round(Style.cornerRadius * 0.8))
          zoneKey: "keyboard"
          labelText: "Keyboard"
          subText: "Background"
          labelAlignment: Qt.AlignTop
          labelTopMargin: Style.space(3)
          isHot: mouseHovered && !touchpadZone.mouseHovered

          // Centered touchpad zone sitting inside the keyboard deck
          ZoneMapBlock {
            id: touchpadZone
            width: Math.round(parent.width * 0.64)
            height: Style.space(19)
            anchors.horizontalCenter: parent.horizontalCenter
            anchors.bottom: parent.bottom
            anchors.bottomMargin: Style.space(3)
            radius: Math.max(Style.space(4), Math.round(Style.cornerRadius * 0.6))
            zoneKey: "touchpad"
            labelText: "Touchpad"
            subText: "Touch Area"
            z: 3
          }
        }
      }
    }

    // Single-line selected zone header + per-zone factory subtree reset
    Item {
      width: parent.width
      implicitHeight: Math.max(zoneHeaderLabel.implicitHeight, zoneResetBtn.implicitHeight)

      Text {
        id: zoneHeaderLabel
        anchors.left: parent.left
        anchors.right: zoneResetBtn.left
        anchors.rightMargin: Style.space(6)
        anchors.verticalCenter: parent.verticalCenter
        textFormat: Text.PlainText
        text: bzc.zoneMeta.label + " · " + bzc.zoneMeta.sublabel
        color: root.fg
        font.family: root.fontFam
        font.pixelSize: Style.font.bodySmall
        font.bold: true
        elide: Text.ElideRight
      }

      ResetButton {
        id: zoneResetBtn
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        resetPath: "backlight." + bzc.zoneKey
        resetLabel: "Reset " + bzc.zoneMeta.label.toLowerCase() + " zone"
      }
    }

    // Single-line color header + preset swatches
    Item {
      width: parent.width
      implicitHeight: Math.max(colorHeaderText.implicitHeight, swatchRow.implicitHeight)

      Text {
        id: colorHeaderText
        anchors.left: parent.left
        anchors.verticalCenter: parent.verticalCenter
        textFormat: Text.PlainText
        text: "Color"
        color: root.dim
        font.family: root.fontFam
        font.pixelSize: Style.font.caption
        font.bold: true
      }

      Row {
        id: swatchRow
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        spacing: Style.space(4)

        Repeater {
          model: root.backlightColorPresets
          delegate: Item {
            required property var modelData
            readonly property bool isSelected: bzc.redValue === modelData.red
              && bzc.greenValue === modelData.green
              && bzc.blueValue === modelData.blue
            readonly property color swatchColor: Qt.rgba(
              modelData.red / 255,
              modelData.green / 255,
              modelData.blue / 255,
              1.0
            )
            width: Style.space(18)
            height: Style.space(18)

            Rectangle {
              anchors.fill: parent
              radius: width / 2
              color: "transparent"
              border.width: parent.isSelected
                ? Math.max(2, Style.space(2))
                : (swatchMouse.containsMouse ? Math.max(1, Style.space(1)) : 0)
              border.color: parent.isSelected ? root.accentColor : root.dim

              Rectangle {
                anchors.centerIn: parent
                width: parent.parent.isSelected ? parent.width - Style.space(5) : parent.width - Style.space(3)
                height: width
                radius: width / 2
                color: parent.parent.swatchColor
                border.width: 1
                border.color: Qt.rgba(root.fg.r, root.fg.g, root.fg.b, 0.22)
                scale: swatchMouse.containsMouse && bzc.enabled ? 1.12 : 1.0

                Behavior on scale {
                  NumberAnimation { duration: 110; easing.type: Easing.OutCubic }
                }
              }
            }

            MouseArea {
              id: swatchMouse
              anchors.fill: parent
              hoverEnabled: true
              enabled: bzc.enabled
              cursorShape: bzc.enabled ? Qt.PointingHandCursor : Qt.ArrowCursor
              onClicked: {
                if (!bzc.enabled) return
                root.setProfileField("backlight." + bzc.zoneKey + ".color", {
                  red: modelData.red,
                  green: modelData.green,
                  blue: modelData.blue
                })
                root.flushProfile()
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
    }

    // Single-line Effect header + live indicator ("Live on keyboard")
    Item {
      width: parent.width
      implicitHeight: Math.max(effectHeaderText.implicitHeight, liveIndicatorRow.implicitHeight)

      Text {
        id: effectHeaderText
        anchors.left: parent.left
        anchors.verticalCenter: parent.verticalCenter
        textFormat: Text.PlainText
        text: "Effect"
        color: root.dim
        font.family: root.fontFam
        font.pixelSize: Style.font.caption
        font.bold: true
      }

      Row {
        id: liveIndicatorRow
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        spacing: Style.space(5)

        Rectangle {
          id: livePreviewDot
          visible: bzc.isColorDotEffect
          width: Style.space(7)
          height: Style.space(7)
          radius: width / 2
          anchors.verticalCenter: parent.verticalCenter
          color: bzc.activeColor
          border.width: 1
          border.color: Qt.rgba(root.fg.r, root.fg.g, root.fg.b, 0.3)
          opacity: 1.0

          NumberAnimation on opacity {
            running: bzc.effectValue === "breathing"
            from: 0.2
            to: 1.0
            duration: 900
            loops: Animation.Infinite
            easing.type: Easing.InOutSine
          }

          Binding on opacity {
            when: bzc.effectValue !== "breathing"
            value: 1.0
          }
        }

        Text {
          visible: !bzc.isColorDotEffect
          anchors.verticalCenter: parent.verticalCenter
          textFormat: Text.PlainText
          text: root.effectDisplayLabel(bzc.effectValue)
          color: root.accentColor
          font.family: root.fontFam
          font.pixelSize: Style.font.caption
          font.bold: true
        }

        Text {
          anchors.verticalCenter: parent.verticalCenter
          textFormat: Text.PlainText
          text: "Live on keyboard"
          color: root.dim
          font.family: root.fontFam
          font.pixelSize: Style.font.caption
        }
      }
    }

    CompactButtonGroup {
      id: zoneEffectGroup
      anchors.horizontalCenter: parent.horizontalCenter
      chipPaddingX: Style.space(3)
      options: root.effectOptionsForZone(bzc.zoneKey)
      value: bzc.effectValue
      onChanged: function(v) {
        if (!bzc.enabled) return
        root.setProfileField("backlight." + bzc.zoneKey + ".effect", v)
        root.flushProfile()
      }
    }
  }

  // The host sizes the card from a morph animation between popups. When that
  // animation is cut short the card keeps a blended rect (wider than
  // contentWidth) and stops tracking contentWidth, so the content ends up
  // pinned to the left of an over-wide card with a wide empty band on its
  // right. Re-assert the final rect once the panel has settled, so the card
  // always matches the content it holds.
  readonly property Item hostCard: {
    var it = keyCatcher ? keyCatcher.parent : null
    while (it) {
      if (it.borderSpec !== undefined && it.radius !== undefined) return it
      it = it.parent
    }
    return null
  }
  Timer {
    interval: 260
    running: root.opened
    repeat: false
    onTriggered: {
      if (!hostCard) return
      hostCard.x = panel.cardOrigin.x
      hostCard.width = panel.contentWidth
      if (panel.cardOrigin.y > 0) hostCard.y = panel.cardOrigin.y
    }
  }

  KeyboardPanel {
    id: panel
    anchorItem: root.anchorItem
    owner: root.barIdentity
    bar: root.bar
    open: root.opened
    focusTarget: keyCatcher
    padding: Style.space(10)

    contentWidth: panel.fittedContentWidth(Style.space(300))

    readonly property int maxCardHeight: panel.screenH > 0
      ? Math.min(Style.space(420), Math.round(panel.screenH * 0.6))
      : Style.space(420)

    contentHeight: panel.fittedContentHeight(column.implicitHeight, panel.maxCardHeight)

    // Same shape as the shipped panels: the content fills the host's content
    // area and takes its width from the ScrollView's available width, instead
    // of re-deriving the inner size by hand.
    PanelKeyCatcher {
      id: keyCatcher
      anchors.fill: parent
      clip: true
      onCloseRequested: root.close()
      onTabRequested: function(direction) { root.switchPanel(direction) }

      ScrollView {
        id: scrollArea
        anchors.fill: parent
        contentWidth: availableWidth
        contentHeight: column.implicitHeight
        clip: true
        ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
        ScrollBar.vertical.policy: ScrollBar.AsNeeded

        Column {
          id: column
          width: scrollArea.availableWidth
          spacing: Style.space(5)

          // Hero row: title on the left, then the connection pill and the
          // battery pill grouped on the trailing edge. One row keeps the
          // header compact; the title elides instead of pushing the pills.
          Item {
            id: heroRow
            width: parent.width
            implicitHeight: Math.max(titleLabel.implicitHeight,
                                     statusPill.implicitHeight,
                                     batteryPill.implicitHeight)

            Text {
              id: titleLabel
              anchors.left: parent.left
              anchors.right: statusPill.left
              anchors.rightMargin: Style.space(8)
              anchors.verticalCenter: parent.verticalCenter
              textFormat: Text.PlainText
              text: "CLVX Keyboard"
              color: root.fg
              font.family: root.fontFam
              font.pixelSize: Style.font.title
              font.bold: true
              elide: Text.ElideRight
            }

            BorderSurface {
              id: statusPill
              anchors.right: batteryPill.visible ? batteryPill.left : parent.right
              anchors.rightMargin: Style.space(6)
              anchors.verticalCenter: parent.verticalCenter
              implicitWidth: statusPillRow.implicitWidth + Style.space(10)
              implicitHeight: statusPillRow.implicitHeight + Style.space(4)
              radius: height / 2
              color: Qt.rgba(root.connectionDotColor.r, root.connectionDotColor.g,
                             root.connectionDotColor.b, 0.14)
              borderSpec: Border.flat(Qt.rgba(root.connectionDotColor.r, root.connectionDotColor.g,
                                               root.connectionDotColor.b, 0.45),
                                      Math.max(1, Style.space(1)))

              Row {
                id: statusPillRow
                anchors.centerIn: parent
                spacing: Style.space(4)

                Rectangle {
                  width: Style.space(6)
                  height: width
                  radius: width / 2
                  anchors.verticalCenter: parent.verticalCenter
                  color: root.connectionDotColor
                }

                Text {
                  anchors.verticalCenter: parent.verticalCenter
                  textFormat: Text.PlainText
                  text: root.connectionLabel
                  color: root.connectionDotColor
                  font.family: root.fontFam
                  font.pixelSize: Style.font.caption
                  font.bold: true
                }
              }
            }

            BorderSurface {
              id: batteryPill
              visible: root.detailText !== ""
              anchors.right: parent.right
              anchors.verticalCenter: parent.verticalCenter
              implicitWidth: batteryText.implicitWidth + Style.space(10)
              implicitHeight: batteryText.implicitHeight + Style.space(4)
              color: "transparent"
              borderSpec: Border.controlSpec("normal", root.fg, root.accentColor)
              radius: Style.cornerRadius

              Text {
                id: batteryText
                anchors.centerIn: parent
                textFormat: Text.PlainText
                text: root.detailText
                color: root.dim
                font.family: root.fontFam
                font.pixelSize: Style.font.body
                font.bold: true
              }
            }
          }

          Text {
            width: parent.width
            visible: root.showErrorLine
            text: root.lastError
            textFormat: Text.PlainText
            color: Color.urgent
            font.family: root.fontFam
            font.pixelSize: Style.font.caption
            wrapMode: Text.WordWrap
          }

          PanelSeparator { width: parent.width; foreground: root.fg }

          Column {
            width: parent.width
            spacing: Style.space(4)

            Item {
              width: parent.width
              implicitHeight: slidersHeader.implicitHeight

              PanelSectionHeader {
                id: slidersHeader
                anchors.horizontalCenter: parent.horizontalCenter
                anchors.verticalCenter: parent.verticalCenter
                topPadding: 0
                text: "Sliders"
                foreground: root.fg
                fontFamily: root.fontFam
              }

              Text {
                visible: !root.profileReady
                anchors.right: parent.right
                anchors.verticalCenter: parent.verticalCenter
                textFormat: Text.PlainText
                text: "Loading…"
                color: root.dim
                font.family: root.fontFam
                font.pixelSize: Style.font.caption
              }
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

            Item {
              width: parent.width
              implicitHeight: backlightHeader.implicitHeight

              PanelSectionHeader {
                id: backlightHeader
                anchors.horizontalCenter: parent.horizontalCenter
                anchors.verticalCenter: parent.verticalCenter
                topPadding: 0
                text: "Backlight"
                foreground: root.fg
                fontFamily: root.fontFam
              }

              Text {
                visible: !root.profileReady
                anchors.right: parent.right
                anchors.verticalCenter: parent.verticalCenter
                textFormat: Text.PlainText
                text: "Loading…"
                color: root.dim
                font.family: root.fontFam
                font.pixelSize: Style.font.caption
              }
            }

            BacklightZoneControl {}
          }
        }
      }
    }
  }
}
