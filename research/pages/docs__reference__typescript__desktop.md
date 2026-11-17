> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Desktop

> See and control a machine's desktop with sbx.desktop.

Input and screenshots run commands in the machine, so a failing one throws `CommandExitError`. The desktop
runs on X display `:1` (`DESKTOP_DISPLAY`). Coordinates are pixels from the top left. See
[Desktop](https://instabox.sh/docs/access/desktop.md).

## desktop.start

Start the desktop, and resolve once it runs.

```ts theme={null}
start(opts?: { width?: number; height?: number }): Promise<DesktopStatus>
```

A running desktop is left as it is. On a template without the desktop (`universal`, `t3`), the first call installs
it, which takes about half a minute: it sends `install`, polls `status`, then sends `start`. The HTTP `start`
action alone never installs. The screen is 800 to 2560 by 600 to 1600 (default 1280 x 800).

## desktop.status

Get the desktop's status.

```ts theme={null}
status(): Promise<DesktopStatus>
```

`DesktopStatus` has `installed`, `running`, and `width` and `height` while running.

## desktop.screenshot

Take a PNG of the whole screen.

```ts theme={null}
screenshot(): Promise<Uint8Array>
```

## desktop.click

Click at a point.

```ts theme={null}
click(x: number, y: number, opts?: { button?: MouseButton; double?: boolean }): Promise<void>
```

The left button clicks unless `button` is `'right'` or `'middle'`. `double` clicks twice.

## desktop.move

Move the pointer.

```ts theme={null}
move(x: number, y: number): Promise<void>
```

## desktop.drag

Drag from one point to another, holding the left button.

```ts theme={null}
drag(from: [number, number], to: [number, number]): Promise<void>
```

## desktop.scroll

Turn the wheel at the pointer.

```ts theme={null}
scroll(opts: { direction: ScrollDirection; amount?: number }): Promise<void>
```

`amount` is the number of notches (default 3). `direction` is `'up'`, `'down'`, `'left'` or `'right'`.

## desktop.type

Type the text as is into the focused window, quotes, newlines and Unicode included.

```ts theme={null}
type(text: string, opts?: { delayMs?: number }): Promise<void>
```

## desktop.press

Press a key (`'Enter'`, `'a'`, `'F5'`) or a chord (`['ctrl', 'l']`).

```ts theme={null}
press(keys: string | string[]): Promise<void>
```

## desktop.open

Open an http or https URL in the desktop's browser.

```ts theme={null}
open(url: string): Promise<void>
```

## desktop.cursorPosition

Get the pointer's position.

```ts theme={null}
cursorPosition(): Promise<{ x: number; y: number }>
```

## desktop.streamUrl

Make a link to a page that shows the desktop full-screen, to open or put in an `<iframe>`.

```ts theme={null}
streamUrl(opts?: { viewOnly?: boolean; ttlSeconds?: number }): Promise<DesktopStream>
```

Returns `{ url, expiresAt }`. Anyone with the link can connect, with no instabox account, until
`expiresAt` (`ttlSeconds` 60 to 86 400, default 3 600). The link is view-only unless `viewOnly: false`.
Expiry stops only new connections.

> [!NOTE]
>
> `streamUrl` and `revokeStreamUrls` belong to the machine's creator: only the creator, signed in or with
> their personal key, can call them. A workspace key gets `403 permission_denied`. A workspace key opens
> the desktop over `GET /v1/machines/{id}/vnc` (a WebSocket, `machines:write`) or reads it with
> `screenshot()`. A personal key reaches only the machines of your personal workspace.

## desktop.revokeStreamUrls

End every stream link of the machine, and disconnect open viewers within a few seconds.

```ts theme={null}
revokeStreamUrls(): Promise<void>
```
