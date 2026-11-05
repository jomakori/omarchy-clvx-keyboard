import QtQuick
import Quickshell
import Quickshell.Io

// Resident clvxd facade. Quickshell has no Unix-socket client, so a small
// unbuffered Python child owns the subscription and forwards JSON lines.
Item {
  id: root
  visible: false

  property var shell: null
  property var manifest: null

  readonly property string pluginId: "io.github.joel.clvx-keyboard"
  readonly property string runtimeDir: Quickshell.env("XDG_RUNTIME_DIR") || ""
  readonly property string socketPath: runtimeDir !== "" ? runtimeDir + "/clvx/clvx.sock" : ""

  readonly property bool connected: state.connected
  readonly property string transport: state.transport
  readonly property int batteryPercent: state.batteryPercent
  readonly property bool charging: state.charging
  readonly property int powerMode: state.powerMode
  readonly property int backlightMode: state.backlightMode
  readonly property string lastHeartbeat: state.lastHeartbeat
  readonly property string lastError: state.lastError
  readonly property var settings: state.settings
  readonly property var profile: state.profile
  readonly property var profileRaw: state.profileRaw
  readonly property var deviceInfo: state.deviceInfo
  readonly property var deviceDefaults: state.deviceDefaults

  QtObject {
    id: state
    property bool connected: false
    property string transport: ""
    property int batteryPercent: 0
    property bool charging: false
    property int powerMode: 0
    property int backlightMode: 0
    property string lastHeartbeat: ""
    property string lastError: ""
    property var settings: null
    property var profile: null
    property var profileRaw: null
    property var deviceInfo: null
    property var deviceDefaults: null
    property int restarts: 0
    property var pendingRequest: null
    property var pendingQueue: ([])
    property bool requestHadStatus: false
    property string activeMethod: ""
  }

  function numeric(value, fallback) {
    var number = Number(value)
    return isFinite(number) ? number : fallback
  }

  function applyStatus(status) {
    state.connected = status.connected === true
    state.transport = status.transport === undefined || status.transport === null ? "" : String(status.transport)
    state.batteryPercent = Math.round(numeric(status.batteryPercent, 0))
    state.charging = status.charging === true
    state.powerMode = Math.round(numeric(status.powerMode, 0))
    state.backlightMode = Math.round(numeric(status.backlightMode, 0))
    state.lastHeartbeat = status.lastHeartbeat === undefined || status.lastHeartbeat === null ? "" : String(status.lastHeartbeat)
    state.lastError = status.error === undefined || status.error === null ? "" : String(status.error)
  }

  function hasQueuedMethod(method) {
    var queue = state.pendingQueue || []
    for (var i = 0; i < queue.length; i++) {
      if (queue[i] && queue[i].method === method) return true
    }
    return false
  }

  function handleLine(line, sourceMethod) {
    var text = String(line || "").trim()
    if (text === "") return false
    if (text.length > 65536) {
      state.lastError = "clvxd response was too large"
      return false
    }
    var method = sourceMethod !== undefined ? String(sourceMethod) : state.activeMethod
    try {
      var message = JSON.parse(text)
      if (!message || typeof message !== "object") return false
      if (message.error !== undefined && message.error !== null) {
        state.lastError = typeof message.error === "string" ? message.error : JSON.stringify(message.error)
        return false
      }
      if (!message.result || typeof message.result !== "object" || Array.isArray(message.result)) return false
      if ("connected" in message.result || "transport" in message.result || "batteryPercent" in message.result) {
        applyStatus(message.result)
        if (method === "subscribe" || method === "status") return true
      }
      // Route default_settings exclusively to state.deviceDefaults so it never
      // overwrites live state.settings or state.profile.
      if (method === "default_settings") {
        state.deviceDefaults = message.result
        return true
      }
      if ("fwVersion" in message.result || "barCode" in message.result || "serialNumber" in message.result
          || "deviceMode" in message.result || "kbmInfo" in message.result || method === "device_info") {
        state.deviceInfo = message.result
        return true
      }
      if (method === "get_settings" || method === "set_global"
          || "global" in message.result || "settings" in message.result) {
        if (!hasQueuedMethod("set_global")) {
          state.settings = message.result
        }
        return true
      }
      if (method === "get_profile_settings" || method === "set_profile_settings"
          || ("id" in message.result && "raw" in message.result)
          || (!("global" in message.result) && !("fwVersion" in message.result) && !("kbmInfo" in message.result)
              && ("touchZone" in message.result || "keyboard" in message.result || "fKey" in message.result || "backlight" in message.result))) {
        if (message.result.raw !== undefined) state.profileRaw = message.result.raw
        if (!hasQueuedMethod("set_profile_settings")) {
          state.profile = message.result
        }
        return true
      }
      return true
    } catch (error) {
      state.lastError = "Invalid response from clvxd"
      return false
    }
  }

  function defaultProfileObject() {
    var d = state.deviceDefaults
    if (!d || typeof d !== "object") return null
    if (d.globalProfile && typeof d.globalProfile === "object") return d.globalProfile
    if (d.profile && typeof d.profile === "object") return d.profile
    if (d.settings && typeof d.settings === "object") {
      if (d.settings.globalProfile && typeof d.settings.globalProfile === "object") return d.settings.globalProfile
      if (d.settings.touchZone || d.settings.backlight || d.settings.keyboard || d.settings.gestures) return d.settings
    }
    if (d.touchZone || d.backlight || d.keyboard || d.gestures) return d
    return null
  }

  function extractDefaultZoneColor(zone) {
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

  function markUnavailable(message) {
    state.connected = false
    state.transport = ""
    state.batteryPercent = 0
    state.charging = false
    state.powerMode = 0
    state.backlightMode = 0
    state.lastHeartbeat = ""
    state.lastError = message
  }

  readonly property string subscriberScript: [
    "import socket, sys",
    "try:",
    "    s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)",
    "    s.connect(sys.argv[1])",
    "    f = s.makefile('rwb', buffering=0)",
    "    f.write(b'{\"id\":1,\"method\":\"subscribe\"}\\n')",
    "    while True:",
    "        line = f.readline()",
    "        if not line:",
    "            break",
    "        if line.strip():",
    "            sys.stdout.buffer.write(line)",
    "            sys.stdout.buffer.flush()",
    "except OSError as error:",
    "    print('clvxd: ' + str(error), file=sys.stderr)",
    "    sys.exit(1)"
  ].join("\n")

  Process {
    id: subscriber
    running: false
    command: ["python3", "-I", "-u", "-c", root.subscriberScript, root.socketPath]
    stdout: SplitParser { onRead: function(line) { root.handleLine(line, "subscribe") } }
    onExited: {
      root.markUnavailable("clvxd socket is unavailable")
      restartTimer.interval = Math.min(30000, 1000 * Math.pow(2, Math.min(5, state.restarts)))
      state.restarts++
      restartTimer.restart()
    }
  }

  Timer {
    id: restartTimer
    repeat: false
    onTriggered: root.connect()
  }

  Timer {
    interval: 60000
    running: subscriber.running
    repeat: false
    onTriggered: state.restarts = 0
  }

  function connect() {
    if (root.socketPath === "") {
      root.markUnavailable("XDG_RUNTIME_DIR is not set")
      return false
    }
    if (subscriber.running) return true
    restartTimer.stop()
    subscriber.running = true
    return true
  }

  readonly property string requestScript: [
    "import json, socket, sys",
    "try:",
    "    s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)",
    "    s.settimeout(5.0)",
    "    s.connect(sys.argv[1])",
    "    f = s.makefile('rwb', buffering=0)",
    "    request = {'id': 1, 'method': sys.argv[2]}",
    "    if len(sys.argv) > 3 and sys.argv[3]:",
    "        params = json.loads(sys.argv[3])",
    "        if not isinstance(params, dict):",
    "            raise ValueError('params must be a JSON object')",
    "        request['params'] = params",
    "    f.write((json.dumps(request, separators=(',', ':')) + '\\n').encode())",
    "    line = f.readline()",
    "    if not line:",
    "        raise OSError('empty response')",
    "    sys.stdout.buffer.write(line)",
    "    sys.stdout.buffer.flush()",
    "except (OSError, ValueError) as error:",
    "    print('clvxd: ' + str(error), file=sys.stderr)",
    "    sys.exit(1)"
  ].join("\n")

  Process {
    id: requestProcess
    running: false
    stdout: SplitParser {
      onRead: function(line) {
        if (root.handleLine(line, state.activeMethod)) state.requestHadStatus = true
      }
    }
    onExited: function(code) {
      if (code !== 0) state.lastError = "clvxd request failed"
      state.activeMethod = ""
      Qt.callLater(root.runPendingRequest)
    }
  }

  function isPollMethod(method) {
    return method === "status"
      || method === "get_settings"
      || method === "get_profile_settings"
      || method === "device_info"
      || method === "default_settings"
  }

  function enqueueRequest(request) {
    var queue = state.pendingQueue ? state.pendingQueue.slice() : []
    var incomingIsPoll = isPollMethod(request.method)

    if (request.method === "set_global" && request.params && request.params.global) {
      for (var i = 0; i < queue.length; i++) {
        if (queue[i].method === "set_global" && queue[i].params && queue[i].params.global) {
          var merged = Object.assign({}, queue[i].params.global, request.params.global)
          queue[i] = { method: "set_global", params: { global: merged } }
          state.pendingQueue = queue
          state.pendingRequest = queue[0] || null
          return
        }
      }
    }

    if (request.method === "set_profile_settings") {
      for (var j = 0; j < queue.length; j++) {
        if (queue[j].method === "set_profile_settings") {
          queue[j] = request
          state.pendingQueue = queue
          state.pendingRequest = queue[0] || null
          return
        }
      }
    }

    if (incomingIsPoll) {
      for (var k = 0; k < queue.length; k++) {
        if (queue[k].method === request.method) {
          state.pendingQueue = queue
          state.pendingRequest = queue[0] || null
          return
        }
      }
      queue.push(request)
    } else {
      // Insert writes ahead of queued read-only polls so polls never delay or displace writes.
      var insertIdx = queue.length
      for (var m = 0; m < queue.length; m++) {
        if (isPollMethod(queue[m].method)) {
          insertIdx = m
          break
        }
      }
      queue.splice(insertIdx, 0, request)
    }

    state.pendingQueue = queue
    state.pendingRequest = queue[0] || null
  }

  function startRequest(request) {
    if (!request || typeof request.method !== "string" || request.method === "") return false
    if (root.socketPath === "") {
      root.markUnavailable("XDG_RUNTIME_DIR is not set")
      return false
    }
    if (requestProcess.running) {
      enqueueRequest(request)
      return true
    }
    state.requestHadStatus = false
    state.activeMethod = request.method
    var command = ["python3", "-I", "-u", "-c", root.requestScript, root.socketPath, request.method]
    if (request.params && typeof request.params === "object" && !Array.isArray(request.params)) {
      command.push(JSON.stringify(request.params))
    }
    requestProcess.command = command
    requestProcess.running = true
    return true
  }

  function runPendingRequest() {
    if (requestProcess.running) return
    var queue = state.pendingQueue ? state.pendingQueue.slice() : []
    if (queue.length === 0 && state.pendingRequest !== null) {
      var single = state.pendingRequest
      state.pendingRequest = null
      startRequest(single)
      return
    }
    if (queue.length === 0) return
    var nextReq = queue.shift()
    state.pendingQueue = queue
    state.pendingRequest = queue[0] || null
    startRequest(nextReq)
  }

  function refresh() {
    return startRequest({ method: "status" })
  }

  function getSettings() {
    return startRequest({ method: "get_settings" })
  }

  function get_settings() {
    return getSettings()
  }

  function setGlobal(globalParams) {
    if (!globalParams || typeof globalParams !== "object" || Array.isArray(globalParams)) return false
    return startRequest({ method: "set_global", params: { global: globalParams } })
  }

  function set_global(globalParams) {
    return setGlobal(globalParams)
  }

  function getProfileSettings() {
    return startRequest({ method: "get_profile_settings", params: { profileId: 0 } })
  }

  function get_profile_settings() {
    return getProfileSettings()
  }

  function setProfileSettings(profileParams) {
    if (!profileParams || typeof profileParams !== "object" || Array.isArray(profileParams)) return false
    var payload = ("settings" in profileParams && profileParams.settings && typeof profileParams.settings === "object" && !("id" in profileParams))
      ? profileParams
      : { settings: profileParams }
    return startRequest({ method: "set_profile_settings", params: payload })
  }

  function set_profile_settings(profileParams) {
    return setProfileSettings(profileParams)
  }

  function getDeviceInfo() {
    return startRequest({ method: "device_info" })
  }

  function device_info() {
    return getDeviceInfo()
  }

  function getDefaultSettings() {
    return startRequest({ method: "default_settings" })
  }

  function getDeviceDefaults() {
    return getDefaultSettings()
  }

  function get_device_defaults() {
    return getDefaultSettings()
  }

  function default_settings() {
    return getDefaultSettings()
  }

  function resetGlobal(field) {
    var key = String(field || "").trim()
    if (key === "") return false
    var defs = state.deviceDefaults
    if (!defs || typeof defs !== "object" || !defs.global || typeof defs.global !== "object") return false
    if (defs.global[key] === undefined) return false
    var patch = {}
    patch[key] = JSON.parse(JSON.stringify(defs.global[key]))
    if (state.settings && state.settings.global) {
      var nextSettings = JSON.parse(JSON.stringify(state.settings))
      nextSettings.global[key] = patch[key]
      state.settings = nextSettings
    }
    return setGlobal(patch)
  }

  function reset_global(field) {
    return resetGlobal(field)
  }

  function resetProfileField(path) {
    var rawPath = String(path || "").trim()
    if (rawPath === "") return false
    var defProf = defaultProfileObject()
    if (!defProf || !state.profile || typeof state.profile !== "object") return false

    var normPath = rawPath
    if (normPath.indexOf("slider.") === 0) normPath = "touchZone." + normPath
    else if (normPath.indexOf("touchpad.") === 0) normPath = "touchZone." + normPath
    else if (normPath.indexOf("fKey.") === 0) normPath = "keyboard." + normPath

    var nextProfile = JSON.parse(JSON.stringify(state.profile))

    if (normPath === "touchZone.slider.left" || normPath === "touchZone.slider.right") {
      var sliderSide = normPath.split(".")[2]
      var defSlider = (defProf.touchZone && defProf.touchZone.slider) ? defProf.touchZone.slider[sliderSide] : null
      if (!defSlider || typeof defSlider !== "object") return false
      if (!nextProfile.touchZone) nextProfile.touchZone = {}
      if (!nextProfile.touchZone.slider) nextProfile.touchZone.slider = {}
      nextProfile.touchZone.slider[sliderSide] = JSON.parse(JSON.stringify(defSlider))
    } else if (normPath === "touchZone.slider.left.sensitivity" || normPath === "touchZone.slider.right.sensitivity") {
      var sensSide = normPath.split(".")[2]
      var defSensSlider = (defProf.touchZone && defProf.touchZone.slider) ? defProf.touchZone.slider[sensSide] : null
      if (!defSensSlider || typeof defSensSlider !== "object") return false
      if (!nextProfile.touchZone) nextProfile.touchZone = {}
      if (!nextProfile.touchZone.slider) nextProfile.touchZone.slider = {}
      if (!nextProfile.touchZone.slider[sensSide]) nextProfile.touchZone.slider[sensSide] = JSON.parse(JSON.stringify(defSensSlider))
      nextProfile.touchZone.slider[sensSide].sensitivity = defSensSlider.sensitivity !== undefined ? defSensSlider.sensitivity : 3
    } else if (normPath === "touchZone.slider.left.behavior" || normPath === "touchZone.slider.right.behavior") {
      var behSide = normPath.split(".")[2]
      var defBehSlider = (defProf.touchZone && defProf.touchZone.slider) ? defProf.touchZone.slider[behSide] : null
      if (!defBehSlider || typeof defBehSlider !== "object") return false
      if (!nextProfile.touchZone) nextProfile.touchZone = {}
      if (!nextProfile.touchZone.slider) nextProfile.touchZone.slider = {}
      nextProfile.touchZone.slider[behSide] = JSON.parse(JSON.stringify(defBehSlider))
    } else if (normPath.indexOf("backlight.") === 0 && normPath.split(".").length === 3) {
      var blParts = normPath.split(".")
      var blZoneKey = blParts[1]
      var blProp = blParts[2]
      var defBlZone = (defProf.backlight && defProf.backlight[blZoneKey]) ? defProf.backlight[blZoneKey] : null
      if (!defBlZone || typeof defBlZone !== "object") return false
      if (!nextProfile.backlight) nextProfile.backlight = {}
      if (blProp === "effect") {
        var prevBlZone = nextProfile.backlight[blZoneKey]
        var hadPrevColor = prevBlZone && ((prevBlZone.solidColor && prevBlZone.solidColor.color) || (prevBlZone.breathing && prevBlZone.breathing.color))
        var prevCol = hadPrevColor ? extractDefaultZoneColor(prevBlZone) : null
        var restoredBl = JSON.parse(JSON.stringify(defBlZone))
        if (prevCol) {
          if (restoredBl.solidColor) restoredBl.solidColor.color = prevCol
          else if (restoredBl.breathing) restoredBl.breathing.color = prevCol
        }
        nextProfile.backlight[blZoneKey] = restoredBl
      } else if (blProp === "color") {
        var defCol = extractDefaultZoneColor(defBlZone)
        if (!nextProfile.backlight[blZoneKey] || typeof nextProfile.backlight[blZoneKey] !== "object") {
          nextProfile.backlight[blZoneKey] = JSON.parse(JSON.stringify(defBlZone))
        }
        var curBl = nextProfile.backlight[blZoneKey]
        if (curBl.solidColor && typeof curBl.solidColor === "object") {
          curBl.solidColor.color = JSON.parse(JSON.stringify(defCol))
        } else if (curBl.breathing && typeof curBl.breathing === "object") {
          curBl.breathing.color = JSON.parse(JSON.stringify(defCol))
        } else if (curBl.interactiveAnimation && typeof curBl.interactiveAnimation === "object") {
          curBl.interactiveAnimation.color = JSON.parse(JSON.stringify(defCol))
        } else {
          curBl.color = JSON.parse(JSON.stringify(defCol))
        }
      } else {
        return false
      }
    } else {
      var parts = normPath.split(".")
      var srcCursor = defProf
      for (var i = 0; i < parts.length; i++) {
        if (!srcCursor || typeof srcCursor !== "object" || srcCursor[parts[i]] === undefined) return false
        srcCursor = srcCursor[parts[i]]
      }
      var dstCursor = nextProfile
      for (var j = 0; j < parts.length - 1; j++) {
        var seg = parts[j]
        if (!dstCursor[seg] || typeof dstCursor[seg] !== "object") dstCursor[seg] = {}
        dstCursor = dstCursor[seg]
      }
      dstCursor[parts[parts.length - 1]] = JSON.parse(JSON.stringify(srcCursor))
    }

    if (state.profile.raw !== undefined) nextProfile.raw = state.profile.raw
    else if (state.profileRaw !== undefined && state.profileRaw !== null) nextProfile.raw = state.profileRaw

    state.profile = nextProfile
    return setProfileSettings(nextProfile)
  }

  function reset_profile_field(path) {
    return resetProfileField(path)
  }

  function setOsMode(mode) {
    var m = (mode === undefined || mode === null || String(mode) === "") ? "LINUX" : String(mode)
    return startRequest({ method: "set_os_mode", params: { mode: m } })
  }

  function set_os_mode(mode) {
    return setOsMode(mode)
  }

  function restartDevice() {
    return startRequest({ method: "restart" })
  }

  function restart() {
    return restartDevice()
  }

  function factoryReset(confirmed) {
    if (confirmed !== true && (!confirmed || typeof confirmed !== "object" || confirmed.confirm !== true)) return false
    return startRequest({ method: "factory_reset", params: { confirm: true } })
  }

  function factory_reset(confirmed) {
    return factoryReset(confirmed)
  }

  Component.onCompleted: root.connect()
}
