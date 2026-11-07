# Manter un servidor de LibreQuenda

Como funciona unha instalación e como xestionala. Para instalar, consulta o [README](../README.md) (`install.sh` pide todo o que sexa obrigatorio do que segue).

## Sen o instalador

No servidor só fan falla dous ficheiros: `docker-compose.yaml` e `.env`. As imaxes constrúeas a CI e descárganse de `ghcr.io/tribosolutions/librequenda-*` (`api`, `db`, `proxy` coa app web dentro, `storage`, `zitadel-setup`, `backup`); alí non se constrúe nada.

```sh
mkdir librequenda && cd librequenda
curl -fLO https://raw.githubusercontent.com/TriboSolutions/LibreQuenda/main/docker-compose.yaml
curl -fL -o .env https://raw.githubusercontent.com/TriboSolutions/LibreQuenda/main/.env.example
# fill in every value: domains, SMTP, the first admin, and the secrets (each line says how to generate it)
docker compose up -d --wait
```

`LIBREQUENDA_VERSION` en `.env` escolle a etiqueta das imaxes: `latest` segue `main`; unha etiqueta de versión (p. ex. `1.2.3`) queda nesa versión. Para actualizar: `docker compose pull && docker compose up -d --wait`. Para executar os teus propios cambios, constrúe as imaxes desde este repositorio cos mesmos nomes (p. ex. `docker build -t ghcr.io/tribosolutions/librequenda-api:latest api`).

## Como funciona

A pila négase a arrincar mentres falte un valor obrigatorio. Todo pasa por un único proxy Caddy (`proxy/`), que obtén e renova os certificados TLS:

| Que | URL |
|---|---|
| App web | `https://<WEB_DOMAIN>` |
| API principal | `https://<WEB_DOMAIN>/api/…` (p. ex. `/api/health/db`) |
| Inicio de sesión (Zitadel) | `https://<ZITADEL_DOMAIN>` |
| Descubrimento OIDC | `https://<ZITADEL_DOMAIN>/.well-known/openid-configuration` |

PostgreSQL non se publica no host: usa `docker compose exec db psql -U librequenda`.

Ademais do usuario propio da aplicación, a base de datos ten un de só lectura, `librequenda_analytics` (contrasinal `ANALYTICS_DB_PASSWORD`), que só pode ler as vistas de analítica: o asistente de datos dos concellos executa as súas consultas con el unha vez que unha persoa da superadministración configura un modelo de linguaxe nos axustes da plataforma. Consulta [analytics.md](analytics.md).

A primeira persoa administradora de Zitadel (`ZITADEL_ADMIN_*`), os axustes de correo de Zitadel e a `ZITADEL_MASTERKEY` só se aplican no primeiro arranque; a masterkey non se pode cambiar nunca despois, así que fai unha copia de seguranza dela. Os datos están en volumes de Docker (`postgres_data`, `garage_data`, …).

**LibreQuenda arrinca baleira.** A persoa da superadministración (`SUPERADMIN_EMAIL`, por defecto a persoa administradora de Zitadel) inicia sesión e abre `/admin` para crear concellos, convidar por correo electrónico á persoa administradora de cada concello e xestionar outras persoas da superadministración. A persoa convidada recibe un correo ao momento; o rol concédese cando inicia sesión con ese correo verificado (o inicio de sesión pide o código enviado por correo), cun aviso na campá.

**O teu propio nome e logotipo:** en `/admin`, a lapela «Personalizar a aplicación» define o nome e o logotipo da aplicación para toda a instalación: úsanos a barra superior, os títulos das lapelas e a icona do navegador, as pantallas do lector QR, o feed do calendario e os correos. Déixaos baleiros para volver aos de LibreQuenda. O pé de páxina segue nomeando LibreQuenda como o software libre sobre o que funciona (AGPL-3.0). As pantallas de inicio de sesión son de Zitadel e manteñen a imaxe de `zitadel/setup.sh`.

**Administración do concello:** as persoas administradoras dun concello (e a superadministración) abren `/<council>/admin` para crear instalacións con foto, descrición, enderezo, punto no mapa, normas de reserva, horario semanal e peches. Cada instalación ten a súa propia páxina en `/<council>/<facility>`.

**A veciñanza** nunca ve as pantallas propias de Zitadel máis alá do inicio de sesión: o nome, o correo e o contrasinal cámbianse desde `/perfil`. `zitadel/setup.sh` dálle a Zitadel a imaxe de LibreQuenda.

**Correo:** LibreQuenda envía as confirmacións de reserva, un recordatorio 15 minutos antes co QR de entrada, as cancelacións (co motivo cando o concello pecha unha instalación) e os convites para persoas administradoras. Zitadel envía os códigos de verificación e os restablecementos de contrasinal polo mesmo servidor de correo.

**As imaxes** gárdanse en **Garage** (autoaloxado, compatible con S3; servizo `storage`). Os buckets son privados, un por concello; a base de datos só garda referencias como `<council id>/facilities/<id>.webp`, e a API convérteas en ligazóns `/storage/…` presinadas (válidas durante unha semana) ao responder. **Os mapas** usan OpenStreetMap (Leaflet); «Como chegar» é unha ligazón simple ás indicacións de Google Maps. A busca de enderezos pasa pola API cara a OpenStreetMap Nominatim (ou `GEOCODER_URL`).

**Iniciar sesión con Google (opcional):** crea un ID de cliente OAuth (tipo *Web application*) en Google Cloud co URI de redirección autorizado `https://<ZITADEL_DOMAIN>/idps/callback`, despois define `GOOGLE_CLIENT_ID` e `GOOGLE_CLIENT_SECRET` en `.env` e executa `docker compose up -d`. Déixaos baleiros para quitar outra vez o botón de Google.

**Contrasinais, non chaves de acceso:** o rexistro e o inicio de sesión van directamente a un contrasinal. Define `ZITADEL_PASSKEYS=true` en `.env` e executa `docker compose up -d` para ofrecer tamén chaves de acceso (passkeys).

**Etiquetas das instalacións:** metadatos descritivos para clasificar e filtrar (incluído atopar instalacións parecidas). As etiquetas nunca deben activar funcións nin cambiar as normas de reserva, de acceso ou outras normas de funcionamento; iso corresponde a axustes explícitos da instalación.

**Tempo:** cada instalación ten un interruptor «Amosar previsión meteorolóxica» no seu formulario de edición (desactivado nas instalacións novas, independente das etiquetas). Cando está activado, amosa a predición de MeteoGalicia para o municipio do seu concello, indicado co seu código INE nos datos do concello (p. ex. 36024 para Lalín). A API obtena (sen clave, `servizos.meteogalicia.gal/mgrss/predicion/jsonPredConcellos.action`), gárdaa en caché durante unha hora e non amosa nada se non se pode acceder a ela; `WEATHER_URL` diríxea a outro sitio. Datos © MeteoGalicia, citada onde se amosan.

**Sesións curtas:** os tokens de acceso duran 10 minutos (`ZITADEL_ACCESS_TOKEN_MINUTES`), así que unha conta bloqueada ou un rol retirado deixa de funcionar nese prazo. A web renóvaos sen que se note co seu token de renovación, así que a xente segue coa sesión iniciada mentres a usa; despois de 30 días sen actividade (`ZITADEL_REFRESH_IDLE_DAYS`), ou de 90 días en total (`ZITADEL_REFRESH_DAYS`), volven iniciar sesión. Executa `docker compose up -d` despois de cambialos.

**Segundo factor para as persoas administradoras:** as persoas administradoras dos concellos e a superadministración só entran na administración despois de iniciar sesión cun segundo factor (unha app de autenticación, unha chave de seguranza ou unha chave de acceso), configurado na súa conta de inicio de sesión. Os tokens de acceso de Zitadel non indican como iniciou sesión a persoa, así que a API pregúntallo a Zitadel (introspección de tokens, cunha aplicación de API que crea `zitadel-setup`) unha vez por token. Activado por defecto (`REQUIRE_ADMIN_MFA=true` en `docker-compose.yaml`); a pila de desenvolvemento local déixao desactivado. Non afecta á veciñanza. Antes de activalo nunha instalación xa existente, pídelles primeiro ás persoas administradoras que configuren un segundo factor.

**Límites de rexistro e de inicio de sesión:** o inicio de sesión de Zitadel non ten captcha, así que o proxy limita os envíos dos seus formularios por enderezo IP: 5 rexistros de conta por hora (`SIGNUP_LIMIT_PER_HOUR`) e 30 intentos de inicio de sesión, contrasinal, código ou verificación de correo cada 10 minutos (`SIGNIN_LIMIT_PER_10_MIN`). Ao superar un límite, a persoa recibe «429 Too Many Requests» e pode volver tentalo un pouco despois. Detrás doutro proxy, os límites contan o enderezo de quen visita só se ese proxy está nunha rede privada (`trusted_proxies`).

**Token de administración de Zitadel:** no seu primeiro arranque, Zitadel crea a conta de servizo `librequenda-setup` e escribe o seu token (`admin.pat`) no volume `zitadel-admin`, que só monta `zitadel-setup` (en só lectura); o token propio da app de inicio de sesión vai a `zitadel-login-client`, e a API só ve `api-secrets`. A conta é propietaria da instancia, o rol mínimo que pode definir as políticas de inicio de sesión, de tokens e de imaxe da instancia. Para non deixar ningún token así, define `ZITADEL_SETUP_REVOKE_TOKEN=true`: cando a configuración remata ben, revoga o token, e os arranques posteriores manteñen a configuración tal como está (para cambiala outra vez, crea un novo token para `librequenda-setup` na consola de Zitadel e ponno nese volume). `ZITADEL_ADMIN_PAT_EXPIRATION` define a caducidade do token no primeiro arranque.

*Actualizar unha instalación que aínda ten o volume único `zitadel-bootstrap`:* copia unha vez os dous tokens nos volumes novos e despois arrinca como sempre (substitúe `librequenda` polo nome do teu proxecto de compose):

```sh
docker run --rm -v librequenda_zitadel-bootstrap:/old:ro -v librequenda_zitadel-admin:/admin -v librequenda_zitadel-login-client:/login \
  alpine sh -c 'cp /old/admin.pat /admin/ && cp /old/login-client.pat /login/'
docker compose up -d
docker volume rm librequenda_zitadel-bootstrap   # once everything works
```

Un `docker-compose.dev.yaml` local segue funcionando tal como está; para reproducir produción, monta `zitadel-admin` en `/zitadel/bootstrap` en `zitadel-setup` e `zitadel-login-client` en `/zitadel/login-client` en `zitadel-login` (con `ZITADEL_SERVICE_USER_TOKEN_FILE` e as rutas `ZITADEL_FIRSTINSTANCE_*PATPATH` como en `docker-compose.yaml`).

**TLS detrás doutro proxy:** se un servidor web ou un balanceador de carga situado diante xa termina o TLS, define `CADDY_AUTO_HTTPS=false` e mantén `PUBLIC_SCHEME=https`; entón Caddy sérvelle HTTP simple no porto `PROXY_HTTP_PORT` (p. ex. `127.0.0.1:8080`, só accesible desde o propio servidor). O proxy ten que enviarlle os dous dominios conservando a cabeceira `Host` e con `X-Forwarded-Proto: https`. Se o elixes en `install.sh`, deixa escrito un `nginx-librequenda.conf` xunto a `.env`, listo para activar e para que `certbot --nginx` lle engada o HTTPS.

**Copias de seguranza:** o servizo `backup` fai un volcado de PostgreSQL todos os días ás 03:30 (`BACKUP_AT`, Europe/Madrid): a base de datos de LibreQuenda e a de Zitadel (as contas ás que pertencen as reservas), cada unha con `pg_dump` e relida antes de gardala, ademais dos roles. Van ao bucket `backups` do Garage deste servidor (`BACKUP_BUCKET`), un cartafol por copia, e bórranse aos 14 días (`BACKUP_KEEP_DAYS`). O servizo pasa a estado unhealthy en `docker compose ps` despois dunha copia fallida, ata que a seguinte funcione; `docker compose logs backup` indica por que. Unha copia no mesmo servidor non sobrevive á perda do seu disco: apunta `BACKUP_S3_URL`, `BACKUP_S3_REGION`, `BACKUP_S3_ACCESS_KEY` e `BACKUP_S3_SECRET_KEY` a un almacenamento compatible con S3 noutro lugar (outro Garage, un provedor) para gardalas fóra del. As imaxes subidas quedan en Garage e non están nestas copias. Os datos de Zitadel están cifrados con `ZITADEL_MASTERKEY`: garda unha copia de `.env` á parte, porque sen el non se pode restaurar unha copia.

```sh
docker compose exec backup backup                 # one now
docker compose stop api zitadel-api zitadel-login # nothing may use the databases while restoring
docker compose exec backup restore                # lists the backups
docker compose exec backup restore 20261105T023000Z
docker compose start api zitadel-api zitadel-login
```

**Imaxes desde a CI:** cada push a `main` e cada etiqueta de versión constrúe as imaxes e publícaas en `ghcr.io/<owner>/librequenda-<image>` (`.github/workflows/ci.yml`).

## A túa propia app Android

`./install.sh app` constrúe a app Android de LibreQuenda co teu nome, logotipo e paquete (applicationId), apuntando ao teu servidor, en Docker (`app/builder/`): un APK para instalar directamente e un AAB para Google Play. Garda as túas respostas, o logotipo e a clave de sinatura en `android/`, xunto a `.env`.

- **A clave de sinatura** (`android/release.jks` e `android/keystore.env`) asina cada actualización: os teléfonos e Google Play rexeitan unha actualización asinada con outra clave. Fai unha copia de seguranza dela xunto con `.env`.
- **O inicio de sesión** volve á app a través do seu paquete (`<applicationId>:/oauth2redirect`), así que Zitadel ten que coñecelo: o instalador engádeo a `ANDROID_APP_IDS` en `.env` e reinicia a pila, e `zitadel-setup` rexístrao. O servidor lista os aceptados en `/config.json` (`androidAppIds`). Se se constrúe noutra máquina, engádeo a man ao `.env` do servidor e executa `docker compose up -d --wait`. Con `ZITADEL_SETUP_REVOKE_TOKEN=true` a configuración xa non pode cambiar Zitadel: dálle primeiro un novo token (consulta «Token de administración de Zitadel»).
- **Actualizacións:** despois de actualizar LibreQuenda, executa outra vez `./install.sh app`: o mesmo paquete e a mesma clave, cun código de versión máis alto.
