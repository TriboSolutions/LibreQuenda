# Rexistrar a entrada na instalación

Os concellos precisan saber quen se presentou de verdade: así quedan libres as quendas de quen non vén, aliméntanse os indicadores de asistencia e a instalación queda para as persoas que a reservaron.

**O que fai hoxe LibreQuenda.** A instalación ten un lector na porta (`/lector`): calquera teléfono, tableta ou ordenador pequeno con cámara, vinculado á instalación cunha clave de dispositivo desde `/<council>/admin`. A persoa amosa un destes dous códigos QR:

- **QR da reserva**: cada reserva ten o seu (na app, nos correos de confirmación e de recordatorio, e o persoal pódeo imprimir para alguén sen conta). O lector deixa entrar a persoa desde 15 minutos antes do comezo ata o final.
- **QR de reserva flash**: o QR persoal do menú da conta. Cambia cada minuto e só vale unha vez, así que unha captura de pantalla non serve. Se a quenda en curso (ou a que está a piques de empezar) está libre, ao lelo resérvaa no momento, cos límites e as normas de sempre.

O persoal tamén pode reservar desde o mostrador («Reserva manual»), imprimirlle á persoa o QR de entrada e cancelar.

A persoa tamén pode rexistrar a entrada desde o seu propio teléfono, lendo ela a instalación en vez de ser lida. Ambas as formas actívanse por instalación na lapela **Entrada** do editor da instalación, e as dúas precisan o punto da instalación no mapa:

- **QR impreso na porta**: a lapela imprime unha folla co QR da instalación. Abre `/<council>/<facility>/entrada` cun token aleatorio, así que o enderezo por si só non funciona. «Novo QR» cambia o token e as follas xa impresas deixan de funcionar.
- **Código de entrada**: a lapela amosa o código fixo de 6 díxitos da instalación e imprímeo para a porta. Sae dun HMAC dun segredo propio de cada instalación, así que «Novo código» dá un novo e as copias impresas deixan de funcionar. A persoa escríbeo na mesma páxina `/entrada`; a súa reserva ten que estar aberta para entrar (desde 15 minutos antes do comezo ata que remata), e o endpoint ten o límite de peticións estrito, así que adiviñar códigos é lento.

Nos dous casos é obrigatoria a localización do teléfono: dentro da distancia da instalación (100 m salvo que o concello a cambie, entre 25 e 1000 m) ao seu punto, cunha precisión de 150 m ou mellor. O navegador pídella á persoa; a API comproba a localización e descártaa, nunca se garda. Alguén decidido pode falsear unha localización, así que estas dúas son as opcións baratas, non as estritas. Despois, o rexistro de entrada é o mesmo que no lector: entra a reserva da persoa para esta quenda. Se non a ten, a páxina ofrece **Reservar e entrar**, unha reserva flash da quenda libre coas normas de sempre. O concello pode desactivar a reserva flash para a instalación na mesma lapela, o que tamén a desactiva para o QR persoal nos lectores.

**A confirmación do persoal** é a terceira forma que activa a lapela, e non precisa punto no mapa. No calendario da «Reserva manual», o persoal abre unha reserva e preme **Marcar entrada**, desde 15 minutos antes do comezo ata que remata a quenda, coma nun lector. Tamén funciona coas reservas a nome dunha persoa, e a reserva amosa quen a marcou. Unha entrada marcada polo persoal pódese desfacer durante 5 minutos, por se se escolleu a reserva equivocada; as entradas desde un lector ou un teléfono non.

Cada entrada garda como se fixo (`reader`, `printed_qr`, `entry_code` ou `staff`), e a exportación de datos do concello inclúeo.

**Outras formas en que podería funcionar (aínda sen facer).** As opcións de abaixo permiten que un concello escolla segundo o orzamento e o que lle importen as trampas. Todas rematarían no mesmo rexistro de entrada na reserva, así que os indicadores non cambian.

| Forma | Na porta | Como rexistra a entrada a persoa | Custo | Difícil de facer trampa? |
| --- | --- | --- | --- | --- |
| QR da reserva ou flash (feito) | Un dispositivo con cámara que executa `/lector` | Amosa o QR desde o seu teléfono ou impreso | Un teléfono ou unha tableta baratos | Si: os tokens son por reserva ou cambian cada minuto |
| QR impreso (feito) | Unha folla de papel co QR da instalación | Léo co seu teléfono; a app rexistra a entrada da súa reserva nesa instalación e quenda | Case de balde | Baixo: unha foto da folla funciona en calquera parte, así que sempre se comproba a localización do teléfono |
| Código de entrada (feito) | Unha folla na porta ou no mostrador co código fixo da instalación | Escribe o código no seu teléfono | Case de balde | Baixo por si só: o código non cambia ata que o concello xera un novo; sempre se comproba a localización do teléfono |
| Etiqueta NFC | Un adhesivo na porta | Achega o teléfono, que abre a páxina de rexistro de entrada | Uns poucos euros por porta | Baixo cunha etiqueta simple (pódese copiar). Alto cunha etiqueta que asina cada lectura (p. ex. mensaxes «SUN» de NTAG 424 DNA), que comproba a API |
| Confirmación do persoal (feito) | Ninguén, ou o mostrador | O persoal marca a súa entrada desde a reserva no calendario da «Reserva manual» | Tempo do persoal | Tan bo coma o persoal |

O que precisarían as que aínda non están feitas:

- **Etiqueta NFC:** unha etiqueta simple xa pode gardar hoxe o enderezo `/entrada` do QR impreso. Unha asinada precisa a clave da etiqueta, gardada por porta, e unha comprobación do contador de lecturas para que non se poida reproducir unha lectura gravada.
