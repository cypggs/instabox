> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Public and private ports

> Give a web server in a machine an HTTPS URL, for anyone or only for people with the link.

A published port gives a web server inside a machine its own HTTPS URL. You can open the app in a
browser, share it with a teammate, or point a webhook at it.

Choose who can reach the URL:

* **Public:** anyone who has the URL. Use it for a demo or a webhook receiver.
* **Private:** only the people you send a link to. Use it for work that is not ready to show.

```bash theme={null}
instabox port publish dev 3000
```

The command prints the URL. It stays the same until you unpublish the port.

```text theme={null}
https://<port>-<machine id without m_>.<domain>/
```

A request to that URL goes to the InstaBox proxy, which passes it to your server on that port inside
the machine. For a private port, the proxy first checks that the request carries the port's token.

```mermaid theme={null}
flowchart LR
  A["Browser, teammate, or webhook"] -->|"HTTPS URL"| B["InstaBox proxy"]
  B -->|"port 3000"| C["Your server in the machine"]
```

## Publish a port

By default a published port is public, so anyone who has the URL can open it. Publishing only creates
the URL and does not start a server, so start your server on that port first. You can give the port a
name to tell it apart in the list.

```bash CLI theme={null}
instabox port publish dev 3000 --name 'Web app'
```

```bash In the machine theme={null}
instabox-port publish 3000 --name 'Web app'
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../ports -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Content-Type: application/json" -d '{"port": 3000, "name": "Web app"}'
```

You can also publish and unpublish ports in the dashboard, on the machine's **Ports** tab.

> [!WARNING]
>
> The URL is not a secret. A public port lets anyone reach the whole service until you unpublish it.

## Make a port private

A private port answers only requests that carry its token. The publish prints a link that contains
the token. The link is shown once, so copy it and share it with the people who need the app.

```bash CLI theme={null}
instabox port publish dev 3000 --private
```

```bash In the machine theme={null}
instabox-port publish 3000 --private
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../ports -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Content-Type: application/json" -d '{"port": 3000, "access": "private"}'
```

In the API response, the link is the `tokenUrl` field.

* **In a browser,** a person opens the link once. The link sets a cookie, so every later request
  from that browser gets through, WebSockets included.
* **From a script,** send the token in an `Authorization: Bearer ibp_…` header.

A request that does not carry the token gets `401`, and it does not wake a paused machine.

## Replace a private link

When a link has reached someone who should not have it, rotate the token to get a new link. The old
link stops working at once, and the connections that used it are closed.

```bash CLI theme={null}
instabox port publish dev 3000 --private --rotate-token
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../ports -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Content-Type: application/json" -d '{"port": 3000, "access": "private", "rotateToken": true}'
```

When the app is ready for everyone, you can make the port public again by publishing it with
`--public`, or `"access": "public"` in the API.

## List and unpublish ports

List a machine's ports to see their URLs and which ones are private.

```bash CLI theme={null}
instabox port ls dev
```

```bash API theme={null}
curl https://api.instabox.sh/v1/machines/m_.../ports \
  -H "X-API-Key: $INSTABOX_API_KEY"
```

Unpublish a port when you no longer want it reachable. Unpublishing removes the URL and closes the
connections that are open on it.

```bash CLI theme={null}
instabox port unpublish dev 3000
```

```bash API theme={null}
curl -X DELETE https://api.instabox.sh/v1/machines/m_.../ports/3000 \
  -H "X-API-Key: $INSTABOX_API_KEY"
```

## Serve a folder

An agent that works inside the machine often wants to show a result, such as a report it built. Inside
the machine, `instabox-port serve` starts a static server for a folder that has an `index.html` and
publishes it in one step.

```bash theme={null}
instabox-port serve ./report --port 8080
```

If `instabox-port` is missing from a machine, run `instabox port setup dev` to install it.

## Reach a paused machine

A visit to the URL wakes a machine that paused itself after 30 idle minutes. The
request waits for the resume, so a phone app or a webhook can call the URL without starting the
machine first. `instabox wake dev` does the same without a request.

A machine that **you** paused or stopped stays offline until you resume or start it. The URL is the
same afterwards. After a stop, start your server again.

## Reach a port only from your computer

When a service should not have a URL at all, you can forward its port over SSH and reach it on your
own computer. This works for any TCP service, such as a database or a debugger. See
[SSH](https://instabox.sh/docs/access/ssh.md#forward-a-port-to-your-computer).

```bash theme={null}
instabox port-forward dev 8080:3000
```

## Limits

* A machine has up to 16 published ports.
* Ports 1024 to 65535 can be published. The desktop port 5901 and the Orca port 6768 are reserved.
* A private link is shown once, because the server keeps only a hash of the token. Rotate the token
  to get a new one.
* Renaming a private port keeps it private. Making it public with `--public` drops the token.
* The private link carries the token as `?_token=ibp_…`. Opening it sets the cookie
  `__Host-instabox_port_auth` and redirects to the plain URL.
* A response header (the status line and all fields) may be up to 24 KiB. A larger one, such as one
  with many large cookies, gets the platform's `502` page instead of the app's response.

> [accordion] Raw TCP ports (preview)
>
> Where the server enables it, `instabox port publish dev 5432 --tcp` prints
> `5432-<id>.<tcp domain>:443` for raw TCP, such as Postgres or Redis. Connect with TLS and that name
> as SNI:
>
> ```bash theme={null}
> psql "host=5432-<id>.<tcp domain> port=443 sslmode=require user=postgres"
> openssl s_client -connect <address> -servername <host>
> ```
>
> psql and libpq 14 and later send SNI. PG17 and later may add `sslnegotiation=direct`. For older
> clients, use stunnel or `instabox port-forward`.
>
> * The service in the machine sees plain TCP and must use its own password.
> * A raw TCP port is always public: `--private` does not apply to it.
> * It is in preview, for admins only, until the server opens it to everyone. Then it is for paid
>   plans only, with a per-workspace limit.
> * A connection counts as activity only while bytes flow. An idle pooled connection does not keep
>   the machine awake, so keep pool idle timeouts under 30 minutes or send keepalives.

> [card] [Preview URLs](https://instabox.sh/docs/use-cases/preview-urls.md)
>
> Share a running app from an agent's machine.

> [card] [Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md)
>
> When a machine pauses itself and what wakes it.
