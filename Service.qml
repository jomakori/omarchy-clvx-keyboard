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
    property int restarts: 0
    property var pendingRequest: null
    property bool requestHadStatus: false
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

  function handleLine(line) {
    var text = String(line || "").trim()
    if (text === "") return false
    if (text.length > 65536) {
      state.lastError = "clvxd response was too large"
      return false
    }
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
      }
      if ("global" in message.result || "settings" in message.result) {
        state.settings = message.result
      }
      return true
    } catch (error) {
      state.lastError = "Invalid response from clvxd"
      return false
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
    stdout: SplitParser { onRead: function(line) { root.handleLine(line) } }
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
        if (root.handleLine(line)) state.requestHadStatus = true
      }
    }
    onExited: function(code) {
      if (code !== 0) state.lastError = "clvxd request failed"
      Qt.callLater(root.runPendingRequest)
    }
  }

  function startRequest(request) {
    if (!request || typeof request.method !== "string" || request.method === "") return false
    if (root.socketPath === "") {
      root.markUnavailable("XDG_RUNTIME_DIR is not set")
      return false
    }
    if (requestProcess.running) {
      // A status/get_settings poll must not displace a queued write request.
      var incomingIsPoll = request.method === "status" || request.method === "get_settings"
      if (state.pendingRequest === null || !incomingIsPoll) state.pendingRequest = request
      return true
    }
    state.pendingRequest = null
    state.requestHadStatus = false
    var command = ["python3", "-I", "-u", "-c", root.requestScript, root.socketPath, request.method]
    if (request.params && typeof request.params === "object" && !Array.isArray(request.params)) {
      command.push(JSON.stringify(request.params))
    }
    requestProcess.command = command
    requestProcess.running = true
    return true
  }

  function runPendingRequest() {
    if (requestProcess.running || state.pendingRequest === null) return
    var request = state.pendingRequest
    state.pendingRequest = null
    startRequest(request)
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

  Component.onCompleted: root.connect()
}
