> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Computer use

> Have a model drive a browser on a machine's desktop while you watch it live or share a link.

An agent that uses a computer the way a person does needs a screen to look at and a mouse and
keyboard to drive. Every `agent-browser` machine (the default template) has a Linux desktop with a
browser for that. You write a short loop
that sends screenshots to a vision model and carries out the clicks and keys that the model answers
with. While the model works, you watch the same screen live in the dashboard, or you share a link
to it.

**1. Install the SDK**

Install the TypeScript SDK in the project that runs your agent.

```bash theme={null}
npm install @insforge/instabox-sdk
```

The SDK reads your API key from `INSTABOX_API_KEY`, so set that variable to your own key. Keep
the key on your server, never in a browser.

```bash theme={null}
export INSTABOX_API_KEY=ibk_...
```

**2. Start the desktop**

Create a machine and start its desktop at the screen size you want the model to see. You can
call `start()` again at any time, because it leaves a running desktop alone.

```ts theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const sbx = await Sandbox.create('agent-browser')
await sbx.desktop.start({ width: 1280, height: 800 })
```

Then call `open()`, which opens the desktop's browser at the URL you give it.

```ts theme={null}
await sbx.desktop.open('https://example.com')
```

The machine's browser now shows `https://example.com` on the desktop.

**3. Let the model drive**

The model works in a loop. Each pass takes a screenshot, which `screenshot()` returns as PNG
bytes for a vision model, and then carries out the action that the model returns. In this loop,
`yourModel.nextAction` stands for your own call to a vision model. It returns one of
`{ type: 'click', x, y }`, `{ type: 'type', text }`, or `{ type: 'done' }`.

```ts theme={null}
for (let step = 0; step < 20; step++) {
  const png = await sbx.desktop.screenshot()
  const action = await yourModel.nextAction(png)
  if (action.type === 'done') break
  if (action.type === 'click') {
    await sbx.desktop.click(action.x, action.y)
  }
  if (action.type === 'type') await sbx.desktop.type(action.text)
}
```

The loop ends when the model answers `done` or after 20 steps have run.

The desktop also takes key presses, scrolling, mouse moves, drags, and right and double clicks,
so you can give the model more actions. [Let an agent drive it](https://instabox.sh/docs/access/desktop.md#let-an-agent-drive-it)
shows each call.

**4. Watch the desktop**

In the dashboard, open the machine and choose the **Computer** tab. You see the desktop as the
model drives it. You can click the desktop to take control and use the mouse and keyboard
yourself. Control goes back to the agent after a minute without input, or 30 seconds after you
close the tab.

**5. Share a link**

You can share a link when someone else should see the desktop too. `streamUrl()` makes a link
that opens a full-screen viewer of this machine's desktop with no sign-in. The link is not an
API key.

By default the link is view-only, so the viewer can watch but cannot type or click. The link
works for one hour by default, and you can set `ttlSeconds` to change that, from 60 to 86400.

```ts theme={null}
const { url, expiresAt } = await sbx.desktop.streamUrl({
  viewOnly: true,
  ttlSeconds: 3600,
})
```

The call returns the link as `url`, and the time when the link expires as `expiresAt`. Open
`url` in a browser and you see the desktop full screen.

You can also show the desktop inside your own page. Put `url` in place of `LINK_URL`.

```html theme={null}
<iframe
  src="LINK_URL"
  width="1280"
  height="800"
  style="border:0"
></iframe>
```

When the people you shared a link with should no longer see the desktop, revoke the links.
This call ends every link for the machine and closes open viewers within a few seconds.

```ts theme={null}
await sbx.desktop.revokeStreamUrls()
```

> [!WARNING]
>
> Links belong to the machine's creator. Only the person who created the machine, signed in or with
> their personal key, can make or revoke them. A workspace API key gets `403 permission_denied`. It
> opens the desktop over `GET /v1/machines/{id}/vnc` instead, as
> [Open the desktop without a link](https://instabox.sh/docs/access/desktop.md#open-the-desktop-without-a-link) describes.

## Build on the desktop

The [Desktop](https://instabox.sh/docs/access/desktop.md) page covers what comes after this first run.

* When each of your own users has a machine, your server can make a fresh link each time a user
  opens their desktop. [Build a service for your own users](https://instabox.sh/docs/access/desktop.md#build-a-service-for-your-own-users)
  has the server route.
* When your server runs on a workspace key, it can relay the desktop over VNC and use no links.
  The same section explains the relay and how many connections each user can hold open.
* When you do not use the SDK, you can start the desktop and make or revoke links over HTTP. The
  **API** tab of each example on the [Desktop](https://instabox.sh/docs/access/desktop.md) page shows the request.

> [card] [Desktop](https://instabox.sh/docs/access/desktop.md)
>
> Every desktop call, viewers, and link rules.

> [card] [SDK reference](https://instabox.sh/docs/reference/sdk.md)
>
> `Sandbox` and `sbx.desktop` in full.
