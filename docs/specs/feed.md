# Seguinte: un inicio persoal (o feed)

*Proposta, aínda sen facer.* Hoxe `/` é igual para todo o mundo: todas as instalacións nunha lista xunto a un mapa, con filtros. É a páxina axeitada para quen visita, pero unha veciña que reserva a mesma pista todos os martes non debería ter que buscala cada semana. A idea:

- **Sen sesión iniciada:** `/` queda como está (lista, mapa e filtros).
- **Con sesión iniciada:** `/` pasa a ser un feed feito a partir do propio uso desa persoa. A busca (lista e mapa) pasa a ter o seu propio enderezo, por exemplo `/buscar`, a un toque do feed e da barra superior.

**O que amosaría o feed, o máis urxente primeiro** (cada tarxeta coa súa acción alí mesmo):

1. **Agora e a seguir:** a reserva de hoxe ou a seguinte, co QR de entrada, como chegar, «convidar» cando quedan prazas, e cancelar mentres aínda se poida.
2. **O que as agarda:** convites para unirse a unha reserva, e peches ou cancelacións que afectan ás súas reservas (o que xa sabe a campá).
3. **Volve reservar o de sempre:** a instalación, o día da semana e a hora que máis reservou nas últimas semanas («Pádel, os martes ás 19:00»), cando está libre esta semana, como reserva cun toque. Indica por que o amosa.
4. **As súas instalacións hoxe:** as instalacións que máis reserva (ou que marca como favoritas), coas quendas libres de hoxe.
5. **Libre en breve preto:** instalacións con quendas libres nas próximas horas que comparten unha etiqueta de deporte co que reserva habitualmente, no seu concello ou preto da súa localización se o permite.
6. **Horas tranquilas:** as horas nas que a súa instalación habitual adoita estar tranquila. A lóxica existía como as «Suxestións» da páxina da instalación (eliminadas; consulta `api/core/tenants/suggestions.go` no historial de git) e pódese reutilizar aquí.
7. **Novidades para ela:** instalacións ou deportes engadidos recentemente no seu concello.

**Quen aínda non ten historial** recibe no seu lugar un comezo breve: escoller o seu concello e os deportes que practica (do catálogo de etiquetas). Iso gárdase como as súas preferencias e enche as tarxetas 4 e 5 ata que teña reservas.

**Como se podería facer:**

- `GET /me/feed` responde cunha lista de tarxetas con tipo (`next_booking`, `invitation`, `usual_turn`, `favourite_today`, `free_soon`, `quiet_hours`, `new_facility`), cada unha cos seus datos e o motivo polo que está alí. A web debuxa un compoñente por cada tipo de tarxeta. Pódense engadir novos tipos de tarxeta sen cambiar os outros.
- Regras simples ordenadas por urxencia, non aprendizaxe automática: fáciles de explicar a unha veciña ou a un concello («porque reservas pádel os martes») e fáciles de probar.
- Datos novos: as preferencias da persoa (concello, etiquetas de deporte) e instalacións favoritas opcionais, por exemplo as táboas `user_preference` e `user_favourite_facility`. Todo o demais xa existe (reservas, convites, avisos, etiquetas, horarios).

**Privacidade:** o feed só usa o que LibreQuenda xa garda para a propia reserva (as reservas da persoa e as preferencias que ela escolle). Non segue as páxinas vistas nin os clics, non comparte nada entre a veciñanza nin co concello, e a localización só se usa no navegador cando o permite, nunca se garda. `/perfil` explicaría que usa o feed e permitiría restablecer as preferencias.

**Preguntas abertas:** que tarxetas lles importan máis aos concellos para a demostración do GOBTEC, se as favoritas son explícitas (unha estrela) ou se deducen das reservas, e se o feed debería chegar tamén por correo (un semanal «a túa quenda de sempre está libre»).
