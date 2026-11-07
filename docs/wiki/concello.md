# 🏛️ Concello

O que fan as persoas administradoras dun concello desde a web (`/<concello>/admin`): as súas instalacións, como se reserva e se entra nelas, a veciñanza verificada, os eventos e os indicadores. Cando algo chega á veciñanza, vai tamén a captura da app.

[← Volver á wiki](README.md) · [Veciñanza](vecinanza.md) · [Superadministración](superadmin.md)

**Índice:** [Alta e mantemento de instalacións](#alta-e-mantemento-de-instalacións) · [Horarios e peches](#horarios-e-peches) · [Ventá de reserva](#ventá-de-reserva) · [Formas de entrada](#formas-de-entrada) · [Reserva manual](#reserva-manual) · [Veciñanza empadroada](#veciñanza-empadroada) · [Eventos do concello](#eventos-do-concello) · [Incidencias](#incidencias) · [Participación](#participación) · [Analítica de uso](#analítica-de-uso) · [Fala cos datos](#fala-cos-datos) · [Widget para a web do concello](#widget-para-a-web-do-concello) · [Administración do concello](#administración-do-concello)

---

## Alta e mantemento de instalacións

Creación e edición de cada instalación con nome, descrición, foto, enderezo e punto no mapa, etiquetas de deporte e tipo de espazo, aforo, idade mínima, accesibilidade, contacto, duración das quendas e predición do tempo. Desactivación cancelando, se se quere, as reservas futuras con aviso.

![Lista de instalacións do concello](../screenshots/web/concello/facilities/01-list.webp)

**Editar unha instalación.** Os datos, a foto e o punto no mapa; as etiquetas cun buscador; a predición de MeteoGalicia cun interruptor.

![Editor dunha instalación](../screenshots/web/concello/facilities/02-editor.webp)

| Etiquetas de deporte | Predición do tempo |
|:-:|:-:|
| ![Etiquetas](../screenshots/web/concello/facilities/04-sport-tags.webp) | ![Predición](../screenshots/web/concello/facilities/05-weather-forecast.webp) |

**Unha instalación nova**, enteira: datos, foto, mapa, normas, idade mínima e horario.

| Formulario cuberto | Creada |
|:-:|:-:|
| <img src="../screenshots/web/concello/facilities/08-new-facility-filled-full.webp" width="420"> | <img src="../screenshots/web/concello/facilities/09-facility-created.webp" width="420"> |

**Desactivar.** As reservas próximas consérvanse, salvo que se marque cancelalas: entón cada persoa recibe un aviso e un correo co motivo.

| Desactivar? | Cancelar tamén as reservas, co motivo |
|:-:|:-:|
| ![Diálogo](../screenshots/web/concello/facilities/10-deactivate-dialog.webp) | ![Cancelar as reservas](../screenshots/web/concello/facilities/11-deactivate-cancel-bookings.webp) |

| Desactivada: 34 reservas canceladas e avisadas | Na lista |
|:-:|:-:|
| ![Desactivada](../screenshots/web/concello/facilities/12-deactivated.webp) | ![Lista](../screenshots/web/concello/facilities/13-list-inactive.webp) |

## Horarios e peches

Horario semanal con varios tramos por día. Peches por festivos ou mantemento que cancelan as reservas afectadas e avisan cunha desculpa e o motivo.

| Horario de mañá e tarde | Escoller o día do peche |
|:-:|:-:|
| <img src="../screenshots/web/concello/hours-closures/02-two-windows-a-day.webp" width="420"> | <img src="../screenshots/web/concello/hours-closures/03-closure-pick-day.webp" width="420"> |

| O peche, co motivo | Confirmar: cancela 2 reservas | Gardado |
|:-:|:-:|:-:|
| ![Formulario](../screenshots/web/concello/hours-closures/04-closure-form.webp) | ![Confirmar](../screenshots/web/concello/hours-closures/05-closure-confirm.webp) | ![Gardado](../screenshots/web/concello/hours-closures/06-closure-added.webp) |

**O que recibe quen tiña reserva:** un aviso na app e un correo de desculpas co motivo.

| Aviso na app | Correo de desculpas |
|:-:|:-:|
| <img src="../screenshots/web/concello/hours-closures/07-resident-app-notice.webp" width="230"> | <img src="../screenshots/web/concello/hours-closures/08-resident-apology-email.webp" width="420"> |

## Ventá de reserva

As regras de reserva valen para todo o concello e cada instalación pode herdalas ou ter as súas: canto antes e ata canto antes se reserva, ata cando se cancela, reservas activas por persoa, por día e por semana, quendas seguidas e penalización por non presentarse.

| Ventá do concello | Unha instalación coa súa propia |
|:-:|:-:|
| ![Ventá do concello](../screenshots/web/concello/booking-rules/01-council-booking-window.webp) | ![Ventá propia](../screenshots/web/concello/booking-rules/03-facility-own-window.webp) |

## Formas de entrada

Na lapela Entrada escóllense as formas de rexistrar a entrada de cada instalación: lector QR, QR impreso, código, etiqueta NFC e confirmación do persoal, con persoas de contacto. O QR e o código imprímense nunha folla, o código tamén en braille.

| Lapela Entrada | Folla co QR | Folla co código, en braille |
|:-:|:-:|:-:|
| <img src="../screenshots/web/concello/checkin-setup/02-entrance-tab-full.webp" width="380"> | <img src="../screenshots/web/concello/checkin-setup/03-printable-sheet.webp" width="260"> | <img src="../screenshots/web/concello/checkin-setup/04-printable-code-braille.webp" width="260"> |

## Reserva manual

Para quen chama ou vén en persoa: o persoal reserva unha quenda libre para alguén con conta ou só cun nome e un teléfono (bórranse aos 30 días), imprime o QR de entrada, marca a entrada e cancela calquera reserva.

![Calendario da reserva manual](../screenshots/web/concello/staff-booking/01-calendar.webp)

| Reservar por teléfono | O QR para imprimir | Unha reserva |
|:-:|:-:|:-:|
| ![Reservar](../screenshots/web/concello/staff-booking/03-book-by-phone.webp) | ![QR](../screenshots/web/concello/staff-booking/04-booked-with-qr.webp) | ![Accións](../screenshots/web/concello/staff-booking/05-booking-actions.webp) |

## Veciñanza empadroada

Instalacións «Só para veciños», con solicitude de verificación desde a app que o concello contrasta co padrón; os datos bórranse ao decidir ou aos 15 días.

| A petición pendente, co DNI e o enderezo | Verificada: a petición xa se borrou |
|:-:|:-:|
| ![Petición](../screenshots/web/concello/residents/01-residents-tab.webp) | ![Verificada](../screenshots/web/concello/residents/04-verified.webp) |

| A veciña recibe o aviso | …e xa pode reservar |
|:-:|:-:|
| <img src="../screenshots/web/concello/residents/06-resident-app-verified-notice.webp" width="230"> | <img src="../screenshots/web/concello/residents/05-resident-app-can-book.webp" width="230"> |

## Eventos do concello

Actividades cun título, aforo opcional e calquera horario, ás que se apunta a veciñanza. A plataforma avisa das reservas ou peches que coinciden e informa as persoas apuntadas se o evento cambia.

![Eventos do concello](../screenshots/web/concello/events/06-event-created.webp)

| Un novo evento | Coincide cun peche | Coincide cunha reserva |
|:-:|:-:|:-:|
| ![Novo](../screenshots/web/concello/events/05-event-filled.webp) | ![Peche](../screenshots/web/concello/events/03-closure-in-the-way.webp) | ![Reserva](../screenshots/web/concello/events/04-bookings-in-the-way.webp) |

| Cambiar a hora | Quen se apuntou recibe o aviso |
|:-:|:-:|
| <img src="../screenshots/web/concello/events/07-edit-event.webp" width="480"> | <img src="../screenshots/web/concello/events/09-resident-app-event-changed.webp" width="230"> |

## Incidencias

A lista de mantemento: os problemas que avisou a veciñanza, coa foto e a urxencia. Ao marcalos como arranxados ou rexeitados, quen avisou recibe un aviso e un correo coa nota ([ver o lado da veciñanza](vecinanza.md#participación)).

| Lista de mantemento | Marcar como arranxado, cunha nota |
|:-:|:-:|
| ![Lista](../screenshots/web/concello/issues/01-issues-list.webp) | ![Arranxado](../screenshots/web/concello/issues/03-mark-solved.webp) |

## Participación

A lapela **Participación** é a canle propia do concello coa veciñanza, en dúas partes, cada unha co que está en marcha e o seu **historial**, para que a lista teña só o actual.

**Comunicados.** Un título e un texto, sobre unha instalación ou todo o concello, visibles ata o día que se escolla. Con «Avisar tamén», chegan á campá, á app e ao correo de quen reservou alí nos últimos 30 días (ou en todo o concello en 90 días, e á veciñanza verificada); a lista di a cantas persoas. Pódense retirar mentres están vixentes; os caducados pasan ao historial.

| Vixentes | Novo comunicado | Historial |
|:-:|:-:|:-:|
| ![Comunicados](../screenshots/web/concello/participation/01-notices.webp) | ![Novo](../screenshots/web/concello/participation/02-new-notice.webp) | ![Historial](../screenshots/web/concello/participation/03-notices-history.webp) |

**Enquisas.** Unha pregunta con 2 a 6 respostas, aberta ata o día que se escolla (como moito 90 días), opcionalmente só para a veciñanza verificada. Ao creala pregúntaselle á xente do concello, na campá e por correo; ao pechar (no seu día ou co botón «Pechar xa»), avísaselle de que xa hai resultados. A administración ve os votos de cada resposta en todo momento; a veciñanza, só ao pechar. Unha enquisa aberta pódese borrar (un erro ao creala, por exemplo); unha pechada non: queda no historial cos seus resultados. Cada voto é anónimo: só se gardan as contas.

| Abertas | Nova enquisa | Historial |
|:-:|:-:|:-:|
| ![Enquisas](../screenshots/web/concello/participation/04-polls.webp) | ![Nova](../screenshots/web/concello/participation/05-new-poll.webp) | ![Historial](../screenshots/web/concello/participation/06-polls-history.webp) |

Todo queda no rexistro de actividade (sen o texto dos comunicados nin as preguntas) e na exportación de datos do concello. O lado da veciñanza está en [O concello informa e pregunta](vecinanza.md#o-concello-informa-e-pregunta).

## Analítica de uso

Os indicadores van máis alá do básico: comparación con outro período, reparto e evolución das valoracións, cancelacións segundo quen as fixo e non presentacións contadas só cando a instalación admitía algunha forma de entrada.

<p align="center"><img src="../screenshots/web/concello/analytics/02-overview-full.webp" width="620" alt="Indicadores do concello"></p>

| Resumo, fronte ao período anterior | Fronte ao mesmo período hai un ano |
|:-:|:-:|
| ![Período anterior](../screenshots/web/concello/analytics/05-kpis-vs-previous.webp) | ![Hai un ano](../screenshots/web/concello/analytics/04-kpis-vs-last-year.webp) |

| Valoracións: reparto e evolución | Reservas por día |
|:-:|:-:|
| ![Valoracións](../screenshots/web/concello/analytics/09-ratings.webp) | ![Reservas por día](../screenshots/web/concello/analytics/06-bookings-chart.webp) |

| Ocupación por día da semana | Días e horas con máis reservas |
|:-:|:-:|
| ![Días da semana](../screenshots/web/concello/analytics/07-weekdays.webp) | ![Horas](../screenshots/web/concello/analytics/08-busy-hours.webp) |

| Por instalación | Detalle dunha instalación |
|:-:|:-:|
| <img src="../screenshots/web/concello/analytics/10-by-facility.webp" width="420"> | <img src="../screenshots/web/concello/analytics/12-facility-detail-full.webp" width="420"> |

## Fala cos datos

Un asistente de IA que pregunta aos datos do concello en linguaxe natural, só de lectura e sen datos persoais, e responde con táboas ou gráficas.

| A burbulla, desde calquera páxina | Unha pregunta e a súa resposta |
|:-:|:-:|
| ![Burbulla](../screenshots/web/concello/data-assistant/01-bubble.webp) | ![Resposta](../screenshots/web/concello/data-assistant/03-answer.webp) |

## Widget para a web do concello

As quendas libres do concello, para pegar na súa propia web: todas as instalacións ou unha, claro ou escuro, en calquera dos catro idiomas.

| O código e a vista previa | O widget incrustado |
|:-:|:-:|
| ![Axustes do widget](../screenshots/web/concello/widget/01-widget-settings.webp) | ![Widget](../screenshots/web/concello/widget/02-widget-embedded.webp) |

## Administración do concello

Invitación e retirada de administradores, servidor de correo propio, descarga completa dos datos e rexistro de actividade de todos os cambios.

<p align="center"><img src="../screenshots/web/concello/council-admin/02-settings-full.webp" width="560" alt="Axustes do concello"></p>

| Invitar a outra persoa administradora | Retirar a invitación |
|:-:|:-:|
| ![Invitada](../screenshots/web/concello/council-admin/03-admin-invited.webp) | ![Retirada](../screenshots/web/concello/council-admin/09-invitation-withdrawn.webp) |

| Servidor de correo propio | Descarga completa dos datos |
|:-:|:-:|
| ![SMTP](../screenshots/web/concello/council-admin/04-own-smtp.webp) | ![Exportar](../screenshots/web/concello/council-admin/05-full-export.webp) |

**Rexistro de actividade:** quen cambiou que e cando, nunca datos persoais da veciñanza; descargable en CSV.

![Rexistro de actividade do concello](../screenshots/web/concello/council-admin/06-activity-log.webp)
