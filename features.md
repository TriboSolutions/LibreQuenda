# Funcións de LibreQuenda

O que fai hoxe a aplicación, o que se está a construír e ideas para máis adiante. Para o plan e o que queda antes do certame, consulta [todo.md](todo.md).

Estado: **feito** · **en curso** (traballo aberto) · **idea** (sen comezar).

## Veciñanza

| Función | Estado |
|---|---|
| Páxina de inicio con todas as instalacións de todos os concellos: lista xunto a un mapa, busca, filtros por concello, deporte, tipo de espazo (catálogo de etiquetas) e tipo de reserva (partidas públicas, eventos), «preto de min» coa distancia | feito |
| Páxinas de concello e de instalación: fotos, descrición, enderezo e como chegar, contacto (o da instalación ou, se non, o do concello), horario, peches, normas de uso, idade mínima, a ventá de reserva que se aplica, a valoración media da veciñanza | feito |
| Vista do día coas quendas libres, reservadas, pechadas e pasadas | feito |
| Predición do tempo nas instalacións ao aire libre (datos abertos de MeteoGalicia, polo municipio do concello): o día na páxina da instalación e o período da quenda no diálogo de reserva, cun aviso en palabras para choiva, treboada, neve ou sarabia, vento forte, calor ou xeada | feito |
| Reservar unha quenda libre aceptando as normas de uso; cambiala a outra hora da mesma instalación; cancelala ata o prazo límite | feito |
| Convidar a outras persoas a encher o aforo dunha reserva cunha ligazón, con conta ou como convidadas (nome e idade); saír dunha reserva; quen reserva pode quitar xente | feito |
| Reservas públicas: facer pública unha reserva e unirse ás reservas públicas doutras persoas desde «Para ti» | feito |
| Eventos do concello: apuntarse aos eventos do concello (un título, aforo e horario propios) desde a páxina da instalación, «Para ti» ou a parte superior do inicio da app, e recibir aviso cando un se move ou se cancela | feito |
| As miñas reservas: próximas e historial, conta atrás ata cada quenda («empeza en 2 h», «xa podes entrar»), asistencia ás pasadas | feito |
| «Para ti» na páxina de inicio: a túa quenda habitual cando cae hoxe e está libre, as instalacións que máis reservas con quendas libres hoxe, outras parecidas que aínda non probaches, partidas públicas e eventos aos que unirse; pódese desactivar | feito |
| Valoracións: ao rematar unha reserva, valorar a instalación de 1 a 5 estrelas e, despois dunha reserva pública, ás outras persoas con conta; anónimas (só se amosan a media e o número), definitivas unha vez dadas; medias nas instalacións e no perfil de cada persoa | feito |
| Avisos na app (campá, actualizada cada 30 s) e correos: confirmación, recordatorio 15 minutos antes, cambio, cancelación por ti ou polo concello (co motivo e as datas do peche) | feito |
| Os correos explican como rexistrar a entrada nesa instalación (só as súas formas), con indicacións para chegar e os contactos da instalación e do concello | feito |
| Calendario (.ics) para subscribirse desde Google, Outlook ou Apple, con botóns dun só clic | feito |
| Perfil: foto encadrada nun círculo (copiada á conta de acceso), data de nacemento, teléfono; preferencias á parte (idioma dos correos, recordatorios, «Para ti») | feito |
| Conta en Zitadel: rexistro e acceso con contrasinal ou Google (chaves de acceso opcionais), cambio de nome, correo e contrasinal desde o perfil | feito |
| Correo verificado obrigatorio para reservar, unirse a unha quenda ou reservar na porta (impide que contas de bots collan quendas) | feito |
| Os teus datos nun ficheiro (JSON e CSV) e borrado da conta (reservas anonimizadas, conta de acceso borrada) | feito |
| Instalacións só para a veciñanza marcadas «Só para veciños» na súa páxina, nas súas tarxetas e nos seus eventos; quen non está verificado alí pode pedilo: chamar ao concello ou enviar unha solicitude (DNI ou NIE, nome, código postal, rúa), unha á semana como máximo, e retirala mentres está pendente; recibe aviso e correo cando o concello decide | feito |
| Avisar dun problema nunha instalación (rede rota, sen luz, vestiarios sucios…) desde unha reserva ou desde a páxina da instalación, cunha categoría, unha foto e se é urxente; recibe aviso e correo cando o concello o arranxa ou o rexeita | feito |
| Páxinas de condicións de uso e de política de privacidade; contacto de soporte no pé de páxina | feito |
| Catro idiomas (galego, castelán, inglés, catalán), tema claro e escuro | feito |

## App Android

| Función | Estado |
|---|---|
| App da veciñanza (Kotlin, Jetpack Compose) sobre a mesma API e o mesmo acceso de Zitadel ca web (app OIDC nativa): inicio con eventos, «Para ti» e as partidas públicas de hoxe; sen sesión iniciada, a lista de instalacións para consultar | feito |
| Busca de instalacións nunha lista ou nun mapa («Ver no mapa»), páxina da instalación e reserva | feito |
| Reservas e historial: cambiar a quenda dunha reserva, avisar dun problema, valorala, rexistrar a entrada de todas as formas (QR no lector, código de entrada, QR da porta, etiqueta NFC, persoal) | feito |
| Reserva flash desde a app, tamén cunha etiqueta NFC | feito |
| Avisos sen push (comprobación en segundo plano), recordatorios locais, notificacións de Android para valorar unha reserva cando remata | feito |
| Perfil, preferencias, idioma, soporte e textos legais; textos xerados a partir das traducións da web | feito |
| Instalacións só para a veciñanza marcadas «Só para veciños» (páxina da instalación, tarxetas, eventos e partidas públicas), e petición de verificación desde unha folla inferior: chamar ao concello ou enviar a solicitude, vela pendente e retirala (a xestión segue na web) | feito |

## Rexistro de entrada na porta

| Función | Estado |
|---|---|
| Lector QR na porta (`/lector`, calquera móbil ou tableta con cámara, emparellado cunha clave de dispositivo): le o QR da reserva ou o QR persoal de reserva flash | feito |
| Apagar os lectores dunha instalación sen retiralos (avariados, desconectados) | feito |
| QR impreso na porta, lido coa cámara do móbil desde a app; a localización do móbil ten que estar dentro da distancia da instalación (compróbase antes de abrir a cámara e nunca se garda) | feito |
| Código de entrada fixo por instalación, imprimible (tamén en braille, a escala para papel microcapsulado ou unha impresora braille), escrito na app comprobando a localización | feito |
| Etiquetas NFC na porta: o enderezo da etiqueta e «novo enderezo» na administración, lidas pola app Android ou abertas na páxina de entrada da web | feito |
| Confirmación do persoal desde o calendario de reservas, que se pode desfacer durante 5 minutos; persoas de contacto por instalación (a quen preguntar alí), e se non, o contacto da instalación e o do concello | feito |
| Diálogo «Rexistrar entrada» en As miñas reservas cunha lapela por cada forma que admite a instalación | feito |
| Reserva flash: reservar a quenda libre en curso ou a piques de comezar, con calquera forma de rexistrar a entrada (QR persoal nun lector, código de entrada, QR da porta, NFC, persoal); pódese desactivar por instalación | feito |

## Concellos

| Función | Estado |
|---|---|
| Instalacións: crear, editar, desactivar (cancelando, se se quere, as reservas próximas cun aviso), foto, punto no mapa (busca por enderezo), etiquetas, aforo, idade mínima, normas de uso | feito |
| Instalacións só para a veciñanza: as persoas administradoras verifican a xente como veciñanza do concello (lapela Veciños, por correo) e retiran a verificación (con confirmación); as persoas administradoras do concello, e as da superadministración en todos, contan como verificadas sen solicitude; só elas reservan e se unen alí (reservas, partidas públicas, eventos, ligazóns de convite), as persoas convidadas sen conta non | feito |
| Solicitudes de verificación: chegan á lapela Veciños (distintivo coas pendentes), á campá das persoas administradoras e ao seu correo («X quere ser veciño»); o persoal contrasta o DNI e o enderezo co seu propio padrón e despois verifica ou descarta. En calquera caso a solicitude bórrase no momento, co seu DNI e enderezo, e a que ninguén decidiu bórrase aos 15 días | feito |
| Horario semanal e peches (cancelando as reservas afectadas, cunha desculpa e o motivo) | feito |
| Formas de rexistrar a entrada por instalación (lapela Entrada) e follas imprimibles co QR e o código | feito |
| Ventá de reserva por concello, herdada polas súas instalacións ou substituída na lapela Reserva de cada unha: con canta antelación (horas ou días) ou apertura da semana seguinte a unha hora fixa (p. ex., domingo ás 20:00), canto antes do comezo se pode reservar desde a app, prazo para cancelar e cambiar, reservas próximas, reservas por día e por semana, quendas seguidas, penalización opcional por non presentarse | feito |
| Un límite para todo o concello das reservas próximas de cada persoa en todas as súas instalacións; un só concello á vez por conta (axuste da plataforma, activado por defecto); ninguén pode ter dúas reservas á mesma hora | feito |
| Eventos (lapela Eventos): un tipo de reserva que só crean as persoas administradoras, cun título, aforo opcional, calquera inicio e fin e sen límites por persoa; amosa as reservas ou peches que coinciden, que o persoal cancela antes; movelo ou cancelalo avisa a quen se apuntou | feito |
| Reserva do persoal por teléfono ou no mostrador, para unha conta ou só cun nome e un teléfono; cancelar calquera reserva; imprimir o QR de entrada | feito |
| Indicadores: reservas, ocupación, cancelacións, non presentacións (contando todas as formas de rexistrar a entrada), días e horas con máis reservas, valoracións (media, reparto de 1 a 5, evolución), comparación con outro período, por instalación; exportación a CSV e Excel | feito |
| As non presentacións lembran cando estaba desactivado o rexistro de entrada (historial das formas de rexistrar a entrada da instalación) | feito |
| Widget de quendas libres para incrustar na propia web do concello (`/<council>/widget`, todas as instalacións ou unha, claro ou escuro, en calquera dos catro idiomas), co código para pegar e unha vista previa nos axustes do concello | feito |
| Lista de mantemento (lapela Incidencias): avisos da veciñanza cun distintivo para os novos, filtros por instalación e estado, estado e nota para quen avisou, o correo de quen avisou só se o compartiu | feito |
| Páxina de axustes do concello: datos, persoas administradoras (invitar, retirar), ventá de reserva, servidor de correo propio (SMTP) con correo de proba, descarga completa dos datos (CSV + datapackage de Frictionless + imaxes, opcionalmente cos datos persoais da veciñanza) | feito |

## Plataforma

| Función | Estado |
|---|---|
| Superadministración: crear, editar, desactivar e borrar concellos, invitar as súas persoas administradoras, xestionar as persoas da superadministración | feito |
| Catálogo de etiquetas por categorías, compartido por todos os concellos | feito |
| Nome e logotipo propios da instalación do servizo (autoaloxamento), tamén nos correos e nas páxinas e correos de acceso de Zitadel | feito |
| Autoaloxamento con Docker Compose: PostgreSQL con seguridade a nivel de fila por concello, Zitadel para as contas, Garage para os ficheiros, Caddy diante | feito |
| Almacenamento privado: un bucket por concello, imaxes só mediante ligazóns presinadas | feito |
| Accesibilidade (WCAG 2.1 AA e os criterios AAA que se aplican), con comprobacións automáticas na CI e unha páxina de declaración de accesibilidade | feito |
| CI: comprobacións de Go, web e Android (filtradas por rutas nos pull requests), análise de vulnerabilidades e de imaxes (govulncheck, pnpm audit, Trivy), imaxes publicadas en ghcr.io, APK de versión asinado nas etiquetas de versión | feito |
| Tokens de acceso de curta duración (10 min, renovados en segundo plano), segundo factor obrigatorio para as persoas administradoras, arranque de Zitadel co mínimo privilexio | feito |
| Límite de peticións no rexistro e no acceso, no proxy (sen captcha: o login de Zitadel non ten) | feito |
| Rexistro de actividade de todos os cambios administrativos, para a superadministración (todos) e para as persoas administradoras de cada concello (os seus), só de engadir e cun período de conservación, descargable en CSV | feito |
| API: ordenación, filtrado e paxinación no backend, con parámetros de consulta validados | feito |
| Despregamento de demostración, kit de produción (`deploy/`), especificación OpenAPI | idea (ver todo.md) |

## Ideas para máis adiante

Agrupadas segundo quen máis gaña; ningunha está comezada.

**Veciñanza**

- **Lista de espera:** poñerse na cola dunha quenda completa e recibir un aviso (e uns minutos para collela) cando alguén cancela.
- **Atopar xogadores:** indicar que buscas xente para un deporte e un nivel, e ver quen máis busca, a partir das reservas públicas.
- **Notificacións push** (web push, e push real na app Android en vez da comprobación en segundo plano) e **SMS** para quen non ten correo, a través do provedor do concello.
- **Aplicación web instalable que funcione sen conexión:** o manifesto xa existe; un service worker deixaría dispoñibles As miñas reservas e o QR de entrada sen cobertura na porta.

**Concellos**

- **Reservas periódicas para clubs e centros de ensino:** as franxas dunha tempada reservadas dunha vez polo persoal, cos seus propios límites, e ventás de prioridade para entidades rexistradas.
- **Taxas e pagamento:** un prezo por quenda ou un bono, pagado a través da Pasarela de Pagamentos das administracións públicas, con exencións.
- **Límites por deporte entre instalacións:** p. ex., como máximo dúas reservas próximas de pádel en todas as pistas do concello (hoxe o límite de todo o concello conta todas as instalacións).
- **Pantalla quiosco na entrada:** as quendas de hoxe e quen está dentro, na tableta do lector.
- **Datos abertos:** publicar a ocupación anonimizada como datos abertos (DCAT-AP, datos.gob.es / abertos.xunta.gal) de forma periódica.

**Plataforma**

- Acceso con **Cl@ve / Chave365** para a veciñanza (por confirmar con AMTEGA/AGASOL).
