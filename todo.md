# Tarefas pendentes

Prazo: **18 nov 2026 23:59** (certame@agasol.gal). Presentación final: 26 nov 2026.
Alcance: bases §1.1 (mínimo) + §1.2 (técnico).

## Veciñanza (bases §1.1 — Cidadanía)

- [x] Catálogo de instalacións e dispoñibilidade por concello
- [x] Reservar unha quenda libre
- [x] Cancelar a propia reserva
- [x] Reserva flash, como en Enjoy: o QR persoal (menú da conta → «Reserva flash») reserva unha quenda libre no lector, desde agora ata que remate a quenda en curso ou a que está a piques de comezar
- [x] Cambiar a propia reserva (`PATCH /me/bookings/{id}`): outro día ou outra hora da mesma instalación; quen reserva pode quitar as persoas que se uniron
- [x] Historial de reservas (pasadas e próximas)
- [x] Ventá de reserva: o concello fixa unha por defecto que herdan todas as instalacións, e cada instalación pode ter a súa: con canta antelación (horas ou días), canto antes do comezo se pode reservar desde a app, prazo para cancelar/cambiar, reservas próximas, reservas por persoa e día, quendas seguidas e unha penalización opcional por non presentarse. O persoal que reserva para alguén sáltase os límites por persoa
- [x] Lista de avisos na app (`GET /me/notifications`)
- [x] Decidir se os avisos na app precisan push en tempo real (WebSockets, SSE) ou se abonda con consultas periódicas: consulta cada 30 s
- [x] Avisos por correo: reserva confirmada / recordatorio (15 min antes) / cancelada; só a quen reserva
- [x] Avisar as persoas afectadas cando o concello cancela reservas (peche, ou desactivación cando o concello decide cancelalas)

## Concellos (bases §1.1 — Concello)

- [x] Crear / editar / desactivar instalacións
- [x] Horario de apertura
- [x] Peches (cancelan as reservas afectadas, aviso e correo de desculpas co motivo)
- [x] Normas de uso editables por instalación
- [x] Panel de indicadores: reservas por instalación, ocupación, taxa de cancelación, días/horas con máis reservas, persoas usuarias activas
- [x] Exportación dos indicadores a CSV
- [x] Exportación completa dos datos do concello (`…/admin/export`): ZIP de CSV + `datapackage.json` de Frictionless + imaxes, para a portabilidade dos datos; `?personal=1` engade os datos persoais da veciñanza
- [x] Xestión das persoas membro do concello (`…/admin/members`)
- [x] Axustes SMTP por concello (se non os ten, usa o relay `SMTP_*`; sen ningún dos dous, os correos do concello quedan á espera)

## Privacidade (RXPD / RGPD)

- [x] Exportación dos datos para a persoa usuaria (`GET /me/export`, JSON/CSV)
- [x] Borrado da conta: anonimizar as reservas, borrar a identidade en Zitadel

## Accesibilidade (WCAG 2.1 AA, máis os criterios AAA que se aplican)

- [x] Revisión da navegación con teclado e do foco (selector de quendas, diálogos)
- [x] Comprobacións de axe na CI (`pnpm test:a11y`, web/e2e)
- [x] Páxina de declaración de accesibilidade (`/accesibilidade`, modelo do RD 1112/2018)

## Interoperabilidade e documentación

- [ ] Decidir como xerar a especificación OpenAPI e despois escribila en `docs/openapi/`
- [x] Traducións completas en gl / es / en / ca: as mesmas claves nos catro, web e API (comprobado o 30 out)
- [x] Revisión nativa da redacción en es / en / ca (30 out): en inglés «slot» en vez do calco «turn», en castelán «ayuntamiento» en todas partes, en catalán botóns en imperativo, apóstrofos tipográficos; as cadeas engadidas despois teñen a mesma revisión

## Despregamento e seguridade

- [x] CI en `.github/workflows/ci.yml`: Go (gofmt, diferenzas de sqlc, vet, probas), web (lint, tipos, build), as cinco imaxes (publicadas en ghcr.io en main e nas etiquetas)
- [x] Análise de dependencias e de imaxes na CI (govulncheck, pnpm audit, Trivy)
- [ ] Kit de autoaloxamento en produción `deploy/`
- [x] Reforzo do arranque de Zitadel: volumes separados (o token de administración á parte do do cliente de login), unha caducidade para `admin.pat`, revogación opcional do token de configuración unha vez configurado — arquitectura §9.1 (#35). O usuario de configuración segue sendo IAM_OWNER: as políticas da instancia precísano
- [x] Tokens de acceso curtos: 10 min por defecto, renovados en segundo plano co refresh token (30 días sen uso, 90 en total) (#28)
- [x] Segundo factor para a administración: o token de acceso de Zitadel non leva `amr`, así que a API o le por introspección do token; as persoas administradoras do concello e da plataforma precísano cando `REQUIRE_ADMIN_MFA` (activado no compose de produción) (#32)
- [x] Un correo verificado antes de reservar, unirse a unha quenda ou reservar na porta (#40)
- [x] Límite de peticións no proxy para o rexistro (5 por hora e IP) e para os envíos de acceso (30 cada 10 min) (#39)
- [ ] Captcha no rexistro, se o límite de peticións non abonda (precisa un fork da app de login de Zitadel)
- [ ] Cl@ve / Chave365: confirmar a vía de integración con AMTEGA / AGASOL

## Rexistro de actividade

- [x] Rexistro de auditoría dos cambios administrativos (superadministración: todos; persoas administradoras do concello: os do seu concello), só de engadir, conservación segundo `AUDIT_RETENTION_DAYS`, pseudonimizado ao borrar a conta, descargable en CSV
- [ ] Engadir unha liña en `audit/actions.go` por cada nova ruta administrativa (mentres tanto rexístranse como "other")

## Presentación ao certame

- [ ] Memoria do proxecto para as bases
- [ ] Despregamento de demostración con datos de proba (os datos de proba teñen tres concellos reais: Lalín, Lugo e Ourense)
- [ ] Presentación para o 26 nov

## Rexistro de entrada e reservas (feito do 28 ao 30 out, seguimento)

- [x] Formas de rexistrar a entrada por instalación: lector QR (pódese apagar sen retiralo), QR impreso na porta e código de entrada fixo (os dous precisan o móbil dentro da distancia da instalación), confirmación do persoal con persoas de contacto (e se non, o contacto da instalación e despois o do concello)
- [x] «Rexistrar entrada» desde As miñas reservas (conta atrás, unha lapela por forma, cámara para o QR da porta) e reserva flash con calquera forma
- [x] Os correos explican só as formas de rexistrar a entrada da instalación, con indicacións para chegar e contactos
- [x] Os indicadores de non presentacións contan todas as formas de rexistrar a entrada, non só os lectores QR
- [x] Ventá de reserva por concello, herdada polas súas instalacións ou substituída na lapela «Reserva» de cada unha; penalización opcional por non presentarse
- [x] Recomendacións «Para ti» na páxina de inicio (pódense desactivar nas Preferencias do perfil)
- [x] Reservas públicas: quen reserva fai pública unha reserva (ao reservar, ou despois desde As miñas reservas) nas instalacións que admiten convites; «Para ti» amosa as reservas públicas con prazas libres nas instalacións da persoa, en instalacións parecidas ou a menos de 10 km cando «preto de min» está activado, e únese a elas alí (coas mesmas normas que a ligazón de convite)
- [ ] Reservas públicas: quen reserva non recibe aviso cando alguén se une (tampouco hoxe desde unha ligazón de convite)
- [x] Foto de perfil copiada á conta de Zitadel da persoa; Zitadel permite rexistrarse e acceder con contrasinal (chaves de acceso con `ZITADEL_PASSKEYS=true`)
- [x] A veciñanza avisa de problemas nunha instalación (desde unha reserva ou desde a súa páxina) na lista de mantemento do concello, e recibe aviso cando se arranxa ou se rexeita
- [ ] Etiqueta NFC: aínda é unha vista previa na lapela Entrada (precisa un enderezo público de rexistro de entrada por etiqueta e, para as etiquetas asinadas, verificación NTAG 424 DNA)
- [x] Indicadores: un historial de cando cada instalación admitía rexistros de entrada (`facility_checkin_period`), para que as quendas co lector QR apagado, ou con todas as formas desactivadas, non conten como non presentacións (#31)
- [x] Foto de perfil: comprobada de principio a fin o 30 out, a foto da app chega á conta de Zitadel. As propias páxinas de acceso de Zitadel (login v2, v4.20) amosan sempre as iniciais, nunca a foto; a súa consola si a amosa
- [x] Foto de perfil: encadrada nun círculo (mover e ampliar) antes de subila, e reencadrable despois (#27)
- [ ] Foto de perfil: borrar unha foto que só existe en Zitadel (ningunha na app) non a borra alí
- [ ] Probar os fluxos de cámara e localización nun móbil real con HTTPS (QR da porta, código de entrada, lector)
- [x] Ventá de reserva: reservas por persoa e semana, e apertura de todas as quendas da semana seguinte nun día e hora fixos en vez da ventá móbil
- [x] Límites de reserva entre instalacións: un límite por persoa para todo o concello, e un só concello á vez por conta (axuste da plataforma)
- [ ] Ventá de reserva, máis adiante se se quere: un límite por deporte en todas as instalacións do concello
- [ ] «Para ti» como o feed completo con sesión iniciada da proposta de docs/specs/feed.md (busca na súa propia páxina, convites, horas de silencio, instalacións novas)

## Tempo (30 out)

- [x] Predición de MeteoGalicia cando se activa nos axustes dunha instalación (independente das etiquetas), na páxina da instalación e no diálogo de reserva, con avisos de mal tempo; o código INE de cada concello nos seus datos
- [ ] Liña do tempo no diálogo de cambio de reserva, en As miñas reservas e no correo de recordatorio
- [ ] Predición por horas para cada punto (MeteoSIX, precisa unha clave de API) en vez de tres períodos por municipio

## Opcional (bases §1.3, só se sae barato)

- [x] Calendario `.ics` coas reservas da persoa usuaria (subscrición desde /perfil)
- [x] Código QR en cada reserva (en «Rexistrar entrada» e nos correos de confirmación e de recordatorio, onde a instalación ten lector)
- [ ] App móbil (`app/`)
