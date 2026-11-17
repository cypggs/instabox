"use client";

import { useEffect, useRef } from "react";
import { Terminal as XTerm } from "xterm";
import { FitAddon } from "@xterm/addon-fit";
import "xterm/css/xterm.css";
import { API_BASE } from "../lib/api";

// Wire protocol: binary frames are raw keystrokes; JSON text frames {"cols","rows"} resize.
export default function Terminal({ machineId }: { machineId: string }) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!ref.current) return;
    const term = new XTerm({
      cursorBlink: true,
      fontFamily: '"JetBrains Mono", monospace',
      fontSize: 13,
      theme: { background: "#0a0a0f", foreground: "#e4e4ef", cursor: "#7c6cf5" },
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(ref.current);
    fit.fit();

    const proto = location.protocol === "https:" ? "wss" : "ws";
    const ws = new WebSocket(`${proto}://${new URL(API_BASE).host}/v1/machines/${machineId}/terminal`);
    ws.binaryType = "arraybuffer";

    ws.onopen = () => term.writeln("\x1b[90m… connected\x1b[0m\r");
    ws.onmessage = (ev) => {
      if (ev.data instanceof ArrayBuffer) term.write(new Uint8Array(ev.data));
      else term.write(String(ev.data));
    };
    ws.onclose = (ev) => term.writeln(`\r\n\x1b[90m[disconnected${ev.reason ? `: ${ev.reason}` : ""}]\x1b[0m`);
    ws.onerror = () => {};

    const dataSub = term.onData((d) => {
      if (ws.readyState === WebSocket.OPEN) ws.send(d); // raw bytes as binary frame
    });
    const sendResize = () => {
      if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ cols: term.cols, rows: term.rows }));
    };
    const ro = new ResizeObserver(() => { fit.fit(); sendResize(); });
    ro.observe(ref.current);
    // initial size once shell spawns
    const t1 = setTimeout(sendResize, 500);
    const t2 = setTimeout(sendResize, 2000);

    return () => {
      clearTimeout(t1); clearTimeout(t2);
      ro.disconnect();
      dataSub.dispose();
      ws.close();
      term.dispose();
    };
  }, [machineId]);

  return <div ref={ref} style={{ height: 440, background: "#0a0a0f", borderRadius: 8, padding: 8 }} />;
}
