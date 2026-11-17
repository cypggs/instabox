> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Preview URLs

> Run your app's dev server in a machine and give a reviewer, a phone, or a webhook sender an HTTPS link to it.

An app that runs in a machine is only useful to others when they can open it: a reviewer, your
phone, or a service that sends webhooks. This guide gives the app's dev server an HTTPS URL that you
can send to anyone. When the app is not ready for everyone, you can make the link private, so that
only the people who have the link can open the app.

**1. Put your app in a machine**

Create a machine for the app.

```bash theme={null}
instabox create --name dev
```

Go to your app's folder on your computer. On this page, `my-app` and `npm run dev` stand for your
own app's folder and its start command.

```bash theme={null}
cd my-app
```

Then push the folder to the machine.

```bash theme={null}
instabox push dev .
```

`push` reports how many files it pushed to `dev:/data/home/my-app`.

**2. Start the dev server**

Start the server as a background job, so that it keeps running after the command returns. The
steps below assume that your server listens on port 3000.

```bash theme={null}
instabox exec --background dev \
  --cwd /data/home/my-app -- npm run dev
```

The command prints the job's ID and the `instabox jobs logs` command that follows the job's
output.

**3. Publish the port**

Publish the server's port to get a URL for it. You can give the port a name with `--name`, which
tells it apart in the list of ports.

```bash theme={null}
instabox port publish dev 3000 --name 'Web app'
```

The command prints the URL, which has this form and stays the same until you unpublish the port.

```text theme={null}
https://<port>-<machine id without m_>.<domain>/
```

> [!WARNING]
>
> A public URL is not a secret. Anyone can reach the whole service until you unpublish it.

**4. Share the link**

Open the printed URL in a browser, and you see your app, because the request goes to your server
on port 3000 in the machine. Then send the URL to the people or the service that need it.

**5. Remove the link when you are done**

List the machine's ports to check what is published. The list shows each published port with its
access and its URL.

```bash theme={null}
instabox port ls dev
```

When nobody needs the link any longer, unpublish the port, which removes the URL and closes the
connections that are open on it.

```bash theme={null}
instabox port unpublish dev 3000
```

A visit to the URL wakes a machine that paused itself for being idle, but a machine that you paused
stays offline until you resume it. See [Reach a paused machine](https://instabox.sh/docs/access/ports.md#reach-a-paused-machine).

## Share a private link instead

When only some people should open the app, publish the port as private. A private port opens only
with the link that carries its token, and the command prints that link, with `?_token=` in it, in
place of the plain URL.

```bash theme={null}
instabox port publish dev 3000 --private
```

Send the printed link to the people who need the app. A browser that opens the link gets a cookie and
goes on to the plain URL, WebSockets included. A script sends the token in an
`Authorization: Bearer ibp_...` header instead, and every other request gets `401`.

When the link has reached someone who should not have it, you can rotate the token. The command
prints a new link, and the old link stops working.

```bash theme={null}
instabox port publish dev 3000 --private --rotate-token
```

When nobody else needs the app at all, you can skip the URL and forward the port to your own
computer. See [Reach a port only from your computer](https://instabox.sh/docs/access/ports.md#reach-a-port-only-from-your-computer).

## Publish from inside the machine

An agent that works inside the machine can publish a port itself with `instabox-port`, so it can hand
you a link to what it built.

```bash theme={null}
instabox-port publish 3000 --name 'Web app'
```

The agent can also serve a folder that has an `index.html`, such as a report, and publish it in one
step.

```bash theme={null}
instabox-port serve ./report --port 8080
```

If `instabox-port` is missing from a machine, run this command on your computer to install it.

```bash theme={null}
instabox port setup dev
```

## Limits

A machine can publish a limited number of ports, and a response header has a size limit. See
[Limits](https://instabox.sh/docs/access/ports.md#limits) on the Public and private ports page.

> [accordion] Start a server with a plain exec
>
> `--background` keeps the server running after the command returns. With a plain `exec`, start
> the server with `nohup ./server >server.log 2>&1 &`, and redirect both stdout and stderr.

> [card] [Public and private ports](https://instabox.sh/docs/access/ports.md)
>
> Public, private, and raw TCP ports in full.

> [card] [Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md)
>
> When a machine pauses and what wakes it.
