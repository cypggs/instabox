> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Run AI-generated code

> Give a model a machine to run the code it writes, and send the output back to it.

A model that writes code needs somewhere to run it. Running it on your own server is a risk, because
you did not write the code and cannot review every line. On InstaBox the code runs in a machine of its
own, so a bad script cannot touch your server, and you delete the machine when you are done.

This page builds the loop that a code interpreter or a data-analysis agent uses: your program takes
code from a model, runs it in a machine, and returns the output, the errors, and the exit code to the
model.

**1. Install the SDK**

Install the SDK for your language.

```bash TypeScript theme={null}
npm install @insforge/instabox-sdk
```

```bash Python theme={null}
pip install instabox-sdk
```

Then set an API key, which both SDKs read from `INSTABOX_API_KEY`. Create the key on
**API keys** in the dashboard, and keep it on your server.

```bash theme={null}
export INSTABOX_API_KEY=ibk_...
```

**2. Create a machine**

Create one machine for the conversation. The call returns when the machine is running, so the
next step can use it at once.

```ts TypeScript theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const sbx = await Sandbox.create()
```

```python Python theme={null}
from instabox import Client

client = Client()
box = client.machines.create()
```

**3. Run the model's code**

Write the code to a file in the machine and run it with a timeout, so that a script that never
ends is stopped. Here `code` is the text that your model returned.

```ts TypeScript theme={null}
import { CommandExitError } from '@insforge/instabox-sdk'

async function runCode(code: string) {
  await sbx.files.write('/data/home/job.py', code)
  try {
    const r = await sbx.commands.run('python3 /data/home/job.py', {
      timeoutMs: 30_000,
    })
    return {
      stdout: r.stdout,
      stderr: r.stderr,
      exitCode: 0,
      truncated: r.truncated,
    }
  } catch (e) {
    if (e instanceof CommandExitError) {
      return {
        stdout: e.stdout,
        stderr: e.stderr,
        exitCode: e.exitCode,
        truncated: e.truncated,
      }
    }
    throw e
  }
}
```

```python Python theme={null}
def run_code(code: str):
    client.files.write(box.id, "/data/home/job.py", code)
    r = client.machines.exec(
        box.id, ["python3", "/data/home/job.py"], timeout=30
    )
    return {
        "stdout": r.stdout,
        "stderr": r.stderr,
        "exit_code": r.exit_code,
        "truncated": r.truncated,
    }
```

A script that prints `hello` returns `hello` in `stdout` with exit code 0. A script that raises an
error returns the traceback in `stderr` with a non-zero exit code, and a script that runs past
the timeout is stopped with exit code 124. In TypeScript a non-zero exit throws
`CommandExitError`, which carries the same fields. In Python it is a normal result.

**4. Send the result back to the model**

Give the model the output and let it decide what to do next. When the code failed, the model
reads the error and writes a fix, and your program runs the new code in the same machine. The
loop stops after five attempts, so that a model that keeps writing failing code does not run
without end.

```ts TypeScript theme={null}
let result = await runCode(await yourModel.writeCode(task))
for (let attempt = 1; attempt < 5 && result.exitCode !== 0; attempt++) {
  result = await runCode(await yourModel.fixCode(task, result))
}
```

```python Python theme={null}
result = run_code(your_model.write_code(task))
for attempt in range(4):
    if result["exit_code"] == 0:
        break
    result = run_code(your_model.fix_code(task, result))
```

`yourModel.writeCode` and `yourModel.fixCode`, and their Python forms, stand for your own calls
to a model.

**5. Delete the machine**

Delete the machine when the conversation ends. Its files are deleted with it. Put the delete in
your program's cleanup path, such as a `finally` block, so that the machine is also deleted when a
model call or an SDK call fails.

```ts TypeScript theme={null}
await sbx.kill()
```

```python Python theme={null}
client.machines.delete(box.id)
```

## Keep one machine for a whole conversation

Files and installed packages stay on the machine between runs, so a later script can read a file that
an earlier one wrote, and a package the model installed is still there. Keep the machine for as long
as the conversation lasts and you do not pay to set it up again for each run.

A machine that nobody uses pauses itself, and it wakes for the next command. This means an idle
conversation costs no runtime, and the next message finds the machine as it was. See
[Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md).

## Keep your keys and private data out of the machine

Code that a model wrote can read anything in the machine, and the machine can reach the internet. So
the code can also send out anything it can read, such as a repo or a data file that you copied in.
Put only what the task needs into the machine.

Do not put API keys there. When the code needs to call an API, save the key as an Org Secret. The
machine then holds only a placeholder, and the outbound proxy adds the real key to requests for the
hosts you allow. See [Org Secrets](https://instabox.sh/docs/security/secrets.md).

## Limits

* A command is stopped after 60 seconds unless you set a timeout, and the longest timeout is 600
  seconds. For longer work, run the command in the background. See
  [Run commands](https://instabox.sh/docs/machines/run-commands.md#run-a-long-command-in-the-background).
* Each of `stdout` and `stderr` is cut at 1 MiB, and `truncated` in the result says so. To get all
  of a long output, stream it. See
  [Run commands](https://instabox.sh/docs/machines/run-commands.md#stream-output-as-it-is-written).

> [card] [Run commands](https://instabox.sh/docs/machines/run-commands.md)
>
> Timeouts, streaming, input, and background jobs.

> [card] [Files](https://instabox.sh/docs/machines/files.md)
>
> Write and read files, and copy folders in and out.

> [card] [Snapshots](https://instabox.sh/docs/machines/snapshots.md)
>
> Start every machine with your packages already installed.

> [card] [Coding agents](https://instabox.sh/docs/use-cases/coding-agents.md)
>
> Run Claude Code, Codex, or OpenCode in a machine.
