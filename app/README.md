# LibreQuenda for Android

The residents' app, native Kotlin / Jetpack Compose, on the same API as the web:

- **Home**: the next booking, «For you» (if the preferences allow it), public games nearby.
- **Facilities**: search by text, date, time, sport, tags and council, sorted by distance
  in the API; a facility's page, turns and public games to join; checkout with its rules.
- **Bookings**: upcoming and history, each booking's page (invite, public, cancel, calendar).
- **Ratings**: once a booking ends, the facility's stars and, after a public one, the others'
  with an account; given once, for good. The averages show on facilities and the profile.
- **Checking in** with every way the facility takes: the booking's QR at the reader, the
  entry code, the door's printed QR, the NFC tag on the door, or the staff.
- **Flash booking**: book a free turn on the spot and go in, the same ways, with the
  personal QR for the readers.
- **Notices** without push: reminders as local alarms, the rest checked in the background,
  and right after each booking ends, so the request to rate it comes on time.
- A first-run tour (after signing in or «Explore without an account»), skippable, and again from the menu.
- Profile, preferences, the app's language (gl, es, ca, en), support and legal texts.

Texts shared with the web are generated from its locales: after changing
`web/src/locales`, run `python3 app/tools/web_strings.py web/src app/app/src/main/res`
from the repository root (CI checks it).

## Requirements

JDK 17, Android SDK platform 36 and build tools 36.0.0, Android CLI, and a running
Android emulator. Gradle and Kotlin are pinned by the project and downloaded
automatically. Android Studio is optional.

Set `ANDROID_HOME` to your SDK directory (usually `$HOME/Android/Sdk`), or set
`sdk.dir` in an untracked `local.properties`.

From this directory:

```sh
android emulator start lq_dev
"$ANDROID_HOME/platform-tools/adb" root
"$ANDROID_HOME/platform-tools/adb" reverse tcp:80 tcp:80
"$ANDROID_HOME/platform-tools/adb" reverse tcp:8080 tcp:80
./gradlew :app:assembleDebug
android run --apks=app/build/outputs/apk/debug/app-debug.apk
```

**Signing in needs port 80 on the emulator.** Zitadel's issuer in the development
stack is `http://auth.localhost` (port 80), and it picks its instance from that host,
so the sign-in page must be reached at exactly that address. Inside the emulator
`*.localhost` is 127.0.0.1, and `adb reverse tcp:80 tcp:80` forwards it to the
laptop's Caddy, but binding port 80 needs `adb root`, which only works on images
without Google Play. Create the AVD from `system-images;android-36;google_apis;x86_64`
(here `lq_dev`); a Play Store image can browse councils but cannot sign in.

Repeat the last two commands after code changes. This rebuilds and restarts the
app; it does not provide hot reload.

## Development API

The default is `http://localhost:8080/api`. The adb reverse command above forwards
emulator port 8080 to the laptop's Caddy proxy on port 80. It preserves the
localhost hostname required by the development Caddy configuration. Start the existing
LibreQuenda development stack first. Repeat the port-forward command after restarting the emulator. Without it,
localhost refers to the emulator itself.

Override the URL at build time, without a server picker:

```sh
ANDROID_API_BASE_URL=https://reservas.example.org/api ./gradlew :app:assembleDebug
```

Alternatively use `-PapiBaseUrl=https://reservas.example.org/api` (takes precedence).
Only this public URL is embedded; never pass server secrets to the Android build.
Plain HTTP is enabled only in debug builds. Release deployments need HTTPS.

## Checks

```sh
./gradlew :app:assembleDebug :app:lintDebug :app:testDebugUnitTest :app:connectedDebugAndroidTest
```

In debug builds, a long press on the «Reserva flash» title plays the flash booking's
celebration, to try it without booking.

## CI/CD

The `Android app` job in `.github/workflows/ci.yml` runs on every pull request and push:
the texts match the web, lint, unit tests and a debug APK (kept as an artifact for 14
days). On a version tag (`v1.2.3`) it also builds a signed release APK (versionName
`1.2.3`) and attaches it to the GitHub release.

Set in the repository (Settings → Secrets and variables → Actions):

| Name | Kind | What |
|---|---|---|
| `ANDROID_API_BASE_URL` | variable | the installation's API, e.g. `https://reservas.example.gal/api` |
| `ANDROID_KEYSTORE_BASE64` | secret | the release keystore: `base64 -w0 release.jks` |
| `ANDROID_KEYSTORE_PASSWORD` | secret | its password |
| `ANDROID_KEY_ALIAS` | secret | the key's alias |
| `ANDROID_KEY_PASSWORD` | secret | the key's password |

Without the secrets the release step is skipped; the keystore never enters the repository.

## Sign-in

Zitadel's own sign-in page in a Custom Tab (OIDC authorization code with PKCE, through
AppAuth), as the web does it. `zitadel/setup.sh` creates the native OIDC app
`android` (redirect `gal.tribo.librequenda:/oauth2redirect`) and publishes its client
id as `androidClientId` in `/config.json`, which the app reads at sign-in. Tokens are
kept in the app's private storage and renewed with the refresh token; signing out
also ends the session at Zitadel.

Not on the Play Store yet: the release APK is installed by hand (or through another store).

## Third-party

The app's typeface, DM Sans (`app/src/main/res/font/dm_sans.ttf`), is under the SIL Open Font
License 1.1: see [third-party/DM-Sans-OFL.txt](third-party/DM-Sans-OFL.txt).
