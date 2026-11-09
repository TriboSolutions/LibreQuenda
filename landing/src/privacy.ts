// /privacidade: the privacy policy of what Tribo Solutions runs itself (this site, the
// demo and the LibreQuenda app on Google Play), in every language of the landing.
// A council with its own installation is the controller there, with its own policy.
import type { Lang } from "./i18n";

export type Privacy = {
  title: string;
  description: string;
  eyebrow: string;
  updated: string;
  intro: string;
  sections: { title: string; body: string[] }[];
  contact: string;
};

const gl: Privacy = {
  title: "Política de privacidade",
  description: "Que datos trata LibreQuenda (a web, a demo e a app de Android), para que, con quen se comparten e os teus dereitos.",
  eyebrow: "Privacidade",
  updated: "Última actualización: 9 de novembro de 2026",
  intro:
    "Esta política explica que datos persoais tratamos en librequenda.app, na demo (demo.librequenda.app) e na app LibreQuenda para Android publicada en Google Play (gal.tribo.librequenda), para que, con quen se comparten e que dereitos tes, segundo o Regulamento Xeral de Protección de Datos (RXPD) e a Lei Orgánica 3/2018.",
  sections: [
    {
      title: "Quen é responsable",
      body: [
        "Tribo Solutions (tribosolutions.es), que desenvolve LibreQuenda. Podes escribirnos a contact@tribosolutions.es.",
        "LibreQuenda é software libre: cada concello pode instalalo no seu servidor e publicar a súa propia app. Nese caso o responsable é ese concello, coa súa propia política de privacidade.",
      ],
    },
    {
      title: "Esta web",
      body: [
        "librequenda.app non usa cookies, analítica nin rastrexadores. O teu navegador pídelle a GitHub o número de estrelas do proxecto, así que GitHub recibe o teu enderezo IP. O servidor garda rexistros técnicos (enderezo IP e páxina pedida) só para a seguridade e o funcionamento.",
      ],
    },
    {
      title: "Que datos trata a app",
      body: [
        "Conta: o teu nome, o teu correo e o contrasinal, xestionados polo noso servizo de inicio de sesión (Zitadel, nos nosos servidores). O contrasinal gárdase cifrado e nunca o vemos.",
        "Perfil: data de nacemento, teléfono, foto (opcional) e idioma.",
        "Uso: as túas reservas, as partidas ás que te unes, os rexistros de entrada, as ausencias, as incidencias que comunicas, os avisos, as votacións nas que participas e as túas preferencias.",
        "Localización: só cando a pides, para amosar distancias ou comprobar que estás na instalación ao rexistrar a entrada. Compróbase no momento e nunca se garda.",
        "Cámara: só para ler o código QR da porta ao rexistrar a entrada. As imaxes non se gardan nin se envían.",
        "NFC: só para ler a etiqueta da instalación ao rexistrar a entrada.",
        "Notificacións: os recordatorios das túas reservas prográmanse no propio teléfono.",
        "A app non recolle os teus contactos, ficheiros, identificadores publicitarios nin datos doutras apps, e non leva SDK de publicidade nin de analítica.",
      ],
    },
    {
      title: "Para que",
      body: [
        "Para xestionar as túas reservas e comprobar os requisitos de cada instalación (como a idade mínima), rexistrar a túa entrada, enviarche avisos e recordatorios e amosarche «Para ti» a partir das túas reservas.",
        "Os concellos ven estatísticas de uso agregadas, sen datos de persoas concretas.",
      ],
    },
    {
      title: "Base legal",
      body: [
        "Prestarche o servizo que pides (art. 6.1.b RXPD). O opcional, como a foto, os correos ou «Para ti», depende de ti e podes cambialo nas preferencias.",
      ],
    },
    {
      title: "Con quen se comparten",
      body: [
        "Non vendemos nin cedemos os teus datos e non os usamos para publicidade.",
        "O persoal do concello onde reservas ve o teu nome e as túas reservas nas súas instalacións. Nas partidas públicas, as demais persoas ven o teu nome. Ningún concello ve as reservas doutro.",
        "Provedores que tratan datos no noso nome: Amazon Web Services aloxa os servidores en España (rexión de Madrid); Google envía os correos de confirmación e os recordatorios; OpenStreetMap serve o mapa, así que recibe o teu enderezo IP e a zona que miras cando o abres. Google Play distribúe a app segundo a súa propia política.",
      ],
    },
    {
      title: "Canto tempo",
      body: [
        "Mentres teñas conta. Se a eliminas, bórranse os teus datos persoais, as reservas futuras cancélanse e as pasadas quedan sen ninguén asociado, só para as estatísticas.",
        "O rexistro do que fan as persoas administradoras gárdase ata dous anos; os rexistros técnicos do servidor, o tempo xusto para a seguridade.",
      ],
    },
    {
      title: "Eliminar a túa conta",
      body: [
        "Na app ou na web: O meu perfil → Os teus datos → Eliminar a miña conta. Tamén podes pedirnolo en contact@tribosolutions.es desde o correo da túa conta, e eliminámola nun prazo de 30 días.",
      ],
    },
    {
      title: "Os teus dereitos",
      body: [
        "Podes acceder aos teus datos, corrixilos, eliminalos, descargalos (no teu perfil, «Os teus datos»), opoñerte ao seu tratamento ou limitalo. Escríbenos a contact@tribosolutions.es para calquera deles.",
        "Se cres que non os tratamos ben, podes reclamar ante a Axencia Española de Protección de Datos (www.aepd.es).",
      ],
    },
    {
      title: "Menores",
      body: ["Para crear unha conta cómpre ter 14 anos ou máis. As persoas menores de 14 anos necesitan o consentimento dos seus pais ou titores."],
    },
    {
      title: "Seguridade",
      body: [
        "As conexións van cifradas (HTTPS), cada persoa só accede ao que lle corresponde e as persoas administradoras entran con verificación en dous pasos.",
      ],
    },
    {
      title: "Cambios",
      body: ["Se esta política cambia de forma importante, avisarémolo na app antes de que se aplique."],
    },
  ],
  contact: "Dúbidas sobre a túa privacidade",
};

const es: Privacy = {
  title: "Política de privacidad",
  description: "Qué datos trata LibreQuenda (la web, la demo y la app de Android), para qué, con quién se comparten y tus derechos.",
  eyebrow: "Privacidad",
  updated: "Última actualización: 9 de noviembre de 2026",
  intro:
    "Esta política explica qué datos personales tratamos en librequenda.app, en la demo (demo.librequenda.app) y en la app LibreQuenda para Android publicada en Google Play (gal.tribo.librequenda), para qué, con quién se comparten y qué derechos tienes, según el Reglamento General de Protección de Datos (RGPD) y la Ley Orgánica 3/2018.",
  sections: [
    {
      title: "Quién es responsable",
      body: [
        "Tribo Solutions (tribosolutions.es), que desarrolla LibreQuenda. Puedes escribirnos a contact@tribosolutions.es.",
        "LibreQuenda es software libre: cada ayuntamiento puede instalarlo en su servidor y publicar su propia app. En ese caso el responsable es ese ayuntamiento, con su propia política de privacidad.",
      ],
    },
    {
      title: "Esta web",
      body: [
        "librequenda.app no usa cookies, analítica ni rastreadores. Tu navegador le pide a GitHub el número de estrellas del proyecto, así que GitHub recibe tu dirección IP. El servidor guarda registros técnicos (dirección IP y página pedida) solo para la seguridad y el funcionamiento.",
      ],
    },
    {
      title: "Qué datos trata la app",
      body: [
        "Cuenta: tu nombre, tu correo y la contraseña, gestionados por nuestro servicio de inicio de sesión (Zitadel, en nuestros servidores). La contraseña se guarda cifrada y nunca la vemos.",
        "Perfil: fecha de nacimiento, teléfono, foto (opcional) e idioma.",
        "Uso: tus reservas, las partidas a las que te unes, los registros de entrada, las ausencias, las incidencias que comunicas, los avisos, las votaciones en las que participas y tus preferencias.",
        "Ubicación: solo cuando la pides, para mostrar distancias o comprobar que estás en la instalación al registrar la entrada. Se comprueba en el momento y nunca se guarda.",
        "Cámara: solo para leer el código QR de la puerta al registrar la entrada. Las imágenes no se guardan ni se envían.",
        "NFC: solo para leer la etiqueta de la instalación al registrar la entrada.",
        "Notificaciones: los recordatorios de tus reservas se programan en el propio teléfono.",
        "La app no recoge tus contactos, archivos, identificadores publicitarios ni datos de otras apps, y no lleva SDK de publicidad ni de analítica.",
      ],
    },
    {
      title: "Para qué",
      body: [
        "Para gestionar tus reservas y comprobar los requisitos de cada instalación (como la edad mínima), registrar tu entrada, enviarte avisos y recordatorios y mostrarte «Para ti» a partir de tus reservas.",
        "Los ayuntamientos ven estadísticas de uso agregadas, sin datos de personas concretas.",
      ],
    },
    {
      title: "Base legal",
      body: [
        "Prestarte el servicio que pides (art. 6.1.b RGPD). Lo opcional, como la foto, los correos o «Para ti», depende de ti y puedes cambiarlo en las preferencias.",
      ],
    },
    {
      title: "Con quién se comparten",
      body: [
        "No vendemos ni cedemos tus datos y no los usamos para publicidad.",
        "El personal del ayuntamiento donde reservas ve tu nombre y tus reservas en sus instalaciones. En las partidas públicas, las demás personas ven tu nombre. Ningún ayuntamiento ve las reservas de otro.",
        "Proveedores que tratan datos en nuestro nombre: Amazon Web Services aloja los servidores en España (región de Madrid); Google envía los correos de confirmación y los recordatorios; OpenStreetMap sirve el mapa, así que recibe tu dirección IP y la zona que miras cuando lo abres. Google Play distribuye la app según su propia política.",
      ],
    },
    {
      title: "Cuánto tiempo",
      body: [
        "Mientras tengas cuenta. Si la eliminas, se borran tus datos personales, las reservas futuras se cancelan y las pasadas quedan sin nadie asociado, solo para las estadísticas.",
        "El registro de lo que hacen los administradores se guarda hasta dos años; los registros técnicos del servidor, el tiempo justo para la seguridad.",
      ],
    },
    {
      title: "Eliminar tu cuenta",
      body: [
        "En la app o en la web: Mi perfil → Tus datos → Eliminar mi cuenta. También puedes pedírnoslo en contact@tribosolutions.es desde el correo de tu cuenta, y la eliminamos en un plazo de 30 días.",
      ],
    },
    {
      title: "Tus derechos",
      body: [
        "Puedes acceder a tus datos, corregirlos, eliminarlos, descargarlos (en tu perfil, «Tus datos»), oponerte a su tratamiento o limitarlo. Escríbenos a contact@tribosolutions.es para cualquiera de ellos.",
        "Si crees que no los tratamos bien, puedes reclamar ante la Agencia Española de Protección de Datos (www.aepd.es).",
      ],
    },
    {
      title: "Menores",
      body: ["Para crear una cuenta hay que tener 14 años o más. Los menores de 14 años necesitan el consentimiento de sus padres o tutores."],
    },
    {
      title: "Seguridad",
      body: [
        "Las conexiones van cifradas (HTTPS), cada persona solo accede a lo que le corresponde y los administradores entran con verificación en dos pasos.",
      ],
    },
    {
      title: "Cambios",
      body: ["Si esta política cambia de forma importante, lo avisaremos en la app antes de que se aplique."],
    },
  ],
  contact: "Dudas sobre tu privacidad",
};

const ca: Privacy = {
  title: "Política de privacitat",
  description: "Quines dades tracta LibreQuenda (el web, la demo i l'app d'Android), per a què, amb qui es comparteixen i els teus drets.",
  eyebrow: "Privacitat",
  updated: "Darrera actualització: 9 de novembre de 2026",
  intro:
    "Aquesta política explica quines dades personals tractem a librequenda.app, a la demo (demo.librequenda.app) i a l'app LibreQuenda per a Android publicada a Google Play (gal.tribo.librequenda), per a què, amb qui es comparteixen i quins drets tens, segons el Reglament General de Protecció de Dades (RGPD) i la Llei Orgànica 3/2018.",
  sections: [
    {
      title: "Qui n'és responsable",
      body: [
        "Tribo Solutions (tribosolutions.es), que desenvolupa LibreQuenda. Pots escriure'ns a contact@tribosolutions.es.",
        "LibreQuenda és programari lliure: cada ajuntament el pot instal·lar al seu servidor i publicar la seva pròpia app. En aquest cas el responsable és aquell ajuntament, amb la seva pròpia política de privacitat.",
      ],
    },
    {
      title: "Aquest web",
      body: [
        "librequenda.app no fa servir galetes, analítica ni rastrejadors. El teu navegador demana a GitHub el nombre d'estrelles del projecte, de manera que GitHub rep la teva adreça IP. El servidor guarda registres tècnics (adreça IP i pàgina demanada) només per a la seguretat i el funcionament.",
      ],
    },
    {
      title: "Quines dades tracta l'app",
      body: [
        "Compte: el teu nom, el teu correu i la contrasenya, gestionats pel nostre servei d'inici de sessió (Zitadel, als nostres servidors). La contrasenya es guarda xifrada i no la veiem mai.",
        "Perfil: data de naixement, telèfon, foto (opcional) i idioma.",
        "Ús: les teves reserves, les partides a què t'uneixes, els registres d'entrada, les absències, les incidències que comuniques, els avisos, les votacions en què participes i les teves preferències.",
        "Ubicació: només quan la demanes, per mostrar distàncies o comprovar que ets a la instal·lació en registrar l'entrada. Es comprova al moment i no es guarda mai.",
        "Càmera: només per llegir el codi QR de la porta en registrar l'entrada. Les imatges no es guarden ni s'envien.",
        "NFC: només per llegir l'etiqueta de la instal·lació en registrar l'entrada.",
        "Notificacions: els recordatoris de les teves reserves es programen al mateix telèfon.",
        "L'app no recull els teus contactes, fitxers, identificadors publicitaris ni dades d'altres apps, i no porta SDK de publicitat ni d'analítica.",
      ],
    },
    {
      title: "Per a què",
      body: [
        "Per gestionar les teves reserves i comprovar els requisits de cada instal·lació (com l'edat mínima), registrar la teva entrada, enviar-te avisos i recordatoris i mostrar-te «Per a tu» a partir de les teves reserves.",
        "Els ajuntaments veuen estadístiques d'ús agregades, sense dades de persones concretes.",
      ],
    },
    {
      title: "Base legal",
      body: [
        "Prestar-te el servei que demanes (art. 6.1.b RGPD). L'opcional, com la foto, els correus o «Per a tu», depèn de tu i ho pots canviar a les preferències.",
      ],
    },
    {
      title: "Amb qui es comparteixen",
      body: [
        "No venem ni cedim les teves dades i no les fem servir per a publicitat.",
        "El personal de l'ajuntament on reserves veu el teu nom i les teves reserves a les seves instal·lacions. A les partides públiques, les altres persones veuen el teu nom. Cap ajuntament no veu les reserves d'un altre.",
        "Proveïdors que tracten dades en nom nostre: Amazon Web Services allotja els servidors a Espanya (regió de Madrid); Google envia els correus de confirmació i els recordatoris; OpenStreetMap serveix el mapa, de manera que rep la teva adreça IP i la zona que mires quan l'obres. Google Play distribueix l'app segons la seva pròpia política.",
      ],
    },
    {
      title: "Quant de temps",
      body: [
        "Mentre tinguis compte. Si l'elimines, s'esborren les teves dades personals, les reserves futures es cancel·len i les passades queden sense ningú associat, només per a les estadístiques.",
        "El registre del que fan els administradors es guarda fins a dos anys; els registres tècnics del servidor, el temps just per a la seguretat.",
      ],
    },
    {
      title: "Eliminar el teu compte",
      body: [
        "A l'app o al web: El meu perfil → Les teves dades → Elimina el meu compte. També ens ho pots demanar a contact@tribosolutions.es des del correu del teu compte, i l'eliminem en un termini de 30 dies.",
      ],
    },
    {
      title: "Els teus drets",
      body: [
        "Pots accedir a les teves dades, corregir-les, eliminar-les, descarregar-les (al teu perfil, «Les teves dades»), oposar-te al seu tractament o limitar-lo. Escriu-nos a contact@tribosolutions.es per a qualsevol d'aquests drets.",
        "Si creus que no les tractem bé, pots reclamar davant l'Agència Espanyola de Protecció de Dades (www.aepd.es).",
      ],
    },
    {
      title: "Menors",
      body: ["Per crear un compte cal tenir 14 anys o més. Els menors de 14 anys necessiten el consentiment dels seus pares o tutors."],
    },
    {
      title: "Seguretat",
      body: [
        "Les connexions van xifrades (HTTPS), cada persona només accedeix al que li correspon i els administradors entren amb verificació en dos passos.",
      ],
    },
    {
      title: "Canvis",
      body: ["Si aquesta política canvia de manera important, ho avisarem a l'app abans que s'apliqui."],
    },
  ],
  contact: "Dubtes sobre la teva privacitat",
};

const en: Privacy = {
  title: "Privacy policy",
  description: "What data LibreQuenda (the website, the demo and the Android app) processes, why, who it is shared with and your rights.",
  eyebrow: "Privacy",
  updated: "Last updated: 9 November 2026",
  intro:
    "This policy explains what personal data we process on librequenda.app, on the demo (demo.librequenda.app) and in the LibreQuenda Android app published on Google Play (gal.tribo.librequenda), why, who it is shared with and what rights you have, under the General Data Protection Regulation (GDPR) and Spain's Organic Law 3/2018.",
  sections: [
    {
      title: "Who is responsible",
      body: [
        "Tribo Solutions (tribosolutions.es), which develops LibreQuenda. You can write to us at contact@tribosolutions.es.",
        "LibreQuenda is free software: any council can install it on its own server and publish its own app. In that case that council is the controller, with its own privacy policy.",
      ],
    },
    {
      title: "This website",
      body: [
        "librequenda.app uses no cookies, analytics or trackers. Your browser asks GitHub for the project's star count, so GitHub receives your IP address. The server keeps technical logs (IP address and page requested) only for security and operation.",
      ],
    },
    {
      title: "What data the app processes",
      body: [
        "Account: your name, your email and your password, managed by our sign-in service (Zitadel, on our own servers). The password is stored hashed and we never see it.",
        "Profile: date of birth, phone, photo (optional) and language.",
        "Usage: your bookings, the games you join, your check-ins, no-shows, the issues you report, notices, the polls you take part in and your preferences.",
        "Location: only when you ask for it, to show distances or confirm you are at the facility when you check in. It is checked on the spot and never stored.",
        "Camera: only to read the door's QR code when you check in. Images are neither stored nor sent.",
        "NFC: only to read the facility's tag when you check in.",
        "Notifications: reminders of your bookings are scheduled on the phone itself.",
        "The app does not collect your contacts, files, advertising identifiers or data from other apps, and it contains no advertising or analytics SDK.",
      ],
    },
    {
      title: "Why",
      body: [
        "To manage your bookings and check each facility's requirements (such as minimum age), record your check-in, send you notices and reminders, and show you «For you» based on your bookings.",
        "Councils see aggregated usage statistics, with no data about specific people.",
      ],
    },
    {
      title: "Legal basis",
      body: [
        "Providing the service you ask for (art. 6.1.b GDPR). Optional things, such as your photo, emails or «For you», are up to you and can be changed in your preferences.",
      ],
    },
    {
      title: "Who it is shared with",
      body: [
        "We do not sell or hand over your data and we do not use it for advertising.",
        "Staff of the council where you book see your name and your bookings at their facilities. In public games, other people see your name. No council sees another's bookings.",
        "Providers that process data on our behalf: Amazon Web Services hosts the servers in Spain (Madrid region); Google sends confirmation and reminder emails; OpenStreetMap serves the map, so it receives your IP address and the area you look at when you open it. Google Play distributes the app under its own policy.",
      ],
    },
    {
      title: "How long",
      body: [
        "For as long as you have an account. If you delete it, your personal data is erased, future bookings are cancelled and past ones are left with nobody attached, only for statistics.",
        "The log of what administrators do is kept for up to two years; the server's technical logs, only as long as security needs.",
      ],
    },
    {
      title: "Deleting your account",
      body: [
        "In the app or on the web: My profile → Your data → Delete my account. You can also ask us at contact@tribosolutions.es from your account's email address, and we delete it within 30 days.",
      ],
    },
    {
      title: "Your rights",
      body: [
        "You can access, correct, delete or download your data (in your profile, «Your data»), object to its processing or restrict it. Write to us at contact@tribosolutions.es for any of them.",
        "If you think we are not handling it properly, you can complain to the Spanish Data Protection Agency (www.aepd.es).",
      ],
    },
    {
      title: "Children",
      body: ["You must be 14 or older to create an account. Children under 14 need the consent of their parents or guardians."],
    },
    {
      title: "Security",
      body: [
        "Connections are encrypted (HTTPS), each person only reaches what is theirs and administrators sign in with two-step verification.",
      ],
    },
    {
      title: "Changes",
      body: ["If this policy changes in an important way, we will say so in the app before it applies."],
    },
  ],
  contact: "Questions about your privacy",
};

const pt: Privacy = {
  title: "Política de privacidade",
  description: "Que dados trata o LibreQuenda (o site, a demo e a app Android), para quê, com quem são partilhados e os teus direitos.",
  eyebrow: "Privacidade",
  updated: "Última atualização: 9 de novembro de 2026",
  intro:
    "Esta política explica que dados pessoais tratamos em librequenda.app, na demo (demo.librequenda.app) e na app LibreQuenda para Android publicada no Google Play (gal.tribo.librequenda), para quê, com quem são partilhados e que direitos tens, segundo o Regulamento Geral sobre a Proteção de Dados (RGPD) e a Lei Orgânica espanhola 3/2018.",
  sections: [
    {
      title: "Quem é responsável",
      body: [
        "A Tribo Solutions (tribosolutions.es), que desenvolve o LibreQuenda. Podes escrever-nos para contact@tribosolutions.es.",
        "O LibreQuenda é software livre: cada município pode instalá-lo no seu servidor e publicar a sua própria app. Nesse caso o responsável é esse município, com a sua própria política de privacidade.",
      ],
    },
    {
      title: "Este site",
      body: [
        "O librequenda.app não usa cookies, analítica nem rastreadores. O teu navegador pede ao GitHub o número de estrelas do projeto, por isso o GitHub recebe o teu endereço IP. O servidor guarda registos técnicos (endereço IP e página pedida) só para a segurança e o funcionamento.",
      ],
    },
    {
      title: "Que dados trata a app",
      body: [
        "Conta: o teu nome, o teu e-mail e a palavra-passe, geridos pelo nosso serviço de início de sessão (Zitadel, nos nossos servidores). A palavra-passe é guardada cifrada e nunca a vemos.",
        "Perfil: data de nascimento, telefone, foto (opcional) e idioma.",
        "Utilização: as tuas reservas, os jogos a que te juntas, os registos de entrada, as faltas, os problemas que comunicas, os avisos, as votações em que participas e as tuas preferências.",
        "Localização: só quando a pedes, para mostrar distâncias ou confirmar que estás na instalação ao registar a entrada. É verificada no momento e nunca é guardada.",
        "Câmara: só para ler o código QR da porta ao registar a entrada. As imagens não são guardadas nem enviadas.",
        "NFC: só para ler a etiqueta da instalação ao registar a entrada.",
        "Notificações: os lembretes das tuas reservas são agendados no próprio telemóvel.",
        "A app não recolhe os teus contactos, ficheiros, identificadores de publicidade nem dados de outras apps, e não inclui SDK de publicidade nem de analítica.",
      ],
    },
    {
      title: "Para quê",
      body: [
        "Para gerir as tuas reservas e verificar os requisitos de cada instalação (como a idade mínima), registar a tua entrada, enviar-te avisos e lembretes e mostrar-te «Para ti» a partir das tuas reservas.",
        "Os municípios veem estatísticas de utilização agregadas, sem dados de pessoas concretas.",
      ],
    },
    {
      title: "Base legal",
      body: [
        "Prestar-te o serviço que pedes (art. 6.º, n.º 1, alínea b) do RGPD). O opcional, como a foto, os e-mails ou «Para ti», depende de ti e podes alterá-lo nas preferências.",
      ],
    },
    {
      title: "Com quem são partilhados",
      body: [
        "Não vendemos nem cedemos os teus dados e não os usamos para publicidade.",
        "O pessoal do município onde reservas vê o teu nome e as tuas reservas nas suas instalações. Nos jogos públicos, as outras pessoas veem o teu nome. Nenhum município vê as reservas de outro.",
        "Fornecedores que tratam dados em nosso nome: a Amazon Web Services aloja os servidores em Espanha (região de Madrid); a Google envia os e-mails de confirmação e os lembretes; o OpenStreetMap serve o mapa, por isso recebe o teu endereço IP e a zona que vês quando o abres. O Google Play distribui a app segundo a sua própria política.",
      ],
    },
    {
      title: "Durante quanto tempo",
      body: [
        "Enquanto tiveres conta. Se a eliminares, os teus dados pessoais são apagados, as reservas futuras são canceladas e as passadas ficam sem ninguém associado, só para as estatísticas.",
        "O registo do que fazem os administradores é guardado até dois anos; os registos técnicos do servidor, o tempo estritamente necessário para a segurança.",
      ],
    },
    {
      title: "Eliminar a tua conta",
      body: [
        "Na app ou no site: O meu perfil → Os teus dados → Eliminar a minha conta. Também nos podes pedir em contact@tribosolutions.es a partir do e-mail da tua conta, e eliminamo-la num prazo de 30 dias.",
      ],
    },
    {
      title: "Os teus direitos",
      body: [
        "Podes aceder aos teus dados, corrigi-los, eliminá-los, descarregá-los (no teu perfil, «Os teus dados»), opor-te ao seu tratamento ou limitá-lo. Escreve-nos para contact@tribosolutions.es para qualquer um deles.",
        "Se achares que não os tratamos bem, podes apresentar reclamação à Agência Espanhola de Proteção de Dados (www.aepd.es).",
      ],
    },
    {
      title: "Menores",
      body: ["Para criar uma conta é preciso ter 14 anos ou mais. Os menores de 14 anos precisam do consentimento dos pais ou tutores."],
    },
    {
      title: "Segurança",
      body: [
        "As ligações são cifradas (HTTPS), cada pessoa só acede ao que lhe corresponde e os administradores entram com verificação em dois passos.",
      ],
    },
    {
      title: "Alterações",
      body: ["Se esta política mudar de forma importante, avisaremos na app antes de ela se aplicar."],
    },
  ],
  contact: "Dúvidas sobre a tua privacidade",
};

export const PRIVACY: Record<Lang, Privacy> = { gl, es, ca, en, pt };
export const PRIVACY_EMAIL = "contact@tribosolutions.es";
