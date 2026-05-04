
import { io } from "socket.io-client";
import { GetSongs } from "./db";
let Io;
function ConnectBackend(on,dis) {
    console.log("ws conntecting");
    
    Io = io("ws://127.0.0.1:54199", {
        transports: ["websocket"],
        reconnectionDelay: 500,        // 基础重试间隔 500ms
        reconnectionDelayMax: 500,     // 最大间隔也是 500ms（固定间隔）
        randomizationFactor: 0,        // 不加入随机延时
        reconnectionAttempts: Infinity,
    });
    Io.on("connect", function () {
        console.log("Connect Backend");
        on(Io)
    });
    Io.on("disconnect", function () {
        console.log("Disconnect Backend");
        dis()
    });
}

function GetIo() {
    return Io
}

export { ConnectBackend, GetIo }