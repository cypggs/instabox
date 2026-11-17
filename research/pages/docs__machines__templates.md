> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Templates

> What a new machine starts from, how to list the templates, and how to pick one.

A template is what a new machine starts from: its operating system, its root disk, and the tools that
come preinstalled. When you name no template, a machine starts from `agent-browser`.

The templates, with the operating system, root disk size and tools of each, are listed in
[Limits](https://instabox.sh/docs/account/limits.md#machine-specs). GPU machines start from `gpu` ([GPU templates](https://instabox.sh/docs/concepts.md#gpu-templates)),
and macOS machines from `macos` ([macOS machines](https://instabox.sh/docs/machines/macos.md)).

> [!NOTE]
>
> The free plan creates from agent-browser, android, omarchy, ci only. Any other template is refused with `403 template_not_allowed` on the free plan. A machine started from one of your snapshots is not limited by this.

## List the templates

```bash theme={null}
instabox templates
```

This prints one row per template: the version a new machine gets when you name the template without
a version. `instabox templates --all` lists every version, old ones too, with a `CREATABLE` column.
The list is not filtered by your plan.

Over the API, `GET /v1/templates` returns every version. Read these fields on each entry:

* `creatable` is `true` when a create may name this version today.
* `latest` is `true` on the one version per template that a create without a version gets.
* `defaultSpec.rootDiskGiB` is the size of the root disk a new machine gets.

```bash theme={null}
curl https://api.instabox.sh/v1/templates \
  -H "X-API-Key: $INSTABOX_API_KEY"
```

## Create a machine from a template

Name the template when you create the machine. Over the API the field is `templateId` on
`POST /v1/machines`.

```bash CLI theme={null}
instabox create --name dev --template agent-browser
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"name": "dev", "templateId": "agent-browser"}'
```

A new machine gets the version marked `latest`. To get one exact version, for example to keep a test
run reproducible, [pin a template version](https://instabox.sh/docs/machines/sizes.md#pin-a-template-version).

> [card] [Limits](https://instabox.sh/docs/account/limits.md#machine-specs)
>
> Root disk, OS and tool versions of each template.

> [card] [Custom images](https://instabox.sh/docs/machines/custom-images.md)
>
> Start machines from your own Docker or OCI image.
