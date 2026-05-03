
import { io } from "socket.io-client";
let io = null
function ConnectBackend() {
    if (io != null) return
    io = io("ws://127.0.0.1:54199", {
        transports: ["websocket"],
    });
    io.on("connect", function () {
        console.log("Connect Backend");
        VoxBackendStates.WsIsConect.set(true)
    });
    io.on("disconnect", function () {
        console.log("Disconnect Backend");
        VoxBackendStates.WsIsConect.set(false)
    });
}

function GetIo() {
    return io
}

export { ConnectBackend, GetIo }