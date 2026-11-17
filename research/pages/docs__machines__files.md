> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Files

> Copy files and folders between your computer and a machine, and read or write a single file.

The file commands copy files and folders between your computer and a machine. You can use them to
copy a project into a machine before a build, to copy the results back out afterwards, or to read
and write a single file from your own code.

```bash theme={null}
instabox push dev ./my-app
```

This copies the folder to `/data/home/my-app` in the machine named `dev`. Files under `/data`, which
is where the home directory `/data/home` lives, stay on the machine when you pause or stop it.

## Copy a folder into a machine

Push a local folder when you want to work on it in the machine, for example to build or test a
project there. When you give no destination, the folder keeps its name and lands in the home
directory. In a git repository, only the files that git tracks are copied, so anything that
you have not added to git is left out.

```bash CLI theme={null}
instabox push dev ./my-app
```

```ts TypeScript theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const sbx = await Sandbox.connect('dev')
await sbx.files.upload('./my-app', '/data/home/my-app')
```

```python Python theme={null}
from instabox import Client

client = Client()
client.files.upload("dev", "./my-app", "/data/home/my-app")
```

When you also need untracked files, add `--all` in the CLI, or pass `all: true` in the TypeScript SDK
and `all=True` in the Python SDK. This adds the untracked files that git does not ignore. Files that
git ignores, such as `node_modules`, are still left out.

## Copy a folder out of a machine

Pull a folder when you want something that the machine produced, such as build output or logs, on
your own computer. Everything under the folder is copied, dot files and empty directories included.
When you give no destination, the CLI copies into the current directory.

```bash CLI theme={null}
instabox pull dev /data/home/my-app/dist
```

```ts TypeScript theme={null}
await sbx.files.download('/data/home/my-app/dist', './dist')
```

```python Python theme={null}
client.files.download("dev", "/data/home/my-app/dist", "./dist")
```

## Copy one file

When you only need to move a single file, `instabox cp` copies it in either direction, in the style
of `scp`. You write the machine side as the machine's name, a colon, and the path.

```bash theme={null}
instabox cp ./backup.tgz dev:/data/home/backup.tgz
```

To copy the file out of the machine, put the machine side first and the local path second.

```bash theme={null}
instabox cp dev:/data/home/backup.tgz .
```

## Write a file from code

When your program needs to put a config file or some generated content into a machine, you can
write text or bytes straight to a path. The SDKs create the parent directories for you, so the
folder does not have to exist yet. The API creates them only when you add `parents=true`. Without
it, a missing folder gives `404 missing_parent`.

```ts TypeScript theme={null}
await sbx.files.write('/data/home/app/.env', 'NODE_ENV=production\n')
```

```python Python theme={null}
client.files.write("dev", "/data/home/app/.env", "NODE_ENV=production\n")
```

```bash API theme={null}
curl -X PUT \
  "https://api.instabox.sh/v1/machines/m_.../files?path=/data/home/app/.env&parents=true" \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @.env
```

## Read a file from code

You can read a file into memory as text, which suits a config file or a result that your program
wants to inspect. For a large file, read it as a stream instead, as
[TypeScript files](https://instabox.sh/docs/reference/typescript/files.md) shows.

```ts TypeScript theme={null}
const text = await sbx.files.read('/data/home/app/package.json')
```

```python Python theme={null}
text = client.files.read("dev", "/data/home/app/package.json")
```

```bash API theme={null}
curl "https://api.instabox.sh/v1/machines/m_.../files/content?path=/data/home/app/package.json" \
  -H "X-API-Key: $INSTABOX_API_KEY"
```

## List a directory

List a directory to see what is in it before you copy it, or to check that a file you expect is
there. The list shows the entries of that one directory and is not recursive.

```bash CLI theme={null}
instabox files dev /data/home/app
```

```ts TypeScript theme={null}
const entries = await sbx.files.list('/data/home/app')
```

```python Python theme={null}
entries = client.files.list("dev", "/data/home/app")
```

`instabox files` also prints a text file when the path is a file, so you can read a small file
without copying it.

## How a path is copied

Where a folder ends up depends on the destination that you give and on whether the source ends in
a slash. `push`, `pull`, and `cp` follow the same four rules for a directory. To copy a directory
with `cp`, add `-r`.

| You write | Where the files go |
| - | - |
| `push dev app /data/w`, and `/data/w` exists | `/data/w/app`: the folder lands inside an existing destination |
| `push dev app /data/w`, and `/data/w` does not exist | `/data/w`: a missing destination becomes the copy |
| `push dev app/ /data/w` | `/data/w`: a trailing slash copies the contents, not the folder |
| `push dev .` | `HOME/<dirname>`: with no destination, the copy keeps its name |

The destination itself does not have to exist, but its parent folder does. When the parent is
missing, `push` and `cp` fail with `missing_parent`. Add `-p` to create the missing parent folders.

## Limits

* `push` streams a single file of up to 5 GiB. For a larger file, split it, push the parts, and join
  them in the machine with `cat`.
* `pull` streams a single file of any size.
* A folder is copied as one compressed archive, and the archive has a size limit: 1 GiB for `push`
  and 5 GiB for `pull`. A `pull` past 32 MiB packs the archive in the machine's `/tmp` first, so it
  needs that much free space there. For a larger folder, copy it over `instabox ssh`.
* For large files, use `push` or `cp` and not `scp` or `rsync`. See
  [SSH](https://instabox.sh/docs/access/ssh.md#copy-large-files-with-the-cli-not-scp) for the reason.
* The file routes of the API wake a machine that paused itself when it was idle. A machine that is
  stopped, or that you paused, answers `409 machine_not_running`.

> [card] [Run commands](https://instabox.sh/docs/machines/run-commands.md)
>
> Build, test, and start servers on the files you copied.

> [card] [Snapshots](https://instabox.sh/docs/machines/snapshots.md)
>
> Save a machine's disk and start new machines from it.
