# LibreQuenda para Android

A app da veciñanza, nativa en Kotlin / Jetpack Compose, sobre a mesma API ca a web:

- **Inicio**: a próxima reserva, «Para ti» (se as preferencias o permiten), partidas públicas próximas.
- **Instalacións**: busca por texto, data, hora, deporte, etiquetas e concello, ordenada por distancia
  na API; a páxina dunha instalación, as quendas e as partidas públicas ás que unirse; a confirmación da reserva coas súas normas.
- **Reservas**: próximas e historial, a páxina de cada reserva (convidar, pública, cancelar, calendario).
- **Valoracións**: cando remata unha reserva, as estrelas da instalación e, despois dunha pública, as das outras persoas
  con conta; dáse unha vez e para sempre. As medias aparecen nas instalacións e no perfil.
- **Rexistrar a entrada** con todas as formas que admite a instalación: o QR da reserva no lector, o
  código de entrada, o QR impreso da porta, a etiqueta NFC da porta ou o persoal.
- **Reserva flash**: reserva unha quenda libre no momento e entra, das mesmas formas, co
  QR persoal para os lectores.
- **Avisos** sen notificacións push: os recordatorios como alarmas locais, o resto comprobado en segundo plano
  e xusto despois de que remate cada reserva, para que a petición de valorala chegue a tempo.
- Un percorrido inicial (despois de iniciar sesión ou de «Explorar sen conta»), que se pode saltar, e que se pode repetir desde o menú.
- Perfil, preferencias, o idioma da app (gl, es, ca, en), asistencia e textos legais.

Os textos compartidos coa web xéranse a partir das súas traducións: despois de cambiar
`web/src/locales`, executa `python3 app/tools/web_strings.py web/src app/app/src/main/res`
desde a raíz do repositorio (a CI compróbao).

## Requisitos

JDK 17, a plataforma 36 do Android SDK e as build tools 36.0.0, Android CLI e un emulador
de Android en marcha. Gradle e Kotlin teñen a versión fixada no proxecto e descárganse
automaticamente. Android Studio é opcional.

Define `ANDROID_HOME` co directorio do teu SDK (normalmente `$HOME/Android/Sdk`) ou define
`sdk.dir` nun `local.properties` sen seguimento en git.

Desde este directorio:

```sh
android emulator start lq_dev
"$ANDROID_HOME/platform-tools/adb" root
"$ANDROID_HOME/platform-tools/adb" reverse tcp:80 tcp:80
"$ANDROID_HOME/platform-tools/adb" reverse tcp:8080 tcp:80
./gradlew :app:assembleDebug
android run --apks=app/build/outputs/apk/debug/app-debug.apk
```

**Para iniciar sesión cómpre o porto 80 no emulador.** O emisor (issuer) de Zitadel na pila de
desenvolvemento é `http://auth.localhost` (porto 80), e Zitadel escolle a súa instancia a partir dese host,
así que hai que chegar á páxina de inicio de sesión exactamente nese enderezo. Dentro do emulador
`*.localhost` é 127.0.0.1, e `adb reverse tcp:80 tcp:80` reenvíao ao Caddy do
portátil, pero para usar o porto 80 cómpre `adb root`, que só funciona en imaxes
sen Google Play. Crea o AVD a partir de `system-images;android-36;google_apis;x86_64`
(aquí `lq_dev`); unha imaxe con Play Store pode navegar polos concellos pero non pode iniciar sesión.

Repite as dúas últimas ordes despois de cambiar o código. Isto recompila e reinicia a
app; non ofrece recarga en quente.

## API de desenvolvemento

O valor predeterminado é `http://localhost:8080/api`. A orde adb reverse de enriba reenvía
o porto 8080 do emulador ao proxy Caddy do portátil no porto 80. Así consérvase o nome de host
localhost que require a configuración de Caddy de desenvolvemento. Arrinca primeiro a pila de
desenvolvemento de LibreQuenda existente. Repite a orde de reenvío de portos despois de reiniciar o emulador. Sen ela,
localhost é o propio emulador.

Cambia o URL no momento da compilación, sen selector de servidor:

```sh
ANDROID_API_BASE_URL=https://reservas.example.org/api ./gradlew :app:assembleDebug
```

Tamén podes usar `-PapiBaseUrl=https://reservas.example.org/api` (ten prioridade).
Só se inclúe este URL público; nunca lle pases segredos do servidor á compilación de Android.
O HTTP sen cifrar só está activado nas compilacións de depuración. Os despregamentos con compilacións de publicación precisan HTTPS.

## Comprobacións

```sh
./gradlew :app:assembleDebug :app:lintDebug :app:testDebugUnitTest :app:connectedDebugAndroidTest
```

Nas compilacións de depuración, unha pulsación longa no título «Reserva flash» reproduce a
celebración da reserva flash, para probala sen reservar.

## CI/CD

O traballo `Android app` de `.github/workflows/ci.yml` execútase en cada pull request e en cada push:
que os textos coincidan coa web, lint, probas unitarias e un APK de depuración (gardado como artefacto durante 14
días). Cunha etiqueta de versión (`v1.2.3`) tamén compila un APK de publicación asinado (versionName
`1.2.3`) e engádeo á release de GitHub.

Configura no repositorio (Settings → Secrets and variables → Actions):

| Nome | Tipo | Que é |
|---|---|---|
| `ANDROID_API_BASE_URL` | variable | a API da instalación, p. ex. `https://reservas.example.gal/api` |
| `ANDROID_KEYSTORE_BASE64` | segredo | o almacén de chaves de publicación: `base64 -w0 release.jks` |
| `ANDROID_KEYSTORE_PASSWORD` | segredo | o seu contrasinal |
| `ANDROID_KEY_ALIAS` | segredo | o alias da chave |
| `ANDROID_KEY_PASSWORD` | segredo | o contrasinal da chave |

Sen os segredos sáltase o paso de publicación; o almacén de chaves nunca entra no repositorio.

## Inicio de sesión

A propia páxina de inicio de sesión de Zitadel nunha Custom Tab (código de autorización OIDC con PKCE, mediante
AppAuth), igual que fai a web. `zitadel/setup.sh` crea a app OIDC nativa
`android` (redirección `gal.tribo.librequenda:/oauth2redirect`) e publica o seu client
id como `androidClientId` en `/config.json`, que a app le ao iniciar sesión. Os tokens
gárdanse no almacenamento privado da app e renóvanse co refresh token; pechar a sesión
tamén remata a sesión en Zitadel.

Aínda non está na Play Store: o APK de publicación instálase a man (ou a través doutra tenda).

## Terceiros

A tipografía da app, DM Sans (`app/src/main/res/font/dm_sans.ttf`), está baixo a SIL Open Font
License 1.1: consulta [third-party/DM-Sans-OFL.txt](third-party/DM-Sans-OFL.txt).
