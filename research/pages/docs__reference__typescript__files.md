> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Files

> Read, write, stream, upload, download, list and remove files in a machine with sbx.files.

`read`, `readStream` and `write` use the API's file routes (`GET /v1/machines/{id}/files/content` and
`PUT /v1/machines/{id}/files`), the path `instabox cp` takes. They need the same scope as commands. A
file can be up to 5 GiB (`MAX_UPLOAD_BYTES`). `list`, `exists` and `remove` run as commands. Relative
paths start at HOME, `/data/home`, on the persistent disk.

```ts theme={null}
import { createReadStream, createWriteStream } from 'node:fs'
import { Readable } from 'node:stream'
import { pipeline } from 'node:stream/promises'

await sbx.files.write('data.bin', createReadStream('data.bin'))
await pipeline(Readable.fromWeb(await sbx.files.readStream('data.bin')), createWriteStream('copy.bin'))
await sbx.files.upload('my-app', 'my-app')
await sbx.files.download('my-app/dist', 'dist')
```

## files.read

Read a file into memory, as UTF-8 text, or as bytes with `{ format: 'bytes' }`.

```ts theme={null}
read(path: string, opts?: { format?: 'text'; maxBytes?: number }): Promise<string>
read(path: string, opts: { format: 'bytes'; maxBytes?: number }): Promise<Uint8Array>
```

A file over `maxBytes` (default 10 MiB, `MAX_FILE_BYTES`) throws `file_too_large`. Pass a larger
`maxBytes`, or use `readStream`. A text read of a file that is not UTF-8 throws `not_utf8`.

## files.readStream

Read a file of any size as a stream, in bounded memory.

```ts theme={null}
readStream(path: string, opts?: { signal?: AbortSignal }): Promise<FileStream>
```

Returns a `ReadableStream<Uint8Array>` that also works with `for await`. In Node,
`Readable.fromWeb(stream)` makes it a Node stream. A missing file or a directory throws `not_found`
before any byte. A stream that ends short of the file's size errors with `file_changed`.

## files.write

Write a file: create its parent directories and replace the file atomically.

```ts theme={null}
write(path: string, data: WriteData, opts?: { signal?: AbortSignal }): Promise<WriteInfo>
```

`data` is a string, bytes (`Uint8Array`, `Buffer`, `ArrayBuffer`) or a stream: a Node `Readable`, a web
`ReadableStream`, or an async iterable of chunks. A stream is sent as it is read, so memory stays flat.
The SDK checks the size and SHA-256 the server reports.

Returns `{ path }`. Writing to a directory throws `is_a_directory`. An error from your stream throws
`source_error`, and the file keeps its old content. Past 5 GiB the server answers `too_large`.

## files.upload

Upload the contents of a local directory into `remoteDir`, as one gzipped tar. This is what
`instabox push` does.

```ts theme={null}
upload(localDir: string, remoteDir: string, opts?: { git?: boolean; all?: boolean; signal?: AbortSignal }): Promise<UploadInfo>
```

* In a git work tree, only the files git tracks go up. `all: true` adds untracked files that git does
  not ignore. `git: false` sends everything under the directory.
* Outside a git work tree, everything goes up, dot files and empty directories included.
* Symlinks stay symlinks, and file modes are kept.

Returns `{ path, files, bytes }`: the machine directory, the regular files sent and the archive size. A
missing local directory throws `local_not_found`.

## files.download

Download a machine directory into `localDir`, as one gzipped tar. This is what `instabox pull` does.

```ts theme={null}
download(remoteDir: string, localDir: string, opts?: { signal?: AbortSignal }): Promise<DownloadInfo>
```

Everything under `remoteDir` comes, dot files and empty directories included. Returns
`{ path, remotePath, files, bytes }`. A missing machine directory throws `not_found`.

`upload` and `download` need Node.js, Bun or Deno. In a browser or a Worker they throw `unsupported`.
Other errors: `local_io_error` (the local disk), `not_a_directory` (a file where a directory was
expected) and `transfer_failed` (`tar` failed in the machine).

## files.list

List a directory. The list is not recursive.

```ts theme={null}
list(path: string): Promise<EntryInfo[]>
```

Returns `{ name, path, type }` entries, with `type` `'file'` or `'dir'`.

## files.remove

Delete a file, or a directory recursively.

```ts theme={null}
remove(path: string): Promise<void>
```

A missing path is not an error.

## files.exists

Check whether a path exists.

```ts theme={null}
exists(path: string): Promise<boolean>
```
