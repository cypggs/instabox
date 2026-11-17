> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Android apps

> Run your app on Android 14 in a machine, drive it with adb, and watch the phone in the dashboard.

You can test an Android app, or build an agent that drives a phone, without a device or an emulator
on your computer. You create a machine from the `android` template, install your app on its Android 14 with adb, and then tap
the screen and take screenshots from the command line while you watch the phone in the dashboard.
Android runs as a container on the machine's own Linux kernel, with no emulator and no nested VM.
The machine also has everything `agent-browser` has: Docker, Node.js, Python, and the desktop.

**1. Create an Android machine**

Create a machine from the `android` template. Use the default size or larger, because Android
uses 1 to 2 GiB of memory while it runs.

```bash theme={null}
instabox create --name phone --template android
```

Android is up about 20 seconds after the machine starts.

**2. Wait for Android to boot**

Right after a create or a fork, `adb devices` is empty until Android has booted. This command
waits in the machine until Android reports `sys.boot_completed`, so run it before you use adb.

```bash theme={null}
instabox exec phone -- 'until [ "$(docker exec instabox-android \
  getprop sys.boot_completed 2>/dev/null)" = 1 ]; do sleep 1; done'
```

The command prints nothing. When it returns, Android has booted.

**3. Install your app**

Copy your app's APK into the machine. In this step, `app-debug.apk` stands for your own APK,
and `push` puts it in the machine's home.

```bash theme={null}
instabox push phone ./app-debug.apk
```

Then install the APK on Android with adb, which runs in the machine.

```bash theme={null}
instabox exec phone -- adb install app-debug.apk
```

`adb install` prints its result when the install ends.

**4. Drive and test the app**

Open a shell in the machine, so that you can run the adb commands of this step there.

```bash theme={null}
instabox ssh phone
```

First check that adb sees the phone. The list shows it as `emulator-5554` or as
`127.0.0.1:5555`.

```bash theme={null}
adb devices
```

You can then send input to the phone. This command taps the screen at x 360 and y 640.

```bash theme={null}
adb shell input tap 360 640
```

To see the result of the tap, take a screenshot of the phone. This command saves the
screenshot to `screen.png` in the machine.

```bash theme={null}
adb exec-out screencap -p > screen.png
```

**5. See the phone**

Open the machine in the dashboard and choose the **Computer** tab. The phone is the **Android**
window on the desktop, and it takes clicks and typing. An agent drives the phone the same way,
with the desktop API or with `adb shell input`, as [Computer use](https://instabox.sh/docs/use-cases/computer-use.md)
shows.

> [!NOTE]
>
> The free plan creates from agent-browser, android, omarchy, ci, so `android` works on every plan.

## Limits of the android template

* **No Google Play or Google services.** Install apps with `adb install`. Apps that need Google
  services for sign-in, maps, or push do not fully work.
* **x86\_64 only.** Apps that ship only ARM native code do not run, because there is no ARM
  translation. Apps with no native code, or with x86\_64 libraries, run.
* **Software rendering.** It is fine for app testing and UI automation, and slow for games and heavy 3D.

> [accordion] adb lists nothing
>
> If `adb devices` still lists nothing after Android has booted, connect adb to the phone once.
>
> ```bash theme={null}
> adb connect 127.0.0.1:5555
> ```
>
> Do not connect while `emulator-5554` is listed. When the same phone is listed twice,
> `adb shell` asks which one to use.

> [accordion] Pause, idle pause, and fork
>
> Pause and resume, and idle pause and wake, keep the same running Android, so apps, their state,
> and the clock carry on. A fork copies Android's data, which is the installed apps and their
> data, and boots its own Android from it about 20 seconds later. Apps that were running are not
> carried over to the fork.

> [accordion] Free Android's memory
>
> When you do not need Android for a while, you can stop it in the machine to free its memory.
>
> ```bash theme={null}
> sudo systemctl stop instabox-android
> ```
>
> To keep Android off across restarts, also disable its service.
>
> ```bash theme={null}
> sudo systemctl disable instabox-android
> ```

> [card] [Computer use](https://instabox.sh/docs/use-cases/computer-use.md)
>
> Screenshots, clicks, and keys from your code.

> [card] [Parallel forks](https://instabox.sh/docs/use-cases/parallel-forks.md)
>
> Copy a set-up phone into several machines.
