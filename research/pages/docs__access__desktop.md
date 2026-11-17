> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Desktop

> Watch a machine's desktop, let an agent drive it, and share a link to it.

A machine can run a Linux desktop with a browser, which gives an agent a real screen to work on.
You can watch the desktop in the dashboard while the agent works, let the agent click and type in
it from code, and share a link that shows the desktop to someone else.

Machines from the `agent-browser` template (shown as **standard**) have the desktop in the image,
so the quickest way to get one is to create a machine from that template and start the desktop.

```ts theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const sbx = await Sandbox.create('agent-browser')
await sbx.desktop.start({ width: 1280, height: 800 })
```

## Watch the desktop in the dashboard

When you want to see what an agent is doing, open the machine in the dashboard and choose the
**Computer** tab. You can click the desktop to take control and use the mouse and keyboard
yourself. Control goes back to the agent after a minute without input from you, or 30 seconds after
you close the tab.

## Let an agent drive it

An agent drives the desktop through calls on `sbx.desktop`, and the desktop has to be started
before the first of them. You can call `start()` again at any time, because it leaves a running
desktop alone.

```ts TypeScript theme={null}
await sbx.desktop.start({ width: 1280, height: 800 })
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../desktop \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"action": "start", "width": 1280, "height": 800}'
```

Over HTTP, `start` does not install anything. If your machine comes from a template without the
desktop, see [Install the desktop over HTTP](#install-the-desktop-over-http).

The agent then works in a loop: it takes a screenshot, decides what to do, and sends input. The
sections below show the calls that the loop is built from. The other calls you can use in the
loop are `move(x, y)`, `drag([x1, y1], [x2, y2])`, `cursorPosition()`, and `status()`.

> [accordion] A minimal agent loop
>
> In this loop, `yourModel.nextAction` stands for your own model call. It returns one of
> `{ type: 'click', x, y }`, `{ type: 'type', text }`, or `{ type: 'done' }`.
>
> ```ts theme={null}
> for (let step = 0; step < 20; step++) {
>   const png = await sbx.desktop.screenshot()
>   const action = await yourModel.nextAction(png)
>   if (action.type === 'done') break
>   if (action.type === 'click') {
>     await sbx.desktop.click(action.x, action.y)
>   }
>   if (action.type === 'type') await sbx.desktop.type(action.text)
> }
> ```

### Open a page in the browser

To send the agent to a web page, call `open()`, which opens the desktop's browser at the URL you
give it.

```ts theme={null}
await sbx.desktop.open('https://example.com')
```

### Take a screenshot

A screenshot is how the agent sees the desktop. `screenshot()` returns the screen as PNG bytes,
which you can pass to a vision model to decide the next step.

```ts theme={null}
const png = await sbx.desktop.screenshot()
```

### Click and scroll

To click, give `click()` the x and y of a point on the screen. When you need a right click or a
double click, pass `{ button: 'right' }` or `{ double: true }` as the third argument.

```ts theme={null}
await sbx.desktop.click(640, 400)
```

To move through a page that is longer than the screen, call `scroll()` with a direction and an
amount.

```ts theme={null}
await sbx.desktop.scroll({ direction: 'down', amount: 5 })
```

### Type text and press keys

To fill in a field, call `type()` with the text. It sends any text, including quotes and Unicode.

```ts theme={null}
await sbx.desktop.type('instabox.sh')
```

For a key that is not text, such as Enter or a shortcut, use `press()`. It takes one key, or an
array of keys for a chord. This example presses Ctrl+L and then Enter.

```ts theme={null}
await sbx.desktop.press(['ctrl', 'l'])
await sbx.desktop.press('Enter')
```

## Share a link to the desktop

You can share a link when someone without an InstaBox account should see the desktop, such as a
customer who watches their agent work. `streamUrl()` makes a link to a full-screen viewer of one
machine's desktop, and the link works without signing in.

By default the link is view-only, so the viewer can watch but cannot type or click. When the viewer
should be able to control the desktop, pass `viewOnly: false`. The link works for one hour by
default, and you can set `ttlSeconds` to change that, from 60 to 86400.

```ts TypeScript theme={null}
const { url, expiresAt } = await sbx.desktop.streamUrl({
  viewOnly: true,
  ttlSeconds: 3600,
})
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../desktop/stream \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"viewOnly": true, "ttlSeconds": 3600}'
```

Both calls return `url`, which is the link to share, and `expiresAt`, which says when it stops
working.

> [!WARNING]
>
> Links belong to the machine's creator. Only the creator, signed in or with their personal key, may
> make or revoke them. A workspace API key, or another member of the workspace, gets
> `403 permission_denied`. A personal key reaches only the machines of your personal workspace.

Over HTTP, the stream calls follow the same rule, so call them with a personal key that has
`machines:write`.

### Embed the desktop in a page

You can also show the desktop inside your own page by putting the link in an `<iframe>`. Put the
link's `url` in place of `LINK_URL`.

```html theme={null}
<iframe src="LINK_URL" width="1280" height="800" style="border:0"></iframe>
```

### Revoke the links

Revoke the links when the people you shared them with should no longer see the desktop.
`revokeStreamUrls()` invalidates every link for the machine and closes open viewers within a few
seconds. Deleting the machine does the same.

```ts TypeScript theme={null}
await sbx.desktop.revokeStreamUrls()
```

```bash API theme={null}
curl -X DELETE https://api.instabox.sh/v1/machines/m_.../desktop/stream \
  -H "X-API-Key: $INSTABOX_API_KEY"
```

## Open the desktop without a link

Only the machine's creator can make a link. When you work with a workspace key, you can still
reach the desktop in two ways, and both work with that key:

* Open the desktop over `GET /v1/machines/{id}/vnc`, which carries VNC in a WebSocket for a viewer
  such as noVNC. It needs `machines:write`.
* Read the screen with `sbx.desktop.screenshot()`.

## Build a service for your own users

If you run a product where each of your users has a machine, you can show each user their own
desktop with links. Keep your API key on your server, never in a browser. Make a fresh link each
time a user opens their desktop, and revoke it when their access ends.

Links belong to the machine's creator, so this server runs on the personal key of the account that
creates the machines, and it creates them with that key too.

This Express route returns a link for the signed-in user. `machineOf` stands for your own mapping from
a user to a machine.

```ts theme={null}
app.get('/api/desktop-link', async (req, res) => {
  const machineId = await machineOf(req.user)
  const sbx = await Sandbox.connect(machineId)
  res.json(
    await sbx.desktop.streamUrl({
      viewOnly: !req.user.canControl,
      ttlSeconds: 600,
    }),
  )
})
```

When the user's access ends, revoke the machine's links so that an open viewer closes too.

```ts theme={null}
await (await Sandbox.connect(machineId)).desktop.revokeStreamUrls()
```

You will need to find a user's machines again later. To make that possible with
`Sandbox.list({ metadata })`, tag the machines with metadata such as `{ customer: 'acme' }`.

> [accordion] Relay the desktop with a workspace key
>
> You can also build the service on a workspace key, when you relay the desktop through your own
> server and use no links. Your server opens `GET /v1/machines/{id}/vnc`, a WebSocket, with a key
> that has `machines:write`. It passes the VNC stream to your user's viewer, or sends them
> `sbx.desktop.screenshot()` images. Your server then decides who sees what, and revoking means
> closing that connection.
>
> Each user has at most 8 open terminals and desktops at once, and a workspace key counts as its
> workspace's owner. A ninth `/vnc` connection gets a retryable `429 quota_exceeded`, so close a
> viewer's connection when they leave.

## Install the desktop over HTTP

The `universal` and `t3` templates have no desktop in the image, and over HTTP `start` does not
install anything. When you use the API with a machine from one of those templates, send `install`
first.

```bash theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../desktop \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"action": "install"}'
```

The install takes about half a minute, so poll `status` until `installed` is true before you send
`start`. This loop waits up to 5 minutes and stops early when the install fails.

```bash theme={null}
for i in $(seq 60); do
  curl -s -X POST https://api.instabox.sh/v1/machines/m_.../desktop \
    -H "X-API-Key: $INSTABOX_API_KEY" \
    -H "Content-Type: application/json" \
    -d '{"action": "status"}' \
    | jq -e '.installed or .failed' >/dev/null && break
  sleep 5
done
```

After the install, send `start` as shown in [Let an agent drive it](#let-an-agent-drive-it). If the
install failed, `start` says why. A `start` that started nothing answers with `installed` false and
a `hint` that names the next call to make.

## Limits

* **One machine.** A link opens that machine's desktop only. It is not an API key and cannot call the
  API.
* **View-only is enforced by the server.** Keyboard, mouse, and clipboard input from a view-only link
  never reach the machine.
* **Expiry stops new connections.** A viewer that is already connected stays connected until it
  leaves.
* **Revoke ends everything.** Revoking closes open viewers as well as stopping new ones.

> [card] [Computer use](https://instabox.sh/docs/use-cases/computer-use.md)
>
> An agent that browses and clicks on its own machine.

> [card] [SDK quick start](https://instabox.sh/docs/integrations/sdk-quickstart.md)
>
> Create, run and pause machines from code.
